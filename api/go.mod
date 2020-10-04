module github.com/tchardin/myel-ethonline/api

go 1.14

require (
	github.com/filecoin-project/go-address v0.0.4
	github.com/filecoin-project/go-data-transfer v0.6.7
	github.com/filecoin-project/go-ds-versioning v0.1.0
	github.com/filecoin-project/go-fil-markets v0.7.0
	github.com/filecoin-project/go-multistore v0.0.3
	github.com/filecoin-project/go-state-types v0.0.0-20200928172055-2df22083d8ab
	github.com/filecoin-project/go-statemachine v0.0.0-20200925024713-05bd7c71fbfe
	github.com/filecoin-project/go-storedcounter v0.0.0-20200421200003-1c99c62e8a5b
	github.com/filecoin-project/lotus v0.8.1
	github.com/filecoin-project/specs-actors v0.9.12
	github.com/hannahhoward/go-pubsub v0.0.0-20200423002714-8d62886cc36e
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-datastore v0.4.5
	github.com/ipfs/go-graphsync v0.2.1
	github.com/ipfs/go-ipfs v0.7.0
	github.com/ipfs/go-ipfs-blockstore v1.0.1
	github.com/ipfs/go-ipfs-config v0.9.0
	github.com/ipfs/go-ipfs-files v0.0.8
	github.com/ipfs/interface-go-ipfs-core v0.4.0
	github.com/ipld/go-ipld-prime v0.5.1-0.20200828233916-988837377a7f
	github.com/libp2p/go-libp2p-core v0.6.1
	github.com/libp2p/go-libp2p-peer v0.2.0
	github.com/libp2p/go-libp2p-peerstore v0.2.6
	github.com/multiformats/go-multiaddr v0.3.1
)

replace github.com/filecoin-project/filecoin-ffi => ../extern/filecoin-ffi
