package rtmkt

import (
	"context"
	"fmt"
	"io"

	datatransfer "github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-fil-markets/shared"
	"github.com/filecoin-project/go-multistore"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/libp2p/go-libp2p-core/peer"
)

type providerValidationEnvironment struct {
	p *Provider
}

// CheckDealParams verifies the given deal params are acceptable
func (pve *providerValidationEnvironment) CheckDealParams(pricePerByte abi.TokenAmount, paymentInterval uint64, paymentIntervalIncrease uint64) error {
	ask := pve.p.GetAsk()
	if pricePerByte.LessThan(ask.PricePerByte) {
		return fmt.Errorf("Price per byte too low")
	}
	if paymentInterval > ask.PaymentInterval {
		return fmt.Errorf("Payment interval too large")
	}
	if paymentIntervalIncrease > ask.PaymentIntervalIncrease {
		return fmt.Errorf("Payment interval increase too large")
	}
	return nil
}

// RunDealDecisioningLogic runs custom deal decision logic to decide if a deal is accepted, if present
func (pve *providerValidationEnvironment) RunDealDecisioningLogic(ctx context.Context, state ProviderDealState) (bool, string, error) {
	if pve.p.dealDecider == nil {
		return true, "", nil
	}
	return pve.p.dealDecider(ctx, state)
}

// StateMachines returns the FSM Group to begin tracking with
func (pve *providerValidationEnvironment) BeginTracking(pds ProviderDealState) error {
	err := pve.p.stateMachines.Begin(pds.Identifier(), &pds)
	if err != nil {
		return err
	}
	return pve.p.stateMachines.Send(pds.Identifier(), ProviderEventOpen)
}

func (pve *providerValidationEnvironment) NextStoreID() (multistore.StoreID, error) {
	storeID := pve.p.multiStore.Next()
	_, err := pve.p.multiStore.Get(storeID)
	return storeID, err
}

type providerRevalidatorEnvironment struct {
	p *Provider
}

func (pre *providerRevalidatorEnvironment) Node() RetrievalNode {
	return pre.p.node
}

func (pre *providerRevalidatorEnvironment) SendEvent(dealID ProviderDealIdentifier, evt ProviderEvent, args ...interface{}) error {
	return pre.p.stateMachines.Send(dealID, evt, args...)
}

func (pre *providerRevalidatorEnvironment) Get(dealID ProviderDealIdentifier) (ProviderDealState, error) {
	var deal ProviderDealState
	err := pre.p.stateMachines.GetSync(context.TODO(), dealID, &deal)
	return deal, err
}

type providerStoreGetter struct {
	p *Provider
}

func (psg *providerStoreGetter) Get(otherPeer peer.ID, dealID DealID) (*multistore.Store, error) {
	var deal ProviderDealState
	err := psg.p.stateMachines.GetSync(context.TODO(), ProviderDealIdentifier{Receiver: otherPeer, DealID: dealID}, &deal)
	if err != nil {
		return nil, err
	}
	return psg.p.multiStore.Get(deal.StoreID)
}

type providerDealEnvironment struct {
	p *Provider
}

// Node returns the node interface for this deal
func (pde *providerDealEnvironment) Node() RetrievalNode {
	return pde.p.node
}

func (pde *providerDealEnvironment) ReadIntoBlockstore(storeID multistore.StoreID, pieceData io.Reader) error {
	return nil
}

func (pde *providerDealEnvironment) TrackTransfer(deal ProviderDealState) error {
	pde.p.revalidator.TrackChannel(deal)
	return nil
}

func (pde *providerDealEnvironment) UntrackTransfer(deal ProviderDealState) error {
	pde.p.revalidator.UntrackChannel(deal)
	return nil
}

func (pde *providerDealEnvironment) ResumeDataTransfer(ctx context.Context, chid datatransfer.ChannelID) error {
	return pde.p.dataTransfer.ResumeDataTransferChannel(ctx, chid)
}

func (pde *providerDealEnvironment) CloseDataTransfer(ctx context.Context, chid datatransfer.ChannelID) error {
	return pde.p.dataTransfer.CloseDataTransferChannel(ctx, chid)
}

func (pde *providerDealEnvironment) DeleteStore(storeID multistore.StoreID) error {
	return pde.p.multiStore.Delete(storeID)
}

type clientDealEnvironment struct {
	c *Client
}

// Node returns the node interface for this deal
func (cde *clientDealEnvironment) Node() RetrievalNode {
	return cde.c.node
}

func (cde *clientDealEnvironment) OpenDataTransfer(ctx context.Context, to peer.ID, proposal *DealProposal) (datatransfer.ChannelID, error) {
	sel := shared.AllSelector()
	var vouch datatransfer.Voucher = proposal
	fmt.Println("Opening data transfer")
	return cde.c.dataTransfer.OpenPullDataChannel(ctx, to, vouch, proposal.PayloadCID, sel)
}

func (cde *clientDealEnvironment) SendDataTransferVoucher(ctx context.Context, channelID datatransfer.ChannelID, payment *DealPayment) error {
	var vouch datatransfer.Voucher = payment
	fmt.Println("SendDataTransferVoucher")
	return cde.c.dataTransfer.SendVoucher(ctx, channelID, vouch)
}

func (cde *clientDealEnvironment) CloseDataTransfer(ctx context.Context, channelID datatransfer.ChannelID) error {
	return cde.c.dataTransfer.CloseDataTransferChannel(ctx, channelID)
}

type clientStoreGetter struct {
	c *Client
}

func (csg *clientStoreGetter) Get(otherPeer peer.ID, dealID DealID) (*multistore.Store, error) {
	var deal ClientDealState
	err := csg.c.stateMachines.Get(dealID).Get(&deal)
	if err != nil {
		return nil, err
	}
	if deal.StoreID == nil {
		return nil, nil
	}
	return csg.c.multiStore.Get(*deal.StoreID)
}
