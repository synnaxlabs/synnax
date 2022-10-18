package index_test

import (
	"bytes"
	"encoding/binary"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/kv"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/cesium/internal/storage"
	"github.com/synnaxlabs/x/kfs"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/telem"
)

type indexWriter struct {
	channelKey core.ChannelKey
	*kv.DB
	storage.Writer
}

func (i indexWriter) write(
	alignment position.Position,
	timestamps []telem.TimeStamp,
) {
	buf := bytes.NewBuffer(make([]byte, 0, len(timestamps)*int(telem.TimeStampDensity)))
	for _, ts := range timestamps {

		lo.Must0(binary.Write(buf, binary.BigEndian, int64(ts)))
	}
	b := buf.Bytes()
	seg := core.SugaredSegment{
		Data: b,
		SegmentMD: core.SegmentMD{
			ChannelKey: i.channelKey,
			Alignment:  alignment,
			FileKey:    1,
		},
	}
	mds := lo.Must(i.Writer.Write([]core.SugaredSegment{seg}))
	mdw := lo.Must(i.DB.NewWriter())
	lo.Must0(mdw.Write(mds))
	lo.Must0(mdw.Commit())
	lo.Must0(mdw.Close())
}

var _ = Describe("Reader", Ordered, func() {
	var (
		newR  func() *index.Reader
		i     indexWriter
		db    *kv.DB
		store *storage.Storage
		r     *index.Reader
	)
	BeforeAll(func() {
		_kfs := lo.Must(kfs.New[core.FileKey]("", kfs.WithFS(kfs.NewMem())))
		store = storage.Wrap(_kfs)
		db = lo.Must(kv.Open(memkv.New()))

	})
	BeforeEach(func() {
		key := lo.Must(db.NextChannelKey())
		Expect(db.SetChannel(core.Channel{
			Key:     key,
			IsIndex: true,
			Density: telem.TimeStampDensity,
		})).To(Succeed())
		newR = func() *index.Reader {
			r = &index.Reader{
				Reader: store.NewReader(),
				Iter:   lo.Must(db.NewIterator(key)),
			}
			return r
		}
		i = indexWriter{
			channelKey: key,
			DB:         db,
			Writer:     store.NewWriter(),
		}
	})
	AfterEach(func() {
		Expect(r.Release()).To(Succeed())
	})

	AfterAll(func() {
		Expect(db.Close()).To(Succeed())
	})
	Context("Empty", func() {
		Describe("SearchP", func() {
			It("Should return a completely uncertain approximation", func() {
				Expect(newR().SearchP(1000, position.Uncertain)).To(Equal(position.Uncertain))
			})
		})
		Describe("SearchTS", func() {
			It("Should return a completely uncertain approximation", func() {
				Expect(newR().SearchTS(1000, telem.Uncertain)).To(Equal(telem.Uncertain))
			})
		})
	})
	Context("Exact Match", func() {
		Describe("SearchP", func() {
			It("Should return the exact position", func() {
				i.write(100, []telem.TimeStamp{1000, 1200, 1300, 1500})
				Expect(newR().SearchP(1300, position.Uncertain)).To(Equal(position.ExactlyAt(102)))
			})
		})
		Describe("SearchTS", func() {
			It("Should return the exact timestamp", func() {
				i.write(100, []telem.TimeStamp{1000, 1200, 1300, 1500})
				Expect(newR().SearchTS(103, telem.Uncertain)).To(Equal(telem.ExactlyAt(1500)))
			})
		})
	})
	Context("Inexact Match", func() {
		Describe("SearchP", func() {
			It("Should return an approximation between the two closest positions", func() {
				i.write(100, []telem.TimeStamp{1000, 1200, 1300, 1500})
				Expect(newR().SearchP(1250, position.Uncertain)).To(Equal(position.Between(101, 102)))
			})
		})
		Describe("SearchTS", func() {
			It("Should return an approximation between the two closest timestamps", func() {
				i.write(100, []telem.TimeStamp{1000, 1200, 1300, 1500})
				i.write(105, []telem.TimeStamp{1800, 1900, 2000, 2100})
				Expect(newR().SearchTS(104, telem.Uncertain)).To(Equal(telem.Between(1500, 1800)))
			})
		})
	})
	Context("Before First", func() {
		Describe("SearchP", func() {
			It("Should return an approximation before the first value", func() {
				i.write(100, []telem.TimeStamp{1000, 1200, 1300, 1500})
				Expect(newR().SearchP(900, position.Uncertain)).To(Equal(position.Before(100)))
			})
		})
		Describe("SearchTS", func() {
			It("Should return an approximation before the first value", func() {
				i.write(100, []telem.TimeStamp{1000, 1200, 1300, 1500})
				Expect(newR().SearchTS(99, telem.Uncertain)).To(Equal(telem.Before(1000)))
			})
		})
	})
	Context("After Last", func() {
		Describe("SearchP", func() {
			It("Should return an approximation after the last value", func() {
				i.write(100, []telem.TimeStamp{1000, 1200, 1300, 1500})
				Expect(newR().SearchP(1600, position.Uncertain)).To(Equal(position.After(103)))
			})
		})
		Describe("SearchTS", func() {
			It("Should return an approximation after the last value", func() {
				i.write(100, []telem.TimeStamp{1000, 1200, 1300, 1500})
				Expect(newR().SearchTS(110, telem.Uncertain)).To(Equal(telem.After(1500)))
			})
		})
	})
})
