package rtmkt

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"path/filepath"
	"sync"

	config "github.com/ipfs/go-ipfs-config"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	libp2p "github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/plugin/loader"
	fsrepo "github.com/ipfs/go-ipfs/repo/fsrepo"
	icore "github.com/ipfs/interface-go-ipfs-core"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

type ipfsNode struct {
	api  icore.CoreAPI
	node *core.IpfsNode
}

func connectToPeers(ctx context.Context, ipfs icore.CoreAPI, peers []string, coPeers chan *peerstore.PeerInfo) error {
	var wg sync.WaitGroup
	peerInfos := make(map[peer.ID]*peerstore.PeerInfo, len(peers))
	for _, addrStr := range peers {
		addr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			return err
		}
		pii, err := peerstore.InfoFromP2pAddr(addr)
		if err != nil {
			return err
		}
		pi, ok := peerInfos[pii.ID]
		if !ok {
			pi = &peerstore.PeerInfo{ID: pii.ID}
			peerInfos[pi.ID] = pi
		}
		pi.Addrs = append(pi.Addrs, pii.Addrs...)
	}

	wg.Add(len(peerInfos))
	for _, peerInfo := range peerInfos {
		go func(peerInfo *peerstore.PeerInfo) {
			defer wg.Done()
			err := ipfs.Swarm().Connect(ctx, *peerInfo)
			if err != nil {
				fmt.Printf("failed to connect to %s: %s", peerInfo.ID, err)
			}
			fmt.Printf("Connected to peer %s\r\n", peerInfo.ID)
			coPeers <- peerInfo
		}(peerInfo)
	}
	wg.Wait()
	return nil
}

func NewIpfsNode(ctx context.Context) (*ipfsNode, error) {
	// ======== Temp repo ==========
	// Load plugins if available
	plugins, err := loader.NewPluginLoader(filepath.Join("", "plugins"))
	if err != nil {
		return nil, fmt.Errorf("Unable to load plugins: %v", err)
	}
	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return nil, fmt.Errorf("Unable to initialize plugins: %v", err)
	}
	if err := plugins.Inject(); err != nil {
		return nil, fmt.Errorf("Unable to inject plugins: %v", err)
	}
	// Create temporary dir
	repoPath, err := ioutil.TempDir("", "ipfs-shell")
	if err != nil {
		return nil, fmt.Errorf("Unable to get temp dir: %v", err)
	}
	// Set private network key
	// swarmkey := []byte("/key/swarm/psk/1.0.0/\n/base16/\n3bafac1973088aceaa01fab233dc2c250da22286c308e7b59b450149d8c08af5")
	// tmpfn := filepath.Join(repoPath, "swarm.key")
	// if err := ioutil.WriteFile(tmpfn, swarmkey, 0666); err != nil {
	// 	return C.CString(fmt.Sprintf("Unable to create swarm key file: %v", err))
	// }
	// Create config with default options and a 2048 bit key
	cfg, err := config.Init(ioutil.Discard, 2048)
	if err != nil {
		return nil, fmt.Errorf("Unable to create config: %v", err)
	}
	// Initialize the repo
	err = fsrepo.Init(repoPath, cfg)
	if err != nil {
		return nil, fmt.Errorf("Unable to initialize repo: %v", err)
	}
	// Open the repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, fmt.Errorf("Unable to open repo: %v", err)
	}

	// Put node configs together
	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption, // full DHT node
		Repo:    repo,
	}
	// Construct the node
	inode, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, fmt.Errorf("Unable to create new ipfs node: %v", err)
	}
	fmt.Printf("Node id: %v\r\n", inode.Identity.String())

	// Attach the core API to the constructed node
	ipfs, err := coreapi.NewCoreAPI(inode)
	if err != nil {
		return nil, fmt.Errorf("Unable to attach api to ipfs node: %v", err)
	}
	// If needed:
	//bootstrapNodes := []string{
	//	//Our Boostrapper node
	//	"/ip4/40.65.198.241/tcp/4001/ipfs/QmPBSxF2LN95dk8d4WgooBnVzAUk8hkGXu569q8ctquwSa",
	//}
	//// Channel to wait for connected peers
	//coPeers := make(chan *peerstore.PeerInfo)

	//// Connect to a bootstrapper peer so we can easily find other peers in our private network
	//go connectToPeers(ctx, ipfs, bootstrapNodes, coPeers)

	// <-coPeers

	n := &ipfsNode{
		api:  ipfs,
		node: inode,
	}
	return n, nil
}

func (n *ipfsNode) GetFile(ctx context.Context, cidStr string) error {
	cid := icorepath.New(cidStr)
	rootNode, err := n.api.Unixfs().Get(ctx, cid)
	if err != nil {
		return fmt.Errorf("Unable to get file from Unixfs: %v", err)
	}
	outputBasePath := "./"
	outputPath := outputBasePath + cidStr

	err = files.WriteTo(rootNode, outputPath)
	if err != nil {
		return fmt.Errorf("Unable to write file for cid: %v", err)
	}
	return nil
}

func (n *ipfsNode) AddWebFile(ctx context.Context, urlStr string) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("Unable to parse url: %v", err)
	}
	wf := files.NewWebFile(u)
	cidFile, err := n.api.Unixfs().Add(ctx, wf)
	if err != nil {
		return "", fmt.Errorf("Unable to add file to ipfs: %v", err)
	}
	return cidFile.String(), nil
}
