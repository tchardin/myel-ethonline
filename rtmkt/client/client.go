package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/ipfs/go-cid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	rtmkt "github.com/tchardin/myel-ethonline/rtmkt/lib"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	n, err := rtmkt.SpawnNode(rtmkt.NodeTypeClient)
	defer n.Close()
	if err != nil {
		log.Error().Err(err).Msg("Unable to create new myel node")
	}

	dealFinished := make(chan int)
	n.Client.SubscribeToEvents(func(event rtmkt.ClientEvent, state rtmkt.ClientDealState) {
		log.Info().
			Str("ClientEvent", rtmkt.ClientEvents[event]).
			Str("ClientDealStatus", rtmkt.DealStatuses[state.Status]).
			Uint64("TotalReceived", state.TotalReceived).
			Str("TotalFunds", state.TotalFunds.String()).
			Str("FundsSpent", state.FundsSpent.String()).
			Str("VoucherShortfall", state.VoucherShortfall.String()).
			Msg("Updating")

		if event == rtmkt.ClientEventCompleteVerified {
			dealFinished <- 1
		}
	})

	peer := n.Store.GetFirstPeer()

	log.Info().Str("peer.ID", peer.ID().Pretty()).Msg("Connected to peer")

	rp := rtmkt.RetrievalPeer{
		ID: peer.ID(),
	}

	mcidString := "QmZNrFiWmW2QUZuqRLgJjG7AVmdDgGYqXwFZcMpjiZ14Qi"
	mcid, _ := cid.Decode(mcidString)

	res, err := n.Client.Query(n.Ctx, rp, mcid, rtmkt.QueryParams{})
	if err != nil {
		log.Error().Err(err).Msg("Unable to send Client Query")
	}

	log.Info().
		Interface("QueryResponse", res).
		Msg("Received from provider")

	clientStoreID := n.Client.NewStoreID()
	clientPaymentChannel, err := n.Wallet.GetDefault()
	if err != nil {
		log.Error().Err(err).Msg("Unable to get default address")
	}

	log.Info().Str("Address", clientPaymentChannel.String()).Msg("Wallet using")

	paymentInterval := uint64(10000)
	paymentIntervalIncrease := uint64(1000)
	pricePerByte := abi.NewTokenAmount(3)
	params, _ := rtmkt.NewParams(pricePerByte, paymentInterval, paymentIntervalIncrease)
	// Seems like the files are sometimes a tiny bit larget than advertized by the deal
	// so we add little leeway
	total := big.Add(big.Mul(pricePerByte, abi.NewTokenAmount(int64(res.Size))), abi.NewTokenAmount(int64(500)))

	log.Info().Str("Address", res.PaymentAddress.String()).Msg("Provider")

	did, err := n.Client.Retrieve(n.Ctx, mcid, params, total, rp, clientPaymentChannel, res.PaymentAddress, clientStoreID)
	if err != nil {
		log.Error().Err(err).Msg("Unable to retrieve content")
	}

	log.Info().Uint64("DealID", uint64(did)).Str("Total", total.String()).Msg("Started retrieval deal")

	<-dealFinished
	// We make an offline api to make sure ipfs doesn't load it directly
	n.Store.Offline()
	err = n.Store.GetFile(mcidString, "/file")
	if err != nil {
		log.Error().Err(err).Msg("Unable to spin up offline ipfs api")
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	select {
	case <-stop:
		fmt.Println("Shutting down")
		os.Exit(0)
	}
}
