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
	"math/rand/v2"
	"os"
	"path"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap/zapcore"
)

// oldPointer is a pointer of an old index.domain file.
type oldPointer struct {
	file         uint16
	offset, size int64
}

// newestPointer returns the pointer with the latest start in the index.domain file of
// key.
func newestPointer(fs xfs.FS, key cesium.ChannelKey) oldPointer {
	GinkgoHelper()
	b := fileBytes(fs, fmt.Sprintf("%d/index.domain", key))
	var (
		p     oldPointer
		start uint64
	)
	for i := 0; i+26 <= len(b); i += 26 {
		if s := binary.LittleEndian.Uint64(b[i:]); i == 0 || s > start {
			start = s
			p = oldPointer{
				file:   binary.LittleEndian.Uint16(b[i+16:]),
				offset: int64(binary.LittleEndian.Uint32(b[i+18:])),
				size:   int64(binary.LittleEndian.Uint32(b[i+22:])),
			}
		}
	}
	return p
}

// fileBytes returns the bytes of the file name of fs.
func fileBytes(fs xfs.FS, name string) []byte {
	GinkgoHelper()
	info := MustSucceed(fs.Stat(name))
	f := MustSucceed(fs.Open(name, os.O_RDONLY))
	b := make([]byte, info.Size())
	MustSucceed(f.ReadAt(b, 0))
	Expect(f.Close()).To(Succeed())
	return b
}

// oldFiles returns the bytes of each file in a channel directory of fs.
func oldFiles(fs xfs.FS) (map[string][]byte, error) {
	files := make(map[string][]byte)
	dirs, err := fs.List("")
	if err != nil {
		return nil, err
	}
	for _, dir := range dirs {
		if _, err = strconv.ParseUint(dir.Name(), 10, 32); err != nil || !dir.IsDir() {
			continue
		}
		infos, err := fs.List(dir.Name())
		if err != nil {
			return nil, err
		}
		for _, info := range infos {
			name := path.Join(dir.Name(), info.Name())
			f, err := fs.Open(name, os.O_RDONLY)
			if err != nil {
				return nil, err
			}
			b := make([]byte, info.Size())
			// A whole read can end at the end of the file with io.EOF.
			if n, err := f.ReadAt(b, 0); n < len(b) {
				return nil, errors.Join(err, f.Close())
			}
			if err = f.Close(); err != nil {
				return nil, err
			}
			files[name] = b
		}
	}
	return files, nil
}

// sameOldFiles returns an error when a file of before differs in fs.
func sameOldFiles(fs xfs.FS, before map[string][]byte) error {
	after, err := oldFiles(fs)
	if err != nil {
		return err
	}
	for name, b := range before {
		if !bytes.Equal(after[name], b) {
			return errors.Newf("old file %s changed", name)
		}
	}
	return nil
}

// truncate cuts the file name of fs to size.
func truncate(fs xfs.FS, name string, size int64) {
	GinkgoHelper()
	f := MustSucceed(fs.Open(name, os.O_RDWR))
	Expect(f.Truncate(size)).To(Succeed())
	Expect(f.Close()).To(Succeed())
}

// zero writes n zeros at off of the file name of fs.
func zero(fs xfs.FS, name string, off, n int64) {
	GinkgoHelper()
	f := MustSucceed(fs.Open(name, os.O_RDWR))
	MustSucceed(f.WriteAt(make([]byte, n), off))
	Expect(f.Close()).To(Succeed())
}

const (
	pastFileCut = "cut a pointer that ends past its file to the whole samples inside it"
	noStampsCut = "cut the samples of a data pointer that its index has no stamps for"
	newestCut   = "cut the newest pointer of an index after its last stamp inside its " +
		"time range"
)

var _ = Describe("Conversion of damaged tails", func() {
	const (
		idx cesium.ChannelKey = 1
		i32 cesium.ChannelKey = 2
		str cesium.ChannelKey = 3
		n                     = 40
	)
	var (
		keys   = []cesium.ChannelKey{idx, i32, str}
		disk   *fault.FS
		stamps []telem.TimeStamp
		ints   []int32
		strs   []string
	)
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
		stamps, ints, strs = make([]telem.TimeStamp, n), make([]int32, n), make(
			[]string,
			n,
		)
		for i := range n {
			stamps[i] = telem.TimeStamp(i+1) * telem.SecondTS
			ints[i] = int32(i + 1)
			strs[i] = strings.Repeat("x", i%7+1)
		}
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Start:    stamps[0],
		}))
		for c := 0; c < n; c += 5 {
			MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
				telem.NewSeries(stamps[c : c+5]),
				telem.NewSeries(ints[c : c+5]),
				telem.NewSeries(strs[c : c+5]),
			}), nil))
			MustSucceed2(w.Commit(ctx, nil))
		}
		Expect(w.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
		// The cuts below need more stamps and samples in each newest pointer than they
		// cut.
		Expect(newestPointer(disk, idx).size / 8).To(BeNumerically(">", 2))
		Expect(newestPointer(disk, i32).size / 4).To(BeNumerically(">", 3))
	})

	// expected returns the bytes of each key that a read returns when it lost the last
	// lost[key] samples.
	expected := func(lost map[cesium.ChannelKey]int) map[cesium.ChannelKey][]byte {
		return map[cesium.ChannelKey][]byte{
			idx: telem.NewSeries(stamps[:n-lost[idx]]).Data,
			i32: telem.NewSeries(ints[:n-lost[i32]]).Data,
			str: telem.NewSeries(strs[:n-lost[str]]).Data,
		}
	}
	// openConverted opens fs on the pointer log as openSeeded does, with ins.
	openConverted := func(
		ctx context.Context,
		fs xfs.FS,
		ins alamos.Instrumentation,
	) (*cesium.DB, func() error) {
		GinkgoHelper()
		db := MustSucceed(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithRandom(rand.NewPCG(1, 2)),
			cesium.WithIdentity(rand.NewPCG(1, 2).Uint64()),
			cesium.WithInstrumentation(ins),
		))
		stop := GoRun(ctx, db.Run)
		return db, func() error { return errors.Join(stop(), db.Close()) }
	}
	// file returns the name of the old file of the newest pointer of key.
	file := func(key cesium.ChannelKey) (string, oldPointer) {
		p := newestPointer(disk, key)
		return fmt.Sprintf("%d/%d.domain", key, p.file), p
	}

	DescribeTable("Should read the whole samples that a damaged tail keeps, and log "+
		"each cut", func(
		ctx SpecContext,
		damage func(),
		lost map[cesium.ChannelKey]int,
		cuts map[string]int,
	) {
		damage()
		disk.Writeback()
		ins, logs := ObservedInstrumentation(zapcore.WarnLevel)
		db, closeDB := openConverted(ctx, disk, ins)
		Expect(dataOf(ctx, db, keys)).To(Equal(expected(lost)))
		Expect(closeDB()).To(Succeed())
		for msg, count := range cuts {
			Expect(logs.FilterMessage(msg).Len()).To(Equal(count), msg)
		}
	},
		Entry("the end of the newest file of a fixed type", func() {
			name, p := file(i32)
			truncate(disk, name, p.offset+p.size-2*4)
		}, map[cesium.ChannelKey]int{i32: 2}, map[string]int{pastFileCut: 1}),
		Entry("zeros at the end of the newest index pointer", func() {
			name, p := file(idx)
			zero(disk, name, p.offset+p.size-2*8, 2*8)
		},
			map[cesium.ChannelKey]int{idx: 2, i32: 2, str: 2},
			map[string]int{newestCut: 1, noStampsCut: 2},
		),
		Entry("the end of the newest index file", func() {
			name, p := file(idx)
			truncate(disk, name, p.offset+p.size-2*8)
		},
			map[cesium.ChannelKey]int{idx: 2, i32: 2, str: 2},
			map[string]int{pastFileCut: 1, noStampsCut: 2},
		),
		Entry("the end of the newest file of a variable type, inside a sample", func() {
			name, p := file(str)
			truncate(disk, name, p.offset+p.size-1)
		}, map[cesium.ChannelKey]int{str: 1}, map[string]int{pastFileCut: 1}),
	)

	It("Should convert with the same cuts after a crash at each event of the first "+
		"open", func(ctx SpecContext) {
		name, p := file(i32)
		truncate(disk, name, p.offset+p.size-3*4)
		name, p = file(idx)
		zero(disk, name, p.offset+p.size-2*8, 2*8)
		disk.Writeback()
		want := expected(map[cesium.ChannelKey]int{idx: 2, i32: 3, str: 2})
		before := MustSucceed(oldFiles(disk))
		c := &crashes{disk: disk, taking: true, check: func(image *fault.FS) error {
			if err := sameOldFiles(image, before); err != nil {
				return err
			}
			db, closeDB, err := openSeeded(ctx, image)
			if err != nil {
				return err
			}
			data, err := dataOf(ctx, db, keys)
			if err = errors.Join(err, closeDB()); err != nil {
				return err
			}
			if !maps.EqualFunc(data, want, bytes.Equal) {
				return errors.New("the read differs from the cut samples")
			}
			return sameOldFiles(image, before)
		}}
		db, closeDB := MustSucceed2(
			openSeeded(ctx, durableFS{FS: disk, onEvent: c.take}),
		)
		c.mu.Lock()
		c.taking = false
		c.mu.Unlock()
		Expect(dataOf(ctx, db, keys)).To(Equal(want))
		Expect(closeDB()).To(Succeed())
		Expect(c.images).To(BeNumerically(">", 10))
		Expect(c.failures).To(BeEmpty())
		Expect(sameOldFiles(disk, before)).To(Succeed())
	})
})
