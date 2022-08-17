package mock

import (
	"github.com/arya-analytics/delta/pkg/storage"
	"github.com/arya-analytics/x/errutil"
	"github.com/arya-analytics/x/fsutil"
	"os"
)

type Builder struct {
	cfg    storage.Config
	Stores []*storage.Store
}

func NewBuilder(cfg ...storage.Config) *Builder {
	var _cfg storage.Config
	if len(cfg) > 0 {
		_cfg = cfg[0]
	} else {
		_cfg = storage.Config{MemBacked: true}
	}
	_cfg.Perm = fsutil.OS_USER_RWX
	_cfg = _cfg.Merge(storage.DefaultConfig())

	if !_cfg.MemBacked {
		if err := os.MkdirAll(_cfg.Dirname, _cfg.Perm); err != nil {
			panic(err)
		}
	}

	return &Builder{cfg: _cfg}
}

func (b *Builder) New() (store *storage.Store) {
	if b.cfg.MemBacked {
		store = b.newMemBacked()
	} else {
		store = b.newFSBacked()
	}
	b.Stores = append(b.Stores, store)
	return store
}

func (b *Builder) Cleanup() error {
	if b.cfg.MemBacked {
		return nil
	}
	return os.RemoveAll(b.cfg.Dirname)
}

func (b *Builder) Close() error {
	c := errutil.NewCatchSimple(errutil.WithAggregation())
	for _, store := range b.Stores {
		c.Exec(store.Close)
	}
	return c.Error()
}

func (b *Builder) newMemBacked() *storage.Store {
	store, err := storage.Open(b.cfg)
	if err != nil {
		panic(err)
	}
	return store
}

func (b *Builder) newFSBacked() *storage.Store {
	// open a temporary directory prefixed with cfg.dirname
	tempDir, err := os.MkdirTemp(b.cfg.Dirname, "delta-test-")
	if err != nil {
		panic(err)
	}
	nCfg := b.cfg
	nCfg.Dirname = tempDir
	store, err := storage.Open(nCfg)
	if err != nil {
		panic(err)
	}
	return store
}
