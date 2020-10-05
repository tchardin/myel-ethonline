package main

import (
	"fmt"

	datatransfer "github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-multistore"
	"github.com/ipld/go-ipld-prime"
	"github.com/libp2p/go-libp2p-core/peer"
)

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
			fmt.Printf("attempting to configure data store: %w", err)
			return
		}
		if store == nil {
			return
		}
		err = gsTransport.UseStore(channelID, store.Loader, store.Storer)
		if err != nil {
			fmt.Printf("attempting to configure data store: %w", err)
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
