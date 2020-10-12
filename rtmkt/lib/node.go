package rtmkt

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/actors/builtin/paych"

	"github.com/filecoin-project/go-fil-markets/shared"
	testnet "github.com/filecoin-project/go-fil-markets/shared_testutil"
	lapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
)

// TestMode disables payments when needed
type TestMode uint64

const (
	// TestModeOff means the payments are enabled as normal
	TestModeOff TestMode = iota
	// TestModeOn means we return mock values
	TestModeOn
)

// RetrievalNode are the node depedencies for a RetrievalAgent
// It is based on lotus RetrievalProviderNode but changed not to rely on miner infra
type RetrievalNode interface {
	GetChainHead(ctx context.Context) (shared.TipSetToken, abi.ChainEpoch, error)

	// GetOrCreatePaymentChannel sets up a new payment channel if one does not exist
	// between a client and a miner and ensures the client has the given amount of funds available in the channel
	GetOrCreatePaymentChannel(ctx context.Context, clientAddress, minerAddress address.Address,
		clientFundsAvailable abi.TokenAmount, tok shared.TipSetToken) (address.Address, cid.Cid, error)

	// CheckAvailableFunds returns the amount of current and incoming funds in a channel
	CheckAvailableFunds(ctx context.Context, paymentChannel address.Address) (ChannelAvailableFunds, error)

	// Allocate late creates a lane within a payment channel so that calls to
	// CreatePaymentVoucher will automatically make vouchers only for the difference
	// in total
	AllocateLane(ctx context.Context, paymentChannel address.Address) (uint64, error)

	// CreatePaymentVoucher creates a new payment voucher in the given lane for a
	// given payment channel so that all the payment vouchers in the lane add up
	// to the given amount (so the payment voucher will be for the difference)
	CreatePaymentVoucher(ctx context.Context, paymentChannel address.Address, amount abi.TokenAmount,
		lane uint64, tok shared.TipSetToken) (*paych.SignedVoucher, error)

	// WaitForPaymentChannelReady just waits for the payment channel's pending operations to complete
	WaitForPaymentChannelReady(ctx context.Context, waitSentinel cid.Cid) (address.Address, error)

	// GetKnownAddresses gets any on known multiaddrs for a given address, so we can add to the peer store
	GetKnownAddresses(ctx context.Context, p RetrievalPeer, tok shared.TipSetToken) ([]ma.Multiaddr, error)

	SavePaymentVoucher(ctx context.Context, paymentChannel address.Address, voucher *paych.SignedVoucher, proof []byte, expectedAmount abi.TokenAmount, tok shared.TipSetToken) (abi.TokenAmount, error)
}

type retrievalNode struct {
	tmode TestMode
	api   lapi.FullNode
	pm    *paychManager
}

// Get the current chain head. Return its TipSetToken and its abi.ChainEpoch.
func (rn *retrievalNode) GetChainHead(ctx context.Context) (shared.TipSetToken, abi.ChainEpoch, error) {
	if rn.tmode == TestModeOn {
		return shared.TipSetToken{}, 0, nil
	}

	head, err := rn.api.ChainHead(ctx)
	if err != nil {
		return nil, 0, err
	}

	return head.Key().Bytes(), head.Height(), nil
}

// GetOrCreatePaymentChannel sets up a new payment channel if one does not exist
// between a client and a miner and ensures the client has the given amount of
// funds available in the channel.
func (rn *retrievalNode) GetOrCreatePaymentChannel(ctx context.Context, clientAddress, minerAddress address.Address, clientFundsAvailable abi.TokenAmount, tok shared.TipSetToken) (address.Address, cid.Cid, error) {
	if rn.tmode == TestModeOn {
		fmt.Println("GetOrCreatePaymentChannel")
		payCh, _ := address.NewActorAddress([]byte("testing"))
		return payCh, testnet.GenerateCids(1)[0], nil
	}
	// TODO: respect the provided TipSetToken (a serialized TipSetKey) when
	// querying the chain
	ci, err := rn.pm.PaychGet(ctx, clientAddress, minerAddress, clientFundsAvailable)
	if err != nil {
		return address.Undef, cid.Undef, err
	}
	return ci.Channel, ci.WaitSentinel, nil
}

// Allocate late creates a lane within a payment channel so that calls to
// CreatePaymentVoucher will automatically make vouchers only for the difference
// in total
func (rn *retrievalNode) AllocateLane(ctx context.Context, paymentChannel address.Address) (uint64, error) {
	if rn.tmode == TestModeOn {
		fmt.Println("AllocateLane")
		return 10, nil
	}
	return rn.pm.PaychAllocateLane(paymentChannel)
}

// CreatePaymentVoucher creates a new payment voucher in the given lane for a
// given payment channel so that all the payment vouchers in the lane add up
// to the given amount (so the payment voucher will be for the difference)
func (rn *retrievalNode) CreatePaymentVoucher(ctx context.Context, paymentChannel address.Address, amount abi.TokenAmount, lane uint64, tok shared.TipSetToken) (*paych.SignedVoucher, error) {
	if rn.tmode == TestModeOn {
		fmt.Println("CreatePaymentVoucher")
		testVoucher := testnet.MakeTestSignedVoucher()
		return testVoucher, nil
	}
	// TODO: respect the provided TipSetToken (a serialized TipSetKey) when
	// querying the chain
	voucher, err := rn.pm.PaychVoucherCreate(ctx, paymentChannel, amount, lane)
	if err != nil {
		fmt.Printf("CreatePaymentVoucher: %v", err)
		return nil, err
	}
	if voucher.Voucher == nil {
		return nil, NewShortfallError(voucher.Shortfall)
	}
	return voucher.Voucher, nil
}

func (rn *retrievalNode) WaitForPaymentChannelReady(ctx context.Context, messageCID cid.Cid) (address.Address, error) {
	if rn.tmode == TestModeOn {
		fmt.Println("WaitForPaymentChannelReady")
		payCh, _ := address.NewActorAddress([]byte("testing"))
		return payCh, nil
	}
	return rn.pm.PaychGetWaitReady(ctx, messageCID)
}

func (rn *retrievalNode) CheckAvailableFunds(ctx context.Context, paymentChannel address.Address) (ChannelAvailableFunds, error) {
	if rn.tmode == TestModeOn {
		fmt.Println("CheckAvailableFunds")
		return ChannelAvailableFunds{
			ConfirmedAmt: abi.NewTokenAmount(10000),
		}, nil
	}
	channelAvailableFunds, err := rn.pm.PaychAvailableFunds(paymentChannel)
	if err != nil {
		return ChannelAvailableFunds{}, err
	}
	return ChannelAvailableFunds{
		ConfirmedAmt:        channelAvailableFunds.ConfirmedAmt,
		PendingAmt:          channelAvailableFunds.PendingAmt,
		PendingWaitSentinel: channelAvailableFunds.PendingWaitSentinel,
		QueuedAmt:           channelAvailableFunds.QueuedAmt,
		VoucherReedeemedAmt: channelAvailableFunds.VoucherReedeemedAmt,
	}, nil
}

func (rn *retrievalNode) GetKnownAddresses(ctx context.Context, p RetrievalPeer, encodedTs shared.TipSetToken) ([]ma.Multiaddr, error) {
	if rn.tmode == TestModeOn {
		fmt.Println("GetKnownAddresses")
		return []ma.Multiaddr{}, nil
	}
	tsk, err := types.TipSetKeyFromBytes(encodedTs)
	if err != nil {
		return nil, err
	}
	mi, err := rn.api.StateMinerInfo(ctx, p.Address, tsk)
	if err != nil {
		return nil, err
	}
	multiaddrs := make([]ma.Multiaddr, 0, len(mi.Multiaddrs))
	for _, a := range mi.Multiaddrs {
		maddr, err := ma.NewMultiaddrBytes(a)
		if err != nil {
			return nil, err
		}
		multiaddrs = append(multiaddrs, maddr)
	}

	return multiaddrs, nil
}

// Get the miner worker address for the given miner owner, as of tok
func (rn *retrievalNode) SavePaymentVoucher(ctx context.Context, paymentChannel address.Address, voucher *paych.SignedVoucher, proof []byte, expectedAmount abi.TokenAmount, tok shared.TipSetToken) (abi.TokenAmount, error) {
	if rn.tmode == TestModeOn {
		fmt.Println("SavePaymentVoucher")
		return expectedAmount, nil
	}
	// TODO: respect the provided TipSetToken (a serialized TipSetKey) when
	// querying the chain
	added, err := rn.pm.AddVoucherInbound(ctx, paymentChannel, voucher, proof, expectedAmount)
	return added, err
}

func NewRetrievalNode(api lapi.FullNode, pm *paychManager, tm TestMode) RetrievalNode {
	return &retrievalNode{tm, api, pm}
}
