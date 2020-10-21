package main

import (
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"os/signal"

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
	starting bool
	clientSubscribed bool
	providerSubscribed bool
}

func (host *ApiServer) StartNode(ctx context.Context, t rtmkt.NodeType) error {
	if host.starting || host.n != nil {
		return nil
	}
	host.starting = true
	var err error
	host.n, err = rtmkt.SpawnNode(t)
	if err != nil {
		log.Error().Err(err).Msg("Unable to spawn myel node")
		return err
	}
	return nil
}

type ProviderEvent struct {
	Status string
	TotalReceived string
}

func (host *ApiServer) RegisterProviderEvents(ctx context.Context) (<-chan ProviderEvent, error) {
	out := make(chan ProviderEvent)
	if host.providerSubscribed {
		return out, nil
	} else {
		host.providerSubscribed = true
	}

	host.n.Provider.SubscribeToEvents(func(event rtmkt.ProviderEvent, state rtmkt.ProviderDealState) {
		log.Info().
			Str("ProviderEvent", rtmkt.ProviderEvents[event]).
			Interface("ProviderDealStatus", rtmkt.DealStatuses[state.Status]).
			Uint64("TotalSent", state.TotalSent).
			Str("FundsReceived", state.FundsReceived.String()).
			Msg("Updating")

		out <- ProviderEvent{
			Status: rtmkt.DealStatuses[state.Status],
			TotalReceived: types.FIL(state.FundsReceived).String(),
		}
	})

	return out, nil
}

func (host *ApiServer) RegisterClientEvents(ctx context.Context) (<-chan ClientEvent, error) {
	out := make(chan ClientEvent)
	if host.clientSubscribed {
		return out, nil
	} else {
		host.clientSubscribed = true
	}
	log.Info().Msg("Registered client events")

	host.n.Client.SubscribeToEvents(func(event rtmkt.ClientEvent, state rtmkt.ClientDealState) {
		log.Info().
			Str("ClientEvent", rtmkt.ClientEvents[event]).
			Str("ClientDealStatus", rtmkt.DealStatuses[state.Status]).
			Uint64("TotalReceived", state.TotalReceived).
			Str("TotalFunds", state.TotalFunds.String()).
			Str("FundsSpent", state.FundsSpent.String()).
			Str("VoucherShortfall", state.VoucherShortfall.String()).
			Msg("Updating")

		out<-ClientEvent{
			Status:        rtmkt.DealStatuses[state.Status],
			TotalReceived: state.TotalReceived,
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

type ClientEvent struct {
	Status string
	TotalReceived uint64
}

func (host *ApiServer) Retrieve(ctx context.Context, order RetrievalOrder) (<-chan ClientEvent, error) {
	out := make(chan ClientEvent)
	host.n.Client.SubscribeToEvents(func(event rtmkt.ClientEvent, state rtmkt.ClientDealState) {
		log.Info().
			Str("ClientEvent", rtmkt.ClientEvents[event]).
			Str("ClientDealStatus", rtmkt.DealStatuses[state.Status]).
			Uint64("TotalReceived", state.TotalReceived).
			Str("TotalFunds", state.TotalFunds.String()).
			Str("FundsSpent", state.FundsSpent.String()).
			Str("VoucherShortfall", state.VoucherShortfall.String()).
			Msg("Updating")

		out<-ClientEvent{
			Status:        rtmkt.DealStatuses[state.Status],
			TotalReceived: state.TotalReceived,
		}
	})

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

	mcid, err := cid.Decode(order.ContentID)
	if err != nil {
		return out, err
	}
	clientStoreID := host.n.Client.NewStoreID()
	rp := rtmkt.RetrievalPeer{
		ID: order.ProviderPeer,
	}
	_, err = host.n.Client.Retrieve(ctx, mcid, params, total, rp, order.Client, order.Provider, clientStoreID)
	return out, err
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

type ProviderResponse struct {
	rtmkt.QueryResponse
	PeerID peer.ID
	IDAddress address.Address
}

// GetFirstPeer is a temp function to find a provider and requested deal terms for a piece of content
func (host *ApiServer) GetFirstPeer(ctx context.Context, p string) (ProviderResponse, error) {
	fp := host.n.Store.GetFirstPeer()
	rp := rtmkt.RetrievalPeer{
		ID: fp.ID(),
	}
	pcid, err := cid.Decode(p)
	if err != nil {
		return ProviderResponse{}, err
	}
	res, err := host.n.Client.Query(ctx, rp, pcid, rtmkt.QueryParams{})
	if err != nil {
		return ProviderResponse{}, err
	}
	idaddr, err := host.n.Lotus.StateLookupID(ctx, res.PaymentAddress, types.EmptyTSK)
	if err != nil {
		return ProviderResponse{}, err
	}
	return ProviderResponse{
		res,
		fp.ID(),
		idaddr,
	}, nil
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
