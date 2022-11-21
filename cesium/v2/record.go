package v2

import (
	"encoding/binary"
	"encoding/json"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/v2/index"
	"github.com/synnaxlabs/cesium/v2/ranger"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"os"
	"strconv"
)

type ChannelKey uint16

type Record struct {
	ChannelKeys []ChannelKey
	Data        [][]byte
}

type Channel struct {
	Key      ChannelKey
	Rate     telem.Rate
	DataType telem.DataType
	Index    ChannelKey
	IsIndex  bool
}

type rangerDB struct {
	ch *Channel
	db *ranger.DB
}

type DB interface {
	CreateChannel(channel *Channel) error
	NewWriter(start telem.TimeStamp) (Writer, error)
	NewIterator()
	Close() error
}

type db struct {
	rootFS     xfs.FS
	rangers    map[ChannelKey]rangerDB
	keyCounter ChannelKey
}

func (db *db) CreateChannel(ch *Channel) error {
	if ch.Index != 0 {
		idx, ok := db.rangers[ch.Index]
		if !ok {
			return errors.Wrap(query.NotFound, "index channel not found")
		}
		if !idx.ch.IsIndex {
			return errors.Wrap(validate.Error, "index channel is not an index")
		}
		db.keyCounter++
		ch.Key = db.keyCounter
		subFS, err := db.rootFS.Sub(strconv.Itoa(int(ch.Key)))
		if err != nil {
			return errors.Wrap(err, "failed to create sub fs")
		}
		// create a file called meta.json in subFS and marshal the channel into it
		metaFile, err := subFS.OpenFile("meta.json", os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return errors.Wrap(err, "failed to create meta file")
		}
		defer metaFile.Close()
		if err := json.NewEncoder(metaFile).Encode(ch); err != nil {
			return err
		}
		rDB, err := ranger.Open(subFS)
		if err != nil {
			return err
		}
		db.rangers[ch.Key] = rangerDB{
			ch: ch,
			db: rDB,
		}
	}
	return nil
}

type writerMeta struct {
	ch      *Channel
	wrapped *ranger.Writer
	idx     index.Index
	hwm     telem.TimeStamp
}

type iterMeta struct {
	ch                    *Channel
	idx                   index.Index
	wrapped               *ranger.Iterator
	artificialStartOffset int64
	artificialEndOffset   int64
}

type iterator struct {
	bounds  telem.TimeRange
	wrapped map[ChannelKey]iterMeta
}

func (i *iterator) SetBounds(bounds telem.TimeRange) {
	i.bounds = bounds
	for _, itMeta := range i.wrapped {
		itMeta.wrapped.SetBounds(bounds)
		if itMeta.wrapped.SeekFirst() && itMeta.wrapped.Range().ContainsStamp(bounds.Start) {
			// we need to artificially change the iterator bounds
			rng := itMeta.wrapped.Range()
			distances, err := itMeta.idx.Distance(i.bounds.Start, []telem.TimeStamp{rng.Start})
			if err != nil {
				panic(err)
			}
			itMeta.artificialStartOffset = distances[0]
		}
		if itMeta.wrapped.SeekFirst() && !itMeta.wrapped.Range().ContainsStamp(bounds.End) {
			rng := itMeta.wrapped.Range()
			distances, err := itMeta.idx.Distance(i.bounds.End, []telem.TimeStamp{rng.End})
			if err != nil {
				panic(err)
			}
			itMeta.artificialEndOffset = distances[0]
		}
	}
}

func (i *iterator) SeekFirst() bool {

}

type writer struct {
	start   telem.TimeStamp
	wrapped map[ChannelKey]writerMeta
}

func (w *writer) Write(record *Record) error {
	for i, chKey := range record.ChannelKeys {
		wr, ok := w.wrapped[chKey]
		if !ok {
			return errors.Wrap(query.NotFound, "channel not found")
		}
		d := record.Data[i]
		if _, err := wr.wrapped.Write(d); err != nil {
			return err
		}
		if wr.ch.IsIndex {
			wr.hwm = decodeTimestamp(d[len(d)-8:])
		}
	}
	return nil
}

func decodeTimestamp(data []byte) telem.TimeStamp {
	return telem.TimeStamp(binary.LittleEndian.Uint64(data))
}

func (w *writer) Commit() error {
	for _, wr := range w.wrapped {
		if wr.ch.IsIndex {
			if err := wr.wrapped.Commit(wr.hwm); err != nil {
				return err
			}
		}
	}
	for _, wr := range w.wrapped {
		if !wr.ch.IsIndex {
			end, err := wr.idx.End(w.start, wr.wrapped.Len())
			if err != nil {
				return err
			}
			contig, err := wr.idx.DefinedAndContiguous(telem.TimeRange{
				Start: w.start,
				End:   end,
			})
			if err != nil {
				return err
			}
			if !contig {
				return errors.Wrap(validate.Error, "index not defined or contiguous")
			}
			if err := wr.wrapped.Commit(end); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *writer) Close() error {
	for _, wr := range w.wrapped {
		if err := wr.wrapped.Close(); err != nil {
			return err
		}
	}
	return nil
}

type Writer interface {
	Write(record *Record) error
	Commit() error
	Close() error
}

type Iterator interface {
	SetBounds(tr telem.TimeRange)
	Next() bool
	Prev() bool
	SeekFirst() bool
	SeekLast() bool
	SeekGE(telem.TimeStamp) bool
	SeekLE(telem.TimeStamp) bool
}
