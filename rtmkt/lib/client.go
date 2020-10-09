package rtmkt

import (
	"context"
	"fmt"

	"github.com/hannahhoward/go-pubsub"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"

	"github.com/filecoin-project/go-address"
	datatransfer "github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-multistore"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-statemachine/fsm"
	"github.com/filecoin-project/go-storedcounter"
)

// ClientSubscriber is a callback that is registered to listen for retrieval events
type ClientSubscriber func(event ClientEvent, state ClientDealState)

// RetrievalClient is a client interface for making retrieval deals
type RetrievalClient interface {
	Start(ctx context.Context) error
	// Find Providers finds retrieval providers who may be storing a given piece
	FindProviders(payloadCID cid.Cid) []RetrievalPeer

	// Query asks a provider for information about a piece it is storing
	Query(
		ctx context.Context,
		p RetrievalPeer,
		payloadCID cid.Cid,
		params QueryParams,
	) (QueryResponse, error)

	// Retrieve retrieves all or part of a piece with the given retrieval parameters
	Retrieve(
		ctx context.Context,
		payloadCID cid.Cid,
		params Params,
		totalFunds abi.TokenAmount,
		p RetrievalPeer,
		clientWallet address.Address,
		minerWallet address.Address,
		storeID *multistore.StoreID,
	) (DealID, error)

	// SubscribeToEvents listens for events that happen related to client retrievals
	SubscribeToEvents(subscriber ClientSubscriber) Unsubscribe

	// TryRestartInsufficientFunds attempts to restart any deals stuck in the insufficient funds state
	// after funds are added to a given payment channel
	TryRestartInsufficientFunds(paymentChannel address.Address) error

	// CancelDeal attempts to cancel an inprogress deal
	CancelDeal(id DealID) error

	// GetDeal returns a given deal by deal ID, if it exists
	GetDeal(dealID DealID) (ClientDealState, error)

	// ListDeals returns all deals
	ListDeals() (map[DealID]ClientDealState, error)

	// Debugging
	NewStoreID() *multistore.StoreID
}

// Client is the production implementation of the RetrievalClient interface
type Client struct {
	network       RetrievalMarketNetwork
	dataTransfer  datatransfer.Manager
	multiStore    *multistore.MultiStore
	node          RetrievalNode
	storedCounter *storedcounter.StoredCounter

	subscribers   *pubsub.PubSub
	resolver      PeerResolver
	stateMachines fsm.Group
}

type internalClientEvent struct {
	evt   ClientEvent
	state ClientDealState
}

func clientDispatcher(evt pubsub.Event, subscriberFn pubsub.SubscriberFn) error {
	ie, ok := evt.(internalClientEvent)
	if !ok {
		return fmt.Errorf("wrong type of event")
	}
	cb, ok := subscriberFn.(ClientSubscriber)
	if !ok {
		return fmt.Errorf("wrong type of event")
	}
	cb(ie.evt, ie.state)
	return nil
}

func NewClient(
	node RetrievalNode,
	network RetrievalMarketNetwork,
	multiStore *multistore.MultiStore,
	dataTransfer datatransfer.Manager,
	ds datastore.Batching,
	resolver PeerResolver,
	storedCounter *storedcounter.StoredCounter,
) (RetrievalClient, error) {
	c := &Client{
		network:       network,
		multiStore:    multiStore,
		dataTransfer:  dataTransfer,
		node:          node,
		resolver:      resolver,
		storedCounter: storedCounter,
		subscribers:   pubsub.New(clientDispatcher),
	}
	sm, err := fsm.New(namespace.Wrap(ds, datastore.NewKey("client-1")), fsm.Parameters{
		Environment:     &clientDealEnvironment{c},
		StateType:       ClientDealState{},
		StateKeyField:   "Status",
		Events:          FsmClientEvents,
		StateEntryFuncs: ClientStateEntryFuncs,
		FinalityStates:  ClientFinalityStates,
		Notifier:        c.notifySubscribers,
	})
	if err != nil {
		return nil, fmt.Errorf("Unable to initialize state machine: %v", err)
	}
	c.stateMachines = sm
	err = dataTransfer.RegisterVoucherResultType(&DealResponse{})
	if err != nil {
		return nil, err
	}
	err = dataTransfer.RegisterVoucherType(&DealProposal{}, nil)
	if err != nil {
		return nil, err
	}
	err = dataTransfer.RegisterVoucherType(&DealPayment{}, nil)
	if err != nil {
		return nil, err
	}

	dataTransfer.SubscribeToEvents(ClientDataTransferSubscriber(c.stateMachines))

	// transportConfigurer := TransportConfigurer(network.ID(), &clientStoreGetter{c})
	// err = dataTransfer.RegisterTransportConfigurer(&DealProposal{}, transportConfigurer)
	// if err != nil {
	// 	return nil, err
	// }
	return c, nil
}

// Start initialized the Client, performing relevant database migrations
func (c *Client) Start(ctx context.Context) error {
	return nil
}

// FindProviders uses PeerResolver interface to locate a list of providers who may have a given payload CID.
func (c *Client) FindProviders(payloadCID cid.Cid) []RetrievalPeer {
	peers, err := c.resolver.GetPeers(payloadCID)
	if err != nil {
		fmt.Printf("failed to get peers: %s", err)
		return []RetrievalPeer{}
	}
	return peers
}

/*
Query sends a retrieval query to a specific retrieval provider, to determine
if the provider can serve a retrieval request and what its specific parameters for
the request are.
The client creates a new `RetrievalQueryStream` for the chosen peer ID,
and calls `WriteQuery` on it, which constructs a data-transfer message and writes it to the Query stream.
*/
func (c *Client) Query(ctx context.Context, p RetrievalPeer, payloadCID cid.Cid, params QueryParams) (QueryResponse, error) {
	// err := c.addMultiaddrs(ctx, p)
	// if err != nil {
	// 	fmt.Printf("Error adding multi address: %v\n", err)
	// 	return QueryResponseUndefined, err
	// }
	s, err := c.network.NewQueryStream(p.ID)
	if err != nil {
		fmt.Printf("Unable to create QueryStream: %v\n", err)
		return QueryResponseUndefined, err
	}
	defer s.Close()

	err = s.WriteQuery(Query{
		PayloadCID:  payloadCID,
		QueryParams: params,
	})
	if err != nil {
		fmt.Printf("Unable to write Query: %v\n", err)
		return QueryResponseUndefined, err
	}

	return s.ReadQueryResponse()
}

/*
Retrieve initiates the retrieval deal flow, which involves multiple requests and responses
To start this processes, the client creates a new `RetrievalDealStream`.  Currently, this connection is
kept open through the entire deal until completion or failure.  Make deals pauseable as well as surviving
a restart is a planned future feature.
Retrieve should be called after using FindProviders and Query are used to identify an appropriate provider to
retrieve the deal from. The parameters identified in Query should be passed to Retrieve to ensure the
greatest likelihood the provider will accept the deal
When called, the client takes the following actions:
1. Creates a deal ID using the next value from its `storedCounter`.
2. Constructs a `DealProposal` with deal terms
3. Tells its statemachine to begin tracking this deal state by dealID.
4. Constructs a `blockio.SelectorVerifier` and adds it to its dealID-keyed map of block verifiers.
5. Triggers a `ClientEventOpen` event on its statemachine.
From then on, the statemachine controls the deal flow in the client. Other components may listen for events in this flow by calling
`SubscribeToEvents` on the Client. The Client handles consuming blocks it receives from the provider, via `ConsumeBlocks` function
Documentation of the client state machine can be found at https://godoc.org/github.com/filecoin-project/go-fil-markets/retrievalmarket/impl/clientstates
*/
func (c *Client) Retrieve(ctx context.Context, payloadCID cid.Cid, params Params, totalFunds abi.TokenAmount, p RetrievalPeer, clientWallet address.Address, minerWallet address.Address, storeID *multistore.StoreID) (DealID, error) {
	// err := c.addMultiaddrs(ctx, p)
	// if err != nil {
	// 	return 0, err
	// }
	next, err := c.storedCounter.Next()
	if err != nil {
		return 0, err
	}
	// make sure the store is loadable
	if storeID != nil {
		_, err = c.multiStore.Get(*storeID)
		if err != nil {
			return 0, err
		}
	}
	dealID := DealID(next)
	dealState := ClientDealState{
		DealProposal: DealProposal{
			PayloadCID: payloadCID,
			ID:         dealID,
			Params:     params,
		},
		TotalFunds:       totalFunds,
		ClientWallet:     clientWallet,
		MinerWallet:      minerWallet,
		TotalReceived:    0,
		CurrentInterval:  params.PaymentInterval,
		BytesPaidFor:     0,
		PaymentRequested: abi.NewTokenAmount(0),
		FundsSpent:       abi.NewTokenAmount(0),
		Status:           DealStatusNew,
		Sender:           p.ID,
		StoreID:          storeID,
	}

	// start the deal processing
	err = c.stateMachines.Begin(dealState.ID, &dealState)
	if err != nil {
		return 0, err
	}

	err = c.stateMachines.Send(dealState.ID, ClientEventOpen)
	if err != nil {
		return 0, err
	}

	return dealID, nil
}

// NewStoreID is for creating new store id from other package
func (c *Client) NewStoreID() *multistore.StoreID {
	id := c.multiStore.Next()
	return &id
}

func (c *Client) notifySubscribers(eventName fsm.EventName, state fsm.StateType) {
	evt := eventName.(ClientEvent)
	ds := state.(ClientDealState)
	_ = c.subscribers.Publish(internalClientEvent{evt, ds})
}

func (c *Client) addMultiaddrs(ctx context.Context, p RetrievalPeer) error {
	tok, _, err := c.node.GetChainHead(ctx)
	if err != nil {
		return err
	}
	maddrs, err := c.node.GetKnownAddresses(ctx, p, tok)
	if err != nil {
		return err
	}
	if len(maddrs) > 0 {
		c.network.AddAddrs(p.ID, maddrs)
	}
	return nil
}

// SubscribeToEvents allows another component to listen for events on the RetrievalClient
// in order to track deals as they progress through the deal flow
func (c *Client) SubscribeToEvents(subscriber ClientSubscriber) Unsubscribe {
	return Unsubscribe(c.subscribers.Subscribe(subscriber))
}

// TryRestartInsufficientFunds attempts to restart any deals stuck in the insufficient funds state
// after funds are added to a given payment channel
func (c *Client) TryRestartInsufficientFunds(paymentChannel address.Address) error {
	var deals []ClientDealState
	err := c.stateMachines.List(&deals)
	if err != nil {
		return err
	}
	for _, deal := range deals {
		if deal.Status == DealStatusInsufficientFunds && deal.PaymentInfo.PayCh == paymentChannel {
			if err := c.stateMachines.Send(deal.ID, ClientEventRecheckFunds); err != nil {
				return err
			}
		}
	}
	return nil
}

// CancelDeal attempts to cancel an in progress deal
func (c *Client) CancelDeal(dealID DealID) error {
	return c.stateMachines.Send(dealID, ClientEventCancel)
}

// GetDeal returns a given deal by deal ID, if it exists
func (c *Client) GetDeal(dealID DealID) (ClientDealState, error) {
	var out ClientDealState
	if err := c.stateMachines.Get(dealID).Get(&out); err != nil {
		return ClientDealState{}, err
	}
	return out, nil
}

// ListDeals lists all known retrieval deals
func (c *Client) ListDeals() (map[DealID]ClientDealState, error) {
	var deals []ClientDealState
	err := c.stateMachines.List(&deals)
	if err != nil {
		return nil, err
	}
	dealMap := make(map[DealID]ClientDealState)
	for _, deal := range deals {
		dealMap[deal.ID] = deal
	}
	return dealMap, nil
}
