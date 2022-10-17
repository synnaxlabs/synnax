package kv

import (
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/x/gorp"
	kvx "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
)

type DB struct {
	kvx.DB
	channel *channelService
}

func Wrap(db kvx.DB) *DB {
	return &DB{
		DB:      db,
		channel: newChannelService(db),
	}
}

func (k *DB) NewIterator(key channel.Key) (core.MDPositionIterator, error) {
	ch, err := k.GetChannel(key)
	if err != nil {
		return nil, err
	}
	return newPositionIterator(k.DB, ch), nil
}

func (k *DB) NewWriter() *Writer {
	return &Writer{
		KVWriter: gorp.WrapKVBatch[[]byte, segment.MD](k.NewBatch(), gorp.WithEncoderDecoder(segmentEncoderDecoder), gorp.WithoutTypePrefix()),
	}
}

func (k *DB) GetChannel(key channel.Key) (channel.Channel, error) {
	chs, err := k.GetChannels(key)
	if err != nil {
		return channel.Channel{}, err
	}
	if len(chs) == 0 {
		return channel.Channel{}, query.NotFound
	}
	return chs[0], nil
}

func (k *DB) GetChannels(keys ...channel.Key) ([]channel.Channel, error) {
	return k.channel.Get(keys...)
}

func (k *DB) SetChannel(ch channel.Channel) error {
	return k.channel.Set(ch)
}

func (k *DB) ChannelExists(key channel.Key) (bool, error) {
	return k.channel.Exists(key)
}
