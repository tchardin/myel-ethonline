package main

import (
	"context"
	"io"

	datatransfer "github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-multistore"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/libp2p/go-libp2p-core/peer"
)

type providerValidationEnvironment struct {
	p *Provider
}

// CheckDealParams verifies the given deal params are acceptable
func (pve *providerValidationEnvironment) CheckDealParams(pricePerByte abi.TokenAmount, paymentInterval uint64, paymentIntervalIncrease uint64, unsealPrice abi.TokenAmount) error {
	return nil
}

// RunDealDecisioningLogic runs custom deal decision logic to decide if a deal is accepted, if present
func (pve *providerValidationEnvironment) RunDealDecisioningLogic(ctx context.Context, state ProviderDealState) (bool, string, error) {
	return true, "", nil
}

// StateMachines returns the FSM Group to begin tracking with
func (pve *providerValidationEnvironment) BeginTracking(pds ProviderDealState) error {
	return nil
}

func (pve *providerValidationEnvironment) NextStoreID() (multistore.StoreID, error) {
	storeID := pve.p.multiStore.Next()
	_, err := pve.p.multiStore.Get(storeID)
	return storeID, err
}

type providerRevalidatorEnvironment struct {
	p *Provider
}

func (pre *providerRevalidatorEnvironment) Node() RetrievalProviderNode {
	return pre.p.node
}

func (pre *providerRevalidatorEnvironment) SendEvent(dealID ProviderDealIdentifier, evt ProviderEvent, args ...interface{}) error {
	return pre.p.stateMachines.Send(dealID, evt, args...)
}

func (pre *providerRevalidatorEnvironment) Get(dealID ProviderDealIdentifier) (ProviderDealState, error) {
	var deal ProviderDealState
	// err := pre.p.stateMachines.GetSync(context.TODO(), dealID, &deal)
	return deal, nil
}

type providerStoreGetter struct {
	p *Provider
}

func (psg *providerStoreGetter) Get(otherPeer peer.ID, dealID DealID) (*multistore.Store, error) {
	var deal ProviderDealState
	// err := psg.p.stateMachines.GetSync(context.TODO(), ProviderDealIdentifier{Receiver: otherPeer, DealID: dealID}, &deal)
	// if err != nil {
	// 	return nil, err
	// }
	return psg.p.multiStore.Get(deal.StoreID)
}

type providerDealEnvironment struct {
	p *Provider
}

// Node returns the node interface for this deal
func (pde *providerDealEnvironment) Node() RetrievalProviderNode {
	return pde.p.node
}

func (pde *providerDealEnvironment) ReadIntoBlockstore(storeID multistore.StoreID, pieceData io.Reader) error {
	return nil
}

func (pde *providerDealEnvironment) TrackTransfer(deal ProviderDealState) error {
	// pde.p.revalidator.TrackChannel(deal)
	return nil
}

func (pde *providerDealEnvironment) UntrackTransfer(deal ProviderDealState) error {
	// pde.p.revalidator.UntrackChannel(deal)
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
