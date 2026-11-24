// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"maps"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// samples holds the bytes of each key that a read returns, and the series count.
type samples struct {
	data   map[cesium.ChannelKey][]byte
	series map[cesium.ChannelKey]int
}

// crashes checks a crash image of disk at each call of take while taking is true, and
// keeps a line for each image that check fails. It holds one image at a time, since
// each one holds the files of two logs.
type crashes struct {
	mu       sync.Mutex
	disk     *fault.FS
	taking   bool
	check    func(image *fault.FS) error
	images   int
	failures []string
}

func (c *crashes) take() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.taking {
		return
	}
	image, err := c.disk.Crash(fault.CrashConfig{
		RunKey:            uint64(c.images),
		EntrySurvivalRate: 0.5,
		DataSurvivalRate:  0.5,
	})
	if err == nil {
		err = c.check(image)
	}
	if err != nil {
		c.failures = append(c.failures, fmt.Sprintf("image %d: %v", c.images, err))
	}
	c.images++
}

// dataOf returns the bytes of each of keys that a read of db returns.
func dataOf(
	ctx context.Context,
	db *cesium.DB,
	keys []cesium.ChannelKey,
) (map[cesium.ChannelKey][]byte, error) {
	fr, err := db.Read(ctx, telem.TimeRangeMax, keys...)
	if err != nil {
		return nil, err
	}
	data := make(map[cesium.ChannelKey][]byte)
	for _, key := range keys {
		for _, series := range fr.Get(key).Series {
			data[key] = append(data[key], series.Data...)
		}
	}
	return data, nil
}

var _ = Describe("Conversion on the pointer log", func() {
	const (
		idx cesium.ChannelKey = 1
		i32 cesium.ChannelKey = 2
		str cesium.ChannelKey = 3
	)
	var (
		keys = []cesium.ChannelKey{idx, i32, str}
		disk *fault.FS
		held samples
	)
	read := func(ctx context.Context, db *cesium.DB) samples {
		GinkgoHelper()
		s := samples{
			data:   make(map[cesium.ChannelKey][]byte),
			series: make(map[cesium.ChannelKey]int),
		}
		fr := MustSucceed(db.Read(ctx, telem.TimeRangeMax, keys...))
		for _, key := range keys {
			for _, series := range fr.Get(key).Series {
				s.data[key] = append(s.data[key], series.Data...)
				s.series[key]++
			}
		}
		return s
	}
	// write writes 5 samples of each key per commit, in 4 commits from start seconds.
	write := func(ctx context.Context, db *cesium.DB, start telem.TimeStamp) {
		GinkgoHelper()
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Clock:    keys[0],
			Start:    start * telem.SecondTS,
		}))
		for c := range telem.TimeStamp(4) {
			var (
				stamps = make([]telem.TimeStamp, 5)
				ints   = make([]int32, 5)
				strs   = make([]string, 5)
			)
			for j := range telem.TimeStamp(5) {
				n := c*5 + j
				stamps[j] = (start + n) * telem.SecondTS
				ints[j] = int32(start + n)
				strs[j] = strings.Repeat("x", int(n%7))
			}
			MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
				telem.NewSeries(stamps),
				telem.NewSeries(ints),
				telem.NewSeries(strs),
			}), nil))
			MustSucceed2(w.Commit(ctx, nil))
		}
		Expect(w.Close()).To(Succeed())
	}
	// parted opens disk with a part size that sends every sealed record to a part.
	parted := func(ctx context.Context) (*cesium.DB, func() error) {
		GinkgoHelper()
		db := MustSucceed(cesium.Open(
			ctx,
			"",
			cesium.WithFS(disk),
			cesium.WithPointerLog(),
			cesium.WithPartSize(1),
		))
		stop := GoRun(ctx, db.Run)
		return db, func() error { return errors.Join(stop(), db.Close()) }
	}
	BeforeEach(func(ctx SpecContext) {
		disk = MustSucceed(fault.New(fault.Config{}))
		// Small files roll each key over at its own commits.
		db := openDBOnFS(ctx, disk, cesium.WithFileSizeCap(48*telem.Byte))
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key:      idx,
				Name:     "time",
				IsIndex:  true,
				DataType: telem.TimestampT,
			},
			cesium.Channel{Key: i32, Name: "i32", Index: idx, DataType: telem.Int32T},
			cesium.Channel{Key: str, Name: "str", Index: idx, DataType: telem.StringT},
		)).To(Succeed())
		write(ctx, db, 1)
		write(ctx, db, 1000)
		held = read(ctx, db)
		Expect(db.Close()).To(Succeed())
		// The old layout never syncs. Writeback makes its files durable, as time does.
		disk.Writeback()
	})

	It("Should read every sample that the old layout held", func(ctx SpecContext) {
		db, closeDB := MustSucceed2(openLog(ctx, disk))
		got := read(ctx, db)
		Expect(got.data).To(Equal(held.data))
		Expect(got.series[i32]).To(BeNumerically(">", 2), "data pointers are split")
		Expect(
			MustSucceed(db.RetrieveChannel(ctx, str)).DataType,
		).To(Equal(telem.StringT))
		fr := MustSucceed(db.Read(ctx, telem.TimeRangeMax, keys...))
		stretches := make(map[uint32]telem.Series)
		for _, s := range fr.Get(idx).Series {
			Expect(s.Alignment.SampleIndex()).To(BeZero())
			stretches[s.Alignment.DomainIndex()] = s
		}
		for _, key := range keys[1:] {
			for _, s := range fr.Get(key).Series {
				clock, ok := stretches[s.Alignment.DomainIndex()]
				Expect(ok).To(BeTrue(), "series at %s has no stretch", s.Alignment)
				Expect(uint64(s.Alignment.SampleIndex()) + uint64(s.Len())).
					To(BeNumerically("<=", clock.Len()))
			}
		}
		Expect(closeDB()).To(Succeed())
		db, closeDB = MustSucceed2(openLog(ctx, disk))
		Expect(read(ctx, db).data).To(Equal(held.data))
		Expect(closeDB()).To(Succeed())
	})

	It("Should read each converted sample once when head 1 holds parts, across "+
		"reopens and a switch", func(ctx SpecContext) {
		db, closeDB := parted(ctx)
		h, _ := MustSucceed2(checkpoint.Read(disk, 1))
		Expect(h.Parts).ToNot(BeEmpty())
		Expect(read(ctx, db).data).To(Equal(held.data))
		Expect(closeDB()).To(Succeed())
		// The reopen reads the parts through the ranges of head 1.
		db, closeDB = parted(ctx)
		Expect(read(ctx, db).data).To(Equal(held.data))
		Expect(switchLog(db, disk, 2)).To(Succeed())
		waitHead(disk, 2)
		Expect(read(ctx, db).data).To(Equal(held.data))
		Expect(closeDB()).To(Succeed())
		db, closeDB = parted(ctx)
		Expect(read(ctx, db).data).To(Equal(held.data))
		Expect(closeDB()).To(Succeed())
	})

	It(
		"Should unlink the old files of a dropped key that a part holds at its last read",
		func(ctx SpecContext) {
			_, closeDB := parted(ctx)
			Expect(closeDB()).To(Succeed())
			// The reopen holds the converted records only in the parts of head 1.
			db, closeDB := parted(ctx)
			iter := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
				Bounds:   telem.TimeRangeMax,
				Channels: []cesium.ChannelKey{i32},
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(db.Drop(ctx, []cesium.ChannelKey{i32})).To(Succeed())
			dir := strconv.FormatUint(uint64(i32), 10)
			Expect(disk.List(dir)).To(ContainElement(
				WithTransform(xfs.FileInfo.Name, HaveSuffix(".domain")),
			), "the iterator pins the old files")
			Expect(iter.Close()).To(Succeed())
			Expect(disk.List(dir)).To(BeEmpty())
			Expect(closeDB()).To(Succeed())
		},
	)

	It("Should keep a channel directory with no meta.json through the first Sweep",
		func(ctx SpecContext) {
			// The conversion skips such a directory, which a power loss can leave, and
			// leaves it in place.
			sub := MustSucceed(disk.Sub("7"))
			f := MustSucceed(sub.Open("1.domain", os.O_CREATE|os.O_RDWR))
			MustSucceed(f.Write([]byte{1, 2, 3, 4}))
			Expect(f.Close()).To(Succeed())
			disk.Writeback()
			db, closeDB := MustSucceed2(openLog(ctx, disk))
			MustSucceed2(db.Sweep(ctx, nil))
			Expect(closeDB()).To(Succeed())
			Expect(disk.Exists("7/1.domain")).To(BeTrue())
		})

	It("Should write after the converted samples", func(ctx SpecContext) {
		db, closeDB := MustSucceed2(openLog(ctx, disk))
		write(ctx, db, 5000)
		got := read(ctx, db)
		for _, key := range keys {
			Expect(got.data[key]).To(HavePrefix(string(held.data[key])))
			Expect(len(got.data[key])).To(BeNumerically(">", len(held.data[key])))
		}
		Expect(closeDB()).To(Succeed())
	})

	It("Should commit no head when the converted state breaks a rule", func(
		ctx SpecContext,
	) {
		fs := xfs.NewMem()
		put := func(name string, b []byte) {
			GinkgoHelper()
			sub := MustSucceed(fs.Sub(path.Dir(name)))
			f := MustSucceed(sub.Open(path.Base(name), os.O_CREATE|os.O_RDWR))
			Expect(f.WriteAt(b, 0)).To(Equal(len(b)))
			Expect(f.Close()).To(Succeed())
		}
		put("1/meta.json", MustSucceed(json.NewCodec().Encode(cesium.Channel{
			Key:      1,
			Name:     "time",
			IsIndex:  true,
			DataType: telem.TimestampT,
			Version:  3,
		})))
		put("1/1.domain", telem.NewSeriesSecondsTSV(1, 2, 3).Data)
		put("1/2.domain", telem.NewSeriesSecondsTSV(6, 5, 4).Data)
		// The second old index pointer holds stamps that fall.
		ptrs := make([]byte, 52)
		for i, p := range []struct {
			start, end telem.TimeStamp
			file       uint16
		}{{1, 4, 1}, {4, 7, 2}} {
			e := ptrs[26*i:]
			binary.LittleEndian.PutUint64(e, uint64(p.start*telem.SecondTS))
			binary.LittleEndian.PutUint64(e[8:], uint64(p.end*telem.SecondTS))
			binary.LittleEndian.PutUint16(e[16:], p.file)
			binary.LittleEndian.PutUint32(e[22:], 24)
		}
		put("1/index.domain", ptrs)
		Expect(openLog(ctx, fs)).Error().To(SatisfyAll(
			MatchError(ContainSubstring("refused by the state")),
			MatchError(ContainSubstring("from 6000000000 to 4000000000")),
		))
		Expect(fs.Exists(checkpoint.Name(1))).To(BeFalse())
	})

	It("Should leave a root that both binaries open with every sample after a crash "+
		"at each step", func(ctx SpecContext) {
		// same reads keys from db and compares their bytes with held.
		same := func(db *cesium.DB, closeDB func() error) error {
			data, err := dataOf(ctx, db, keys)
			if err = errors.Join(err, closeDB()); err != nil {
				return err
			}
			if !maps.EqualFunc(data, held.data, bytes.Equal) {
				return errors.New("the read differs from the old layout")
			}
			return nil
		}
		c := &crashes{disk: disk, taking: true, check: func(image *fault.FS) error {
			fork, err := image.Crash(fault.CrashConfig{
				DataSurvivalRate:  1,
				EntrySurvivalRate: 1,
			})
			if err != nil {
				return err
			}
			old, err := cesium.Open(ctx, "", cesium.WithFS(fork))
			if err != nil {
				return errors.Wrap(err, "old binary")
			}
			if err = same(old, old.Close); err != nil {
				return errors.Wrap(err, "old binary")
			}
			db, closeDB, err := openLog(ctx, image)
			if err != nil {
				return errors.Wrap(err, "pointer log")
			}
			return errors.Wrap(same(db, closeDB), "pointer log")
		}}
		_, closeDB := MustSucceed2(openLog(ctx, durableFS{FS: disk, onEvent: c.take}))
		c.mu.Lock()
		c.taking = false
		c.mu.Unlock()
		Expect(closeDB()).To(Succeed())
		Expect(c.images).To(BeNumerically(">", 10))
		Expect(c.failures).To(BeEmpty())
	})
})
