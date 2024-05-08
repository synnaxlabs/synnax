// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// WriterMode sets the operating mode of the writer, optionally enabling or disabling
// persistence and streaming.
type WriterMode uint8

// Persist returns true if the current mode should persist data.
func (w WriterMode) Persist() bool { return w != WriterStreamOnly }

// Stream returns true if the current mode should stream data.
func (w WriterMode) Stream() bool { return w != WriterPersistOnly }

const (
	WriterPersistStream = iota + 1
	WriterPersistOnly
	WriterStreamOnly
)

// WriterConfig sets the configuration used to open a new writer on the DB.
type WriterConfig struct {
	// Name sets the human-readable name for the writer, which is useful for identifying
	// it in control transfer scenarios.
	// [OPTIONAL] - Defaults to an empty string.
	ControlSubject control.Subject
	// Start marks the starting timestamp of the first sample to be written by the
	// writer. If a sample exists for any channel at this timestamp, the writer will
	// fail to open.
	Start telem.TimeStamp
	// Channels sets the channels that the writer will write to. If a channel does
	// not exist, the writer fill fail to open.
	Channels []core.ChannelKey
	// Authorities marks the starting control authorities of the writer.
	Authorities []control.Authority
	// ErrOnUnauthorized controls whether the writer will return an error when
	// attempting to write to a channel that it does not have authority over.
	// In non-control scenarios, this value should be set to true. In scenarios
	// that require control handoff, this value should be set to false.
	ErrOnUnauthorized *bool
	// Mode sets the persistence and streaming mode of the writer. The default
	// mode is WriterModePersistStream. See the WriterMode documentation for more.
	// [OPTIONAL] - Defaults to WriterModePersistStream.
	Mode WriterMode
	// EnableAutoCommit determines whether the writer will automatically commit after each write.
	// If EnableAutoCommit is true, then the writer will commit after each write, and will
	// persist that commit to index after the specified AutoPersistInterval.
	// [OPTIONAL] - Defaults to false.
	EnableAutoCommit *bool
	// AutoPersistInterval is the interval at which commits to the index will be persisted.
	// To persist every commit to guarantee minimal loss of data, set AutoPersistInterval
	// to AlwaysAutoPersist.
	// [OPTIONAL] - Defaults to 1s.
	AutoPersistInterval telem.TimeSpan
}

const AlwaysAutoPersist telem.TimeSpan = -1

var (
	_ config.Config[WriterConfig] = WriterConfig{}
)

func DefaultWriterConfig() WriterConfig {
	return WriterConfig{
		ControlSubject: control.Subject{
			Key: uuid.New().String(),
		},
		Authorities:         []control.Authority{control.Absolute},
		ErrOnUnauthorized:   config.False(),
		Mode:                WriterPersistStream,
		EnableAutoCommit:    config.Bool(false),
		AutoPersistInterval: 1 * telem.Second,
	}
}

// Validate implements config.GateConfig.
func (w WriterConfig) Validate() error {
	v := validate.New("cesium.WriterConfig")
	validate.NotEmptySlice(v, "Channels", w.Channels)
	validate.NotNil(v, "ErrOnUnauthorized", w.ErrOnUnauthorized)
	validate.NotEmptyString(v, "ControlSubject.Key", w.ControlSubject.Key)
	v.Ternary(
		len(w.Authorities) != len(w.Channels) && len(w.Authorities) != 1,
		"authority count must be 1 or equal to channel count",
	)
	v.Ternary(w.AutoPersistInterval != 1*telem.Second && !*w.EnableAutoCommit, "AutoPersist interval cannot be set without EnableAutoCommit")
	v.Ternary(w.AutoPersistInterval < 0 && w.AutoPersistInterval != -1, "AutoIndexPersistInterval cannot be a negative number except for AlwaysAutoPersist")
	return v.Error()
}

// Override implements config.GateConfig.
func (w WriterConfig) Override(other WriterConfig) WriterConfig {
	w.Start = override.Zero(w.Start, other.Start)
	w.Channels = override.Slice(w.Channels, other.Channels)
	w.Authorities = override.Slice(w.Authorities, other.Authorities)
	w.ControlSubject.Name = override.String(w.ControlSubject.Name, other.ControlSubject.Name)
	w.ControlSubject.Key = override.String(w.ControlSubject.Key, other.ControlSubject.Key)
	w.ErrOnUnauthorized = override.Nil(w.ErrOnUnauthorized, other.ErrOnUnauthorized)
	w.Mode = override.Numeric(w.Mode, other.Mode)
	w.EnableAutoCommit = override.Nil(w.EnableAutoCommit, other.EnableAutoCommit)
	w.AutoPersistInterval = override.Zero(w.AutoPersistInterval, other.AutoPersistInterval)
	return w
}

func (w WriterConfig) authority(i int) control.Authority {
	if len(w.Authorities) == 1 {
		return w.Authorities[0]
	}
	return w.Authorities[i]
}

// NewStreamWriter implements DB.
func (db *DB) NewStreamWriter(ctx context.Context, cfgs ...WriterConfig) (StreamWriter, error) {
	return db.newStreamWriter(ctx, cfgs...)
}

// OpenWriter implements DB.
func (db *DB) OpenWriter(ctx context.Context, cfgs ...WriterConfig) (*Writer, error) {
	internal, err := db.newStreamWriter(ctx, cfgs...)
	if err != nil {
		return nil, err
	}
	return wrapStreamWriter(internal), nil
}

func (db *DB) newStreamWriter(ctx context.Context, cfgs ...WriterConfig) (w *streamWriter, err error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	cfg, err := config.New(DefaultWriterConfig(), cfgs...)
	if err != nil {
		return nil, err
	}

	var (
		domainWriters  map[ChannelKey]*idxWriter
		rateWriters    map[telem.Rate]*idxWriter
		virtualWriters map[ChannelKey]*virtual.Writer
		controlUpdate  ControlUpdate
	)

	defer func() {
		if err == nil {
			return
		}
		for _, idx := range domainWriters {
			_, err_ := idx.Close(ctx)
			err = errors.CombineErrors(err_, err)
		}
		for _, idx := range rateWriters {
			_, err_ := idx.Close(ctx)
			err = errors.CombineErrors(err_, err)
		}
	}()

	for i, key := range cfg.Channels {
		u, uOk := db.unaryDBs[key]
		v, vOk := db.virtualDBs[key]
		if !vOk && !uOk {
			return nil, core.ChannelNotFound
		}
		var (
			auth     = cfg.authority(i)
			transfer controller.Transfer
		)
		if vOk {
			if virtualWriters == nil {
				virtualWriters = make(map[ChannelKey]*virtual.Writer)
			}
			virtualWriters[key], transfer, err = v.OpenWriter(ctx, virtual.WriterConfig{
				Subject:   cfg.ControlSubject,
				Start:     cfg.Start,
				Authority: auth,
			})
			if err != nil {
				return nil, err
			}
		} else {
			var w *unary.Writer
			w, transfer, err = u.OpenWriter(ctx, unary.WriterConfig{
				Subject:   cfg.ControlSubject,
				Start:     cfg.Start,
				Authority: auth,
				Persist:   config.Bool(cfg.Mode.Persist()),

				EnableAutoCommit:         cfg.EnableAutoCommit,
				AutoIndexPersistInterval: cfg.AutoPersistInterval,
			})
			if err != nil {
				return nil, err
			}
			if u.Channel.Index != 0 {
				// Hot path optimization: in the common case we only write to a rate based
				// index or a domain indexed channel, not both. In either case we can avoid a
				// map allocation.
				if domainWriters == nil {
					domainWriters = make(map[ChannelKey]*idxWriter)
				}
				idxW, exists := domainWriters[u.Channel.Index]
				if !exists {
					// If there is no existing index writer for this index-group.
					idxW, err = db.openDomainIdxWriter(u.Channel.Index, cfg)
					if err != nil {
						return nil, err
					}
					idxW.writingToIdx = u.Channel.IsIndex
					domainWriters[u.Channel.Index] = idxW
				} else if u.Channel.IsIndex {
					idxW.writingToIdx = true
					domainWriters[u.Channel.Index] = idxW
				}

				idxW.internal[key] = &unaryWriterState{Writer: *w}
			} else {
				// Hot path optimization: in the common case we only write to a rate based
				// index or an indexed channel, not both. In either case we can avoid a
				// map allocation.
				if rateWriters == nil {
					rateWriters = make(map[telem.Rate]*idxWriter)
				}

				idxW, ok := rateWriters[u.Channel.Rate]
				if !ok {
					idxW = db.openRateIdxWriter(u.Channel.Rate, cfg)
					rateWriters[u.Channel.Rate] = idxW
				}

				idxW.internal[key] = &unaryWriterState{Writer: *w}
			}
			if transfer.Occurred() {
				controlUpdate.Transfers = append(controlUpdate.Transfers, transfer)
			}
		}
	}

	if len(controlUpdate.Transfers) > 0 {
		db.updateControlDigests(ctx, controlUpdate)
	}

	w = &streamWriter{
		WriterConfig:    cfg,
		internal:        make([]*idxWriter, 0, len(domainWriters)+len(rateWriters)),
		relay:           db.relay.inlet,
		virtual:         &virtualWriter{internal: virtualWriters, digestKey: db.digests.key},
		updateDBControl: db.updateControlDigests,
	}
	for _, idx := range domainWriters {
		w.internal = append(w.internal, idx)
	}
	for _, idx := range rateWriters {
		w.internal = append(w.internal, idx)
	}
	return w, nil
}

func (db *DB) openDomainIdxWriter(
	idxKey ChannelKey,
	cfg WriterConfig,
) (*idxWriter, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	u, ok := db.unaryDBs[idxKey]
	if !ok {
		return nil, core.ChannelNotFound
	}
	idx := &index.Domain{DB: u.Domain, Instrumentation: db.Instrumentation}
	w := &idxWriter{internal: make(map[ChannelKey]*unaryWriterState)}
	w.idx.key = idxKey
	w.idx.Index = idx
	w.idx.highWaterMark = cfg.Start
	w.writingToIdx = false
	w.start = cfg.Start
	return w, nil
}

func (db *DB) openRateIdxWriter(
	rate telem.Rate,
	cfg WriterConfig,
) *idxWriter {
	idx := index.Rate{Rate: rate}
	w := &idxWriter{internal: make(map[ChannelKey]*unaryWriterState)}
	w.idx.Index = idx
	w.start = cfg.Start
	return w
}
