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

	log.Info().Str("cid", hcid).Msg("Serving content")

	n.Provider.SubscribeToEvents(func(event rtmkt.ProviderEvent, state rtmkt.ProviderDealState) {
		log.Info().
			Str("ProviderEvent", rtmkt.ProviderEvents[event]).
			Interface("ProviderDealStatus", rtmkt.DealStatuses[state.Status]).
			Msg("Updating")
	})

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	select {
	case <-stop:
		fmt.Println("Shutting down")
		os.Exit(0)
	}

}
