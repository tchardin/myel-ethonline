package rtmkt

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"
	datatransfer "github.com/filecoin-project/go-data-transfer"
	versioning "github.com/filecoin-project/go-ds-versioning/pkg"
	"github.com/filecoin-project/go-multistore"
	"github.com/filecoin-project/go-statemachine/fsm"
	"github.com/hannahhoward/go-pubsub"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
)

// DealDecider is a function that makes a decision about whether to accept a deal
type DealDecider func(ctx context.Context, state ProviderDealState) (bool, string, error)

// ProviderSubscriber is a callback that is registered to listen for retrieval events on a provider
type ProviderSubscriber func(event ProviderEvent, state ProviderDealState)

// Unsubscribe is a function that unsubscribes a subscriber for either the
// client or the provider
type Unsubscribe func()

type RetrievalProvider interface {
	// Start begins listening for deals on the given host
	Start(ctx context.Context) error

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

type Provider struct {
	network      RetrievalMarketNetwork
	dataTransfer datatransfer.Manager
	multiStore   *multistore.MultiStore
	node         RetrievalNode

	// TODO integrate properly with multistore
	store *ipfsStore

	requestValidator *ProviderRequestValidator
	revalidator      *ProviderRevalidator

	minerAddress address.Address
	subscribers  *pubsub.PubSub

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

func NewProvider(
	minerAddress address.Address,
	node RetrievalNode,
	network RetrievalMarketNetwork,
	multiStore *multistore.MultiStore,
	store *ipfsStore,
	dataTransfer datatransfer.Manager,
	ds datastore.Batching,
) (RetrievalProvider, error) {
	p := &Provider{
		store:        store,
		multiStore:   multiStore,
		dataTransfer: dataTransfer,
		node:         node,
		network:      network,
		minerAddress: minerAddress,
		subscribers:  pubsub.New(providerDispatcher),
	}

	askStore, err := NewAskStore(namespace.Wrap(ds, datastore.NewKey("retrieval-ask")), datastore.NewKey("latest"))
	if err != nil {
		return nil, err
	}
	p.askStore = askStore
	p.stateMachines, err = fsm.New(namespace.Wrap(ds, datastore.NewKey("provider-"+string(versioning.VersionKey("1")))), fsm.Parameters{
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
	p.requestValidator = NewProviderRequestValidator(&providerValidationEnvironment{p})

	p.revalidator = NewProviderRevalidator(&providerRevalidatorEnvironment{p})

	err = p.dataTransfer.RegisterVoucherType(&DealProposal{}, p.requestValidator)
	if err != nil {
		return nil, err
	}
	err = p.dataTransfer.RegisterRevalidator(&DealPayment{}, p.revalidator)
	if err != nil {
		return nil, err
	}
	err = p.dataTransfer.RegisterVoucherResultType(&DealResponse{})
	if err != nil {
		return nil, err
	}
	// transportConfigurer := TransportConfigurer(network.ID(), &providerStoreGetter{p})
	// err = p.dataTransfer.RegisterTransportConfigurer(&DealProposal{}, transportConfigurer)
	// if err != nil {
	// 	return nil, err
	// }
	dataTransfer.SubscribeToEvents(ProviderDataTransferSubscriber(p.stateMachines))

	return p, nil
}

// Stop stops handling incoming requests.
func (p *Provider) Stop() error {
	return p.network.StopHandlingRequests()
}

// Start begins listening for deals on the given host.
// Start must be called in order to accept incoming deals.
func (p *Provider) Start(ctx context.Context) error {
	return p.network.SetDelegate(p)
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
		fmt.Printf("Error setting retrieval ask: %v", err)
	}
}

// ListDeals lists all known retrieval deals
func (p *Provider) ListDeals() map[ProviderDealIdentifier]ProviderDealState {
	var deals []ProviderDealState
	_ = p.stateMachines.List(&deals)
	dealMap := make(map[ProviderDealIdentifier]ProviderDealState)
	for _, deal := range deals {
		dealMap[ProviderDealIdentifier{Receiver: deal.Receiver, DealID: deal.ID}] = deal
	}
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
		PaymentAddress:             p.minerAddress,
		MinPricePerByte:            ask.PricePerByte,
		MaxPaymentInterval:         ask.PaymentInterval,
		MaxPaymentIntervalIncrease: ask.PaymentIntervalIncrease,
	}
	size, err := p.store.GetSize(query.PayloadCID)
	if err == nil && size > 0 {
		answer.Status = QueryResponseAvailable
		answer.Size = uint64(size)
	}

	if err := stream.WriteQueryResponse(answer); err != nil {
		fmt.Printf("Retrieval query: WriteCborRPC: %s", err)
		return
	}
}
