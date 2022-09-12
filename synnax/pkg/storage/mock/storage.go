package mock

import (
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/arya-analytics/x/config"
	"github.com/arya-analytics/x/errutil"
	"github.com/arya-analytics/x/fsutil"
	"os"
)

type Builder struct {
	cfg    storage.Config
	Stores []*storage.Store
}

func NewBuilder(cfg ...storage.Config) *Builder {
	_cfg, err := config.OverrideAndValidate(storage.DefaultConfig, append([]storage.Config{{
		MemBacked: config.BoolPointer(true),
		Perm:      fsutil.OS_USER_RWX,
	}}, cfg...)...)
	if err != nil {
		panic(err)
	}

	if !*_cfg.MemBacked {
		if err := os.MkdirAll(_cfg.Dirname, _cfg.Perm); err != nil {
			panic(err)
		}
	}

	return &Builder{cfg: _cfg}
}

func (b *Builder) New() (store *storage.Store) {
	if *b.cfg.MemBacked {
		store = b.newMemBacked()
	} else {
		store = b.newFSBacked()
	}
	b.Stores = append(b.Stores, store)
	return store
}

func (b *Builder) Cleanup() error {
	if *b.cfg.MemBacked {
		return nil
	}
	return os.RemoveAll(b.cfg.Dirname)
}

func (b *Builder) Close() error {
	c := errutil.NewCatch(errutil.WithAggregation())
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
