# Myel | ETHOnline

We started this repo to isolate development for ETHOnline hackathon. We have built a
MacOS app in a private repo in parallel to interact with the retrieval market. We may release an open source web
UI in the future so anyone can fork and create their own client for the retrieval market however our current focus
is on speed and delivering a high quality experience over the retrieval market.

This repo features our first working retrieval market implementation in golang as well as 
prototype contracts for our retrieval payment management and FIL lending systems.

You can find information about how to run the oracle system in its eponymous directory.

## Getting Started

The retieval market relies on a remote lotus node we run ourselves. There is a token
in the source that can be used to run the app however if we see increase usage we may 
disable it at any time. We are also testing on Infura however they are missing some endpoints around gas estimation.

Both nodes need wallets with addresses on chain to properly function. The provider node will look for a `provider.private` key
file and client node looks for a `client.private`. There is a program to generate new addresses in
the filground directory but you still need to start with a stocked wallet.

### Running the provider

First we run the provider node

```
cd rtmkt/provider
go run .
```

This will load an arbitrary file from a web url into a temporary IPFS repo and start
looking for peers on the local network.

Once a retrieval is completed our provider node will update the payment channel and settle it.

### Running the client

```
cd rtmkt/client
go run .
```

Our client node will look for a provider peer on the local network and immediately start 
a retrieval.

### Running the api

This starts a node which can both provide and retieve content. It uses separate addresses 
for easier debugging experience.

```
cd rtmkt/api
go run .
```

## UI

You can watch a live demo of retrieval via the UI [here](https://youtu.be/56HrKnlPiDs?t=1183).

We will update this part soon with instructions on how to download the macOS app.
We plan on building a windows version too but feel free to open an issue if you'd 
like us to prioritize it in our roadmap.

## Contributions

Best thing is to reach out if you'd like to contribute so we can sync up properly. 
We'll review PRs and issues too.

If you want to buy us coffee you can drop some FIL at `f13t4qv2lvlwowq67d2txl7auiddhlppca3nw5yxa`.


