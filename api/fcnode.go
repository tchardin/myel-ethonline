package main

import "C"
import (
	"fmt"
	"net/http"

	datatransfer "github.com/filecoin-project/go-data-transfer"
	dtfimpl "github.com/filecoin-project/go-data-transfer/impl"
	dtnet "github.com/filecoin-project/go-data-transfer/network"
	gstransport "github.com/filecoin-project/go-data-transfer/transport/graphsync"
	rmnet "github.com/filecoin-project/go-fil-markets/retrievalmarket/network"
	"github.com/filecoin-project/go-multistore"
	"github.com/filecoin-project/go-storedcounter"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/ipfs/go-datastore"
	graphsync "github.com/ipfs/go-graphsync/impl"
	gsnet "github.com/ipfs/go-graphsync/network"
	storeutil "github.com/ipfs/go-graphsync/storeutil"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	"github.com/libp2p/go-libp2p-core/host"
)

func NewDataTransfer(host host.Host, ds datastore.Batching) (datatransfer.Manager, error) {
	// Create a graphsync network
	gsNet := gsnet.NewFromLibp2pHost(host)
	// Create a datastore network which is technically a graphsync network but the interface
	// doesn't exactly match so we can't reuse the same one. Not sure if intential?
	dtNet := dtnet.NewFromLibp2pHost(host)
	// Integrate with Blockstore from IPFS
	var bs blockstore.Blockstore
	loader := storeutil.LoaderForBlockstore(bs)
	storer := storeutil.StorerForBlockstore(bs)
	// Create a graphsync exchange
	exchange := graphsync.New(ctx, gsNet, loader, storer)
	// Build transport interface
	tp := gstransport.NewTransport(host.ID(), exchange)
	// A counter that persists to the datastore as it increments
	// not sure exactly what it's for but required by NewDataTransfer method
	key := datastore.NewKey("counter")
	storedCounter := storedcounter.New(ds, key)
	// Finally we initialize the new instance of data transfer manager
	return dtfimpl.NewDataTransfer(ds, dtNet, tp, storedCounter)
}

func mockProvider(node RetrievalProviderNode, network rmnet.RetrievalMarketNetwork, multiStore *multistore.MultiStore, dataTransfer datatransfer.Manager, ds datastore.Batching, opts ...RetrievalProviderOption) {
}

func SpawnFilecoinNode() *C.char {
	nodeApi, ncloser, err := client.NewFullNodeRPC(ctx, "ws://localhost:1234/rpc/v0", http.Header{})
	if err != nil {
		return C.CString(fmt.Sprintf("Unable to create Lotus RPC client: %v", err))
	}
	defer ncloser()

	radapter := NewRetrievalProviderNode(nodeApi)
	netwk := rmnet.NewFromLibp2pHost(inode.PeerHost)
	ds := inode.Repo.Datastore()
	multiDs, err := multistore.NewMultiDstore(ds)
	if err != nil {
		return C.CString(fmt.Sprintf("Unable to create multistore: %v", err))
	}
	dataTransfer, err := NewDataTransfer(inode.PeerHost, ds)
	if err != nil {
		return C.CString(fmt.Sprintf("Unable to create graphsync data transfer: %v", err))
	}
	// TODO: We need an address for the miner, or do we?
	mockProvider(radapter, netwk, multiDs, dataTransfer, ds)
	return nil
}
