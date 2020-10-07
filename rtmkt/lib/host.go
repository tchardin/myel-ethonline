package rtmkt
import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery"
)

// Eventually we can construct our own libp2p node to pass over to ipfs
const DiscoveryInterval = time.Hour
const DiscoveryServiceTag = "Myel"

type DiscoveryNotifee struct {
	h              host.Host
	ConnectedPeers chan *peer.AddrInfo
}

func (n *DiscoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	err := n.h.Connect(context.Background(), pi)
	if err != nil {
		fmt.Printf("error connecting to peer %s: %s\n", pi.ID.Pretty(), err)
	}
	n.ConnectedPeers <- &pi
}

func NewLibp2pHost(ctx context.Context) (host.Host, error) {
	h, err := libp2p.New(ctx, libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	if err != nil {
		return nil, err
	}

	disc, err := discovery.NewMdnsService(ctx, h, DiscoveryInterval, DiscoveryServiceTag)
	if err != nil {
		return nil, err
	}
	n := &DiscoveryNotifee{
		h:              h,
		ConnectedPeers: make(chan *peer.AddrInfo),
	}
	disc.RegisterNotifee(n)

	p := <-n.ConnectedPeers
	fmt.Printf("Connected with peer %v\n", p.ID.Pretty())
	return h, nil
}
