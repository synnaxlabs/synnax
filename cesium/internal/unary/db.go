package unary

import (
	"fmt"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/ranger"
	"github.com/synnaxlabs/x/binary"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"os"
)

type DB struct {
	Config
	Ranger *ranger.DB
	_idx   index.Index
}

func (db *DB) Index() index.Index {
	if !db.Channel.IsIndex {
		panic(fmt.Sprintf("[ranger.unary] - database %s does not support indexing", db.Channel.Key))
	}
	return db.index()
}

func (db *DB) index() index.Index {
	if db._idx == nil {
		panic("[ranger.unary] - index is not set")
	}
	return db._idx
}

func (db *DB) SetIndex(idx index.Index) { db._idx = idx }

func (db *DB) NewWriter(cfg ranger.WriterConfig) (*Writer, error) {
	w, err := db.Ranger.NewWriter(cfg)
	return &Writer{start: cfg.Start, Channel: db.Channel, internal: w, idx: db.index()}, err
}

type IteratorConfig struct {
	Bounds telem.TimeRange
	// AutoChunkSize sets the maximum size of a chunk that will be returned by the
	// iterator when using AutoSpan in calls ot Next or Prev.
	AutoChunkSize int64
}

func IterRange(tr telem.TimeRange) IteratorConfig {
	return IteratorConfig{Bounds: ranger.IterRange(tr).Bounds, AutoChunkSize: 0}
}

var (
	DefaultIteratorConfig = IteratorConfig{AutoChunkSize: 100000}
)

func (i IteratorConfig) Override(other IteratorConfig) IteratorConfig {
	i.Bounds.Start = override.Numeric(i.Bounds.Start, other.Bounds.Start)
	i.Bounds.End = override.Numeric(i.Bounds.End, other.Bounds.End)
	i.AutoChunkSize = override.Numeric(i.AutoChunkSize, other.AutoChunkSize)
	return i
}

func (i IteratorConfig) toRanger() ranger.IteratorConfig {
	return ranger.IteratorConfig{Bounds: i.Bounds}
}

func (db *DB) NewIterator(cfg IteratorConfig) *Iterator {
	cfg = DefaultIteratorConfig.Override(cfg)
	iter := db.Ranger.NewIterator(cfg.toRanger())
	i := &Iterator{
		idx:            db.index(),
		Channel:        db.Channel,
		internal:       iter,
		logger:         db.Logger,
		IteratorConfig: cfg,
	}
	i.SetBounds(cfg.Bounds)
	return i
}

func (db *DB) Close() error { return db.Ranger.Close() }

const metaFile = "meta.json"

func readOrCreateMeta(cfg Config) (core.Channel, error) {
	exists, err := cfg.FS.Exists(metaFile)
	if err != nil {
		return cfg.Channel, err
	}
	if !exists {
		if cfg.Channel.Key == "" {
			return cfg.Channel, errors.Wrap(
				validate.Error,
				"[ranger.unary] - a channel is required when creating a new database",
			)
		}
		return cfg.Channel, createMeta(cfg.FS, cfg.MetaECD, cfg.Channel)
	}
	return readMeta(cfg.FS, cfg.MetaECD)
}

func readMeta(fs xfs.FS, ecd binary.EncoderDecoder) (core.Channel, error) {
	metaF, err := fs.Open(metaFile, os.O_RDONLY)
	var ch core.Channel
	if err != nil {
		return ch, err
	}
	if err := ecd.DecodeStream(metaF, &ch); err != nil {
		return ch, err
	}
	return ch, metaF.Close()
}

func createMeta(fs xfs.FS, ecd binary.EncoderDecoder, ch core.Channel) error {
	metaF, err := fs.Open(metaFile, os.O_CREATE|os.O_WRONLY)
	if err != nil {
		return err
	}
	b, err := ecd.Encode(ch)
	if err != nil {
		return err
	}
	if _, err := metaF.Write(b); err != nil {
		return err
	}
	return metaF.Close()
}
