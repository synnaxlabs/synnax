package cesium

import (
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/errutil"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

var (
	// ChannelNotFound is returned when a channel or a range of data cannot be found in the DB.
	ChannelNotFound = errors.Wrap(query.NotFound, "[cesium] - channel not found")
)

type Channel = core.Channel

func Keys(channels ...Channel) []string {
	return lo.Uniq(lo.Map(channels, func(c Channel, _ int) string { return c.Key }))
}

type DB interface {
	CreateChannel(channels ...Channel) error
	Write(start telem.TimeStamp, frame telem.Frame) error
	WriteArray(start telem.TimeStamp, arr telem.Array) error
	NewWriter(cfg WriterConfig) (Writer, error)
	NewStreamWriter(cfg WriterConfig) (StreamWriter, error)
	Read(tr telem.TimeRange, keys ...string) (telem.Frame, error)
	NewIterator(cfg IteratorConfig) (Iterator, error)
	NewStreamIterator(cfg IteratorConfig) (StreamIterator, error)
	Close() error
}

type cesium struct {
	*options
	dbs map[string]unary.DB
}

// Write implements DB.
func (db *cesium) Write(start telem.TimeStamp, frame telem.Frame) error {
	var config WriterConfig
	config.Channels = make([]string, len(frame.Arrays))
	for i, arr := range frame.Arrays {
		config.Channels[i] = arr.Key
	}
	config.Start = start
	w, err := db.NewWriter(config)
	if err != nil {
		return err
	}
	w.Write(frame)
	w.Commit()
	return w.Close()
}

func (db *cesium) WriteArray(start telem.TimeStamp, arr telem.Array) error {
	return db.Write(start, telem.Frame{Arrays: []telem.Array{arr}})
}

// Read implements DB.
func (db *cesium) Read(tr telem.TimeRange, keys ...string) (frame telem.Frame, err error) {
	var config IteratorConfig
	config.Channels = keys
	config.Bounds = tr
	iter, err := db.NewIterator(config)
	if err != nil {
		return
	}
	defer func() {
		err = iter.Close()
	}()
	if !iter.SeekFirst() {
		return
	}
	for iter.Next(telem.TimeSpanMax) {
		frame.Arrays = append(frame.Arrays, iter.Value().Arrays...)
	}
	return
}

// Close implements DB.
func (db *cesium) Close() error {
	c := errutil.NewCatch(errutil.WithAggregation())
	for _, u := range db.dbs {
		c.Exec(u.Close)
	}
	return nil
}
