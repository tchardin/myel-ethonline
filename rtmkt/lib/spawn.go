package rtmkt

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-multistore"
	"github.com/filecoin-project/go-storedcounter"
	lclient "github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/wallet"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	cbor "github.com/ipfs/go-ipld-cbor"
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
	Store    *ipfsStore
	Client   RetrievalClient
	Provider RetrievalProvider
	Wallet   *wallet.LocalWallet
	PaychMgr *paychManager
	lcloser  jsonrpc.ClientCloser
}

func (mn *MyelNode) Close() {
	mn.lcloser()
	mn.Store.node.Close()
}

func SpawnNode(nt NodeType) (*MyelNode, error) {
	ctx := context.Background()
	// Establish connection with a remote (or local) lotus node
	lapi, lcloser, err := lclient.NewFullNodeRPC(ctx, "ws://40.65.198.241:8080/rpc/v0", http.Header{
		// This token can write msgs to mempool but not sign them
		"Authorization": []string{fmt.Sprintf("Bearer %s", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJyZWFkIiwid3JpdGUiXX0.zaQXQevDkk_1oEvGrgFhqJsJfjIA_e4ksuDuec2kplU")},
	})
	if err != nil {
		return nil, fmt.Errorf("Unable to start lotus rpc: %v", err)
	}
	// Create an underlying ipfs node
	ipfs, err := NewIpfsStore(ctx)
	if err != nil {
		return nil, fmt.Errorf("Unable to start ipfs node: %v", err)
	}

	// Create a retrieval network protocol from the ipfs node libp2p host
	net := NewFromLibp2pHost(ipfs.node.PeerHost)
	// Get the Datastore from ipfs
	ds := ipfs.node.Repo.Datastore()
	multiDs, err := multistore.NewMultiDstore(ds)
	if err != nil {
		return nil, fmt.Errorf("Unable to create multistore: %v", err)
	}
	dataTransfer, err := NewDataTransfer(ctx, ipfs.node, ds)
	if err != nil {
		return nil, fmt.Errorf("Unable to create graphsync data transfer: %v", err)
	}
	err = dataTransfer.Start(ctx)
	if err != nil {
		return nil, fmt.Errorf("Unable to start data transfer: %v", err)
	}
	memks := wallet.NewMemKeyStore()
	w, err := wallet.NewWallet(memks)
	if err != nil {
		return nil, fmt.Errorf("Unable to create new wallet: %v", err)
	}
	actStore := cbor.NewCborStore(ipfs.node.Blockstore)
	paychMgr := NewPaychManager(ctx, lapi, w, ds, actStore)

	// Wrap the full node api to provide an adapted interface
	radapter := NewRetrievalNode(lapi, paychMgr, TestModeOff)

	node := &MyelNode{
		Ctx:      ctx,
		Store:    ipfs,
		Wallet:   w,
		PaychMgr: paychMgr,
		lcloser:  lcloser,
	}
	if nt == NodeTypeClient || nt == NodeTypeFull {
		if _, err := node.WalletImport("client.private"); err != nil {
			return nil, err
		}
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
		providerAddress, err := node.WalletImport("provider.private")
		if err != nil {
			return nil, err
		}
		pds := namespace.Wrap(ds, datastore.NewKey("/retrieval/provider"))

		provider, err := NewProvider(providerAddress, radapter, net, multiDs, ipfs, dataTransfer, pds)
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

func (mn *MyelNode) WalletImport(name string) (address.Address, error) {
	wd, err := os.Getwd()
	if err != nil {
		return address.Undef, fmt.Errorf("Unable to get current directory: %v", err)
	}
	fdata, err := ioutil.ReadFile(path.Join(wd, name))
	if err != nil {
		return address.Undef, fmt.Errorf("Unable to import private key file: %v", err)
	}
	var ki types.KeyInfo
	data, err := hex.DecodeString(strings.TrimSpace(string(fdata)))
	if err != nil {
		return address.Undef, fmt.Errorf("Unable to decode hex string: %v", err)
	}
	if err := json.Unmarshal(data, &ki); err != nil {
		return address.Undef, fmt.Errorf("Unable to unmarshal keyinfo: %v", err)
	}
	addr, err := mn.Wallet.WalletImport(mn.Ctx, &ki)
	if err := mn.Wallet.SetDefault(addr); err != nil {
		return address.Undef, err
	}
	return addr, nil
}
