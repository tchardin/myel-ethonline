package rtmkt

import (
	"bytes"

	cborutil "github.com/filecoin-project/go-cbor-util"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	dshelp "github.com/ipfs/go-ipfs-ds-help"
)

//go:generate cbor-gen-for --map-encoding RetrievalPeers

// RetrievalPeers is a convenience struct for encoding slices of RetrievalPeer
type RetrievalPeers struct {
	Peers []RetrievalPeer
}

// PeerResolver is an interface for looking up providers that may have a piece
type PeerResolver interface {
	GetPeers(payloadCID cid.Cid) ([]RetrievalPeer, error) // TODO: channel
}

type Local struct {
	ds datastore.Datastore
}

func NewLocalPeerResolver(ds datastore.Batching) *Local {
	d := namespace.Wrap(ds, datastore.NewKey("localpeers-1"))
	return &Local{d}
}

func (l *Local) AddPeer(cid cid.Cid, peer RetrievalPeer) error {
	key := dshelp.MultihashToDsKey(cid.Hash())
	exists, err := l.ds.Has(key)
	if err != nil {
		return err
	}

	var newRecord bytes.Buffer

	if !exists {
		peers := RetrievalPeers{Peers: []RetrievalPeer{peer}}
		err = cborutil.WriteCborRPC(&newRecord, &peers)
		if err != nil {
			return err
		}
	} else {
		entry, err := l.ds.Get(key)
		if err != nil {
			return err
		}
		var peers RetrievalPeers
		if err = cborutil.ReadCborRPC(bytes.NewReader(entry), &peers); err != nil {
			return err
		}
		if hasPeer(peers, peer) {
			return nil
		}
		peers.Peers = append(peers.Peers, peer)
		err = cborutil.WriteCborRPC(&newRecord, &peers)
		if err != nil {
			return err
		}
	}

	return l.ds.Put(key, newRecord.Bytes())
}

func hasPeer(peerList RetrievalPeers, peer RetrievalPeer) bool {
	for _, p := range peerList.Peers {
		if p == peer {
			return true
		}
	}
	return false
}

func (l *Local) GetPeers(payloadCID cid.Cid) ([]RetrievalPeer, error) {
	entry, err := l.ds.Get(dshelp.MultihashToDsKey(payloadCID.Hash()))
	if err == datastore.ErrNotFound {
		return []RetrievalPeer{}, nil
	}
	if err != nil {
		return nil, err
	}
	var peers RetrievalPeers
	if err := cborutil.ReadCborRPC(bytes.NewReader(entry), &peers); err != nil {
		return nil, err
	}
	return peers.Peers, nil
}
