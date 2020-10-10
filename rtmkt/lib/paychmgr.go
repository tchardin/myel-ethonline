package rtmkt

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/hannahhoward/go-pubsub"

	"github.com/filecoin-project/go-address"
	cborrpc "github.com/filecoin-project/go-cbor-util"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/actors/builtin/paych"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/wallet"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	init2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/init"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	dsq "github.com/ipfs/go-datastore/query"
)

type paychManager struct {
	ctx    context.Context
	api    api.FullNode
	wallet *wallet.Wallet
	store  *channelStore

	lk       sync.RWMutex
	channels map[string]*channelAccessor
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
		from:  from,
		to:    to,
		chctx: pm.ctx,
		api:   pm.api,
		wal:   pm.wallet,
		store: pm.store,
		lk:    &channelLock{globalLock: &pm.lk},
	}
	// TODO: Use LRU
	pm.channels[key] = ca
	return ca
}

func NewPaychManager(ctx context.Context, node api.FullNode, w *wallet.Wallet) *paychManager {
	return &paychManager{
		ctx:      ctx,
		api:      node,
		wallet:   w,
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

type channelAccessor struct {
	from          address.Address
	to            address.Address
	chctx         context.Context
	api           api.FullNode
	wal           *wallet.Wallet
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
		_, pchState, err := ca.sa.loadPaychActorState(ca.chctx, ch)
		if err != nil {
			return nil, err
		}

		laneStates, err := ca.laneState(pchState, ch)
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

func (ca *channelAccessor) createPaych(ctx context.Context, amt types.BigInt) (cid.Cid, error) {
	nwVersion, err := ca.api.StateNetworkVersion(ctx, types.EmptyTSK)
	if err != nil {
		return cid.Undef, err
	}

	mb := paych.Message(actors.VersionForNetwork(nwVersion), ca.from)

	msg, err := mb.Create(ca.to, amt)
	cp := *msg
	msg = &cp
	if err != nil {
		return cid.Undef, err
	}

	msg, err = ca.api.GasEstimateMessageGas(ctx, msg, nil, types.EmptyTSK)
	if err != nil {
		return cid.Undef, err
	}
	// TODO save nonce in local data store
	nonce, err := ca.api.MpoolGetNonce(ctx, msg.From)
	if err != nil {
		return cid.Undef, err
	}

	msg.Nonce = nonce
	mbl, err := msg.ToStorageBlock()
	if err != nil {
		return cid.Undef, err
	}

	sig, err := ca.wal.Sign(ctx, msg.From, mbl.Cid().Bytes())
	if err != nil {
		return cid.Undef, fmt.Errorf("Unable to sign msg: %v", err)
	}

	smsg := &types.SignedMessage{
		Message:   *msg,
		Signature: *sig,
	}

	if _, err := ca.api.MpoolPush(ctx, smsg); err != nil {
		return cid.Undef, fmt.Errorf("MpoolPush failed with error: %v", err)
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
		fmt.Printf("wait msg: %w", err)
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

// addFunds sends a message to add funds to the channel and returns the message cid
func (ca *channelAccessor) addFunds(ctx context.Context, channelInfo *ChannelInfo, amt types.BigInt) (*cid.Cid, error) {
	msg := &types.Message{
		To:     *channelInfo.Channel,
		From:   channelInfo.Control,
		Value:  amt,
		Method: 0,
	}

	smsg, err := ca.api.MpoolPushMessage(ctx, msg, nil)
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

var ErrChannelNotTracked = errors.New("channel not tracked")

type channelStore struct {
	ds datastore.Batching
}

func NewStore(ds dtypes.MetadataDS) *channelStore {
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
