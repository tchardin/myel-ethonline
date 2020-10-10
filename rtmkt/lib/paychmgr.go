package rtmkt

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/actors/builtin/paych"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/wallet"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/paychmgr"
	"github.com/ipfs/go-cid"
)

type managerAPI struct {
	ctx    context.Context
	api    api.FullNode
	wallet *wallet.Wallet
}

func (ma *managerAPI) StateAccountKey(ctx context.Context, addr address.Address, tip types.TipSetKey) (address.Address, error) {
	return address.TestAddress, nil
}

func (ma *managerAPI) StateWaitMsg(ctx context.Context, msg cid.Cid, confidence uint64) (*api.MsgLookup, error) {
	return nil, nil
}

func (ma *managerAPI) MpoolPushMessage(ctx context.Context, msg *types.Message, maxFee *api.MessageSendSpec) (*types.SignedMessage, error) {
	return nil, nil
}

func (ma *managerAPI) WalletHas(ctx context.Context, addr address.Address) (bool, error) {
	return false, nil
}

func (ma *managerAPI) StateNetworkVersion(context.Context, types.TipSetKey) (network.Version, error) {
	return network.Version0, nil
}

func (ma *managerAPI) ResolveToKeyAddress(ctx context.Context, addr address.Address, ts *types.TipSet) (address.Address, error) {
	return address.TestAddress, nil
}

func (ma *managerAPI) GetPaychState(ctx context.Context, addr address.Address, ts *types.TipSet) (*types.Actor, paych.State, error) {
	return nil, nil, nil
}

func (ma *managerAPI) Call(ctx context.Context, msg *types.Message, ts *types.TipSet) (*api.InvocResult, error) {
	return nil, nil
}

func NewPaychManager(ctx context.Context, node api.FullNode, w *wallet.Wallet, ds dtypes.MetadataDS) *paychmgr.Manager {
	ma := &managerAPI{ctx, node, w}
	store := paychmgr.NewStore(ds)
	return paychmgr.NewManager(ctx, nil, ma, store, ma)
	// return &Manager{
	// 	ctx:      ctx,
	// 	shutdown: shutdown,
	// 	store:    store,
	// 	sa:       &stateAccessor{sm: ma},
	// 	channels: make(map[string]*interface{}),
	// 	pchapi:   ma,
	// }

}

type stateAccessor struct {
	sm *managerAPI
}

func (ca *stateAccessor) loadPaychActorState(ctx context.Context, ch address.Address) (*types.Actor, paych.State, error) {
	return ca.sm.GetPaychState(ctx, ch, nil)
}

func (ca *stateAccessor) loadStateChannelInfo(ctx context.Context, ch address.Address, dir uint64) (*paychmgr.ChannelInfo, error) {
	_, st, err := ca.loadPaychActorState(ctx, ch)
	if err != nil {
		return nil, err
	}

	// Load channel "From" account actor state
	f, err := st.From()
	if err != nil {
		return nil, err
	}
	from, err := ca.sm.ResolveToKeyAddress(ctx, f, nil)
	if err != nil {
		return nil, err
	}
	t, err := st.To()
	if err != nil {
		return nil, err
	}
	to, err := ca.sm.ResolveToKeyAddress(ctx, t, nil)
	if err != nil {
		return nil, err
	}

	nextLane, err := ca.nextLaneFromState(ctx, st)
	if err != nil {
		return nil, err
	}

	ci := &paychmgr.ChannelInfo{
		Channel:   &ch,
		Direction: dir,
		NextLane:  nextLane,
	}

	if dir == paychmgr.DirOutbound {
		ci.Control = from
		ci.Target = to
	} else {
		ci.Control = to
		ci.Target = from
	}

	return ci, nil
}

func (ca *stateAccessor) nextLaneFromState(ctx context.Context, st paych.State) (uint64, error) {
	laneCount, err := st.LaneCount()
	if err != nil {
		return 0, err
	}
	if laneCount == 0 {
		return 0, nil
	}

	maxID := uint64(0)
	if err := st.ForEachLaneState(func(idx uint64, _ paych.LaneState) error {
		if idx > maxID {
			maxID = idx
		}
		return nil
	}); err != nil {
		return 0, err
	}

	return maxID + 1, nil
}
