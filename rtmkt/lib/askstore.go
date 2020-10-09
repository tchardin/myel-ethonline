package rtmkt

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/ipfs/go-datastore"
	"golang.org/x/xerrors"

	cborutil "github.com/filecoin-project/go-cbor-util"
	versioning "github.com/filecoin-project/go-ds-versioning/pkg"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-datastore/namespace"
)

// DefaultPricePerByte is the charge per byte retrieved if the miner does
// not specifically set it
var DefaultPricePerByte = abi.NewTokenAmount(2)

// DefaultPaymentInterval is the baseline interval, set to 1Mb
// if the miner does not explicitly set it otherwise
var DefaultPaymentInterval = uint64(1 << 20)

// DefaultPaymentIntervalIncrease is the amount interval increases on each payment,
// set to to 1Mb if the miner does not explicitly set it otherwise
var DefaultPaymentIntervalIncrease = uint64(1 << 20)

// AskStore is an interface which provides access to a persisted retrieval Ask
type AskStore interface {
	GetAsk() *Ask
	SetAsk(ask *Ask) error
}

// AskStoreImpl implements AskStore, persisting a retrieval Ask
// to disk. It also maintains a cache of the current Ask in memory
type AskStoreImpl struct {
	lk  sync.RWMutex
	ask *Ask
	ds  datastore.Batching
	key datastore.Key
}

// NewAskStore returns a new instance of AskStoreImpl
// It will initialize a new default ask and store it if one is not set.
// Otherwise it loads the current Ask from disk
func NewAskStore(ds datastore.Batching, key datastore.Key) (*AskStoreImpl, error) {
	askDs := namespace.Wrap(ds, datastore.NewKey(string(versioning.VersionKey("1"))))
	s := &AskStoreImpl{
		ds:  askDs,
		key: key,
	}

	if err := s.tryLoadAsk(); err != nil {
		return nil, err
	}

	if s.ask == nil {
		// for now set a default retrieval ask
		defaultAsk := &Ask{
			PricePerByte:            DefaultPricePerByte,
			PaymentInterval:         DefaultPaymentInterval,
			PaymentIntervalIncrease: DefaultPaymentIntervalIncrease,
		}

		if err := s.SetAsk(defaultAsk); err != nil {
			return nil, fmt.Errorf("failed setting a default retrieval ask: %w", err)
		}
	}
	return s, nil
}

// SetAsk stores retrieval provider's ask
func (s *AskStoreImpl) SetAsk(ask *Ask) error {
	s.lk.Lock()
	defer s.lk.Unlock()

	return s.saveAsk(ask)
}

// GetAsk returns the current retrieval ask, or nil if one does not exist.
func (s *AskStoreImpl) GetAsk() *Ask {
	s.lk.RLock()
	defer s.lk.RUnlock()
	if s.ask == nil {
		return nil
	}
	ask := *s.ask
	return &ask
}

func (s *AskStoreImpl) tryLoadAsk() error {
	s.lk.Lock()
	defer s.lk.Unlock()

	err := s.loadAsk()

	if err != nil {
		if xerrors.Is(err, datastore.ErrNotFound) {
			// this is expected
			return nil
		}
		return err
	}

	return nil
}

func (s *AskStoreImpl) loadAsk() error {
	askb, err := s.ds.Get(s.key)
	if err != nil {
		return xerrors.Errorf("failed to load most recent retrieval ask from disk: %w", err)
	}

	var ask Ask
	if err := cborutil.ReadCborRPC(bytes.NewReader(askb), &ask); err != nil {
		return err
	}

	s.ask = &ask
	return nil
}

func (s *AskStoreImpl) saveAsk(a *Ask) error {
	b, err := cborutil.Dump(a)
	if err != nil {
		return err
	}

	if err := s.ds.Put(s.key, b); err != nil {
		return err
	}

	s.ask = a
	return nil
}
