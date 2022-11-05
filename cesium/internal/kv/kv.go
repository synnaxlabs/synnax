package kv

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/gorp"
	kvx "github.com/synnaxlabs/x/kv"
	"go.uber.org/zap"
)

type DB struct {
	kvx.DB
	core.ChannelWriter
	core.ChannelReader
	core.FileCounter
	logger *zap.Logger
}

func Open(db kvx.DB, logger *zap.Logger) (*DB, error) {
	ce, err := openChannelEngine(db, "channel-counter")
	c, err := openFileCounter(db, []byte("file-counter"))
	if err != nil {
		return nil, err
	}
	_db := &DB{
		DB:            db,
		ChannelWriter: ce,
		ChannelReader: ce,
		FileCounter:   c,
		logger:        logger,
	}
	return _db, err

}

func (k *DB) NewIterator(ch core.Channel) core.PositionIterator {
	return newPositionIterator(k.DB, ch, k.logger)
}

func (k *DB) NewWriter() core.MDWriter {
	return &Writer{
		KVWriter: gorp.WrapKVBatch[[]byte, core.SegmentMD](
			k.NewBatch(),
			gorp.WithEncoderDecoder(segmentEncoderDecoder),
			gorp.WithoutTypePrefix(),
		),
	}
}
