package rtmkt

import (
	"fmt"

	"github.com/filecoin-project/go-address"
	datatransfer "github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-multistore"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/actors/builtin/paych"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

//go:generate cbor-gen-for --map-encoding Query QueryResponse DealProposal DealResponse Params QueryParams DealPayment ProviderDealState PaymentInfo Ask RetrievalPeer ClientDealState

// QueryProtocolID is the protocol for querying information about retrieval
// deal parameters
// let's keep the same as lotus for now
const QueryProtocolID = protocol.ID("/fil/retrieval/qry/1.0.0")

// ClientDealState is the current state of a deal from the point of view
// of a retrieval client
type ClientDealState struct {
	DealProposal
	StoreID              *multistore.StoreID
	ChannelID            datatransfer.ChannelID
	LastPaymentRequested bool
	AllBlocksReceived    bool
	TotalFunds           abi.TokenAmount
	ClientWallet         address.Address
	MinerWallet          address.Address
	PaymentInfo          *PaymentInfo
	Status               DealStatus
	Sender               peer.ID
	TotalReceived        uint64
	Message              string
	BytesPaidFor         uint64
	CurrentInterval      uint64
	PaymentRequested     abi.TokenAmount
	FundsSpent           abi.TokenAmount
	UnsealFundsPaid      abi.TokenAmount
	WaitMsgCID           *cid.Cid // the CID of any message the client deal is waiting for
	VoucherShortfall     abi.TokenAmount
	LegacyProtocol       bool
}

// ProviderDealState is the current state of a deal from the point of view
// of a retrieval provider
type ProviderDealState struct {
	DealProposal
	StoreID         multistore.StoreID
	ChannelID       datatransfer.ChannelID
	Status          DealStatus
	Receiver        peer.ID
	TotalSent       uint64
	FundsReceived   abi.TokenAmount
	Message         string
	CurrentInterval uint64
}

// Identifier provides a unique id for this provider deal
func (pds ProviderDealState) Identifier() ProviderDealIdentifier {
	return ProviderDealIdentifier{Receiver: pds.Receiver, DealID: pds.ID}
}

// ProviderDealIdentifier is a value that uniquely identifies a deal
type ProviderDealIdentifier struct {
	Receiver peer.ID
	DealID   DealID
}

func (p ProviderDealIdentifier) String() string {
	return fmt.Sprintf("%v/%v", p.Receiver, p.DealID)
}

// DealID is an identifier for a retrieval deal (unique to a client)
type DealID uint64

func (d DealID) String() string {
	return fmt.Sprintf("%d", d)
}

// DealProposal is a proposal for a new retrieval deal
type DealProposal struct {
	PayloadCID cid.Cid
	ID         DealID
	Params
}

// Type method makes DealProposal usable as a voucher
func (dp *DealProposal) Type() datatransfer.TypeIdentifier {
	return "RetrievalDealProposal/1"
}

// DealProposalUndefined is an undefined deal proposal
var DealProposalUndefined = DealProposal{}

// Params are the parameters requested for a retrieval deal proposal
type Params struct {
	PricePerByte            abi.TokenAmount
	PaymentInterval         uint64 // when to request payment
	PaymentIntervalIncrease uint64
}

// NewParams generates parameters for a retrieval deal
func NewParams(pricePerByte abi.TokenAmount, paymentInterval uint64, paymentIntervalIncrease uint64) (Params, error) {
	return Params{
		PricePerByte:            pricePerByte,
		PaymentInterval:         paymentInterval,
		PaymentIntervalIncrease: paymentIntervalIncrease,
	}, nil
}

// DealResponse is a response to a retrieval deal proposal
type DealResponse struct {
	Status DealStatus
	ID     DealID

	// payment required to proceed
	PaymentOwed abi.TokenAmount

	Message string
}

// Type method makes DealResponse usable as a voucher result
func (dr *DealResponse) Type() datatransfer.TypeIdentifier {
	return "RetrievalDealResponse/1"
}

// DealResponseUndefined is an undefined deal response
var DealResponseUndefined = DealResponse{}

// DealPayment is a payment for an in progress retrieval deal
type DealPayment struct {
	ID             DealID
	PaymentChannel address.Address
	PaymentVoucher *paych.SignedVoucher
}

// Type method makes DealPayment usable as a voucher
func (dr *DealPayment) Type() datatransfer.TypeIdentifier {
	return "RetrievalDealPayment/1"
}

// DealPaymentUndefined is an undefined deal payment
var DealPaymentUndefined = DealPayment{}

// PaymentInfo is the payment channel and lane for a deal, once it is setup
type PaymentInfo struct {
	PayCh address.Address
	Lane  uint64
}

// Ask is the provider condition for accepting a deal
type Ask struct {
	PricePerByte            abi.TokenAmount
	PaymentInterval         uint64
	PaymentIntervalIncrease uint64
}

// QueryResponseStatus indicates whether a queried piece is available
type QueryResponseStatus uint64

const (
	// QueryResponseAvailable indicates a provider has a piece and is prepared to
	// return it
	QueryResponseAvailable QueryResponseStatus = iota

	// QueryResponseUnavailable indicates a provider either does not have or cannot
	// serve the queried piece to the client
	QueryResponseUnavailable

	// QueryResponseError indicates something went wrong generating a query response
	QueryResponseError
)

// QueryResponse is a miners response to a given retrieval query
type QueryResponse struct {
	Status                     QueryResponseStatus
	Size                       uint64          // Total size of piece in bytes
	PaymentAddress             address.Address // address to send funds to -- may be different than miner addr
	MinPricePerByte            abi.TokenAmount
	MaxPaymentInterval         uint64
	MaxPaymentIntervalIncrease uint64
	Message                    string
}

// QueryResponseUndefined is an empty QueryResponse
var QueryResponseUndefined = QueryResponse{}

// QueryParams - indicate what specific information about a piece that a retrieval
// client is interested in, as well as specific parameters the client is seeking
// for the retrieval deal
type QueryParams struct {
	MaxPricePerByte            abi.TokenAmount // optional, tell miner uninterested if more expensive than this
	MinPaymentInterval         uint64          // optional, tell miner uninterested unless payment interval is greater than this
	MinPaymentIntervalIncrease uint64          // optional, tell miner uninterested unless payment interval increase is greater than this
}

// Query is a query to a given provider to determine information about a piece
// they may have available for retrieval
type Query struct {
	PayloadCID cid.Cid
	QueryParams
}

// QueryUndefined is a query with no values
var QueryUndefined = Query{}

// ChannelAvailableFunds provides information about funds in a channel
type ChannelAvailableFunds struct {
	// ConfirmedAmt is the amount of funds that have been confirmed on-chain
	// for the channel
	ConfirmedAmt abi.TokenAmount
	// PendingAmt is the amount of funds that are pending confirmation on-chain
	PendingAmt abi.TokenAmount
	// PendingWaitSentinel can be used with PaychGetWaitReady to wait for
	// confirmation of pending funds
	PendingWaitSentinel *cid.Cid
	// QueuedAmt is the amount that is queued up behind a pending request
	QueuedAmt abi.TokenAmount
	// VoucherRedeemedAmt is the amount that is redeemed by vouchers on-chain
	// and in the local datastore
	VoucherReedeemedAmt abi.TokenAmount
}

// RetrievalPeer is a provider address/peer.ID pair (everything needed to make
// deals for with a miner)
type RetrievalPeer struct {
	Address  address.Address
	ID       peer.ID
	PieceCID *cid.Cid
}

// ShortfallErorr is an error that indicates a short fall of funds
type ShortfallError struct {
	shortfall abi.TokenAmount
}

// NewShortfallError returns a new error indicating a shortfall of funds
func NewShortfallError(shortfall abi.TokenAmount) error {
	return ShortfallError{shortfall}
}

// Shortfall returns the numerical value of the shortfall
func (se ShortfallError) Shortfall() abi.TokenAmount {
	return se.shortfall
}
func (se ShortfallError) Error() string {
	return fmt.Sprintf("Inssufficient Funds. Shortfall: %s", se.shortfall.String())
}
