package rtmkt

import (
	"bytes"
	"fmt"
	"math"

	datatransfer "github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-multistore"
	"github.com/filecoin-project/go-statemachine/fsm"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	basicnode "github.com/ipld/go-ipld-prime/node/basic"
	"github.com/libp2p/go-libp2p-core/peer"
	cbg "github.com/whyrusleeping/cbor-gen"
)

// EventReceiver is any thing that can receive FSM events
type EventReceiver interface {
	Send(id interface{}, name fsm.EventName, args ...interface{}) (err error)
}

func clientEventForResponse(response *DealResponse) (ClientEvent, []interface{}) {
	switch response.Status {
	case DealStatusRejected:
		return ClientEventDealRejected, []interface{}{response.Message}
	case DealStatusDealNotFound:
		return ClientEventDealNotFound, []interface{}{response.Message}
	case DealStatusAccepted:
		return ClientEventDealAccepted, nil
	case DealStatusFundsNeededLastPayment:
		return ClientEventLastPaymentRequested, []interface{}{response.PaymentOwed}
	case DealStatusCompleted:
		return ClientEventComplete, nil
	case DealStatusFundsNeeded:
		return ClientEventPaymentRequested, []interface{}{response.PaymentOwed}
	default:
		return ClientEventUnknownResponseReceived, nil
	}
}

const noEvent = ClientEvent(math.MaxUint64)

func clientEvent(event datatransfer.Event, channelState datatransfer.ChannelState) (ClientEvent, []interface{}) {
	switch event.Code {
	case datatransfer.Progress:
		return ClientEventBlocksReceived, []interface{}{channelState.Received()}
	case datatransfer.FinishTransfer:
		return ClientEventAllBlocksReceived, nil
	case datatransfer.Cancel:
		return ClientEventProviderCancelled, nil
	case datatransfer.NewVoucherResult:
		response, ok := dealResponseFromVoucherResult(channelState.LastVoucherResult())
		if !ok {
			fmt.Printf("unexpected voucher result received: %s\n", channelState.LastVoucher().Type())
			return noEvent, nil
		}

		return clientEventForResponse(response)
	case datatransfer.Error:
		if channelState.Message() == datatransfer.ErrRejected.Error() {
			return ClientEventDealRejected, []interface{}{"rejected for unknown reasons"}
		}
		return ClientEventDataTransferError, []interface{}{fmt.Errorf("deal data transfer failed: %s", event.Message)}
	default:
	}

	return noEvent, nil
}

// ClientDataTransferSubscriber is the function called when an event occurs in a data
// transfer initiated on the client -- it reads the voucher to verify this even occurred
// in a storage market deal, then, based on the data transfer event that occurred, it dispatches
// an event to the appropriate state machine
func ClientDataTransferSubscriber(deals EventReceiver) datatransfer.Subscriber {
	return func(event datatransfer.Event, channelState datatransfer.ChannelState) {
		dealProposal, ok := dealProposalFromVoucher(channelState.Voucher())

		// if this event is for a transfer not related to retrieval, ignore
		if !ok {
			return
		}

		retrievalEvent, params := clientEvent(event, channelState)
		if retrievalEvent == noEvent {
			return
		}

		// data transfer events for progress do not affect deal state
		err := deals.Send(dealProposal.ID, retrievalEvent, params...)
		if err != nil {
			fmt.Printf("processing dt event: %w\n", err)
		}
	}
}

// StoreGetter retrieves the store for a given proposal cid
type StoreGetter interface {
	Get(otherPeer peer.ID, dealID DealID) (*multistore.Store, error)
}

// StoreConfigurableTransport defines the methods needed to
// configure a data transfer transport use a unique store for a given request
type StoreConfigurableTransport interface {
	UseStore(datatransfer.ChannelID, ipld.Loader, ipld.Storer) error
}

// TransportConfigurer configurers the graphsync transport to use a custom blockstore per deal
func TransportConfigurer(thisPeer peer.ID, storeGetter StoreGetter) datatransfer.TransportConfigurer {
	return func(channelID datatransfer.ChannelID, voucher datatransfer.Voucher, transport datatransfer.Transport) {
		dealProposal, ok := dealProposalFromVoucher(voucher)
		if !ok {
			return
		}
		gsTransport, ok := transport.(StoreConfigurableTransport)
		if !ok {
			return
		}
		otherPeer := channelID.OtherParty(thisPeer)
		store, err := storeGetter.Get(otherPeer, dealProposal.ID)
		if err != nil {
			fmt.Printf("attempting to configure data store: %w\n", err)
			return
		}
		if store == nil {
			return
		}
		err = gsTransport.UseStore(channelID, store.Loader, store.Storer)
		if err != nil {
			fmt.Printf("attempting to configure data store: %w\n", err)
		}
	}
}

func dealProposalFromVoucher(voucher datatransfer.Voucher) (*DealProposal, bool) {
	dealProposal, ok := voucher.(*DealProposal)
	// if this event is for a transfer not related to storage, ignore
	if ok {
		return dealProposal, true
	}

	return nil, false
}

func dealResponseFromVoucherResult(vres datatransfer.VoucherResult) (*DealResponse, bool) {
	dealResponse, ok := vres.(*DealResponse)
	if ok {
		return dealResponse, true
	}
	// if this event is for a transfer not related to storage, ignore
	return nil, false
}

// DecodeNode validates and computes a decoded ipld.Node selector from the
// provided cbor-encoded selector
func DecodeNode(defnode *cbg.Deferred) (ipld.Node, error) {
	reader := bytes.NewReader(defnode.Raw)
	nb := basicnode.Prototype.Any.NewBuilder()
	err := dagcbor.Decoder(nb, reader)
	if err != nil {
		return nil, err
	}
	return nb.Build(), nil
}
