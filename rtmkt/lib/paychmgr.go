package rtmkt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/hannahhoward/go-pubsub"

	"github.com/filecoin-project/go-address"
	cborrpc "github.com/filecoin-project/go-cbor-util"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/actors/adt"
	"github.com/filecoin-project/lotus/chain/actors/builtin/paych"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/wallet"
	"github.com/filecoin-project/lotus/lib/sigs"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	paych0 "github.com/filecoin-project/specs-actors/actors/builtin/paych"
	adt0 "github.com/filecoin-project/specs-actors/actors/util/adt"
	init2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/init"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	dsq "github.com/ipfs/go-datastore/query"
	cbor "github.com/ipfs/go-ipld-cbor"
)

type paychManager struct {
	ctx      context.Context
	api      api.FullNode
	wallet   *wallet.Wallet
	store    *channelStore
	actStore *cbor.BasicIpldStore

	lk       sync.RWMutex
	channels map[string]*channelAccessor
}

func (pm *paychManager) accessorByFromTo(from address.Address, to address.Address) (*channelAccessor, error) {
	key := pm.accessorCacheKey(from, to)

	// First take a read lock and check the cache
	pm.lk.RLock()
	ca, ok := pm.channels[key]
	pm.lk.RUnlock()
	if ok {
		return ca, nil
	}

	// Not in cache, so take a write lock
	pm.lk.Lock()
	defer pm.lk.Unlock()

	// Need to check cache again in case it was updated between releasing read
	// lock and taking write lock
	ca, ok = pm.channels[key]
	if !ok {
		// Not in cache, so create a new one and store in cache
		ca = pm.addAccessorToCache(from, to)
	}

	return ca, nil
}

func (pm *paychManager) accessorByAddress(ch address.Address) (*channelAccessor, error) {
	// Get the channel from / to
	pm.lk.RLock()
	channelInfo, err := pm.store.ByAddress(ch)
	pm.lk.RUnlock()
	if err != nil {
		return nil, err
	}

	// TODO: cache by channel address so we can get by address instead of using from / to
	return pm.accessorByFromTo(channelInfo.Control, channelInfo.Target)
}

func (pm *paychManager) accessorCacheKey(from address.Address, to address.Address) string {
	return from.String() + "->" + to.String()
}

func (pm *paychManager) addAccessorToCache(from address.Address, to address.Address) *channelAccessor {
	key := pm.accessorCacheKey(from, to)
	ca := &channelAccessor{
		from:         from,
		to:           to,
		chctx:        pm.ctx,
		api:          pm.api,
		wal:          pm.wallet,
		actStore:     pm.actStore,
		store:        pm.store,
		lk:           &channelLock{globalLock: &pm.lk},
		msgListeners: newMsgListeners(),
	}
	// TODO: Use LRU
	pm.channels[key] = ca
	return ca
}

func NewPaychManager(ctx context.Context, node api.FullNode, w *wallet.Wallet, ds dtypes.MetadataDS, adts *cbor.BasicIpldStore) *paychManager {
	store := NewChannelStore(ds)
	return &paychManager{
		ctx:      ctx,
		api:      node,
		wallet:   w,
		store:    store,
		actStore: adts,
		channels: make(map[string]*channelAccessor),
	}
	// return &Manager{
	// 	ctx:      ctx,
	// 	shutdown: shutdown,
	// 	store:    store,
	// 	sa:       &stateAccessor{sm: ma},
	// 	channels: make(map[string]*interface{}),
	// 	pchapi:   ma,
	// }

}

func (pm *paychManager) PaychGet(ctx context.Context, from, to address.Address, amt types.BigInt) (*api.ChannelInfo, error) {
	chanAccessor, err := pm.accessorByFromTo(from, to)
	if err != nil {
		return nil, fmt.Errorf("Unable to get or create channel accessor: %v", err)
	}
	addr, pcid, err := chanAccessor.getPaych(ctx, amt)
	if err != nil {
		return nil, fmt.Errorf("Unable to get or create pay channel from accessor: %v", err)
	}
	return &api.ChannelInfo{
		Channel:      addr,
		WaitSentinel: pcid,
	}, nil
}

func (pm *paychManager) PaychAllocateLane(ch address.Address) (uint64, error) {
	ca, err := pm.accessorByAddress(ch)
	if err != nil {
		return 0, fmt.Errorf("Unable to find channel to allocate lane: %v", err)
	}
	return ca.allocateLane(ch)
}

func (pm *paychManager) PaychVoucherCreate(ctx context.Context, ch address.Address, amt types.BigInt, lane uint64) (*api.VoucherCreateResult, error) {
	vouch := paych.SignedVoucher{Amount: amt, Lane: lane}
	ca, err := pm.accessorByAddress(ch)
	if err != nil {
		return nil, fmt.Errorf("Unable to find channel to create voucher for: %v", err)
	}

	return ca.createVoucher(ctx, ch, vouch)
}

func (pm *paychManager) PaychGetWaitReady(ctx context.Context, mcid cid.Cid) (address.Address, error) {
	// Find the channel associated with the message CID
	pm.lk.Lock()
	ci, err := pm.store.ByMessageCid(mcid)
	pm.lk.Unlock()

	if err != nil {
		if err == datastore.ErrNotFound {
			return address.Undef, fmt.Errorf("Could not find wait msg cid %s", mcid)
		}
		return address.Undef, err
	}

	chanAccessor, err := pm.accessorByFromTo(ci.Control, ci.Target)
	if err != nil {
		return address.Undef, err
	}

	return chanAccessor.getPaychWaitReady(ctx, mcid)
}

func (pm *paychManager) PaychAvailableFunds(ch address.Address) (*api.ChannelAvailableFunds, error) {
	ca, err := pm.accessorByAddress(ch)
	if err != nil {
		return nil, err
	}

	ci, err := ca.getChannelInfo(ch)
	if err != nil {
		return nil, err
	}

	return ca.availableFunds(ci.ChannelID)
}

func (pm *paychManager) ListChannels() ([]address.Address, error) {
	// Need to take an exclusive lock here so that channel operations can't run
	// in parallel (see channelLock)
	pm.lk.Lock()
	defer pm.lk.Unlock()

	return pm.store.ListChannels()
}

func (pm *paychManager) ListVouchers(ctx context.Context, ch address.Address) ([]*VoucherInfo, error) {
	ca, err := pm.accessorByAddress(ch)
	if err != nil {
		return nil, err
	}
	return ca.listVouchers(ctx, ch)
}

func (pm *paychManager) RedeemAll(ctx context.Context) error {
	chs, err := pm.ListChannels()
	if err != nil {
		return err
	}
	for _, ch := range chs {
		bestByLane, err := pm.BestSpendableByLane(ctx, ch)
		if err != nil {
			fmt.Printf("Error checking spendable vouchers: %v", err)
			continue
		}

		var wg sync.WaitGroup
		wg.Add(len(bestByLane) + 1)
		// Add an extra Settle call in case the channel hasn't been settled yet
		// it's ok if it fails because it may already be settled
		mcid, err := pm.Settle(ctx, ch)
		if err != nil {
			fmt.Printf("Unable to call Settle on pch: %v", err)
		}
		go func(msgCid cid.Cid) {
			defer wg.Done()
			msgLookup, err := pm.api.StateWaitMsg(ctx, msgCid, build.MessageConfidence)
			if err != nil {
				fmt.Printf("Error waiting for pch to settle: %v", err)
			}
			if msgLookup.Receipt.ExitCode != 0 {
				fmt.Printf("Failed to collect pch - ExitCode: %v", msgLookup.Receipt.ExitCode)
			}
		}(mcid)

		for _, voucher := range bestByLane {
			msgCid, err := pm.SubmitVoucher(ctx, ch, voucher, nil, nil)
			if err != nil {
				fmt.Printf("Unable to submit voucher: %v", err)
				continue
			}
			go func(voucher *paych.SignedVoucher, submitMessageCID cid.Cid) {
				defer wg.Done()
				msgLookup, err := pm.api.StateWaitMsg(ctx, submitMessageCID, build.MessageConfidence)
				if err != nil {
					fmt.Printf("Error waiting for voucher submit: %v", err)
				}
				if msgLookup.Receipt.ExitCode != 0 {
					fmt.Printf("Failed to submit voucher - ExitCode: %v", msgLookup.Receipt.ExitCode)
				}
			}(voucher, msgCid)
		}
		wg.Wait()
	}
	return nil
}

func (pm *paychManager) BestSpendableByLane(ctx context.Context, ch address.Address) (map[uint64]*paych.SignedVoucher, error) {
	vouchers, err := pm.ListVouchers(ctx, ch)
	if err != nil {
		return nil, err
	}

	bestByLane := make(map[uint64]*paych.SignedVoucher)
	for _, vi := range vouchers {
		// TODO: spendable, err := PaychVoucherCheckSpendable(ctx, ch, voucher, nil, nil)
		if bestByLane[vi.Voucher.Lane] == nil || vi.Voucher.Amount.GreaterThan(bestByLane[vi.Voucher.Lane].Amount) {
			bestByLane[vi.Voucher.Lane] = vi.Voucher
		}
	}
	return bestByLane, nil
}

func (pm *paychManager) SubmitVoucher(ctx context.Context, ch address.Address, sv *paych.SignedVoucher, secret []byte, proof []byte) (cid.Cid, error) {
	if len(proof) > 0 {
		return cid.Undef, errProofNotSupported
	}
	ca, err := pm.accessorByAddress(ch)
	if err != nil {
		return cid.Undef, err
	}
	return ca.submitVoucher(ctx, ch, sv, secret)
}

func (pm *paychManager) Settle(ctx context.Context, addr address.Address) (cid.Cid, error) {
	ca, err := pm.accessorByAddress(addr)
	if err != nil {
		return cid.Undef, err
	}
	return ca.settle(ctx, addr)
}

func (pm *paychManager) Collect(ctx context.Context, addr address.Address) (cid.Cid, error) {
	ca, err := pm.accessorByAddress(addr)
	if err != nil {
		return cid.Undef, err
	}
	return ca.collect(ctx, addr)
}

var errProofNotSupported = fmt.Errorf("payment channel proof parameter is not supported")

// AddVoucherInbound adds a voucher for an inbound channel.
// If the channel is not in the store, fetches the channel from state (and checks that
// the channel To address is owned by the wallet).
func (pm *paychManager) AddVoucherInbound(ctx context.Context, ch address.Address, sv *paych.SignedVoucher, proof []byte, minDelta types.BigInt) (types.BigInt, error) {
	if len(proof) > 0 {
		return types.NewInt(0), errProofNotSupported
	}
	// Get an accessor for the channel, creating it from state if necessary
	ca, err := pm.inboundChannelAccessor(ctx, ch)
	if err != nil {
		return types.BigInt{}, err
	}
	ca.lk.Lock()
	defer ca.lk.Unlock()
	return ca.addVoucherUnlocked(ctx, ch, sv, minDelta)
}

// inboundChannelAccessor gets an accessor for the given channel. The channel
// must either exist in the store, or be an inbound channel that can be created
// from state.
func (pm *paychManager) inboundChannelAccessor(ctx context.Context, ch address.Address) (*channelAccessor, error) {
	// Make sure channel is in store, or can be fetched from state, and that
	// the channel To address is owned by the wallet
	ci, err := pm.trackInboundChannel(ctx, ch)
	if err != nil {
		return nil, err
	}

	// This is an inbound channel, so To is the Control address (this node)
	from := ci.Target
	to := ci.Control
	return pm.accessorByFromTo(from, to)
}

func (pm *paychManager) trackInboundChannel(ctx context.Context, ch address.Address) (*ChannelInfo, error) {
	// Need to take an exclusive lock here so that channel operations can't run
	// in parallel (see channelLock)
	pm.lk.Lock()
	defer pm.lk.Unlock()

	// Check if channel is in store
	ci, err := pm.store.ByAddress(ch)
	if err == nil {
		// Channel is in store, so it's already being tracked
		return ci, nil
	}

	// If there's an error (besides channel not in store) return err
	if err != ErrChannelNotTracked {
		return nil, err
	}

	// Channel is not in store, so get channel from state
	stateCi, err := pm.loadStateChannelInfo(ch, DirInbound)
	if err != nil {
		return nil, err
	}

	// Check that channel To address is in wallet
	to := stateCi.Control // Inbound channel so To addr is Control (this node)
	toKey, err := pm.api.StateAccountKey(ctx, to, types.EmptyTSK)
	if err != nil {
		return nil, err
	}
	has, err := pm.wallet.HasKey(toKey)
	if err != nil {
		return nil, err
	}
	if !has {
		msg := "cannot add voucher for channel %s: wallet does not have key for address %s"
		return nil, fmt.Errorf(msg, ch, to)
	}

	// Save channel to store
	return pm.store.TrackChannel(stateCi)
}

func (pm *paychManager) loadStateChannelInfo(ch address.Address, dir uint64) (*ChannelInfo, error) {
	ca := &channelAccessor{
		chctx:        pm.ctx,
		api:          pm.api,
		wal:          pm.wallet,
		actStore:     pm.actStore,
		store:        pm.store,
		lk:           &channelLock{globalLock: &pm.lk},
		msgListeners: newMsgListeners(),
	}
	_, as, err := ca.loadPaychActorState(ch)
	if err != nil {
		return nil, err
	}
	f, err := as.From()
	if err != nil {
		return nil, err
	}
	from, err := pm.api.StateAccountKey(pm.ctx, f, types.EmptyTSK)
	if err != nil {
		return nil, fmt.Errorf("Unable to get account key for From actor state: %v", err)
	}
	t, err := as.To()
	if err != nil {
		return nil, err
	}
	to, err := pm.api.StateAccountKey(pm.ctx, t, types.EmptyTSK)
	if err != nil {
		return nil, fmt.Errorf("Unable to get account key for To actor state: %v", err)
	}
	nextLane, err := nextLaneFromState(as)
	if err != nil {
		return nil, fmt.Errorf("Unable to get next lane from state: %v", err)
	}
	ci := &ChannelInfo{
		Channel:   &ch,
		Direction: dir,
		NextLane:  nextLane,
	}
	if dir == DirOutbound {
		ci.Control = from
		ci.Target = to
	} else {
		ci.Control = to
		ci.Target = from
	}
	return ci, nil

}

func nextLaneFromState(st paych.State) (uint64, error) {
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

type channelAccessor struct {
	from          address.Address
	to            address.Address
	chctx         context.Context
	api           api.FullNode
	wal           *wallet.Wallet
	actStore      *cbor.BasicIpldStore
	store         *channelStore
	lk            *channelLock
	fundsReqQueue []*fundsReq
	msgListeners  msgListeners
}

// paychFundsRes is the response to a create channel or add funds request
type paychFundsRes struct {
	channel address.Address
	mcid    cid.Cid
	err     error
}

// fundsReq is a request to create a channel or add funds to a channel
type fundsReq struct {
	ctx     context.Context
	promise chan *paychFundsRes
	amt     types.BigInt

	lk sync.Mutex
	// merge parent, if this req is part of a merge
	merge *mergedFundsReq
	// whether the req's context has been cancelled
	active bool
}

func newFundsReq(ctx context.Context, amt types.BigInt) *fundsReq {
	promise := make(chan *paychFundsRes)
	return &fundsReq{
		ctx:     ctx,
		promise: promise,
		amt:     amt,
		active:  true,
	}
}

// onComplete is called when the funds request has been executed
func (r *fundsReq) onComplete(res *paychFundsRes) {
	select {
	case <-r.ctx.Done():
	case r.promise <- res:
	}
}

// cancel is called when the req's context is cancelled
func (r *fundsReq) cancel() {
	r.lk.Lock()

	r.active = false
	m := r.merge

	r.lk.Unlock()

	// If there's a merge parent, tell the merge parent to check if it has any
	// active reqs left
	if m != nil {
		m.checkActive()
	}
}

// isActive indicates whether the req's context has been cancelled
func (r *fundsReq) isActive() bool {
	r.lk.Lock()
	defer r.lk.Unlock()

	return r.active
}

// setMergeParent sets the merge that this req is part of
func (r *fundsReq) setMergeParent(m *mergedFundsReq) {
	r.lk.Lock()
	defer r.lk.Unlock()

	r.merge = m
}

func (ca *channelAccessor) allocateLane(ch address.Address) (uint64, error) {
	ca.lk.Lock()
	defer ca.lk.Unlock()

	return ca.store.AllocateLane(ch)
}

// mergedFundsReq merges together multiple add funds requests that are queued
// up, so that only one message is sent for all the requests (instead of one
// message for each request)
type mergedFundsReq struct {
	ctx    context.Context
	cancel context.CancelFunc
	reqs   []*fundsReq
}

func newMergedFundsReq(reqs []*fundsReq) *mergedFundsReq {
	ctx, cancel := context.WithCancel(context.Background())
	m := &mergedFundsReq{
		ctx:    ctx,
		cancel: cancel,
		reqs:   reqs,
	}

	for _, r := range m.reqs {
		r.setMergeParent(m)
	}

	// If the requests were all cancelled while being added, cancel the context
	// immediately
	m.checkActive()

	return m
}

// Called when a fundsReq is cancelled
func (m *mergedFundsReq) checkActive() {
	// Check if there are any active fundsReqs
	for _, r := range m.reqs {
		if r.isActive() {
			return
		}
	}

	// If all fundsReqs have been cancelled, cancel the context
	m.cancel()
}

// onComplete is called when the queue has executed the mergeFundsReq.
// Calls onComplete on each fundsReq in the mergeFundsReq.
func (m *mergedFundsReq) onComplete(res *paychFundsRes) {
	for _, r := range m.reqs {
		if r.isActive() {
			r.onComplete(res)
		}
	}
}

// sum is the sum of the amounts in all requests in the merge
func (m *mergedFundsReq) sum() types.BigInt {
	sum := types.NewInt(0)
	for _, r := range m.reqs {
		if r.isActive() {
			sum = types.BigAdd(sum, r.amt)
		}
	}
	return sum
}

// getPaych ensures that a channel exists between the from and to addresses,
// and adds the given amount of funds.
// If the channel does not exist a create channel message is sent and the
// message CID is returned.
// If the channel does exist an add funds message is sent and both the channel
// address and message CID are returned.
// If there is an in progress operation (create channel / add funds), getPaych
// blocks until the previous operation completes, then returns both the channel
// address and the CID of the new add funds message.
// If an operation returns an error, subsequent waiting operations will still
// be attempted.
func (ca *channelAccessor) getPaych(ctx context.Context, amt types.BigInt) (address.Address, cid.Cid, error) {
	// Add the request to add funds to a queue and wait for the result
	freq := newFundsReq(ctx, amt)
	ca.enqueue(freq)
	select {
	case res := <-freq.promise:
		return res.channel, res.mcid, res.err
	case <-ctx.Done():
		freq.cancel()
		return address.Undef, cid.Undef, ctx.Err()
	}
}

// Queue up an add funds operation
func (ca *channelAccessor) enqueue(task *fundsReq) {
	ca.lk.Lock()
	defer ca.lk.Unlock()

	ca.fundsReqQueue = append(ca.fundsReqQueue, task)
	go ca.processQueue("") // nolint: errcheck
}

// Run the operations in the queue
func (ca *channelAccessor) processQueue(channelID string) (*api.ChannelAvailableFunds, error) {
	ca.lk.Lock()
	defer ca.lk.Unlock()

	// Remove cancelled requests
	ca.filterQueue()

	// If there's nothing in the queue, bail out
	if len(ca.fundsReqQueue) == 0 {
		return ca.currentAvailableFunds(channelID, types.NewInt(0))
	}

	// Merge all pending requests into one.
	// For example if there are pending requests for 3, 2, 4 then
	// amt = 3 + 2 + 4 = 9
	merged := newMergedFundsReq(ca.fundsReqQueue[:])
	amt := merged.sum()
	if amt.IsZero() {
		// Note: The amount can be zero if requests are cancelled as we're
		// building the mergedFundsReq
		return ca.currentAvailableFunds(channelID, amt)
	}

	res := ca.processTask(merged.ctx, amt)

	// If the task is waiting on an external event (eg something to appear on
	// chain) it will return nil
	if res == nil {
		// Stop processing the fundsReqQueue and wait. When the event occurs it will
		// call processQueue() again
		return ca.currentAvailableFunds(channelID, amt)
	}

	// Finished processing so clear the queue
	ca.fundsReqQueue = nil

	// Call the task callback with its results
	merged.onComplete(res)

	return ca.currentAvailableFunds(channelID, types.NewInt(0))
}

func (ca *channelAccessor) currentAvailableFunds(channelID string, queuedAmt types.BigInt) (*api.ChannelAvailableFunds, error) {
	if len(channelID) == 0 {
		return nil, nil
	}

	channelInfo, err := ca.store.ByChannelID(channelID)
	if err != nil {
		return nil, err
	}

	// The channel may have a pending create or add funds message
	waitSentinel := channelInfo.CreateMsg
	if waitSentinel == nil {
		waitSentinel = channelInfo.AddFundsMsg
	}

	// Get the total amount redeemed by vouchers.
	// This includes vouchers that have been submitted, and vouchers that are
	// in the datastore but haven't yet been submitted.
	totalRedeemed := types.NewInt(0)
	if channelInfo.Channel != nil {
		ch := *channelInfo.Channel
		_, as, err := ca.loadPaychActorState(ch)
		if err != nil {
			return nil, err
		}

		laneStates, err := ca.laneState(as, ch)
		if err != nil {
			return nil, err
		}

		for _, ls := range laneStates {
			r, err := ls.Redeemed()
			if err != nil {
				return nil, err
			}
			totalRedeemed = types.BigAdd(totalRedeemed, r)
		}
	}

	return &api.ChannelAvailableFunds{
		Channel:             channelInfo.Channel,
		From:                channelInfo.from(),
		To:                  channelInfo.to(),
		ConfirmedAmt:        channelInfo.Amount,
		PendingAmt:          channelInfo.PendingAmount,
		PendingWaitSentinel: waitSentinel,
		QueuedAmt:           queuedAmt,
		VoucherReedeemedAmt: totalRedeemed,
	}, nil
}

type state0 struct {
	paych0.State
	store adt.Store
	lsAmt *adt0.Array
}

// Channel owner, who has funded the actor
func (s *state0) From() (address.Address, error) {
	return s.State.From, nil
}

// Recipient of payouts from channel
func (s *state0) To() (address.Address, error) {
	return s.State.To, nil
}

// Height at which the channel can be `Collected`
func (s *state0) SettlingAt() (abi.ChainEpoch, error) {
	return s.State.SettlingAt, nil
}

// Amount successfully redeemed through the payment channel, paid out on `Collect()`
func (s *state0) ToSend() (abi.TokenAmount, error) {
	return s.State.ToSend, nil
}

func (s *state0) getOrLoadLsAmt() (*adt0.Array, error) {
	if s.lsAmt != nil {
		return s.lsAmt, nil
	}

	// Get the lane state from the chain
	lsamt, err := adt0.AsArray(s.store, s.State.LaneStates)
	if err != nil {
		return nil, err
	}

	s.lsAmt = lsamt
	return lsamt, nil
}

// Get total number of lanes
func (s *state0) LaneCount() (uint64, error) {
	lsamt, err := s.getOrLoadLsAmt()
	if err != nil {
		return 0, err
	}
	return lsamt.Length(), nil
}

// Iterate lane states
func (s *state0) ForEachLaneState(cb func(idx uint64, dl paych.LaneState) error) error {
	// Get the lane state from the chain
	lsamt, err := s.getOrLoadLsAmt()
	if err != nil {
		return err
	}

	// Note: we use a map instead of an array to store laneStates because the
	// client sets the lane ID (the index) and potentially they could use a
	// very large index.
	var ls paych0.LaneState
	return lsamt.ForEach(&ls, func(i int64) error {
		return cb(uint64(i), &laneState0{ls})
	})
}

type laneState0 struct {
	paych0.LaneState
}

func (ls *laneState0) Redeemed() (big.Int, error) {
	return ls.LaneState.Redeemed, nil
}

func (ls *laneState0) Nonce() (uint64, error) {
	return ls.LaneState.Nonce, nil
}

func (ca *channelAccessor) loadPaychActorState(ch address.Address) (*types.Actor, paych.State, error) {
	actor, err := ca.api.StateGetActor(ca.chctx, ch, types.EmptyTSK)
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to get actor: %v", err)
	}
	actorState, err := ca.api.StateReadState(ca.chctx, ch, types.EmptyTSK)
	stateEncod, err := json.Marshal(actorState.State)

	adtStore := adt0.WrapStore(ca.chctx, ca.actStore)
	state := state0{store: adtStore}
	err = json.Unmarshal(stateEncod, &state)
	if err != nil {
		return nil, nil, fmt.Errorf("Error parsing actor state: %v", err)
	}

	raw, err := ca.api.ChainReadObj(ca.chctx, state.LaneStates)
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to read lane states from chain: %v", err)
	}
	block, err := blocks.NewBlockWithCid(raw, state.LaneStates)
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to make a block with obj: %v", err)
	}
	err = ca.actStore.Blocks.Put(block)
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to set block in store: %v", err)
	}

	return actor, &state, nil
}

type laneState struct {
	redeemed big.Int
	nonce    uint64
}

func (ls laneState) Redeemed() (big.Int, error) {
	return ls.redeemed, nil
}

func (ls laneState) Nonce() (uint64, error) {
	return ls.nonce, nil
}

// laneState gets the LaneStates from chain, then applies all vouchers in
// the data store over the chain state
func (ca *channelAccessor) laneState(state paych.State, ch address.Address) (map[uint64]paych.LaneState, error) {
	// TODO: we probably want to call UpdateChannelState with all vouchers to be fully correct
	//  (but technically dont't need to)

	laneCount, err := state.LaneCount()
	if err != nil {
		return nil, err
	}

	// Note: we use a map instead of an array to store laneStates because the
	// client sets the lane ID (the index) and potentially they could use a
	// very large index.
	laneStates := make(map[uint64]paych.LaneState, laneCount)
	err = state.ForEachLaneState(func(idx uint64, ls paych.LaneState) error {
		laneStates[idx] = ls
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Apply locally stored vouchers
	vouchers, err := ca.store.VouchersForPaych(ch)
	if err != nil && err != ErrChannelNotTracked {
		return nil, err
	}

	for _, v := range vouchers {
		for range v.Voucher.Merges {
			return nil, fmt.Errorf("paych merges not handled yet")
		}

		// Check if there is an existing laneState in the payment channel
		// for this voucher's lane
		ls, ok := laneStates[v.Voucher.Lane]

		// If the voucher does not have a higher nonce than the existing
		// laneState for this lane, ignore it
		if ok {
			n, err := ls.Nonce()
			if err != nil {
				return nil, err
			}
			if v.Voucher.Nonce < n {
				continue
			}
		}

		// Voucher has a higher nonce, so replace laneState with this voucher
		laneStates[v.Voucher.Lane] = laneState{v.Voucher.Amount, v.Voucher.Nonce}
	}

	return laneStates, nil
}

// filterQueue filters cancelled requests out of the queue
func (ca *channelAccessor) filterQueue() {
	if len(ca.fundsReqQueue) == 0 {
		return
	}

	// Remove cancelled requests
	i := 0
	for _, r := range ca.fundsReqQueue {
		if r.isActive() {
			ca.fundsReqQueue[i] = r
			i++
		}
	}

	// Allow GC of remaining slice elements
	for rem := i; rem < len(ca.fundsReqQueue); rem++ {
		ca.fundsReqQueue[i] = nil
	}

	// Resize slice
	ca.fundsReqQueue = ca.fundsReqQueue[:i]
}

// queueSize is the size of the funds request queue (used by tests)
func (ca *channelAccessor) queueSize() int {
	ca.lk.Lock()
	defer ca.lk.Unlock()

	return len(ca.fundsReqQueue)
}

// msgWaitComplete is called when the message for a previous task is confirmed
// or there is an error.
func (ca *channelAccessor) msgWaitComplete(mcid cid.Cid, err error) {
	ca.lk.Lock()
	defer ca.lk.Unlock()

	// Save the message result to the store
	dserr := ca.store.SaveMessageResult(mcid, err)
	if dserr != nil {
		fmt.Printf("saving message result: %s", dserr)
	}

	// Inform listeners that the message has completed
	ca.msgListeners.fireMsgComplete(mcid, err)

	// The queue may have been waiting for msg completion to proceed, so
	// process the next queue item
	if len(ca.fundsReqQueue) > 0 {
		go ca.processQueue("") // nolint: errcheck
	}
}

// processTask checks the state of the channel and takes appropriate action
// (see description of getPaych).
// Note that processTask may be called repeatedly in the same state, and should
// return nil if there is no state change to be made (eg when waiting for a
// message to be confirmed on chain)
func (ca *channelAccessor) processTask(ctx context.Context, amt types.BigInt) *paychFundsRes {
	// Get the payment channel for the from/to addresses.
	// Note: It's ok if we get ErrChannelNotTracked. It just means we need to
	// create a channel.
	channelInfo, err := ca.store.OutboundActiveByFromTo(ca.from, ca.to)
	if err != nil && err != ErrChannelNotTracked {
		return &paychFundsRes{err: err}
	}

	// If a channel has not yet been created, create one.
	if channelInfo == nil {
		mcid, err := ca.createPaych(ctx, amt)
		if err != nil {
			return &paychFundsRes{err: err}
		}

		return &paychFundsRes{mcid: mcid}
	}

	// If the create channel message has been sent but the channel hasn't
	// been created on chain yet
	if channelInfo.CreateMsg != nil {
		// Wait for the channel to be created before trying again
		return nil
	}

	// If an add funds message was sent to the chain but hasn't been confirmed
	// on chain yet
	if channelInfo.AddFundsMsg != nil {
		// Wait for the add funds message to be confirmed before trying again
		return nil
	}

	// We need to add more funds, so send an add funds message to
	// cover the amount for this request
	mcid, err := ca.addFunds(ctx, channelInfo, amt)
	if err != nil {
		return &paychFundsRes{err: err}
	}
	return &paychFundsRes{channel: *channelInfo.Channel, mcid: *mcid}
}

func (ca *channelAccessor) messageBuilder(ctx context.Context, from address.Address) (paych.MessageBuilder, error) {
	nwVersion, err := ca.api.StateNetworkVersion(ctx, types.EmptyTSK)
	if err != nil {
		return nil, err
	}

	return paych.Message(actors.VersionForNetwork(nwVersion), from), nil
}

func (ca *channelAccessor) createPaych(ctx context.Context, amt types.BigInt) (cid.Cid, error) {
	mb, err := ca.messageBuilder(ctx, ca.from)
	if err != nil {
		return cid.Undef, err
	}

	msg, err := mb.Create(ca.to, amt)
	cp := *msg
	msg = &cp
	if err != nil {
		return cid.Undef, err
	}

	smsg, err := ca.mpoolPush(ctx, msg)
	if err != nil {
		return cid.Undef, err
	}

	ci, err := ca.store.CreateChannel(ca.from, ca.to, smsg.Cid(), amt)
	if err != nil {
		return cid.Undef, fmt.Errorf("Unable to create channel in store: %v", err)
	}

	// Wait for the channel to be created on chain
	go ca.waitForPaychCreateMsg(ci.ChannelID, smsg.Cid())

	return smsg.Cid(), nil
}

// waitForPaychCreateMsg waits for mcid to appear on chain and stores the robust address of the
// created payment channel
func (ca *channelAccessor) waitForPaychCreateMsg(channelID string, mcid cid.Cid) {
	err := ca.waitPaychCreateMsg(channelID, mcid)
	ca.msgWaitComplete(mcid, err)
}

func (ca *channelAccessor) waitPaychCreateMsg(channelID string, mcid cid.Cid) error {
	mwait, err := ca.api.StateWaitMsg(ca.chctx, mcid, build.MessageConfidence)
	if err != nil {
		fmt.Printf("wait msg: %v", err)
		return err
	}

	// If channel creation failed
	if mwait.Receipt.ExitCode != 0 {
		ca.lk.Lock()
		defer ca.lk.Unlock()

		// Channel creation failed, so remove the channel from the datastore
		dserr := ca.store.RemoveChannel(channelID)
		if dserr != nil {
			fmt.Printf("failed to remove channel %s: %s", channelID, dserr)
		}

		// Exit code 7 means out of gas
		err := fmt.Errorf("payment channel creation failed (exit code %d)", mwait.Receipt.ExitCode)
		fmt.Printf("Error: %v", err)
		return err
	}

	// TODO: ActorUpgrade abstract over this.
	// This "works" because it hasn't changed from v0 to v2, but we still
	// need an abstraction here.
	var decodedReturn init2.ExecReturn
	err = decodedReturn.UnmarshalCBOR(bytes.NewReader(mwait.Receipt.Return))
	if err != nil {
		fmt.Printf("Error decoding Receipt: %v", err)
		return err
	}

	ca.lk.Lock()
	defer ca.lk.Unlock()

	// Store robust address of channel
	ca.mutateChannelInfo(channelID, func(channelInfo *ChannelInfo) {
		channelInfo.Channel = &decodedReturn.RobustAddress
		channelInfo.Amount = channelInfo.PendingAmount
		channelInfo.PendingAmount = big.NewInt(0)
		channelInfo.CreateMsg = nil
	})

	return nil
}

// mpoolPush preps and sends a message to mpool
func (ca *channelAccessor) mpoolPush(ctx context.Context, msg *types.Message) (*types.SignedMessage, error) {
	// TODO: See if we can make gas estimate method work properly sometime
	msg.GasLimit = int64(8264670)

	msg, err := ca.api.GasEstimateMessageGas(ctx, msg, nil, types.EmptyTSK)
	if err != nil {
		return nil, err
	}
	// TODO save nonce from local data store
	nonce, err := ca.api.MpoolGetNonce(ctx, msg.From)
	if err != nil {
		return nil, err
	}

	msg.Nonce = nonce
	mbl, err := msg.ToStorageBlock()
	if err != nil {
		return nil, err
	}

	sig, err := ca.wal.Sign(ctx, msg.From, mbl.Cid().Bytes())
	if err != nil {
		return nil, err
	}

	smsg := &types.SignedMessage{
		Message:   *msg,
		Signature: *sig,
	}

	if _, err := ca.api.MpoolPush(ctx, smsg); err != nil {
		return nil, fmt.Errorf("MpoolPush failed with error: %v", err)
	}
	return smsg, nil
}

// addFunds sends a message to add funds to the channel and returns the message cid
func (ca *channelAccessor) addFunds(ctx context.Context, channelInfo *ChannelInfo, amt types.BigInt) (*cid.Cid, error) {
	msg := &types.Message{
		To:     *channelInfo.Channel,
		From:   channelInfo.Control,
		Value:  amt,
		Method: 0,
	}

	smsg, err := ca.mpoolPush(ctx, msg)
	if err != nil {
		return nil, err
	}
	mcid := smsg.Cid()

	// Store the add funds message CID on the channel
	ca.mutateChannelInfo(channelInfo.ChannelID, func(ci *ChannelInfo) {
		ci.PendingAmount = amt
		ci.AddFundsMsg = &mcid
	})

	// Store a reference from the message CID to the channel, so that we can
	// look up the channel from the message CID
	err = ca.store.SaveNewMessage(channelInfo.ChannelID, mcid)
	if err != nil {
		fmt.Printf("saving add funds message CID %s: %s", mcid, err)
	}

	go ca.waitForAddFundsMsg(channelInfo.ChannelID, mcid)

	return &mcid, nil
}

// Change the state of the channel in the store
func (ca *channelAccessor) mutateChannelInfo(channelID string, mutate func(*ChannelInfo)) {
	channelInfo, err := ca.store.ByChannelID(channelID)

	// If there's an error reading or writing to the store just log an error.
	// For now we're assuming it's unlikely to happen in practice.
	// Later we may want to implement a transactional approach, whereby
	// we record to the store that we're going to send a message, send
	// the message, and then record that the message was sent.
	if err != nil {
		fmt.Printf("Error reading channel info from store: %s", err)
		return
	}

	mutate(channelInfo)

	err = ca.store.putChannelInfo(channelInfo)
	if err != nil {
		fmt.Printf("Error writing channel info to store: %s", err)
	}
}

// waitForAddFundsMsg waits for mcid to appear on chain and returns error, if any
func (ca *channelAccessor) waitForAddFundsMsg(channelID string, mcid cid.Cid) {
	err := ca.waitAddFundsMsg(channelID, mcid)
	ca.msgWaitComplete(mcid, err)
}

func (ca *channelAccessor) waitAddFundsMsg(channelID string, mcid cid.Cid) error {
	mwait, err := ca.api.StateWaitMsg(ca.chctx, mcid, build.MessageConfidence)
	if err != nil {
		fmt.Printf("Error waiting for chain message: %v", err)
		return err
	}

	if mwait.Receipt.ExitCode != 0 {
		err := fmt.Errorf("voucher channel creation failed: adding funds (exit code %d)", mwait.Receipt.ExitCode)
		fmt.Printf("Error: %v", err)

		ca.lk.Lock()
		defer ca.lk.Unlock()

		ca.mutateChannelInfo(channelID, func(channelInfo *ChannelInfo) {
			channelInfo.PendingAmount = big.NewInt(0)
			channelInfo.AddFundsMsg = nil
		})

		return err
	}

	ca.lk.Lock()
	defer ca.lk.Unlock()

	// Store updated amount
	ca.mutateChannelInfo(channelID, func(channelInfo *ChannelInfo) {
		channelInfo.Amount = types.BigAdd(channelInfo.Amount, channelInfo.PendingAmount)
		channelInfo.PendingAmount = big.NewInt(0)
		channelInfo.AddFundsMsg = nil
	})

	return nil
}

// createVoucher creates a voucher with the given specification, setting its
// nonce, signing the voucher and storing it in the local datastore.
// If there are not enough funds in the channel to create the voucher, returns
// the shortfall in funds.
func (ca *channelAccessor) createVoucher(ctx context.Context, ch address.Address, voucher paych.SignedVoucher) (*api.VoucherCreateResult, error) {
	ca.lk.Lock()
	defer ca.lk.Unlock()

	// Find the channel for the voucher
	ci, err := ca.store.ByAddress(ch)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel info by address: %v", err)
	}

	// Set the voucher channel
	sv := &voucher
	sv.ChannelAddr = ch

	// Get the next nonce on the given lane
	sv.Nonce = ca.nextNonceForLane(ci, voucher.Lane)

	// Sign the voucher
	vb, err := sv.SigningBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get voucher signing bytes: %v", err)
	}

	sig, err := ca.wal.Sign(ctx, ci.Control, vb)
	if err != nil {
		return nil, fmt.Errorf("failed to sign voucher: %v", err)
	}
	sv.Signature = sig

	// Store the voucher
	if _, err := ca.addVoucherUnlocked(ctx, ch, sv, types.NewInt(0)); err != nil {
		fmt.Printf("Error storing voucher: %v", err)
		// If there are not enough funds in the channel to cover the voucher,
		// return a voucher create result with the shortfall
		if ife, ok := err.(insufficientFundsErr); ok {
			return &api.VoucherCreateResult{
				Shortfall: ife.Shortfall(),
			}, nil
		}

		return nil, fmt.Errorf("failed to persist voucher: %v", err)
	}

	return &api.VoucherCreateResult{Voucher: sv, Shortfall: types.NewInt(0)}, nil
}

func (ca *channelAccessor) nextNonceForLane(ci *ChannelInfo, lane uint64) uint64 {
	var maxnonce uint64
	for _, v := range ci.Vouchers {
		if v.Voucher.Lane == lane {
			if v.Voucher.Nonce > maxnonce {
				maxnonce = v.Voucher.Nonce
			}
		}
	}

	return maxnonce + 1
}

func (ca *channelAccessor) addVoucherUnlocked(ctx context.Context, ch address.Address, sv *paych.SignedVoucher, minDelta types.BigInt) (types.BigInt, error) {
	ci, err := ca.store.ByAddress(ch)
	if err != nil {
		return types.BigInt{}, err
	}

	// Check if the voucher has already been added
	for _, v := range ci.Vouchers {
		eq, err := cborrpc.Equals(sv, v.Voucher)
		if err != nil {
			return types.BigInt{}, err
		}
		if eq {
			// Ignore the duplicate voucher.
			fmt.Println("AddVoucher: voucher re-added")
			return types.NewInt(0), nil
		}

	}

	// Check voucher validity
	laneStates, err := ca.checkVoucherValidUnlocked(ctx, ch, sv)
	if err != nil {
		return types.NewInt(0), err
	}

	// The change in value is the delta between the voucher amount and
	// the highest previous voucher amount for the lane
	laneState, exists := laneStates[sv.Lane]
	redeemed := big.NewInt(0)
	if exists {
		redeemed, err = laneState.Redeemed()
		if err != nil {
			return types.NewInt(0), err
		}
	}

	delta := types.BigSub(sv.Amount, redeemed)
	if minDelta.GreaterThan(delta) {
		return delta, fmt.Errorf("addVoucher: supplied token amount too low; minD=%s, D=%s; laneAmt=%s; v.Amt=%s", minDelta, delta, redeemed, sv.Amount)
	}

	ci.Vouchers = append(ci.Vouchers, &VoucherInfo{
		Voucher: sv,
	})

	if ci.NextLane <= sv.Lane {
		ci.NextLane = sv.Lane + 1
	}

	return delta, ca.store.putChannelInfo(ci)
}

func (ca *channelAccessor) checkVoucherValidUnlocked(ctx context.Context, ch address.Address, sv *paych.SignedVoucher) (map[uint64]paych.LaneState, error) {
	if sv.ChannelAddr != ch {
		return nil, fmt.Errorf("voucher ChannelAddr doesn't match channel address, got %s, expected %s", sv.ChannelAddr, ch)
	}

	// Load payment channel actor state
	act, pchState, err := ca.loadPaychActorState(ch)
	if err != nil {
		return nil, err
	}

	// Load channel "From" account actor state
	f, err := pchState.From()
	if err != nil {
		return nil, err
	}

	from, err := ca.api.StateAccountKey(ctx, f, types.EmptyTSK)
	if err != nil {
		return nil, err
	}

	// verify voucher signature
	vb, err := sv.SigningBytes()
	if err != nil {
		return nil, err
	}

	// TODO: technically, either party may create and sign a voucher.
	// However, for now, we only accept them from the channel creator.
	// More complex handling logic can be added later
	if err := sigs.Verify(sv.Signature, from, vb); err != nil {
		return nil, err
	}

	// Check the voucher against the highest known voucher nonce / value
	laneStates, err := ca.laneState(pchState, ch)
	if err != nil {
		return nil, err
	}

	// If the new voucher nonce value is less than the highest known
	// nonce for the lane
	ls, lsExists := laneStates[sv.Lane]
	if lsExists {
		n, err := ls.Nonce()
		if err != nil {
			return nil, err
		}

		if sv.Nonce <= n {
			return nil, fmt.Errorf("nonce too low")
		}

		// If the voucher amount is less than the highest known voucher amount
		r, err := ls.Redeemed()
		if err != nil {
			return nil, err
		}
		if sv.Amount.LessThanEqual(r) {
			return nil, fmt.Errorf("voucher amount is lower than amount for voucher with lower nonce")
		}
	}

	// Total redeemed is the total redeemed amount for all lanes, including
	// the new voucher
	// eg
	//
	// lane 1 redeemed:            3
	// lane 2 redeemed:            2
	// voucher for lane 1:         5
	//
	// Voucher supersedes lane 1 redeemed, therefore
	// effective lane 1 redeemed:  5
	//
	// lane 1:  5
	// lane 2:  2
	//          -
	// total:   7
	totalRedeemed, err := ca.totalRedeemedWithVoucher(laneStates, sv)
	if err != nil {
		return nil, err
	}

	// Total required balance must not exceed actor balance
	if act.Balance.LessThan(totalRedeemed) {
		return nil, newErrInsufficientFunds(types.BigSub(totalRedeemed, act.Balance))
	}

	if len(sv.Merges) != 0 {
		return nil, fmt.Errorf("dont currently support paych lane merges")
	}

	return laneStates, nil
}

// Get the total redeemed amount across all lanes, after applying the voucher
func (ca *channelAccessor) totalRedeemedWithVoucher(laneStates map[uint64]paych.LaneState, sv *paych.SignedVoucher) (big.Int, error) {
	// TODO: merges
	if len(sv.Merges) != 0 {
		return big.Int{}, fmt.Errorf("dont currently support paych lane merges")
	}

	total := big.NewInt(0)
	for _, ls := range laneStates {
		r, err := ls.Redeemed()
		if err != nil {
			return big.Int{}, err
		}
		total = big.Add(total, r)
	}

	lane, ok := laneStates[sv.Lane]
	if ok {
		// If the voucher is for an existing lane, and the voucher nonce
		// is higher than the lane nonce
		n, err := lane.Nonce()
		if err != nil {
			return big.Int{}, err
		}

		if sv.Nonce > n {
			// Add the delta between the redeemed amount and the voucher
			// amount to the total
			r, err := lane.Redeemed()
			if err != nil {
				return big.Int{}, err
			}

			delta := big.Sub(sv.Amount, r)
			total = big.Add(total, delta)
		}
	} else {
		// If the voucher is *not* for an existing lane, just add its
		// value (implicitly a new lane will be created for the voucher)
		total = big.Add(total, sv.Amount)
	}

	return total, nil
}

func (ca *channelAccessor) listVouchers(ctx context.Context, ch address.Address) ([]*VoucherInfo, error) {
	ca.lk.Lock()
	defer ca.lk.Unlock()

	// TODO: just having a passthrough method like this feels odd. Seems like
	// there should be some filtering we're doing here
	return ca.store.VouchersForPaych(ch)
}

// getPaychWaitReady waits for a the response to the message with the given cid
func (ca *channelAccessor) getPaychWaitReady(ctx context.Context, mcid cid.Cid) (address.Address, error) {
	ca.lk.Lock()

	// First check if the message has completed
	msgInfo, err := ca.store.GetMessage(mcid)
	if err != nil {
		ca.lk.Unlock()

		return address.Undef, err
	}

	// If the create channel / add funds message failed, return an error
	if len(msgInfo.Err) > 0 {
		ca.lk.Unlock()

		return address.Undef, fmt.Errorf(msgInfo.Err)
	}

	// If the message has completed successfully
	if msgInfo.Received {
		ca.lk.Unlock()

		// Get the channel address
		ci, err := ca.store.ByMessageCid(mcid)
		if err != nil {
			return address.Undef, err
		}

		if ci.Channel == nil {
			panic(fmt.Sprintf("create / add funds message %s succeeded but channelInfo.Channel is nil", mcid))
		}
		return *ci.Channel, nil
	}

	// The message hasn't completed yet so wait for it to complete
	promise := ca.msgPromise(ctx, mcid)

	// Unlock while waiting
	ca.lk.Unlock()

	select {
	case res := <-promise:
		return res.channel, res.err
	case <-ctx.Done():
		return address.Undef, ctx.Err()
	}
}

type onMsgRes struct {
	channel address.Address
	err     error
}

// msgPromise returns a channel that receives the result of the message with
// the given CID
func (ca *channelAccessor) msgPromise(ctx context.Context, mcid cid.Cid) chan onMsgRes {
	promise := make(chan onMsgRes)
	triggerUnsub := make(chan struct{})
	unsub := ca.msgListeners.onMsgComplete(mcid, func(err error) {
		close(triggerUnsub)

		// Use a go-routine so as not to block the event handler loop
		go func() {
			res := onMsgRes{err: err}
			if res.err == nil {
				// Get the channel associated with the message cid
				ci, err := ca.store.ByMessageCid(mcid)
				if err != nil {
					res.err = err
				} else {
					res.channel = *ci.Channel
				}
			}

			// Pass the result to the caller
			select {
			case promise <- res:
			case <-ctx.Done():
			}
		}()
	})

	// Unsubscribe when the message is received or the context is done
	go func() {
		select {
		case <-ctx.Done():
		case <-triggerUnsub:
		}

		unsub()
	}()

	return promise
}

func (ca *channelAccessor) getChannelInfo(addr address.Address) (*ChannelInfo, error) {
	ca.lk.Lock()
	defer ca.lk.Unlock()

	return ca.store.ByAddress(addr)
}

func (ca *channelAccessor) availableFunds(channelID string) (*api.ChannelAvailableFunds, error) {
	return ca.processQueue(channelID)
}

func (ca *channelAccessor) submitVoucher(ctx context.Context, ch address.Address, sv *paych.SignedVoucher, secret []byte) (cid.Cid, error) {
	ca.lk.Lock()
	defer ca.lk.Unlock()

	ci, err := ca.store.ByAddress(ch)
	if err != nil {
		return cid.Undef, err
	}

	has, err := ci.hasVoucher(sv)
	if err != nil {
		return cid.Undef, err
	}

	// If the channel has the voucher
	if has {
		// Check that the voucher hasn't already been submitted
		submitted, err := ci.wasVoucherSubmitted(sv)
		if err != nil {
			return cid.Undef, err
		}
		if submitted {
			return cid.Undef, fmt.Errorf("cannot submit voucher that has already been submitted")
		}
	}

	mb, err := ca.messageBuilder(ctx, ci.Control)
	if err != nil {
		return cid.Undef, err
	}

	msg, err := mb.Update(ch, sv, secret)
	if err != nil {
		return cid.Undef, err
	}

	smsg, err := ca.mpoolPush(ctx, msg)
	if err != nil {
		return cid.Undef, err
	}

	// If the channel didn't already have the voucher
	if !has {
		// Add the voucher to the channel
		ci.Vouchers = append(ci.Vouchers, &VoucherInfo{
			Voucher: sv,
		})
	}

	// Mark the voucher and any lower-nonce vouchers as having been submitted
	err = ca.store.MarkVoucherSubmitted(ci, sv)
	if err != nil {
		return cid.Undef, err
	}

	return smsg.Cid(), nil
}

func (ca *channelAccessor) settle(ctx context.Context, ch address.Address) (cid.Cid, error) {
	ca.lk.Lock()
	defer ca.lk.Unlock()

	ci, err := ca.store.ByAddress(ch)
	if err != nil {
		return cid.Undef, err
	}

	mb, err := ca.messageBuilder(ctx, ci.Control)
	if err != nil {
		return cid.Undef, err
	}
	msg, err := mb.Settle(ch)
	if err != nil {
		return cid.Undef, err
	}
	smgs, err := ca.mpoolPush(ctx, msg)
	if err != nil {
		return cid.Undef, err
	}

	ci.Settling = true
	err = ca.store.putChannelInfo(ci)
	if err != nil {
		fmt.Printf("Error marking channel as settled: %s", err)
	}

	return smgs.Cid(), err
}

func (ca *channelAccessor) collect(ctx context.Context, ch address.Address) (cid.Cid, error) {
	ca.lk.Lock()
	defer ca.lk.Unlock()

	ci, err := ca.store.ByAddress(ch)
	if err != nil {
		return cid.Undef, err
	}

	mb, err := ca.messageBuilder(ctx, ci.Control)
	if err != nil {
		return cid.Undef, err
	}

	msg, err := mb.Collect(ch)
	if err != nil {
		return cid.Undef, err
	}

	smsg, err := ca.mpoolPush(ctx, msg)
	if err != nil {
		return cid.Undef, err
	}

	return smsg.Cid(), nil
}

var ErrChannelNotTracked = errors.New("channel not tracked")

// insufficientFundsErr indicates that there are not enough funds in the
// channel to create a voucher
type insufficientFundsErr interface {
	Shortfall() types.BigInt
}

type ErrInsufficientFunds struct {
	shortfall types.BigInt
}

func newErrInsufficientFunds(shortfall types.BigInt) *ErrInsufficientFunds {
	return &ErrInsufficientFunds{shortfall: shortfall}
}

func (e *ErrInsufficientFunds) Error() string {
	return fmt.Sprintf("not enough funds in channel to cover voucher - shortfall: %d", e.shortfall)
}

func (e *ErrInsufficientFunds) Shortfall() types.BigInt {
	return e.shortfall
}

type channelStore struct {
	ds datastore.Batching
}

func NewChannelStore(ds dtypes.MetadataDS) *channelStore {
	ds = namespace.Wrap(ds, datastore.NewKey("/paych"))
	return &channelStore{
		ds: ds,
	}
}

const (
	DirInbound  = 1
	DirOutbound = 2
)

const (
	dsKeyChannelInfo = "ChannelInfo"
	dsKeyMsgCid      = "MsgCid"
)

type VoucherInfo struct {
	Voucher   *paych.SignedVoucher
	Proof     []byte // ignored
	Submitted bool
}

// ChannelInfo keeps track of information about a channel
type ChannelInfo struct {
	// ChannelID is a uuid set at channel creation
	ChannelID string
	// Channel address - may be nil if the channel hasn't been created yet
	Channel *address.Address
	// Control is the address of the local node
	Control address.Address
	// Target is the address of the remote node (on the other end of the channel)
	Target address.Address
	// Direction indicates if the channel is inbound (Control is the "to" address)
	// or outbound (Control is the "from" address)
	Direction uint64
	// Vouchers is a list of all vouchers sent on the channel
	Vouchers []*VoucherInfo
	// NextLane is the number of the next lane that should be used when the
	// client requests a new lane (eg to create a voucher for a new deal)
	NextLane uint64
	// Amount added to the channel.
	// Note: This amount is only used by GetPaych to keep track of how much
	// has locally been added to the channel. It should reflect the channel's
	// Balance on chain as long as all operations occur on the same datastore.
	Amount types.BigInt
	// PendingAmount is the amount that we're awaiting confirmation of
	PendingAmount types.BigInt
	// CreateMsg is the CID of a pending create message (while waiting for confirmation)
	CreateMsg *cid.Cid
	// AddFundsMsg is the CID of a pending add funds message (while waiting for confirmation)
	AddFundsMsg *cid.Cid
	// Settling indicates whether the channel has entered into the settling state
	Settling bool
}

func (ci *ChannelInfo) from() address.Address {
	if ci.Direction == DirOutbound {
		return ci.Control
	}
	return ci.Target
}

func (ci *ChannelInfo) to() address.Address {
	if ci.Direction == DirOutbound {
		return ci.Target
	}
	return ci.Control
}

// infoForVoucher gets the VoucherInfo for the given voucher.
// returns nil if the channel doesn't have the voucher.
func (ci *ChannelInfo) infoForVoucher(sv *paych.SignedVoucher) (*VoucherInfo, error) {
	for _, v := range ci.Vouchers {
		eq, err := cborrpc.Equals(sv, v.Voucher)
		if err != nil {
			return nil, err
		}
		if eq {
			return v, nil
		}
	}
	return nil, nil
}

func (ci *ChannelInfo) hasVoucher(sv *paych.SignedVoucher) (bool, error) {
	vi, err := ci.infoForVoucher(sv)
	return vi != nil, err
}

// wasVoucherSubmitted returns true if the voucher has been submitted
func (ci *ChannelInfo) wasVoucherSubmitted(sv *paych.SignedVoucher) (bool, error) {
	vi, err := ci.infoForVoucher(sv)
	if err != nil {
		return false, err
	}
	if vi == nil {
		return false, fmt.Errorf("cannot submit voucher that has not been added to channel")
	}
	return vi.Submitted, nil
}

// MsgInfo stores information about a create channel / add funds message
// that has been sent
type MsgInfo struct {
	// ChannelID links the message to a channel
	ChannelID string
	// MsgCid is the CID of the message
	MsgCid cid.Cid
	// Received indicates whether a response has been received
	Received bool
	// Err is the error received in the response
	Err string
}

func (s *channelStore) CreateChannel(from, to address.Address, mcid cid.Cid, amt types.BigInt) (*ChannelInfo, error) {
	ci := &ChannelInfo{
		Direction:     DirOutbound,
		NextLane:      0,
		Control:       from,
		Target:        to,
		CreateMsg:     &mcid,
		PendingAmount: amt,
	}
	if err := s.putChannelInfo(ci); err != nil {
		return nil, err
	}
	if err := s.SaveNewMessage(ci.ChannelID, mcid); err != nil {
		return nil, err
	}
	return ci, nil
}

// SaveNewMessage is called when a message is sent
func (s *channelStore) SaveNewMessage(channelID string, mcid cid.Cid) error {
	k := dskeyForMsg(mcid)

	b, err := cborrpc.Dump(&MsgInfo{ChannelID: channelID, MsgCid: mcid})
	if err != nil {
		return err
	}

	return s.ds.Put(k, b)
}

// RemoveChannel removes the channel with the given channel ID
func (s *channelStore) RemoveChannel(channelID string) error {
	return s.ds.Delete(dskeyForChannel(channelID))
}

// The datastore key used to identify the channel info
func dskeyForChannel(channelID string) datastore.Key {
	return datastore.KeyWithNamespaces([]string{dsKeyChannelInfo, channelID})
}

// putChannelInfo stores the channel info in the datastore
func (s *channelStore) putChannelInfo(ci *ChannelInfo) error {
	if len(ci.ChannelID) == 0 {
		ci.ChannelID = uuid.New().String()
	}
	k := dskeyForChannel(ci.ChannelID)

	b, err := marshallChannelInfo(ci)
	if err != nil {
		return err
	}

	return s.ds.Put(k, b)
}

// The datastore key used to identify the message
func dskeyForMsg(mcid cid.Cid) datastore.Key {
	return datastore.KeyWithNamespaces([]string{dsKeyMsgCid, mcid.String()})
}

// SaveMessageResult is called when the result of a message is received
func (s *channelStore) SaveMessageResult(mcid cid.Cid, msgErr error) error {
	minfo, err := s.GetMessage(mcid)
	if err != nil {
		return err
	}

	k := dskeyForMsg(mcid)
	minfo.Received = true
	if msgErr != nil {
		minfo.Err = msgErr.Error()
	}

	b, err := cborrpc.Dump(minfo)
	if err != nil {
		return err
	}

	return s.ds.Put(k, b)
}

// ByMessageCid gets the channel associated with a message
func (s *channelStore) ByMessageCid(mcid cid.Cid) (*ChannelInfo, error) {
	minfo, err := s.GetMessage(mcid)
	if err != nil {
		return nil, err
	}

	ci, err := s.findChan(func(ci *ChannelInfo) bool {
		return ci.ChannelID == minfo.ChannelID
	})
	if err != nil {
		return nil, err
	}

	return ci, err
}

// GetMessage gets the message info for a given message CID
func (s *channelStore) GetMessage(mcid cid.Cid) (*MsgInfo, error) {
	k := dskeyForMsg(mcid)

	val, err := s.ds.Get(k)
	if err != nil {
		return nil, err
	}

	var minfo MsgInfo
	if err := minfo.UnmarshalCBOR(bytes.NewReader(val)); err != nil {
		return nil, err
	}

	return &minfo, nil
}

// ByChannelID gets channel info by channel ID
func (s *channelStore) ByChannelID(channelID string) (*ChannelInfo, error) {
	var stored ChannelInfo

	res, err := s.ds.Get(dskeyForChannel(channelID))
	if err != nil {
		if err == datastore.ErrNotFound {
			return nil, ErrChannelNotTracked
		}
		return nil, err
	}

	return unmarshallChannelInfo(&stored, res)
}

// OutboundActiveByFromTo looks for outbound channels that have not been
// settled, with the given from / to addresses
func (s *channelStore) OutboundActiveByFromTo(from address.Address, to address.Address) (*ChannelInfo, error) {
	return s.findChan(func(ci *ChannelInfo) bool {
		if ci.Direction != DirOutbound {
			return false
		}
		if ci.Settling {
			return false
		}
		return ci.Control == from && ci.Target == to
	})
}

// ByAddress gets the channel that matches the given address
func (s *channelStore) ByAddress(addr address.Address) (*ChannelInfo, error) {
	return s.findChan(func(ci *ChannelInfo) bool {
		return ci.Channel != nil && *ci.Channel == addr
	})
}

// VouchersForPaych gets the vouchers for the given channel
func (s *channelStore) VouchersForPaych(ch address.Address) ([]*VoucherInfo, error) {
	ci, err := s.ByAddress(ch)
	if err != nil {
		return nil, err
	}

	return ci.Vouchers, nil
}

// findChan finds a single channel using the given filter.
// If there isn't a channel that matches the filter, returns ErrChannelNotTracked
func (s *channelStore) findChan(filter func(ci *ChannelInfo) bool) (*ChannelInfo, error) {
	cis, err := s.findChans(filter, 1)
	if err != nil {
		return nil, err
	}

	if len(cis) == 0 {
		return nil, ErrChannelNotTracked
	}

	return &cis[0], err
}

// findChans loops over all channels, only including those that pass the filter.
// max is the maximum number of channels to return. Set to zero to return unlimited channels.
func (s *channelStore) findChans(filter func(*ChannelInfo) bool, max int) ([]ChannelInfo, error) {
	res, err := s.ds.Query(dsq.Query{Prefix: dsKeyChannelInfo})
	if err != nil {
		return nil, err
	}
	defer res.Close() //nolint:errcheck

	var stored ChannelInfo
	var matches []ChannelInfo

	for {
		res, ok := res.NextSync()
		if !ok {
			break
		}

		if res.Error != nil {
			return nil, err
		}

		ci, err := unmarshallChannelInfo(&stored, res.Value)
		if err != nil {
			return nil, err
		}

		if !filter(ci) {
			continue
		}

		matches = append(matches, *ci)

		// If we've reached the maximum number of matches, return.
		// Note that if max is zero we return an unlimited number of matches
		// because len(matches) will always be at least 1.
		if len(matches) == max {
			return matches, nil
		}
	}

	return matches, nil
}

// AllocateLane allocates a new lane for the given channel
func (s *channelStore) AllocateLane(ch address.Address) (uint64, error) {
	ci, err := s.ByAddress(ch)
	if err != nil {
		return 0, err
	}

	out := ci.NextLane
	ci.NextLane++

	return out, s.putChannelInfo(ci)
}

// TrackChannel stores a channel, returning an error if the channel was already
// being tracked
func (s *channelStore) TrackChannel(ci *ChannelInfo) (*ChannelInfo, error) {
	_, err := s.ByAddress(*ci.Channel)
	switch err {
	default:
		return nil, err
	case nil:
		return nil, fmt.Errorf("already tracking channel: %s", ci.Channel)
	case ErrChannelNotTracked:
		err = s.putChannelInfo(ci)
		if err != nil {
			return nil, err
		}

		return s.ByAddress(*ci.Channel)
	}
}

// ListChannels returns the addresses of all channels that have been created
func (s *channelStore) ListChannels() ([]address.Address, error) {
	cis, err := s.findChans(func(ci *ChannelInfo) bool {
		return ci.Channel != nil
	}, 0)
	if err != nil {
		return nil, err
	}

	addrs := make([]address.Address, 0, len(cis))
	for _, ci := range cis {
		addrs = append(addrs, *ci.Channel)
	}

	return addrs, nil
}

// markVoucherSubmitted marks the voucher, and any vouchers of lower nonce
// in the same lane, as being submitted.
// Note: This method doesn't write anything to the store.
func (ci *ChannelInfo) markVoucherSubmitted(sv *paych.SignedVoucher) error {
	vi, err := ci.infoForVoucher(sv)
	if err != nil {
		return err
	}
	if vi == nil {
		return fmt.Errorf("cannot submit voucher that has not been added to channel")
	}

	// Mark the voucher as submitted
	vi.Submitted = true

	// Mark lower-nonce vouchers in the same lane as submitted (lower-nonce
	// vouchers are superseded by the submitted voucher)
	for _, vi := range ci.Vouchers {
		if vi.Voucher.Lane == sv.Lane && vi.Voucher.Nonce < sv.Nonce {
			vi.Submitted = true
		}
	}

	return nil
}

func (ps *channelStore) MarkVoucherSubmitted(ci *ChannelInfo, sv *paych.SignedVoucher) error {
	err := ci.markVoucherSubmitted(sv)
	if err != nil {
		return err
	}
	return ps.putChannelInfo(ci)
}

// TODO: This is a hack to get around not being able to CBOR marshall a nil
// address.Address. It's been fixed in address.Address but we need to wait
// for the change to propagate to specs-actors before we can remove this hack.
var emptyAddr address.Address

func init() {
	addr, err := address.NewActorAddress([]byte("empty"))
	if err != nil {
		panic(err)
	}
	emptyAddr = addr
}

func marshallChannelInfo(ci *ChannelInfo) ([]byte, error) {
	// See note above about CBOR marshalling address.Address
	if ci.Channel == nil {
		ci.Channel = &emptyAddr
	}
	return cborrpc.Dump(ci)
}

func unmarshallChannelInfo(stored *ChannelInfo, value []byte) (*ChannelInfo, error) {
	if err := stored.UnmarshalCBOR(bytes.NewReader(value)); err != nil {
		return nil, err
	}

	// See note above about CBOR marshalling address.Address
	if stored.Channel != nil && *stored.Channel == emptyAddr {
		stored.Channel = nil
	}

	return stored, nil
}

type rwlock interface {
	RLock()
	RUnlock()
}

// channelLock manages locking for a specific channel.
// Some operations update the state of a single channel, and need to block
// other operations only on the same channel's state.
// Some operations update state that affects all channels, and need to block
// any operation against any channel.
type channelLock struct {
	globalLock rwlock
	chanLock   sync.Mutex
}

func (l *channelLock) Lock() {
	// Wait for other operations by this channel to finish.
	// Exclusive per-channel (no other ops by this channel allowed).
	l.chanLock.Lock()
	// Wait for operations affecting all channels to finish.
	// Allows ops by other channels in parallel, but blocks all operations
	// if global lock is taken exclusively (eg when adding a channel)
	l.globalLock.RLock()
}

func (l *channelLock) Unlock() {
	l.globalLock.RUnlock()
	l.chanLock.Unlock()
}

type msgListeners struct {
	ps *pubsub.PubSub
}

type msgCompleteEvt struct {
	mcid cid.Cid
	err  error
}

type subscriberFn func(msgCompleteEvt)

func newMsgListeners() msgListeners {
	ps := pubsub.New(func(event pubsub.Event, subFn pubsub.SubscriberFn) error {
		evt, ok := event.(msgCompleteEvt)
		if !ok {
			return fmt.Errorf("wrong type of event")
		}
		sub, ok := subFn.(subscriberFn)
		if !ok {
			return fmt.Errorf("wrong type of subscriber")
		}
		sub(evt)
		return nil
	})
	return msgListeners{ps: ps}
}

// onMsgComplete registers a callback for when the message with the given cid
// completes
func (ml *msgListeners) onMsgComplete(mcid cid.Cid, cb func(error)) pubsub.Unsubscribe {
	var fn subscriberFn = func(evt msgCompleteEvt) {
		if mcid.Equals(evt.mcid) {
			cb(evt.err)
		}
	}
	return ml.ps.Subscribe(fn)
}

// fireMsgComplete is called when a message completes
func (ml *msgListeners) fireMsgComplete(mcid cid.Cid, err error) {
	e := ml.ps.Publish(msgCompleteEvt{mcid: mcid, err: err})
	if e != nil {
		// In theory we shouldn't ever get an error here
		fmt.Printf("unexpected error publishing message complete: %s", e)
	}
}
