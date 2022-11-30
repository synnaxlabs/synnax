package cesium

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/ranger"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/validate"
)

// NewStreamWriter implements DB.
func (db *cesium) NewStreamWriter(cfg WriterConfig) (StreamWriter, error) {
	return db.newStreamWriter(cfg)
}

// NewWriter implements DB.
func (db *cesium) NewWriter(cfg WriterConfig) (Writer, error) {
	internal, err := db.newStreamWriter(cfg)
	if err != nil {
		return nil, err
	}
	return wrapStreamWriter(internal), nil
}

func (db *cesium) newStreamWriter(cfg WriterConfig) (*streamWriter, error) {
	var (
		idx          index.Index
		writingToIdx bool
		idxChannel   Channel
		internal     = make(map[string]unary.Writer, len(cfg.Channels))
	)
	for i, key := range cfg.Channels {
		u, ok := db.dbs[key]
		if !ok {
			return nil, ChannelNotFound
		}
		if u.Channel.IsIndex {
			writingToIdx = true
		}
		if i == 0 {
			if u.Channel.Index != "" {
				idxU, err := db.getUnary(u.Channel.Index)
				if err != nil {
					return nil, err
				}
				idx = &index.Ranger{DB: idxU.Ranger, Logger: db.logger}
				idxChannel = idxU.Channel
			} else {
				idx = index.Rate{Rate: u.Channel.Rate, Logger: db.logger}
				idxChannel = u.Channel
			}
		} else {
			if err := validateSameIndex(u.Channel, idxChannel); err != nil {
				return nil, err
			}
		}
		w, err := u.NewWriter(ranger.WriterConfig{Start: cfg.Start})
		if err != nil {
			return nil, err
		}
		internal[key] = *w
	}

	w := &streamWriter{internal: internal}
	w.Start = cfg.Start
	w.idx.key = idxChannel.Key
	w.writingToIdx = writingToIdx
	w.idx.highWaterMark = cfg.Start
	w.idx.Index = idx
	return w, nil
}

func validateSameIndex(chOne, chTwo Channel) error {
	if chOne.Index == "" && chTwo.Index == "" {
		if chOne.Rate != chTwo.Rate {
			return errors.Wrapf(validate.Error, "channels must have the same rate")
		}
	}
	if chOne.Index != chTwo.Index {
		return errors.Wrapf(validate.Error, "channels must have the same index")
	}
	return nil
}
