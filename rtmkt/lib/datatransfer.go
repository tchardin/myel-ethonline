package rtmkt

import (
	"context"

	datatransfer "github.com/filecoin-project/go-data-transfer"
	dtfimpl "github.com/filecoin-project/go-data-transfer/impl"
	dtnet "github.com/filecoin-project/go-data-transfer/network"
	gstransport "github.com/filecoin-project/go-data-transfer/transport/graphsync"
	"github.com/filecoin-project/go-storedcounter"
	"github.com/ipfs/go-datastore"
	graphsync "github.com/ipfs/go-graphsync/impl"
	gsnet "github.com/ipfs/go-graphsync/network"
	storeutil "github.com/ipfs/go-graphsync/storeutil"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	"github.com/libp2p/go-libp2p-core/host"
)

func NewDataTransfer(ctx context.Context, host host.Host, ds datastore.Batching) (datatransfer.Manager, error) {
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
	key := datastore.NewKey("/retrieval/counter")
	storedCounter := storedcounter.New(ds, key)
	// Finally we initialize the new instance of data transfer manager
	return dtfimpl.NewDataTransfer(ds, dtNet, tp, storedCounter)
}
