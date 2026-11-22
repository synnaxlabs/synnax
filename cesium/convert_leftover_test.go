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
	"maps"
	"math/rand/v2"
	"os"
	"path"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// landingFS renames N.domain_temp to N.domain, once for each armed name N.domain, when
// an open or an exists check misses N.domain: the rename of the orphan pass that lands
// between the first two tries of a read or an unlink. landed holds the call that missed
// each name.
type landingFS struct {
	xfs.FS
	mu     *sync.Mutex
	armed  set.Set[string]
	landed map[string]string
}

func (f landingFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return landingFS{FS: sub, mu: f.mu, armed: f.armed, landed: f.landed}, nil
}

func (f landingFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if errors.Is(err, os.ErrNotExist) {
		if landErr := f.land(name, "open"); landErr != nil {
			return nil, landErr
		}
	}
	return file, err
}

func (f landingFS) Exists(name string) (bool, error) {
	ok, err := f.FS.Exists(name)
	if err == nil && !ok {
		err = f.land(name, "exists")
	}
	return ok, err
}

// arm makes the next miss of name rename its compaction file to name.
func (f landingFS) arm(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.armed.Add(name)
}

func (f landingFS) land(name, call string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.armed.Contains(name) {
		return nil
	}
	f.armed.Remove(name)
	f.landed[name] = call
	return f.Rename(name+"_temp", name)
}

// call returns the call that missed name.
func (f landingFS) call(name string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.landed[name]
}

const (
	keptCompaction = "kept a file of an old compaction, which can hold the bytes of an " +
		"old file"
	movedTemp = "moved the file that the conversion read for a missing old file into " +
		"place"
)

var _ = Describe("Leftovers of the old layout", func() {
	var (
		keys = []cesium.ChannelKey{oldIdx, oldI32, oldStr}
		disk *fault.FS
		old  oldSamples
	)
	BeforeEach(func(ctx SpecContext) {
		disk = MustSucceed(fault.New(fault.Config{}))
		old = writeOld(ctx, disk)
	})
	exists := func(name string) bool {
		GinkgoHelper()
		return MustSucceed(disk.Exists(name))
	}
	write := func(name string, b []byte) {
		GinkgoHelper()
		sub := MustSucceed(disk.Sub(path.Dir(name)))
		f := MustSucceed(sub.Open(path.Base(name), os.O_CREATE|os.O_WRONLY|os.O_TRUNC))
		MustSucceed(f.WriteAt(b, 0))
		Expect(f.Close()).To(Succeed())
	}
	// session opens fs with identity of the source of openSeeded, reads every key and
	// expects want, and closes. It returns the logs of the session.
	session := func(
		ctx SpecContext,
		fs xfs.FS,
		want map[cesium.ChannelKey][]byte,
	) *observer.ObservedLogs {
		GinkgoHelper()
		ins, logs := ObservedInstrumentation(zapcore.InfoLevel)
		db, closeDB := openConverted(ctx, fs, ins)
		Expect(dataOf(ctx, db, keys)).To(Equal(want))
		Expect(closeDB()).To(Succeed())
		return logs
	}
	// without returns the bytes of each key without the string samples in tr.
	without := func(tr telem.TimeRange) map[cesium.ChannelKey][]byte {
		var strs []string
		for i, s := range old.stamps {
			if !tr.ContainsStamp(s) {
				strs = append(strs, old.strs[i])
			}
		}
		b := old.bytes(nil)
		b[oldStr] = telem.NewSeries(strs).Data
		return b
	}

	It("Should read a missing old file from its compaction file, and move it into "+
		"place on the first open that loads a head", func(ctx SpecContext) {
		name := oldName(oldStr, newestPointer(disk, oldStr))
		b := fileBytes(disk, name)
		Expect(disk.Rename(name, name+"_temp")).To(Succeed())
		logs := session(ctx, disk, old.bytes(nil))
		kept := logs.FilterMessage(keptCompaction).All()
		Expect(kept).To(HaveLen(1))
		Expect(kept[0].ContextMap()).To(Equal(map[string]any{
			"file": name + "_temp",
			"read": true,
		}))
		Expect(exists(name)).To(BeFalse())
		Expect(exists(name + "_temp")).To(BeTrue())
		logs = session(ctx, disk, old.bytes(nil))
		Expect(logs.FilterMessage(movedTemp).Len()).To(Equal(1))
		Expect(fileBytes(disk, name)).To(Equal(b))
		Expect(exists(name + "_temp")).To(BeFalse())
	})

	It("Should keep and log each file of an old compaction when the old file exists",
		func(ctx SpecContext) {
			name := oldName(oldI32, newestPointer(disk, oldI32))
			write(name+"_temp", []byte{1, 2, 3, 4})
			write(name+"_gc", []byte{5, 6, 7, 8})
			logs := session(ctx, disk, old.bytes(nil))
			Expect(logs.FilterMessage(keptCompaction).Len()).To(Equal(2))
			for range 2 {
				logs = session(ctx, disk, old.bytes(nil))
				Expect(logs.FilterMessage(keptCompaction).Len()).To(BeZero())
				Expect(logs.FilterMessage(movedTemp).Len()).To(BeZero())
			}
			Expect(fileBytes(disk, name+"_temp")).To(Equal([]byte{1, 2, 3, 4}))
			Expect(fileBytes(disk, name+"_gc")).To(Equal([]byte{5, 6, 7, 8}))
		})

	It("Should keep the files of an old compaction when a delete buries the segment "+
		"of their old file", func(ctx SpecContext) {
		p := newestPointer(disk, oldStr)
		name := oldName(oldStr, p)
		write(name+"_temp", []byte{1})
		write(name+"_gc", []byte{2})
		session(ctx, disk, old.bytes(nil))
		tr := telem.TimeRange{Start: p.start, End: p.end}
		ins, _ := ObservedInstrumentation(zapcore.InfoLevel)
		db, closeDB := openConverted(ctx, disk, ins)
		Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{oldStr}, tr)).To(Succeed())
		Eventually(func() bool { return exists(name) }).Should(BeFalse())
		Expect(closeDB()).To(Succeed())
		Expect(fileBytes(disk, name+"_temp")).To(Equal([]byte{1}))
		Expect(fileBytes(disk, name+"_gc")).To(Equal([]byte{2}))
	})

	It("Should keep a compaction file whose old file the head does not name",
		func(ctx SpecContext) {
			write("2/99.domain_temp", []byte{1})
			session(ctx, disk, old.bytes(nil))
			session(ctx, disk, old.bytes(nil))
			Expect(exists("2/99.domain_temp")).To(BeTrue())
			Expect(exists("2/99.domain")).To(BeFalse())
		})

	It("Should remove the delete directories and the truncate copies on the first "+
		"open that loads a head, and keep them through the conversion and an undo",
		func(ctx SpecContext) {
			write("5-DELETE-77/meta.json", []byte("{}"))
			write("2/1000-2000.truncated.tmp", []byte{1})
			session(ctx, disk, old.bytes(nil))
			Expect(exists("5-DELETE-77")).To(BeTrue())
			Expect(exists("2/1000-2000.truncated.tmp")).To(BeTrue())
			db := MustSucceed(cesium.Open(
				ctx,
				"",
				cesium.WithFS(disk),
				cesium.WithPointerLog(),
				cesium.WithRandom(rand.NewPCG(1, 2)),
				cesium.WithIdentity(0),
			))
			Expect(db.Close()).To(Succeed())
			Expect(exists("5-DELETE-77")).To(BeTrue())
			Expect(exists("2/1000-2000.truncated.tmp")).To(BeTrue())
			session(ctx, disk, old.bytes(nil))
			Expect(exists("5-DELETE-77")).To(BeFalse())
			Expect(exists("2/1000-2000.truncated.tmp")).To(BeFalse())
		})

	It("Should read and remove every leftover after a crash at each event of the "+
		"first open that loads a head", func(ctx SpecContext) {
		str := oldName(oldStr, newestPointer(disk, oldStr))
		i32 := oldName(oldI32, newestPointer(disk, oldI32))
		strBytes := fileBytes(disk, str)
		Expect(disk.Rename(str, str+"_temp")).To(Succeed())
		write(i32+"_temp", []byte{1})
		write(i32+"_gc", []byte{2})
		write("5-DELETE-77/meta.json", []byte("{}"))
		write("2/1000-2000.truncated.tmp", []byte{3})
		session(ctx, disk, old.bytes(nil))
		disk.Writeback()
		want := old.bytes(nil)
		c := &crashes{disk: disk, taking: true, check: func(image *fault.FS) error {
			db, closeDB, err := openSeeded(ctx, image)
			if err != nil {
				return err
			}
			data, err := dataOf(ctx, db, keys)
			if err = errors.Join(err, closeDB()); err != nil {
				return err
			}
			if !maps.EqualFunc(data, want, bytes.Equal) {
				return errors.New("the read differs from the old layout")
			}
			return nil
		}}
		_, closeDB := MustSucceed2(
			openSeeded(ctx, durableFS{FS: disk, onEvent: c.take}),
		)
		c.mu.Lock()
		c.taking = false
		c.mu.Unlock()
		Expect(closeDB()).To(Succeed())
		Expect(c.images).To(BeNumerically(">", 3))
		Expect(c.failures).To(BeEmpty())
		session(ctx, disk, want)
		Expect(fileBytes(disk, str)).To(Equal(strBytes))
		Expect(exists(str + "_temp")).To(BeFalse())
		Expect(exists(i32 + "_temp")).To(BeTrue())
		Expect(exists(i32 + "_gc")).To(BeTrue())
		Expect(exists("5-DELETE-77")).To(BeFalse())
		Expect(exists("2/1000-2000.truncated.tmp")).To(BeFalse())
	})

	It("Should read and bury a segment of a missing old file in the first session, "+
		"then move the other missing file into place at the restart",
		func(ctx SpecContext) {
			ptrs := oldPointers(disk, oldStr)
			a, b := ptrs[len(ptrs)-2], ptrs[len(ptrs)-1]
			aName, bName := oldName(oldStr, a), oldName(oldStr, b)
			Expect(disk.Rename(aName, aName+"_temp")).To(Succeed())
			Expect(disk.Rename(bName, bName+"_temp")).To(Succeed())
			tr := telem.TimeRange{Start: a.start, End: a.end}
			ins, _ := ObservedInstrumentation(zapcore.InfoLevel)
			db, closeDB := openConverted(ctx, disk, ins)
			Expect(dataOf(ctx, db, keys)).To(Equal(old.bytes(nil)))
			Expect(
				db.DeleteTimeRange(ctx, []cesium.ChannelKey{oldStr}, tr),
			).To(Succeed())
			Eventually(func() bool { return exists(aName + "_temp") }).Should(BeFalse())
			Expect(dataOf(ctx, db, keys)).To(Equal(without(tr)))
			Expect(closeDB()).To(Succeed())
			session(ctx, disk, without(tr))
			Expect(exists(aName)).To(BeFalse())
			Expect(exists(bName)).To(BeTrue())
			Expect(exists(bName + "_temp")).To(BeFalse())
		})

	It("Should read and bury a segment whose missing old file moves into place "+
		"between the first two tries", func(ctx SpecContext) {
		ptrs := oldPointers(disk, oldStr)
		a, b := ptrs[len(ptrs)-2], ptrs[len(ptrs)-1]
		aName, bName := oldName(oldStr, a), oldName(oldStr, b)
		Expect(disk.Rename(aName, aName+"_temp")).To(Succeed())
		Expect(disk.Rename(bName, bName+"_temp")).To(Succeed())
		fs := landingFS{
			FS:     disk,
			mu:     &sync.Mutex{},
			armed:  set.New[string](),
			landed: make(map[string]string),
		}
		ins, _ := ObservedInstrumentation(zapcore.InfoLevel)
		db, closeDB := openConverted(ctx, fs, ins)
		fs.arm(aName)
		Expect(dataOf(ctx, db, keys)).To(Equal(old.bytes(nil)))
		Expect(fs.call(aName)).To(Equal("open"))
		Expect(exists(aName)).To(BeTrue())
		fs.arm(bName)
		tr := telem.TimeRange{Start: b.start, End: b.end}
		Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{oldStr}, tr)).To(Succeed())
		Eventually(func() bool {
			return exists(bName) || exists(bName+"_temp")
		}).Should(BeFalse())
		Expect(fs.call(bName)).To(Equal("exists"))
		Expect(dataOf(ctx, db, keys)).To(Equal(without(tr)))
		Expect(closeDB()).To(Succeed())
	})
})
