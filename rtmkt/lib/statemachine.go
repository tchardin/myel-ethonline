package rtmkt

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"
	datatransfer "github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-multistore"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-statemachine/fsm"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
)

// ClientDealEnvironment is a bridge to the environment a client deal is executing in.
// It provides access to relevant functionality on the retrieval client
type ClientDealEnvironment interface {
	// Node returns the node interface for this deal
	Node() RetrievalNode
	OpenDataTransfer(ctx context.Context, to peer.ID, proposal *DealProposal) (datatransfer.ChannelID, error)
	SendDataTransferVoucher(context.Context, datatransfer.ChannelID, *DealPayment) error
	CloseDataTransfer(context.Context, datatransfer.ChannelID) error
}

// ProposeDeal sends the proposal to the other party
func ProposeDeal(ctx fsm.Context, environment ClientDealEnvironment, deal ClientDealState) error {
	channelID, err := environment.OpenDataTransfer(ctx.Context(), deal.Sender, &deal.DealProposal)
	if err != nil {
		return ctx.Trigger(ClientEventWriteDealProposalErrored, err)
	}
	return ctx.Trigger(ClientEventDealProposed, channelID)
}

// SetupPaymentChannelStart initiates setting up a payment channel for a deal
func SetupPaymentChannelStart(ctx fsm.Context, environment ClientDealEnvironment, deal ClientDealState) error {

	tok, _, err := environment.Node().GetChainHead(ctx.Context())
	if err != nil {
		return ctx.Trigger(ClientEventPaymentChannelErrored, err)
	}

	paych, msgCID, err := environment.Node().GetOrCreatePaymentChannel(ctx.Context(), deal.ClientWallet, deal.MinerWallet, deal.TotalFunds, tok)
	if err != nil {
		return ctx.Trigger(ClientEventPaymentChannelErrored, err)
	}

	if paych == address.Undef {
		return ctx.Trigger(ClientEventPaymentChannelCreateInitiated, msgCID)
	}

	return ctx.Trigger(ClientEventPaymentChannelAddingFunds, msgCID, paych)
}

// WaitPaymentChannelReady waits for a pending operation on a payment channel -- either creating or depositing funds
func WaitPaymentChannelReady(ctx fsm.Context, environment ClientDealEnvironment, deal ClientDealState) error {
	paych, err := environment.Node().WaitForPaymentChannelReady(ctx.Context(), *deal.WaitMsgCID)
	if err != nil {
		return ctx.Trigger(ClientEventPaymentChannelErrored, err)
	}
	return ctx.Trigger(ClientEventPaymentChannelReady, paych)
}

// AllocateLane allocates a lane for this retrieval operation
func AllocateLane(ctx fsm.Context, environment ClientDealEnvironment, deal ClientDealState) error {
	lane, err := environment.Node().AllocateLane(ctx.Context(), deal.PaymentInfo.PayCh)
	if err != nil {
		return ctx.Trigger(ClientEventAllocateLaneErrored, err)
	}
	return ctx.Trigger(ClientEventLaneAllocated, lane)
}

// Ongoing just double checks that we may need to move out of the ongoing state cause a payment was previously requested
func Ongoing(ctx fsm.Context, environment ClientDealEnvironment, deal ClientDealState) error {
	if deal.PaymentRequested.GreaterThan(big.Zero()) {
		if deal.LastPaymentRequested {
			return ctx.Trigger(ClientEventLastPaymentRequested, big.Zero())
		}
		return ctx.Trigger(ClientEventPaymentRequested, big.Zero())
	}
	return nil
}

// ProcessPaymentRequested processes a request for payment from the provider
func ProcessPaymentRequested(ctx fsm.Context, environment ClientDealEnvironment, deal ClientDealState) error {
	// see if we need to send payment
	if deal.TotalReceived-deal.BytesPaidFor >= deal.CurrentInterval ||
		deal.AllBlocksReceived {
		return ctx.Trigger(ClientEventSendFunds)
	}
	return nil
}

// SendFunds sends the next amount requested by the provider
func SendFunds(ctx fsm.Context, environment ClientDealEnvironment, deal ClientDealState) error {
	// check that paymentRequest <= (totalReceived - bytesPaidFor) * pricePerByte + (unsealPrice - unsealFundsPaid), or fail
	retrievalPrice := big.Mul(abi.NewTokenAmount(int64(deal.TotalReceived-deal.BytesPaidFor)), deal.PricePerByte)
	if deal.PaymentRequested.GreaterThan(retrievalPrice) {
		return ctx.Trigger(ClientEventBadPaymentRequested, "too much money requested for bytes sent")
	}

	tok, _, err := environment.Node().GetChainHead(ctx.Context())
	if err != nil {
		return ctx.Trigger(ClientEventCreateVoucherFailed, err)
	}

	// create payment voucher with node (or fail) for (fundsSpent + paymentRequested)
	// use correct payCh + lane
	// (node will do subtraction back to paymentRequested... slightly odd behavior but... well anyway)
	voucher, err := environment.Node().CreatePaymentVoucher(ctx.Context(), deal.PaymentInfo.PayCh, big.Add(deal.FundsSpent, deal.PaymentRequested), deal.PaymentInfo.Lane, tok)
	if err != nil {
		shortfallErr, ok := err.(ShortfallError)
		if ok {
			return ctx.Trigger(ClientEventVoucherShortfall, shortfallErr.Shortfall())
		}
		return ctx.Trigger(ClientEventCreateVoucherFailed, err)
	}

	// send payment voucher (or fail)
	err = environment.SendDataTransferVoucher(ctx.Context(), deal.ChannelID, &DealPayment{
		ID:             deal.DealProposal.ID,
		PaymentChannel: deal.PaymentInfo.PayCh,
		PaymentVoucher: voucher,
	})
	if err != nil {
		return ctx.Trigger(ClientEventWriteDealPaymentErrored, err)
	}

	return ctx.Trigger(ClientEventPaymentSent)
}

// CheckFunds examines current available funds in a payment channel after a voucher shortfall to determine
// a course of action -- whether it's a good time to try again, wait for pending operations, or
// we've truly expended all funds and we need to wait for a manual readd
func CheckFunds(ctx fsm.Context, environment ClientDealEnvironment, deal ClientDealState) error {
	// if we already have an outstanding operation, let's wait for that to complete
	if deal.WaitMsgCID != nil {
		return ctx.Trigger(ClientEventPaymentChannelAddingFunds, *deal.WaitMsgCID, deal.PaymentInfo.PayCh)
	}
	availableFunds, err := environment.Node().CheckAvailableFunds(ctx.Context(), deal.PaymentInfo.PayCh)
	if err != nil {
		return ctx.Trigger(ClientEventPaymentChannelErrored, err)
	}
	unredeemedFunds := big.Sub(availableFunds.ConfirmedAmt, availableFunds.VoucherReedeemedAmt)
	shortfall := big.Sub(deal.PaymentRequested, unredeemedFunds)
	if shortfall.LessThanEqual(big.Zero()) {
		return ctx.Trigger(ClientEventPaymentChannelReady, deal.PaymentInfo.PayCh)
	}
	totalInFlight := big.Add(availableFunds.PendingAmt, availableFunds.QueuedAmt)
	if totalInFlight.LessThan(shortfall) || availableFunds.PendingWaitSentinel == nil {
		finalShortfall := big.Sub(shortfall, totalInFlight)
		return ctx.Trigger(ClientEventFundsExpended, finalShortfall)
	}
	return ctx.Trigger(ClientEventPaymentChannelAddingFunds, *availableFunds.PendingWaitSentinel, deal.PaymentInfo.PayCh)
}

// ClientCancelDeal clears a deal that went wrong for an unknown reason
func ClientCancelDeal(ctx fsm.Context, environment ClientDealEnvironment, deal ClientDealState) error {
	// Read next response (or fail)
	err := environment.CloseDataTransfer(ctx.Context(), deal.ChannelID)
	if err != nil {
		return ctx.Trigger(ClientEventDataTransferError, err)
	}

	return ctx.Trigger(ClientEventCancelComplete)
}

// CheckComplete verifies that a provider that completed without a last payment requested did in fact send us all the data
func CheckComplete(ctx fsm.Context, environment ClientDealEnvironment, deal ClientDealState) error {
	if !deal.AllBlocksReceived {
		return ctx.Trigger(ClientEventEarlyTermination)
	}

	return ctx.Trigger(ClientEventCompleteVerified)
}

func recordReceived(deal *ClientDealState, totalReceived uint64) error {
	deal.TotalReceived = totalReceived
	return nil
}

var paymentChannelCreationStates = []fsm.StateKey{
	DealStatusWaitForAcceptance,
	DealStatusAccepted,
	DealStatusPaymentChannelCreating,
	DealStatusPaymentChannelAllocatingLane,
}

// FsmClientEvents are the events that can happen in a retrieval client
var FsmClientEvents = fsm.Events{
	fsm.Event(ClientEventOpen).
		From(DealStatusNew).ToNoChange(),

	// ProposeDeal handler events
	fsm.Event(ClientEventWriteDealProposalErrored).
		FromAny().To(DealStatusErrored).
		Action(func(deal *ClientDealState, err error) error {
			deal.Message = fmt.Errorf("proposing deal: %w", err).Error()
			return nil
		}),
	fsm.Event(ClientEventDealProposed).
		From(DealStatusNew).To(DealStatusWaitForAcceptance).
		Action(func(deal *ClientDealState, channelID datatransfer.ChannelID) error {
			deal.ChannelID = channelID
			deal.Message = ""
			return nil
		}),

	fsm.Event(ClientEventDealNotFound).
		From(DealStatusWaitForAcceptance).To(DealStatusDealNotFound).
		Action(func(deal *ClientDealState, message string) error {
			deal.Message = fmt.Sprintf("deal not found: %s", message)
			return nil
		}),
	fsm.Event(ClientEventDealAccepted).
		From(DealStatusWaitForAcceptance).To(DealStatusAccepted),
	fsm.Event(ClientEventUnknownResponseReceived).
		FromAny().To(DealStatusFailing).
		Action(func(deal *ClientDealState, status DealStatus) error {
			deal.Message = fmt.Sprintf("Unexpected deal response status: %s", DealStatuses[status])
			return nil
		}),

	// Payment channel setup
	fsm.Event(ClientEventPaymentChannelErrored).
		FromMany(DealStatusAccepted, DealStatusPaymentChannelCreating, DealStatusPaymentChannelAddingFunds).To(DealStatusFailing).
		Action(func(deal *ClientDealState, err error) error {
			deal.Message = fmt.Errorf("error from payment channel: %w", err).Error()
			return nil
		}),
	fsm.Event(ClientEventPaymentChannelCreateInitiated).
		From(DealStatusAccepted).To(DealStatusPaymentChannelCreating).
		Action(func(deal *ClientDealState, msgCID cid.Cid) error {
			deal.WaitMsgCID = &msgCID
			return nil
		}),
	fsm.Event(ClientEventPaymentChannelAddingFunds).
		FromMany(DealStatusAccepted).To(DealStatusPaymentChannelAllocatingLane).
		FromMany(DealStatusCheckFunds).To(DealStatusPaymentChannelAddingFunds).
		Action(func(deal *ClientDealState, msgCID cid.Cid, payCh address.Address) error {
			deal.WaitMsgCID = &msgCID
			if deal.PaymentInfo == nil {
				deal.PaymentInfo = &PaymentInfo{
					PayCh: payCh,
				}
			}
			return nil
		}),
	fsm.Event(ClientEventPaymentChannelReady).
		From(DealStatusPaymentChannelCreating).To(DealStatusPaymentChannelAllocatingLane).
		From(DealStatusPaymentChannelAddingFunds).To(DealStatusOngoing).
		From(DealStatusCheckFunds).To(DealStatusOngoing).
		Action(func(deal *ClientDealState, payCh address.Address) error {
			if deal.PaymentInfo == nil {
				deal.PaymentInfo = &PaymentInfo{
					PayCh: payCh,
				}
			}
			deal.WaitMsgCID = nil
			// remove any insufficient funds message
			deal.Message = ""
			return nil
		}),
	fsm.Event(ClientEventAllocateLaneErrored).
		FromMany(DealStatusPaymentChannelAllocatingLane).
		To(DealStatusFailing).
		Action(func(deal *ClientDealState, err error) error {
			deal.Message = fmt.Errorf("allocating payment lane: %w", err).Error()
			return nil
		}),

	fsm.Event(ClientEventLaneAllocated).
		From(DealStatusPaymentChannelAllocatingLane).To(DealStatusOngoing).
		Action(func(deal *ClientDealState, lane uint64) error {
			deal.PaymentInfo.Lane = lane
			return nil
		}),

	// Transfer Channel Errors
	fsm.Event(ClientEventDataTransferError).
		FromAny().To(DealStatusErrored).
		Action(func(deal *ClientDealState, err error) error {
			deal.Message = fmt.Sprintf("error generated by data transfer: %s", err.Error())
			return nil
		}),

	// Receiving requests for payment
	fsm.Event(ClientEventLastPaymentRequested).
		FromMany(
			DealStatusOngoing,
			DealStatusFundsNeededLastPayment,
			DealStatusFundsNeeded).To(DealStatusFundsNeededLastPayment).
		FromMany(
			paymentChannelCreationStates...).ToJustRecord().
		Action(func(deal *ClientDealState, paymentOwed abi.TokenAmount) error {
			deal.PaymentRequested = big.Add(deal.PaymentRequested, paymentOwed)
			deal.LastPaymentRequested = true
			return nil
		}),
	fsm.Event(ClientEventPaymentRequested).
		FromMany(
			DealStatusOngoing,
			DealStatusFundsNeeded).To(DealStatusFundsNeeded).
		FromMany(
			paymentChannelCreationStates...).ToJustRecord().
		Action(func(deal *ClientDealState, paymentOwed abi.TokenAmount) error {
			deal.PaymentRequested = big.Add(deal.PaymentRequested, paymentOwed)
			return nil
		}),

	// Receiving data
	fsm.Event(ClientEventAllBlocksReceived).
		FromMany(
			DealStatusOngoing,
			DealStatusBlocksComplete,
		).To(DealStatusBlocksComplete).
		FromMany(paymentChannelCreationStates...).ToJustRecord().
		FromMany(DealStatusSendFunds, DealStatusFundsNeeded).ToJustRecord().
		From(DealStatusFundsNeededLastPayment).To(DealStatusSendFundsLastPayment).
		Action(func(deal *ClientDealState) error {
			deal.AllBlocksReceived = true
			return nil
		}),
	fsm.Event(ClientEventBlocksReceived).
		FromMany(DealStatusOngoing,
			DealStatusFundsNeeded,
			DealStatusFundsNeededLastPayment).ToNoChange().
		FromMany(paymentChannelCreationStates...).ToJustRecord().
		Action(recordReceived),

	fsm.Event(ClientEventSendFunds).
		From(DealStatusFundsNeeded).To(DealStatusSendFunds).
		From(DealStatusFundsNeededLastPayment).To(DealStatusSendFundsLastPayment),

	// Sending Payments
	fsm.Event(ClientEventFundsExpended).
		FromMany(DealStatusCheckFunds).To(DealStatusInsufficientFunds).
		Action(func(deal *ClientDealState, shortfall abi.TokenAmount) error {
			deal.Message = fmt.Sprintf("not enough current or pending funds in payment channel, shortfall of %s", shortfall.String())
			return nil
		}),
	fsm.Event(ClientEventBadPaymentRequested).
		FromMany(DealStatusSendFunds, DealStatusSendFundsLastPayment).To(DealStatusFailing).
		Action(func(deal *ClientDealState, message string) error {
			deal.Message = message
			return nil
		}),
	fsm.Event(ClientEventCreateVoucherFailed).
		FromMany(DealStatusSendFunds, DealStatusSendFundsLastPayment).To(DealStatusFailing).
		Action(func(deal *ClientDealState, err error) error {
			deal.Message = fmt.Errorf("creating payment voucher: %w", err).Error()
			return nil
		}),
	fsm.Event(ClientEventVoucherShortfall).
		FromMany(DealStatusSendFunds, DealStatusSendFundsLastPayment).To(DealStatusCheckFunds).
		Action(func(deal *ClientDealState, shortfall abi.TokenAmount) error {
			return nil
		}),

	fsm.Event(ClientEventWriteDealPaymentErrored).
		FromAny().To(DealStatusErrored).
		Action(func(deal *ClientDealState, err error) error {
			deal.Message = fmt.Errorf("writing deal payment: %w", err).Error()
			return nil
		}),
	fsm.Event(ClientEventPaymentSent).
		From(DealStatusSendFunds).To(DealStatusOngoing).
		From(DealStatusSendFundsLastPayment).To(DealStatusFinalizing).
		Action(func(deal *ClientDealState) error {
			// paymentRequested = 0
			// fundsSpent = fundsSpent + paymentRequested
			// if paymentRequested / pricePerByte >= currentInterval
			// currentInterval = currentInterval + proposal.intervalIncrease
			// bytesPaidFor = bytesPaidFor + (paymentRequested / pricePerByte)
			deal.FundsSpent = big.Add(deal.FundsSpent, deal.PaymentRequested)

			bytesPaidFor := big.Div(deal.PaymentRequested, deal.PricePerByte).Uint64()
			if bytesPaidFor >= deal.CurrentInterval {
				deal.CurrentInterval += deal.DealProposal.PaymentIntervalIncrease
			}
			deal.BytesPaidFor += bytesPaidFor
			deal.PaymentRequested = abi.NewTokenAmount(0)
			return nil
		}),

	// completing deals
	fsm.Event(ClientEventComplete).
		From(DealStatusOngoing).To(DealStatusCheckComplete).
		From(DealStatusFinalizing).To(DealStatusCompleted),
	fsm.Event(ClientEventCompleteVerified).
		From(DealStatusCheckComplete).To(DealStatusCompleted),
	fsm.Event(ClientEventEarlyTermination).
		From(DealStatusCheckComplete).To(DealStatusErrored).
		Action(func(deal *ClientDealState) error {
			deal.Message = "Provider sent complete status without sending all data"
			return nil
		}),

	// after cancelling a deal is complete
	fsm.Event(ClientEventCancelComplete).
		From(DealStatusFailing).To(DealStatusErrored).
		From(DealStatusCancelling).To(DealStatusCancelled),

	// receiving a cancel indicating most likely that the provider experienced something wrong on their
	// end, unless we are already failing or cancelling
	fsm.Event(ClientEventProviderCancelled).
		From(DealStatusFailing).ToJustRecord().
		From(DealStatusCancelling).ToJustRecord().
		FromAny().To(DealStatusErrored).Action(
		func(deal *ClientDealState) error {
			if deal.Status != DealStatusFailing && deal.Status != DealStatusCancelling {
				deal.Message = "Provider cancelled retrieval due to error"
			}
			return nil
		},
	),

	// user manually cancells retrieval
	fsm.Event(ClientEventCancel).FromAny().To(DealStatusCancelling).Action(func(deal *ClientDealState) error {
		deal.Message = "Retrieval Cancelled"
		return nil
	}),

	// payment channel receives more money, we believe there may be reason to recheck the funds for this channel
	fsm.Event(ClientEventRecheckFunds).From(DealStatusInsufficientFunds).To(DealStatusCheckFunds),
}

// ClientFinalityStates are terminal states after which no further events are received
var ClientFinalityStates = []fsm.StateKey{
	DealStatusErrored,
	DealStatusCompleted,
	DealStatusCancelled,
	DealStatusRejected,
	DealStatusDealNotFound,
}

// ClientStateEntryFuncs are the handlers for different states in a retrieval client
var ClientStateEntryFuncs = fsm.StateEntryFuncs{
	DealStatusNew:                          ProposeDeal,
	DealStatusAccepted:                     SetupPaymentChannelStart,
	DealStatusPaymentChannelCreating:       WaitPaymentChannelReady,
	DealStatusPaymentChannelAllocatingLane: AllocateLane,
	DealStatusOngoing:                      Ongoing,
	DealStatusFundsNeeded:                  ProcessPaymentRequested,
	DealStatusFundsNeededLastPayment:       ProcessPaymentRequested,
	DealStatusSendFunds:                    SendFunds,
	DealStatusSendFundsLastPayment:         SendFunds,
	DealStatusCheckFunds:                   CheckFunds,
	DealStatusPaymentChannelAddingFunds:    WaitPaymentChannelReady,
	DealStatusFailing:                      ClientCancelDeal,
	DealStatusCancelling:                   ClientCancelDeal,
	DealStatusCheckComplete:                CheckComplete,
}

// ProviderDealEnvironment is a bridge to the environment a provider deal is executing in
// It provides access to relevant functionality on the retrieval provider
type ProviderDealEnvironment interface {
	// Node returns the node interface for this deal
	Node() RetrievalNode
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
