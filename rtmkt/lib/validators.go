package rtmkt

import (
	"context"
	"fmt"
	"sync"

	datatransfer "github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-multistore"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

// ValidationEnvironment contains the dependencies needed to validate deals
type ValidationEnvironment interface {
	// GetPiece(c cid.Cid, pieceCID *cid.Cid) (piecestore.PieceInfo, error)
	// CheckDealParams verifies the given deal params are acceptable
	CheckDealParams(pricePerByte abi.TokenAmount, paymentInterval uint64, paymentIntervalIncrease uint64) error
	// RunDealDecisioningLogic runs custom deal decision logic to decide if a deal is accepted, if present
	RunDealDecisioningLogic(ctx context.Context, state ProviderDealState) (bool, string, error)
	// StateMachines returns the FSM Group to begin tracking with
	BeginTracking(pds ProviderDealState) error
	// NextStoreID allocates a store for this deal
	NextStoreID() (multistore.StoreID, error)
}

// ProviderRequestValidator validates incoming requests for the Retrieval Provider
type ProviderRequestValidator struct {
	env ValidationEnvironment
}

// NewProviderRequestValidator returns a new instance of the ProviderRequestValidator
func NewProviderRequestValidator(env ValidationEnvironment) *ProviderRequestValidator {
	return &ProviderRequestValidator{env}
}

// ValidatePush validates a push request received from the peer that will send data
func (rv *ProviderRequestValidator) ValidatePush(sender peer.ID, voucher datatransfer.Voucher, baseCid cid.Cid, selector ipld.Node) (datatransfer.VoucherResult, error) {
	return nil, fmt.Errorf("No pushes accepted")
}

// ValidatePull validates a pull request received from the peer that will receive data
func (rv *ProviderRequestValidator) ValidatePull(receiver peer.ID, voucher datatransfer.Voucher, baseCid cid.Cid, selector ipld.Node) (datatransfer.VoucherResult, error) {
	proposal, ok := voucher.(*DealProposal)
	if !ok {
		return nil, fmt.Errorf("Wrong voucher type")
	}
	if proposal.PayloadCID != baseCid {
		return nil, fmt.Errorf("Incorrect CID for this proposal")
	}
	pds := ProviderDealState{
		DealProposal: *proposal,
		Receiver:     receiver,
	}
	response := DealResponse{
		ID: proposal.ID,
	}
	// check that the deal parameters match our required parameters or
	// reject outright
	err := rv.env.CheckDealParams(pds.PricePerByte, pds.PaymentInterval, pds.PaymentIntervalIncrease)
	if err != nil {
		response.Status = DealStatusRejected
	}

	accepted, reason, err := rv.env.RunDealDecisioningLogic(context.TODO(), pds)
	if !accepted {
		response.Status = DealStatusRejected
		response.Message = reason
		return &response, nil
	}
	if err != nil {
		response.Status = DealStatusErrored
		return &response, nil
	}

	pds.StoreID, err = rv.env.NextStoreID()
	if err != nil {
		response.Status = DealStatusErrored
		return &response, nil
	}
	response.Status = DealStatusAccepted

	err = rv.env.BeginTracking(pds)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// RevalidatorEnvironment are the dependencies needed to
// build the logic of revalidation -- essentially, access to the node at statemachines
type RevalidatorEnvironment interface {
	Node() RetrievalNode
	SendEvent(dealID ProviderDealIdentifier, evt ProviderEvent, args ...interface{}) error
	Get(dealID ProviderDealIdentifier) (ProviderDealState, error)
}

type channelData struct {
	dealID       ProviderDealIdentifier
	totalSent    uint64
	totalPaidFor uint64
	interval     uint64
	pricePerByte abi.TokenAmount
	reload       bool
}

// ProviderRevalidator defines data transfer revalidation logic in the context of
// a provider for a retrieval deal
type ProviderRevalidator struct {
	env               RevalidatorEnvironment
	trackedChannelsLk sync.RWMutex
	trackedChannels   map[datatransfer.ChannelID]*channelData
}

// NewProviderRevalidator returns a new instance of a ProviderRevalidator
func NewProviderRevalidator(env RevalidatorEnvironment) *ProviderRevalidator {
	return &ProviderRevalidator{
		env:             env,
		trackedChannels: make(map[datatransfer.ChannelID]*channelData),
	}
}

// TrackChannel indicates a retrieval deal tracked by this provider. It associates
// a given channel ID with a retrieval deal, so that checks run for data sent
// on the channel
func (pr *ProviderRevalidator) TrackChannel(deal ProviderDealState) {
	pr.trackedChannelsLk.Lock()
	defer pr.trackedChannelsLk.Unlock()
	pr.trackedChannels[deal.ChannelID] = &channelData{
		dealID: deal.Identifier(),
	}
	pr.writeDealState(deal)
}

// UntrackChannel indicates a retrieval deal is finish and no longer is tracked
// by this provider
func (pr *ProviderRevalidator) UntrackChannel(deal ProviderDealState) {
	pr.trackedChannelsLk.Lock()
	defer pr.trackedChannelsLk.Unlock()
	delete(pr.trackedChannels, deal.ChannelID)
}

func (pr *ProviderRevalidator) loadDealState(channel *channelData) error {
	if !channel.reload {
		return nil
	}
	deal, err := pr.env.Get(channel.dealID)
	if err != nil {
		return err
	}
	pr.writeDealState(deal)
	channel.reload = false
	return nil
}

func (pr *ProviderRevalidator) writeDealState(deal ProviderDealState) {
	channel := pr.trackedChannels[deal.ChannelID]
	channel.totalSent = deal.TotalSent
	channel.totalPaidFor = big.Div(big.Max(deal.FundsReceived, big.Zero()), deal.PricePerByte).Uint64()
	channel.interval = deal.CurrentInterval
	channel.pricePerByte = deal.PricePerByte
}

// Revalidate revalidates a request with a new voucher
func (pr *ProviderRevalidator) Revalidate(channelID datatransfer.ChannelID, voucher datatransfer.Voucher) (datatransfer.VoucherResult, error) {
	pr.trackedChannelsLk.RLock()
	defer pr.trackedChannelsLk.RUnlock()
	channel, ok := pr.trackedChannels[channelID]
	if !ok {
		return nil, nil
	}

	// read payment, or fail
	payment, ok := voucher.(*DealPayment)
	if !ok {
		return nil, fmt.Errorf("wrong voucher type")
	}
	response, err := pr.processPayment(channel.dealID, payment)
	if err == nil {
		channel.reload = true
	}
	return response, err
}

func (pr *ProviderRevalidator) processPayment(dealID ProviderDealIdentifier, payment *DealPayment) (*DealResponse, error) {

	tok, _, err := pr.env.Node().GetChainHead(context.TODO())
	if err != nil {
		_ = pr.env.SendEvent(dealID, ProviderEventSaveVoucherFailed, err)
		return errorDealResponse(dealID, err), err
	}

	deal, err := pr.env.Get(dealID)
	if err != nil {
		return errorDealResponse(dealID, err), err
	}

	// attempt to redeem voucher
	// (totalSent * pricePerByte + unsealPrice) - fundsReceived
	paymentOwed := big.Sub(big.Mul(abi.NewTokenAmount(int64(deal.TotalSent)), deal.PricePerByte), deal.FundsReceived)
	received, err := pr.env.Node().SavePaymentVoucher(context.TODO(), payment.PaymentChannel, payment.PaymentVoucher, nil, paymentOwed, tok)
	if err != nil {
		_ = pr.env.SendEvent(dealID, ProviderEventSaveVoucherFailed, err)
		return errorDealResponse(dealID, err), err
	}

	// received = 0 / err = nil indicates that the voucher was already saved, but this may be ok
	// if we are making a deal with ourself - in this case, we'll instead calculate received
	// but subtracting from fund sent
	if big.Cmp(received, big.Zero()) == 0 {
		received = big.Sub(payment.PaymentVoucher.Amount, deal.FundsReceived)
	}

	// check if all payments are received to continue the deal, or send updated required payment
	if received.LessThan(paymentOwed) {
		_ = pr.env.SendEvent(dealID, ProviderEventPartialPaymentReceived, received)
		return &DealResponse{
			ID:          deal.ID,
			Status:      deal.Status,
			PaymentOwed: big.Sub(paymentOwed, received),
		}, datatransfer.ErrPause
	}

	// resume deal
	_ = pr.env.SendEvent(dealID, ProviderEventPaymentReceived, received)
	if deal.Status == DealStatusFundsNeededLastPayment {
		return &DealResponse{
			ID:     deal.ID,
			Status: DealStatusCompleted,
		}, nil
	}
	return nil, nil
}

func errorDealResponse(dealID ProviderDealIdentifier, err error) *DealResponse {
	return &DealResponse{
		ID:      dealID.DealID,
		Message: err.Error(),
		Status:  DealStatusErrored,
	}
}

// OnPullDataSent is called on the responder side when more bytes are sent
// for a given pull request. It should return a VoucherResult + ErrPause to
// request revalidation or nil to continue uninterrupted,
// other errors will terminate the request
func (pr *ProviderRevalidator) OnPullDataSent(chid datatransfer.ChannelID, additionalBytesSent uint64) (bool, datatransfer.VoucherResult, error) {
	pr.trackedChannelsLk.RLock()
	defer pr.trackedChannelsLk.RUnlock()
	channel, ok := pr.trackedChannels[chid]
	if !ok {
		return false, nil, nil
	}

	err := pr.loadDealState(channel)
	if err != nil {
		return true, nil, err
	}

	channel.totalSent += additionalBytesSent
	if channel.totalSent-channel.totalPaidFor >= channel.interval {
		paymentOwed := big.Mul(abi.NewTokenAmount(int64(channel.totalSent-channel.totalPaidFor)), channel.pricePerByte)
		err := pr.env.SendEvent(channel.dealID, ProviderEventPaymentRequested, channel.totalSent)
		if err != nil {
			return true, nil, err
		}
		return true, &DealResponse{
			ID:          channel.dealID.DealID,
			Status:      DealStatusFundsNeeded,
			PaymentOwed: paymentOwed,
		}, datatransfer.ErrPause
	}
	return true, nil, pr.env.SendEvent(channel.dealID, ProviderEventBlockSent, channel.totalSent)
}

// OnPushDataReceived is called on the responder side when more bytes are received
// for a given push request.  It should return a VoucherResult + ErrPause to
// request revalidation or nil to continue uninterrupted,
// other errors will terminate the request
func (pr *ProviderRevalidator) OnPushDataReceived(chid datatransfer.ChannelID, additionalBytesReceived uint64) (bool, datatransfer.VoucherResult, error) {
	return false, nil, nil
}

// OnComplete is called to make a final request for revalidation -- often for the
// purpose of settlement.
// if VoucherResult is non nil, the request will enter a settlement phase awaiting
// a final update
func (pr *ProviderRevalidator) OnComplete(chid datatransfer.ChannelID) (bool, datatransfer.VoucherResult, error) {
	pr.trackedChannelsLk.RLock()
	defer pr.trackedChannelsLk.RUnlock()
	channel, ok := pr.trackedChannels[chid]
	if !ok {
		return false, nil, nil
	}

	err := pr.loadDealState(channel)
	if err != nil {
		return true, nil, err
	}

	err = pr.env.SendEvent(channel.dealID, ProviderEventBlocksCompleted)
	if err != nil {
		return true, nil, err
	}

	paymentOwed := big.Mul(abi.NewTokenAmount(int64(channel.totalSent-channel.totalPaidFor)), channel.pricePerByte)
	if paymentOwed.Equals(big.Zero()) {
		return true, &DealResponse{
			ID:     channel.dealID.DealID,
			Status: DealStatusCompleted,
		}, nil
	}
	err = pr.env.SendEvent(channel.dealID, ProviderEventPaymentRequested, channel.totalSent)
	if err != nil {
		return true, nil, err
	}
	return true, &DealResponse{
		ID:          channel.dealID.DealID,
		Status:      DealStatusFundsNeededLastPayment,
		PaymentOwed: paymentOwed,
	}, datatransfer.ErrPause
}
