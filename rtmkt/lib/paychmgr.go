package rtmkt

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/actors/builtin/paych"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/wallet"
	"github.com/ipfs/go-cid"
)

type paychManager struct {
	ctx    context.Context
	api    api.FullNode
	wallet *wallet.Wallet
}

func (pm *paychManager) StateAccountKey(ctx context.Context, addr address.Address, tip types.TipSetKey) (address.Address, error) {
	return address.TestAddress, nil
}

func (pm *paychManager) StateWaitMsg(ctx context.Context, msg cid.Cid, confidence uint64) (*api.MsgLookup, error) {
	return nil, nil
}

func (pm *paychManager) MpoolPushMessage(ctx context.Context, msg *types.Message, maxFee *api.MessageSendSpec) (*types.SignedMessage, error) {
	return nil, nil
}

func (pm *paychManager) WalletHas(ctx context.Context, addr address.Address) (bool, error) {
	return false, nil
}

func (pm *paychManager) StateNetworkVersion(context.Context, types.TipSetKey) (network.Version, error) {
	return network.Version0, nil
}

func (pm *paychManager) ResolveToKeyAddress(ctx context.Context, addr address.Address, ts *types.TipSet) (address.Address, error) {
	return address.TestAddress, nil
}

func (pm *paychManager) GetPaychState(ctx context.Context, addr address.Address, ts *types.TipSet) (*types.Actor, paych.State, error) {
	return nil, nil, nil
}

func (pm *paychManager) Call(ctx context.Context, msg *types.Message, ts *types.TipSet) (*api.InvocResult, error) {
	return nil, nil
}

func NewPaychManager(ctx context.Context, node api.FullNode, w *wallet.Wallet) *paychManager {
	return &paychManager{ctx, node, w}
	// return &Manager{
	// 	ctx:      ctx,
	// 	shutdown: shutdown,
	// 	store:    store,
	// 	sa:       &stateAccessor{sm: ma},
	// 	channels: make(map[string]*interface{}),
	// 	pchapi:   ma,
	// }

}
