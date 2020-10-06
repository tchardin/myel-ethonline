package main

import (
	"context"
	"fmt"
	"github.com/filecoin-project/go-address"
	datatransfer "github.com/filecoin-project/go-data-transfer"
	versioning "github.com/filecoin-project/go-ds-versioning/pkg"
	"github.com/filecoin-project/go-fil-markets/retrievalmarket"
	"github.com/filecoin-project/go-fil-markets/shared"
	"github.com/filecoin-project/go-multistore"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-statemachine/fsm"
	lapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/specs-actors/actors/builtin/paych"
	"github.com/hannahhoward/go-pubsub"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
)

// RetrievalProvider is an interface by which a provider configures their
// retrieval operations and monitors deals received and process
type RetrievalProvider interface {
	// Start begins listening for deals on the given host
	Start(ctx context.Context) error

	// OnReady registers a listener for when the provider comes on line
	OnReady(shared.ReadyFunc)

	// Stop stops handling incoming requests
	Stop() error

	// SetAsk sets the retrieval payment parameters that this miner will accept
	SetAsk(ask *Ask)

	// GetAsk returns the retrieval providers pricing information
	GetAsk() *Ask

	// SubscribeToEvents listens for events that happen related to client retrievals
	SubscribeToEvents(subscriber ProviderSubscriber) Unsubscribe

	ListDeals() map[ProviderDealIdentifier]ProviderDealState
}

// RetrievalProviderNode are the node depedencies for a RetrevalProvider
// It is based on lotus RetrievalProviderNode but changed not to rely on miner infra
type RetrievalProviderNode interface {
	GetChainHead(ctx context.Context) (shared.TipSetToken, abi.ChainEpoch, error)
	SavePaymentVoucher(ctx context.Context, paymentChannel address.Address, voucher *paych.SignedVoucher, proof []byte, expectedAmount abi.TokenAmount, tok shared.TipSetToken) (abi.TokenAmount, error)
}

type retrievalProviderNode struct {
	full lapi.FullNode
}
// Get the current chain head. Return its TipSetToken and its abi.ChainEpoch.
func (rpn *retrievalProviderNode) GetChainHead(ctx context.Context) (shared.TipSetToken, abi.ChainEpoch, error) {
	head, err := rpn.full.ChainHead(ctx)
	if err != nil {
		return nil, 0, err
	}

	return head.Key().Bytes(), head.Height(), nil
}
// Get the miner worker address for the given miner owner, as of tok
func (rpn *retrievalProviderNode) SavePaymentVoucher(ctx context.Context, paymentChannel address.Address, voucher *paych.SignedVoucher, proof []byte, expectedAmount abi.TokenAmount, tok shared.TipSetToken) (abi.TokenAmount, error) {
	// TODO: respect the provided TipSetToken (a serialized TipSetKey) when
	// querying the chain
	added, err := rpn.full.PaychVoucherAdd(ctx, paymentChannel, voucher, proof, expectedAmount)
	return added, err
}

func NewRetrievalProviderNode(full lapi.FullNode) RetrievalProviderNode {
	return &retrievalProviderNode{full}
}

// RetrievalProviderOption is a function that configures a retrieval provider
type RetrievalProviderOption func(p *Provider)

// DealDecider is a function that makes a decision about whether to accept a deal
type DealDecider func(ctx context.Context, state retrievalmarket.ProviderDealState) (bool, string, error)

// ProviderSubscriber is a callback that is registered to listen for retrieval events on a provider
type ProviderSubscriber func(event ProviderEvent, state ProviderDealState)

// Unsubscribe is a function that unsubscribes a subscriber for either the
// client or the provider
type Unsubscribe func()

type Provider struct {
	multiStore       *multistore.MultiStore
	dataTransfer     datatransfer.Manager
	node             RetrievalProviderNode
	network          RetrievalMarketNetwork
	requestValidator *ProviderRequestValidator
	revalidator      *ProviderRevalidator
	// minerAddress     address.Address
	readySub      *pubsub.PubSub
	subscribers   *pubsub.PubSub
	stateMachines fsm.Group
	dealDecider   DealDecider
	askStore      AskStore
}
type internalProviderEvent struct {
	evt   ProviderEvent
	state ProviderDealState
}

func providerDispatcher(evt pubsub.Event, subscriberFn pubsub.SubscriberFn) error {
	ie, ok := evt.(internalProviderEvent)
	if !ok {
		return fmt.Errorf("wrong type of event")
	}
	cb, ok := subscriberFn.(ProviderSubscriber)
	if !ok {
		return fmt.Errorf("wrong type of event")
	}
	cb(ie.evt, ie.state)
	return nil
}

func NewProvider(node RetrievalProviderNode,
	network RetrievalMarketNetwork,
	multiStore *multistore.MultiStore,
	dataTransfer datatransfer.Manager,
	ds datastore.Batching,
	opts ...RetrievalProviderOption,
) (RetrievalProvider, error) {
	p := &Provider{
		multiStore:   multiStore,
		dataTransfer: dataTransfer,
		node:         node,
		network:      network,
		// minerAddress: minerAddress,
		subscribers: pubsub.New(providerDispatcher),
		readySub:    pubsub.New(shared.ReadyDispatcher),
	}

	askStore, err := NewAskStore(namespace.Wrap(ds, datastore.NewKey("retrieval-ask")), datastore.NewKey("latest"))
	if err != nil {
		return nil, err
	}
	p.askStore = askStore
	p.stateMachines, err = fsm.New(namespace.Wrap(ds, datastore.NewKey(string(versioning.VersionKey("1")))), fsm.Parameters{
		Environment:     &providerDealEnvironment{p},
		StateType:       ProviderDealState{},
		StateKeyField:   "Status",
		Events:          FsmProviderEvents,
		StateEntryFuncs: ProviderStateEntryFuncs,
		FinalityStates:  ProviderFinalityStates,
		Notifier:        p.notifySubscribers,
	})
	if err != nil {
		return nil, fmt.Errorf("Unable to create state machine: %v", err)
	}
	p.Configure(opts...)
	p.requestValidator = NewProviderRequestValidator(&providerValidationEnvironment{p})
	p.revalidator = NewProviderRevalidator(&providerRevalidatorEnvironment{p})

	err = p.dataTransfer.RegisterVoucherType(&retrievalmarket.DealProposal{}, p.requestValidator)
	if err != nil {
		return nil, err
	}
	err = p.dataTransfer.RegisterRevalidator(&retrievalmarket.DealPayment{}, p.revalidator)
	if err != nil {
		return nil, err
	}
	err = p.dataTransfer.RegisterVoucherResultType(&retrievalmarket.DealResponse{})
	if err != nil {
		return nil, err
	}
	transportConfigurer := TransportConfigurer(network.ID(), &providerStoreGetter{p})
	err = p.dataTransfer.RegisterTransportConfigurer(&retrievalmarket.DealProposal{}, transportConfigurer)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// Configure reconfigures a provider after initialization
func (p *Provider) Configure(opts ...RetrievalProviderOption) {
	for _, opt := range opts {
		opt(p)
	}
}

// Stop stops handling incoming requests.
func (p *Provider) Stop() error {
	return p.network.StopHandlingRequests()
}

// Start begins listening for deals on the given host.
// Start must be called in order to accept incoming deals.
func (p *Provider) Start(ctx context.Context) error {
	go func() {
		err := p.readySub.Publish(nil)
		if err != nil {
			fmt.Printf("Publish retrieval provider ready event: %s", err.Error())
		}
	}()
	return p.network.SetDelegate(p)
}

// OnReady registers a listener for when the provider has finished starting up
func (p *Provider) OnReady(ready shared.ReadyFunc) {
	p.readySub.Subscribe(ready)
}

func (p *Provider) notifySubscribers(eventName fsm.EventName, state fsm.StateType) {
	evt := eventName.(ProviderEvent)
	ds := state.(ProviderDealState)
	_ = p.subscribers.Publish(internalProviderEvent{evt, ds})
}

// SubscribeToEvents listens for events that happen related to client retrievals
func (p *Provider) SubscribeToEvents(subscriber ProviderSubscriber) Unsubscribe {
	return Unsubscribe(p.subscribers.Subscribe(subscriber))
}

// GetAsk returns the current deal parameters this provider accepts
func (p *Provider) GetAsk() *Ask {
	return p.askStore.GetAsk()
}

// SetAsk sets the deal parameters this provider accepts
func (p *Provider) SetAsk(ask *Ask) {
	err := p.askStore.SetAsk(ask)

	if err != nil {
		fmt.Printf("Error setting retrieval ask: %w", err)
	}
}

// ListDeals lists all known retrieval deals
func (p *Provider) ListDeals() map[ProviderDealIdentifier]ProviderDealState {
	// TODO
	dealMap := make(map[ProviderDealIdentifier]ProviderDealState)
	return dealMap
}

/*
HandleQueryStream is called by the network implementation whenever a new message is received on the query protocol
A Provider handling a retrieval `Query` does the following:
1. Get the node's chain head in order to get its miner worker address.
2. Look in its piece store to determine if it can serve the given payload CID.
3. Combine these results with its existing parameters for retrieval deals to construct a `retrievalmarket.QueryResponse` struct.
4. Writes this response to the `Query` stream.
The connection is kept open only as long as the query-response exchange.
*/
func (p *Provider) HandleQueryStream(stream RetrievalQueryStream) {
	defer stream.Close()
	query, err := stream.ReadQuery()
	if err != nil {
		return
	}

	ask := p.GetAsk()

	answer := QueryResponse{
		Status:                     QueryResponseUnavailable,
		MinPricePerByte:            ask.PricePerByte,
		MaxPaymentInterval:         ask.PaymentInterval,
		MaxPaymentIntervalIncrease: ask.PaymentIntervalIncrease,
	}
	// TODO: check if cid is available in our store
	cid := query.PayloadCID
	itemAvailable, err := checkCID(cid)
	if err == nil && itemAvailable {
		answer.Status = QueryResponseAvailable
		// TODO answer.Size = uint64(pieceInfo.Deals[0].Length)
	}

	if err := stream.WriteQueryResponse(answer); err != nil {
		fmt.Printf("Retrieval query: WriteCborRPC: %s", err)
		return
	}
}

// TODO
func checkCID(payloadCID cid.Cid) (bool, error) {
	return true, nil
}