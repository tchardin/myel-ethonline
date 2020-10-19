package main

import (
	"context"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	rtmkt "github.com/tchardin/myel-ethonline/rtmkt/lib"
)

const RPCPort = ":5555"

func main() {
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("Received err from api server, exiting...")
	}
}

type ApiServer struct {
	n *rtmkt.MyelNode
}

func (host *ApiServer) StartNode(ctx context.Context, t rtmkt.NodeType) error {
	if host.n != nil {
		return nil
	}
	var err error
	host.n, err = rtmkt.SpawnNode(t)
	if err != nil {
		log.Error().Err(err).Msg("Unable to spawn myel node")
		return err
	}
	return nil
}

type ProviderEvent struct {
	Name string
}

func (host *ApiServer) ProviderEvents(ctx context.Context) (<-chan ProviderEvent, error) {
	out := make(chan ProviderEvent)

	host.n.Provider.SubscribeToEvents(func(event rtmkt.ProviderEvent, state rtmkt.ProviderDealState) {
		log.Info().
			Str("ProviderEvent", rtmkt.ProviderEvents[event]).
			Interface("ProviderDealStatus", rtmkt.DealStatuses[state.Status]).
			Uint64("TotalSent", state.TotalSent).
			Str("FundsReceived", state.FundsReceived.String()).
			Msg("Updating")

		out <- ProviderEvent{
			Name: rtmkt.ProviderEvents[event],
		}
	})

	return out, nil
}

func (host *ApiServer) SetProviderAsk(ctx context.Context, ppb int64, pi, pii uint64) error {
	ask := &rtmkt.Ask{
		PricePerByte:            abi.NewTokenAmount(ppb),
		PaymentInterval:         pi,
		PaymentIntervalIncrease: pii,
	}
	return host.n.Provider.SetAsk(ask)
}

func (host *ApiServer) GetFirstPeer() (peer.ID, error) {
	fp := host.n.Store.GetFirstPeer()
	return fp.ID(), nil
}

func (host *ApiServer) DefaultAddress(ctx context.Context) (address.Address, error) {
	addr, err := host.n.Wallet.GetDefault()
	if err != nil {
		return address.Undef, err
	}
	return addr, nil
}

func (host *ApiServer) AddressBalance(ctx context.Context, addr address.Address) (string, error) {
	amount, err := host.n.Lotus.WalletBalance(ctx, addr)
	if err != nil {
		return types.FIL{}.String(), err
	}
	return types.FIL(amount).String(), err
}

type RetrievalOrder struct {
	PaymentInterval         uint64
	PaymentIntervalIncrease uint64
	PricePerByte            uint64
	Size                    uint64
	ContentID               string
	Client                  address.Address
	Provider                address.Address
	ProviderPeer            peer.ID
}

func (host *ApiServer) Retrieve(ctx context.Context, order RetrievalOrder) (<-chan string, error) {
	ppb := abi.NewTokenAmount(int64(order.PricePerByte))
	params, _ := rtmkt.NewParams(ppb, order.PaymentInterval, order.PaymentIntervalIncrease)
	total := big.Add(big.Mul(ppb, abi.NewTokenAmount(int64(order.Size))), abi.NewTokenAmount(int64(500)))

	log.Info().
		Interface("Params", params).
		Str("Ppb", ppb.String()).
		Str("Total", total.String()).
		Str("Content", order.ContentID).
		Str("Client", order.Client.String()).
		Str("Provider", order.Provider.String()).
		Str("ProviderPeer", order.ProviderPeer.String()).
		Msg("Ready to retrieve")

	out := make(chan string)
	ticker := time.NewTicker(time.Second * 3)

	go func() {
		defer close(out)

		for {
			select {
			case <-ticker.C:
				rand.Seed(time.Now().UnixNano())
				randInt := rand.Intn(14)
				evt := rtmkt.ProviderEvents[rtmkt.ProviderEvent(randInt)]
				out <- evt
			case <-ctx.Done():
				ticker.Stop()
			}
		}
	}()

	return out, nil
}

func (host *ApiServer) AddWebFile(ctx context.Context, url string) (cid.Cid, error) {
	hcid, err := host.n.Store.AddWebFile(url)
	if err != nil {
		return cid.Undef, err
	}
	return hcid, nil
}

func (host *ApiServer) AddFile(ctx context.Context, path string) (cid.Cid, error) {
	return host.n.Store.AddFile(path)
}

func (host *ApiServer) GetFile(ctx context.Context, fid, to string) error {
	return host.n.Store.GetFile(fid, to)
}

func (host *ApiServer) QueryDeal(ctx context.Context, m string, pid peer.ID) (rtmkt.QueryResponse, error) {
	rp := rtmkt.RetrievalPeer{
		ID: pid,
	}
	mcid, err := cid.Decode(m)
	if err != nil {
		return rtmkt.QueryResponse{}, err
	}
	return host.n.Client.Query(ctx, rp, mcid, rtmkt.QueryParams{})
}

func runApiServer(shutdownCh <-chan struct{}) error {
	serverHandler := &ApiServer{}
	rpcServer := jsonrpc.NewServer()
	rpcServer.Register("MyelApi", serverHandler)

	http.Handle("/rpc/v0", rpcServer)
	log.Info().Str("port", RPCPort).Msg("Starting RPC")

	srv := &http.Server{
		Addr:    RPCPort,
		Handler: http.DefaultServeMux,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	<-shutdownCh
	if err := srv.Shutdown(context.TODO()); err != nil {
		log.Error().Err(err).Msg("Failed to shutdown RPC server")
		return err
	}
	log.Info().Msg("Server exited ok")

	if serverHandler.n != nil {
		serverHandler.n.Close()
	}
	return nil
}

func run() error {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	donec := make(chan struct{}, 1)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	go runApiServer(donec)

	select {
	case <-stop:
		log.Info().Msg("Shutting down")
		close(donec)
		os.Exit(0)
	}
	return nil

}
