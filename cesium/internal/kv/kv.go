package kv

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/gorp"
	kvx "github.com/synnaxlabs/x/kv"
)

type DB struct {
	kvx.DB
	core.ChannelWriter
	core.ChannelReader
	core.FileCounter
}

func Open(db kvx.DB) (*DB, error) {
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
	}
	return _db, err

}

func (k *DB) NewIterator(key core.ChannelKey) (core.PositionIterator, error) {
	ch, err := k.GetChannel(key)
	if err != nil {
		return nil, err
	}
	return newPositionIterator(k.DB, ch), nil
}

func (k *DB) NewWriter() (core.MDWriter, error) {
	return &Writer{
		KVWriter: gorp.WrapKVBatch[[]byte, core.SegmentMD](
			k.NewBatch(),
			gorp.WithEncoderDecoder(segmentEncoderDecoder),
			gorp.WithoutTypePrefix(),
		),
	}, nil
}
