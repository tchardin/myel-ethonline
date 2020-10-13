module github.com/tchardin/myel-ethonline/rtmkt

go 1.14

require (
	github.com/filecoin-project/go-address v0.0.4
	github.com/filecoin-project/go-cbor-util v0.0.0-20191219014500-08c40a1e63a2
	github.com/filecoin-project/go-data-transfer v0.6.7
	github.com/filecoin-project/go-ds-versioning v0.1.0
	github.com/filecoin-project/go-fil-markets v0.7.1
	github.com/filecoin-project/go-jsonrpc v0.1.2-0.20201008195726-68c6a2704e49
	github.com/filecoin-project/go-multistore v0.0.3
	github.com/filecoin-project/go-state-types v0.0.0-20201003010437-c33112184a2b
	github.com/filecoin-project/go-statemachine v0.0.0-20200925024713-05bd7c71fbfe
	github.com/filecoin-project/go-storedcounter v0.0.0-20200421200003-1c99c62e8a5b
	github.com/filecoin-project/lotus v0.10.0
	github.com/filecoin-project/specs-actors v0.9.12
	github.com/filecoin-project/specs-actors/v2 v2.1.0
	github.com/google/uuid v1.1.1
	github.com/hannahhoward/cbor-gen-for v0.0.0-20200817222906-ea96cece81f1
	github.com/hannahhoward/go-pubsub v0.0.0-20200423002714-8d62886cc36e
	github.com/ipfs/go-block-format v0.0.2
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-datastore v0.4.5
	github.com/ipfs/go-graphsync v0.2.1
	github.com/ipfs/go-ipfs v0.7.0
	github.com/ipfs/go-ipfs-blockstore v1.0.1 // indirect
	github.com/ipfs/go-ipfs-config v0.9.0
	github.com/ipfs/go-ipfs-ds-help v1.0.0
	github.com/ipfs/go-ipfs-files v0.0.8
	github.com/ipfs/go-ipld-cbor v0.0.5-0.20200428170625-a0bd04d3cbdf
	github.com/ipfs/interface-go-ipfs-core v0.4.0
	github.com/ipld/go-ipld-prime v0.5.1-0.20200828233916-988837377a7f
	github.com/jpillora/backoff v1.0.0
	github.com/libp2p/go-libp2p v0.11.0 // indirect
	github.com/libp2p/go-libp2p-core v0.6.1
	github.com/libp2p/go-libp2p-peer v0.2.0
	github.com/libp2p/go-libp2p-peerstore v0.2.6
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/multiformats/go-multihash v0.0.14
	github.com/prometheus/common v0.10.0
	github.com/rs/zerolog v1.20.0
	github.com/whyrusleeping/cbor-gen v0.0.0-20200826160007-0b9f6c5fb163
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
)

replace github.com/filecoin-project/filecoin-ffi => ../extern/filecoin-ffi

replace github.com/supranational/blst => ../extern/fil-blst/blst

replace github.com/filecoin-project/fil-blst => ../extern/fil-blst

replace github.com/ipfs/go-ipfs => ../extern/go-ipfs
