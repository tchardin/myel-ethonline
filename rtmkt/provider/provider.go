package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	rtmkt "github.com/tchardin/myel-ethonline/rtmkt/lib"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	n, err := rtmkt.SpawnNode(rtmkt.NodeTypeProvider)
	defer n.Close()
	if err != nil {
		log.Error().Err(err).Msg("Unable to create libp2p host")
	}

	hcid, err := n.Store.AddWebFile("https://images.unsplash.com/photo-1601666703585-964591b026c5")
	if err != nil {
		log.Error().Err(err).Msg("Unable to load web content")
	}

	size, err := n.Store.GetSize(hcid)
	if err != nil {
		log.Error().Err(err).Msg("Unable to get file size")
	}

	log.Info().Str("cid", hcid.String()).Int64("Size", size).Msg("Serving content")

	n.Provider.SubscribeToEvents(func(event rtmkt.ProviderEvent, state rtmkt.ProviderDealState) {
		log.Info().
			Str("ProviderEvent", rtmkt.ProviderEvents[event]).
			Interface("ProviderDealStatus", rtmkt.DealStatuses[state.Status]).
			Uint64("TotalSent", state.TotalSent).
			Str("FundsReceived", state.FundsReceived.String()).
			Msg("Updating")

		if event == rtmkt.ProviderEventComplete {
			chs, err := n.PaychMgr.ListChannels()
			if err != nil {
				log.Error().Err(err).Msg("Unable to list channels")
				return
			}
			for _, ch := range chs {
				vchs, err := n.PaychMgr.ListVouchers(n.Ctx, ch)
				if err != nil {
					log.Error().Err(err).Msg("Listing vouchers")
					continue
				}
				for _, vch := range vchs {
					log.Info().Interface("Voucher", vch.Voucher).Msg("Saved")
				}
			}
		}
	})

	addr, err := n.Wallet.GetDefault()
	if err != nil {
		log.Error().Err(err).Msg("Unable to get default address")
	}

	log.Info().Str("Address", addr.String()).Msg("Wallet using")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	select {
	case <-stop:
		fmt.Println("Shutting down")
		os.Exit(0)
	}

}
