package main

import (
	"fmt"
	"os"
	"os/signal"

	rtmkt "github.com/tchardin/myel-ethonline/rtmkt/lib"
)

func main() {
	fmt.Println(rtmkt.QueryProtocolID)
	n, err := rtmkt.SpawnNode(rtmkt.NodeTypeProvider)
	defer n.Close()
	if err != nil {
		fmt.Printf("Unable to create libp2p host: %v", err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	select {
	case <-stop:
		fmt.Println("Shutting down")
		os.Exit(0)
	}

}
