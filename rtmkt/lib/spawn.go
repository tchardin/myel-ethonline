package rtmkt

import (
	"context"
	"fmt"
	"net/http"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-multistore"
	"github.com/filecoin-project/go-storedcounter"
	lclient "github.com/filecoin-project/lotus/api/client"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
)

// NodeType is the role our node can play in the retrieval market
type NodeType uint64

const (
	// NodeTypeClient nodes can only retrieve content
	NodeTypeClient NodeType = iota
	// NodeTypeProvider nodes can only provide content
	NodeTypeProvider
	// NodeTypeFull nodes can do everything
	// mostly TODO here
	NodeTypeFull
)

type MyelNode struct {
	Ctx      context.Context
	Ipfs     *ipfsNode
	Client   RetrievalClient
	Provider RetrievalProvider
	lcloser  jsonrpc.ClientCloser
}

func (mn *MyelNode) Close() {
	mn.lcloser()
	mn.Ipfs.node.Close()
}

func SpawnNode(nt NodeType) (*MyelNode, error) {
	ctx := context.Background()
	// Establish connection with a remote (or local) lotus node
	lapi, lcloser, err := lclient.NewFullNodeRPC(ctx, "ws://localhost:1234/rpc/v0", http.Header{})
	if err != nil {
		return nil, fmt.Errorf("Unable to start lotus rpc: %v", err)
	}
	// Wrap the full node api to provide an adapted interface
	radapter := NewRetrievalNode(lapi)
	// Create an underlying ipfs node
	ipfs, err := NewIpfsNode(ctx)
	if err != nil {
		return nil, fmt.Errorf("Unable to start ipfs node: %v", err)
	}
	node := &MyelNode{
		Ctx:     ctx,
		Ipfs:    ipfs,
		lcloser: lcloser,
	}
	// Create a retrieval network protocol from the ipfs node libp2p host
	net := NewFromLibp2pHost(ipfs.node.PeerHost)
	// Get the Datastore from ipfs
	ds := ipfs.node.Repo.Datastore()
	multiDs, err := multistore.NewMultiDstore(ds)
	if err != nil {
		return nil, fmt.Errorf("Unable to create multistore: %v", err)
	}
	dataTransfer, err := NewDataTransfer(ctx, ipfs.node.PeerHost, ds)
	if err != nil {
		return nil, fmt.Errorf("Unable to create graphsync data transfer: %v", err)
	}
	if nt == NodeTypeClient || nt == NodeTypeFull {
		// Create a new namespace for our metadata store
		nds := namespace.Wrap(ds, datastore.NewKey("/retrieval/client"))
		countKey := datastore.NewKey("/retrieval/client/dealcounter")
		dealCounter := storedcounter.New(ds, countKey)
		resolver := NewLocalPeerResolver(ds)

		client, err := NewClient(radapter, net, multiDs, dataTransfer, nds, resolver, dealCounter)
		if err != nil {
			return nil, fmt.Errorf("Unable to create new retrieval client: %v", err)
		}
		node.Client = client

		err = client.Start(ctx)
		if err != nil {
			return nil, err
		}
	}
	if nt == NodeTypeProvider || nt == NodeTypeFull {
		pds := namespace.Wrap(ds, datastore.NewKey("/retrieval/provider"))
		testAddress := address.TestAddress
		provider, err := NewProvider(testAddress, radapter, net, multiDs, dataTransfer, pds)
		if err != nil {
			return nil, fmt.Errorf("Unable to create new retrieval provider: %v", err)
		}
		node.Provider = provider

		err = provider.Start(ctx)
		if err != nil {
			return nil, fmt.Errorf("Unable to start provider: %v", err)
		}
	}

	return node, nil
}
