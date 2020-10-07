package main

import (
	"fmt"
	"os"
	"os/signal"

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

	n.Client.SubscribeToEvents(func(event rtmkt.ClientEvent, state rtmkt.ClientDealState) {
		log.Info().Str("ClientEvent", rtmkt.ClientEvents[event]).Interface("ClientDealState", state)
	})

	peer := n.Ipfs.GetFirstPeer(n.Ctx)

	log.Info().Str("peer.ID", peer.ID().Pretty()).Msg("Connected to peer")

	rp := rtmkt.RetrievalPeer{
		ID: peer.ID(),
	}

	mcid, _ := cid.Decode("QmZNrFiWmW2QUZuqRLgJjG7AVmdDgGYqXwFZcMpjiZ14Qi")

	res, err := n.Client.Query(n.Ctx, rp, mcid, rtmkt.QueryParams{})
	if err != nil {
		log.Error().Err(err).Msg("Unable to send Client Query")
	}

	log.Info().
		Interface("QueryResponse", res).
		Msg("Received from provider")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	select {
	case <-stop:
		fmt.Println("Shutting down")
		os.Exit(0)
	}
}
