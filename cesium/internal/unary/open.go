package unary

import (
	"github.com/apache/arrow/go/v10/arrow/memory"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/ranger"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/config"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

type Config struct {
	// FS is where the database stores its file. This FS is assumed to be a directory
	// where DB has exclusive write access.
	FS xfs.FS
	// MetaECD is the encoder/decoder used to encode Channel information into the
	// meta file.
	MetaECD binary.EncoderDecoder
	// Channel is the Channel for the database. This only needs to be set when
	// creating a new database.
	Channel   core.Channel
	Logger    *zap.Logger
	Allocator memory.Allocator
}

var _ config.Config[Config] = (*Config)(nil)

func (c Config) Validate() error {
	v := validate.New("cesium.unary")
	validate.NotNil(v, "FS", c.FS)
	validate.NotNil(v, "MetaECD", c.MetaECD)
	validate.NotNil(v, "Logger", c.Logger)
	validate.NotNil(v, "Allocator", c.Allocator)
	return v.Error()
}

func (c Config) Override(other Config) Config {
	c.FS = override.Nil(c.FS, other.FS)
	c.MetaECD = override.Nil(c.MetaECD, other.MetaECD)
	if c.Channel.Key == "" {
		c.Channel = other.Channel
	}
	c.Logger = override.Nil(c.Logger, other.Logger)
	c.Allocator = override.Nil(c.Allocator, other.Allocator)
	return c
}

var DefaultConfig = Config{
	MetaECD:   &binary.JSONIdentEncoderDecoder{},
	Logger:    zap.NewNop(),
	Allocator: memory.DefaultAllocator,
}

func Open(cfg ...Config) (*DB, error) {
	_cfg, err := config.OverrideAndValidate(DefaultConfig, cfg...)
	if err != nil {
		return nil, err
	}
	_cfg.Channel, err = readOrCreateMeta(_cfg)
	if err != nil {
		return nil, err
	}
	rangerDB, err := ranger.Open(ranger.Config{FS: _cfg.FS, Logger: _cfg.Logger})

	db := &DB{Config: _cfg, Ranger: rangerDB}
	if _cfg.Channel.IsIndex {
		db._idx = &index.Ranger{DB: rangerDB, Logger: _cfg.Logger}
	} else if _cfg.Channel.Index == "" {
		db._idx = index.Rate{Rate: _cfg.Channel.Rate, Logger: _cfg.Logger}
	}
	return db, err
}
