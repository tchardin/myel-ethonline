package main

import (
	"context"
	datatransfer "github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-multistore"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-statemachine/fsm"
)

// ProviderDealEnvironment is a bridge to the environment a provider deal is executing in
// It provides access to relevant functionality on the retrieval provider
type ProviderDealEnvironment interface {
	// Node returns the node interface for this deal
	Node() RetrievalProviderNode
	TrackTransfer(deal ProviderDealState) error
	UntrackTransfer(deal ProviderDealState) error
	DeleteStore(storeID multistore.StoreID) error
	ResumeDataTransfer(context.Context, datatransfer.ChannelID) error
	CloseDataTransfer(context.Context, datatransfer.ChannelID) error
}

// TrackTransfer resumes a deal so we can start sending data
func TrackTransfer(ctx fsm.Context, environment ProviderDealEnvironment, deal ProviderDealState) error {
	err := environment.TrackTransfer(deal)
	if err != nil {
		return ctx.Trigger(ProviderEventDataTransferError, err)
	}
	return nil
}

func UnpauseDeal(ctx fsm.Context, environment ProviderDealEnvironment, deal ProviderDealState) error {
	err := environment.TrackTransfer(deal)
	if err != nil {
		return ctx.Trigger(ProviderEventDataTransferError, err)
	}
	err = environment.ResumeDataTransfer(ctx.Context(), deal.ChannelID)
	if err != nil {
		return ctx.Trigger(ProviderEventDataTransferError, err)
	}
	return nil
}

// CancelDeal clears a deal that went wrong for an unknown reason
func CancelDeal(ctx fsm.Context, environment ProviderDealEnvironment, deal ProviderDealState) error {
	// Read next response (or fail)
	err := environment.UntrackTransfer(deal)
	if err != nil {
		return ctx.Trigger(ProviderEventDataTransferError, err)
	}
	err = environment.DeleteStore(deal.StoreID)
	if err != nil {
		return ctx.Trigger(ProviderEventMultiStoreError, err)
	}
	err = environment.CloseDataTransfer(ctx.Context(), deal.ChannelID)
	if err != nil {
		return ctx.Trigger(ProviderEventDataTransferError, err)
	}
	return ctx.Trigger(ProviderEventCancelComplete)
}

// CleanupDeal runs to do memory cleanup for an in progress deal
func CleanupDeal(ctx fsm.Context, environment ProviderDealEnvironment, deal ProviderDealState) error {
	err := environment.UntrackTransfer(deal)
	if err != nil {
		return ctx.Trigger(ProviderEventDataTransferError, err)
	}
	err = environment.DeleteStore(deal.StoreID)
	if err != nil {
		return ctx.Trigger(ProviderEventMultiStoreError, err)
	}
	return ctx.Trigger(ProviderEventCleanupComplete)
}

func recordError(deal *ProviderDealState, err error) error {
	deal.Message = err.Error()
	return nil
}

// ProviderEvents are the events that can happen in a retrieval provider
var FsmProviderEvents = fsm.Events{
	// receiving new deal
	fsm.Event(ProviderEventOpen).
		From(DealStatusNew).ToNoChange().
		Action(
			func(deal *ProviderDealState) error {
				deal.TotalSent = 0
				deal.FundsReceived = abi.NewTokenAmount(0)
				deal.CurrentInterval = deal.PaymentInterval
				return nil
			},
		),

	// accepting
	fsm.Event(ProviderEventDealAccepted).
		From(DealStatusNew).To(DealStatusOngoing).
		Action(func(deal *ProviderDealState, channelID datatransfer.ChannelID) error {
			deal.ChannelID = channelID
			return nil
		}),

	// request payment
	fsm.Event(ProviderEventPaymentRequested).
		From(DealStatusOngoing).To(DealStatusFundsNeeded).
		Action(func(deal *ProviderDealState, totalSent uint64) error {
			deal.TotalSent = totalSent
			return nil
		}),

	// receive and process payment
	fsm.Event(ProviderEventSaveVoucherFailed).
		FromMany(DealStatusFundsNeeded, DealStatusFundsNeededLastPayment).To(DealStatusFailing).
		Action(recordError),
	fsm.Event(ProviderEventPartialPaymentReceived).
		FromMany(DealStatusFundsNeeded, DealStatusFundsNeededLastPayment).ToNoChange().
		Action(func(deal *ProviderDealState, fundsReceived abi.TokenAmount) error {
			deal.FundsReceived = big.Add(deal.FundsReceived, fundsReceived)
			return nil
		}),
	fsm.Event(ProviderEventPaymentReceived).
		From(DealStatusFundsNeeded).To(DealStatusOngoing).
		From(DealStatusFundsNeededLastPayment).To(DealStatusFinalizing).
		Action(func(deal *ProviderDealState, fundsReceived abi.TokenAmount) error {
			deal.FundsReceived = big.Add(deal.FundsReceived, fundsReceived)
			deal.CurrentInterval += deal.PaymentIntervalIncrease
			return nil
		}),

	// completing
	fsm.Event(ProviderEventComplete).From(DealStatusFinalizing).To(DealStatusCompleting),
	fsm.Event(ProviderEventCleanupComplete).From(DealStatusCompleting).To(DealStatusCompleted),

	// Error cleanup
	fsm.Event(ProviderEventCancelComplete).FromMany(DealStatusFailing).To(DealStatusErrored),

	// data transfer errors
	fsm.Event(ProviderEventDataTransferError).
		FromAny().To(DealStatusErrored).
		Action(recordError),

	// multistore errors
	fsm.Event(ProviderEventMultiStoreError).
		FromAny().To(DealStatusErrored).
		Action(recordError),

	fsm.Event(ProviderEventClientCancelled).
		From(DealStatusFailing).ToJustRecord().
		FromAny().To(DealStatusCancelled).Action(
		func(deal *ProviderDealState) error {
			if deal.Status != DealStatusFailing {
				deal.Message = "Client cancelled retrieval"
			}
			return nil
		},
	),
}

// ProviderStateEntryFuncs are the handlers for different states in a retrieval provider
var ProviderStateEntryFuncs = fsm.StateEntryFuncs{
	DealStatusOngoing:    TrackTransfer,
	DealStatusFailing:    CancelDeal,
	DealStatusCompleting: CleanupDeal,
}

// ProviderFinalityStates are the terminal states for a retrieval provider
var ProviderFinalityStates = []fsm.StateKey{
	DealStatusErrored,
	DealStatusCompleted,
	DealStatusCancelled,
}
