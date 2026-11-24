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
	"fmt"
	"maps"
	"math/rand/v2"
	"regexp"
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// layoutName matches the paths of the files of the pointer log layout: the heads, logs,
// and parts of the root, and the segment and channel files of the channel directories.
var layoutName = regexp.MustCompile(
	`^([0-9a-f]{16}\.(head|log|part)|\d+/([0-9a-f]{16}\.(seg|chk)|channel))$`,
)

func isUndone(name string) bool { return strings.HasSuffix(name, ".undone") }

// undoneSuffix is the suffix that an undo gives the files of the database id.
func undoneSuffix(id uint64) string { return fmt.Sprintf(".%016x.undone", id) }

// filesOf returns the paths of the files of the root of fs and of its directories that
// match, sorted.
func filesOf(fs xfs.FS, match func(name string) bool) ([]string, error) {
	infos, err := fs.List("")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, info := range infos {
		if !info.IsDir() {
			if match(info.Name()) {
				names = append(names, info.Name())
			}
			continue
		}
		sub, err := fs.List(info.Name())
		if err != nil {
			return nil, err
		}
		for _, f := range sub {
			if name := info.Name() + "/" + f.Name(); match(name) {
				names = append(names, name)
			}
		}
	}
	slices.Sort(names)
	return names, nil
}

var _ = Describe("Undo of a conversion", func() {
	const (
		idx cesium.ChannelKey = 1
		i32 cesium.ChannelKey = 2
	)
	var (
		keys     = []cesium.ChannelKey{idx, i32}
		channels = []cesium.Channel{
			{Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT},
			{Key: i32, Name: "i32", Index: idx, DataType: telem.Int32T},
		}
		disk *fault.FS
		held map[cesium.ChannelKey][]byte
	)
	// write writes 10 samples of each key from start seconds, in 2 commits.
	write := func(ctx context.Context, db *cesium.DB, start telem.TimeStamp) {
		GinkgoHelper()
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Start:    start * telem.SecondTS,
		}))
		for c := range telem.TimeStamp(2) {
			var (
				stamps = make([]telem.TimeStamp, 5)
				ints   = make([]int32, 5)
			)
			for j := range telem.TimeStamp(5) {
				n := start + c*5 + j
				stamps[j], ints[j] = n*telem.SecondTS, int32(n)
			}
			MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
				telem.NewSeries(stamps),
				telem.NewSeries(ints),
			}), nil))
			MustSucceed2(w.Commit(ctx, nil))
		}
		Expect(w.Close()).To(Succeed())
	}
	// open opens fs on the pointer log with the identity id held and the random source
	// of seed, and runs the DB until the returned close. Each sealed record goes to a
	// part, so an undo moves parts aside too.
	open := func(
		ctx context.Context,
		fs xfs.FS,
		id, seed uint64,
	) (*cesium.DB, func() error, error) {
		db, err := cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithIdentity(id),
			cesium.WithRandom(rand.NewPCG(seed, seed)),
			cesium.WithPartSize(1),
		)
		if err != nil {
			return nil, nil, err
		}
		stop := GoRun(ctx, db.Run)
		return db, func() error { return errors.Join(stop(), db.Close()) }, nil
	}
	// readsHeld returns an error when the read of db differs from the old layout.
	readsHeld := func(ctx context.Context, db *cesium.DB, closeDB func() error) error {
		data, err := dataOf(ctx, db, keys)
		if err = errors.Join(err, closeDB()); err != nil {
			return err
		}
		if !maps.EqualFunc(data, held, bytes.Equal) {
			return errors.New("the read differs from the old layout")
		}
		return nil
	}
	BeforeEach(func(ctx SpecContext) {
		disk = MustSucceed(fault.New(fault.Config{}))
		db := openDBOnFS(ctx, disk, cesium.WithFileSizeCap(48*telem.Byte))
		Expect(db.CreateChannel(ctx, channels...)).To(Succeed())
		write(ctx, db, 1)
		write(ctx, db, 1000)
		held = MustSucceed(dataOf(ctx, db, keys))
		Expect(db.Close()).To(Succeed())
		disk.Writeback()
	})

	It("Should open the state with the identity that it gave", func(ctx SpecContext) {
		db, closeDB := MustSucceed2(open(ctx, disk, 0, 1))
		id := db.Identity()
		Expect(id).ToNot(BeZero())
		write(ctx, db, 5000)
		written := MustSucceed(dataOf(ctx, db, keys))
		Expect(closeDB()).To(Succeed())
		db, closeDB = MustSucceed2(open(ctx, disk, id, 2))
		Expect(db.Identity()).To(Equal(id))
		Expect(dataOf(ctx, db, keys)).To(Equal(written))
		Expect(closeDB()).To(Succeed())
		Expect(filesOf(disk, isUndone)).To(BeEmpty())
	})

	It("Should refuse a root that holds a database when the caller gives no identity",
		func(ctx SpecContext) {
			db, closeDB := MustSucceed2(open(ctx, disk, 0, 1))
			id := db.Identity()
			Expect(closeDB()).To(Succeed())
			layout := MustSucceed(filesOf(disk, layoutName.MatchString))
			Expect(cesium.Open(ctx, "", cesium.WithFS(disk), cesium.WithPointerLog())).
				Error().To(MatchError(ContainSubstring(fmt.Sprintf(
				"the root holds database %016x, and the caller gave no identity",
				id,
			))))
			Expect(filesOf(disk, layoutName.MatchString)).To(Equal(layout))
			Expect(filesOf(disk, isUndone)).To(BeEmpty())
			fresh := MustSucceed(cesium.Open(
				ctx,
				"",
				cesium.WithFS(xfs.NewMem()),
				cesium.WithPointerLog(),
			))
			Expect(GoRun(ctx, fresh.Run)()).To(Succeed())
			Expect(fresh.Close()).To(Succeed())
		})

	It("Should move the new layout aside and convert the files of the old binary "+
		"again after an undo", func(ctx SpecContext) {
		db, closeDB := MustSucceed2(open(ctx, disk, 0, 1))
		id := db.Identity()
		write(ctx, db, 5000)
		Expect(closeDB()).To(Succeed())
		layout := MustSucceed(filesOf(disk, layoutName.MatchString))
		Expect(layout).To(ContainElements(
			MatchRegexp(`^\d+/[0-9a-f]{16}\.seg$`),
			MatchRegexp(`^[0-9a-f]{16}\.part$`),
		))
		// The undo of the old binary puts back each meta.json and changes no other
		// file, so the old binary writes on from the files that the conversion left.
		old := MustSucceed(cesium.Open(ctx, "", cesium.WithFS(disk)))
		write(ctx, old, 9000)
		oldData := MustSucceed(dataOf(ctx, old, keys))
		Expect(old.Close()).To(Succeed())
		disk.Writeback()
		Expect(oldData[i32]).To(HaveLen(len(held[i32]) + 10*4))
		db, closeDB = MustSucceed2(open(ctx, disk, 0, 2))
		again := db.Identity()
		Expect(again).ToNot(Equal(id))
		Expect(dataOf(ctx, db, keys)).To(Equal(oldData))
		Expect(closeDB()).To(Succeed())
		moved := make([]string, len(layout))
		for i, name := range layout {
			moved[i] = name + undoneSuffix(id)
		}
		Expect(filesOf(disk, isUndone)).To(ConsistOf(moved))
		db, closeDB = MustSucceed2(open(ctx, disk, again, 3))
		Expect(db.Identity()).To(Equal(again))
		Expect(closeDB()).To(Succeed())
		Expect(filesOf(disk, isUndone)).To(ConsistOf(moved))
	})

	It("Should take heads of another identity as an undone conversion", func(
		ctx SpecContext,
	) {
		db, closeDB := MustSucceed2(open(ctx, disk, 0, 1))
		id := db.Identity()
		Expect(closeDB()).To(Succeed())
		db, closeDB = MustSucceed2(open(ctx, disk, id+1, 2))
		Expect(db.Identity()).ToNot(Equal(id))
		Expect(readsHeld(ctx, db, closeDB)).To(Succeed())
		Expect(filesOf(disk, isUndone)).
			To(ContainElement(checkpoint.Name(1) + undoneSuffix(id)))
	})

	It("Should keep the files of each undo", func(ctx SpecContext) {
		ids := make([]uint64, 0, 3)
		for seed := range uint64(3) {
			db, closeDB := MustSucceed2(open(ctx, disk, 0, seed+1))
			ids = append(ids, db.Identity())
			write(ctx, db, 5000)
			Expect(closeDB()).To(Succeed())
		}
		undone := MustSucceed(filesOf(disk, isUndone))
		for _, id := range ids[:2] {
			suffix := undoneSuffix(id)
			Expect(undone).To(ContainElements(
				checkpoint.Name(1)+suffix,
				log.Name(1)+suffix,
				MatchRegexp(`^\d+/[0-9a-f]{16}\.seg`+regexp.QuoteMeta(suffix)+`$`),
			))
		}
	})

	It("Should start an empty database after an undo on a root with no old layout",
		func(ctx SpecContext) {
			fs := xfs.NewMem()
			db, closeDB := MustSucceed2(open(ctx, fs, 0, 1))
			id := db.Identity()
			Expect(db.CreateChannel(ctx, channels...)).To(Succeed())
			write(ctx, db, 1)
			Expect(closeDB()).To(Succeed())
			db, closeDB = MustSucceed2(open(ctx, fs, 0, 2))
			Expect(db.Identity()).ToNot(Equal(id))
			Expect(db.Read(ctx, telem.TimeRangeMax, keys...)).Error().
				To(MatchError(cesium.ErrChannelNotFound))
			Expect(closeDB()).To(Succeed())
			suffix := regexp.QuoteMeta(undoneSuffix(id))
			Expect(filesOf(fs, isUndone)).To(ContainElements(
				checkpoint.Name(1)+undoneSuffix(id),
				"1/channel"+undoneSuffix(id),
				MatchRegexp(`^\d+/[0-9a-f]{16}\.seg`+suffix+`$`),
			))
		})

	It("Should keep the undone files of a key when a channel that reuses the key "+
		"drops, through a later open and Sweep", func(ctx SpecContext) {
		fs := xfs.NewMem()
		db, closeDB := MustSucceed2(open(ctx, fs, 0, 1))
		id := db.Identity()
		Expect(db.CreateChannel(ctx, channels...)).To(Succeed())
		write(ctx, db, 1)
		Expect(closeDB()).To(Succeed())
		db, closeDB = MustSucceed2(open(ctx, fs, 0, 2))
		next := db.Identity()
		undone := MustSucceed(filesOf(fs, isUndone))
		Expect(undone).To(ContainElement("1/channel" + undoneSuffix(id)))
		// The restored channel table no longer holds the keys, so new channels take
		// them.
		Expect(db.CreateChannel(ctx, channels...)).To(Succeed())
		write(ctx, db, 100)
		own := func(name string) bool {
			return !isUndone(name) &&
				(strings.HasPrefix(name, "1/") || strings.HasPrefix(name, "2/"))
		}
		Expect(filesOf(fs, own)).
			To(ContainElement(MatchRegexp(`^1/[0-9a-f]{16}\.seg$`)))
		Expect(db.Drop(ctx, keys)).To(Succeed())
		Expect(closeDB()).To(Succeed())
		Expect(filesOf(fs, own)).To(BeEmpty())
		Expect(filesOf(fs, isUndone)).To(Equal(undone))
		// Only a person removes the undone files: no open and no Sweep does.
		db, closeDB = MustSucceed2(open(ctx, fs, next, 3))
		MustSucceed2(db.Sweep(ctx, nil))
		Expect(closeDB()).To(Succeed())
		Expect(filesOf(fs, isUndone)).To(Equal(undone))
		Expect(fs.Exists("1")).To(BeTrue())
		Expect(fs.Exists("2")).To(BeTrue())
	})

	It("Should finish the undo and lose no file after a crash at each of its steps",
		func(ctx SpecContext) {
			db, closeDB := MustSucceed2(open(ctx, disk, 0, 1))
			id := db.Identity()
			write(ctx, db, 5000)
			Expect(closeDB()).To(Succeed())
			layout := MustSucceed(filesOf(disk, layoutName.MatchString))
			c := &crashes{disk: disk, taking: true, check: func(image *fault.FS) error {
				db, closeDB, err := open(ctx, image, 0, 3)
				if err != nil {
					return err
				}
				if err = readsHeld(ctx, db, closeDB); err != nil {
					return err
				}
				undone, err := filesOf(image, isUndone)
				if err != nil {
					return err
				}
				for _, name := range layout {
					if !slices.Contains(undone, name+undoneSuffix(id)) {
						return errors.Newf("%s is not moved aside", name)
					}
				}
				return nil
			}}
			// Each unsynced rename survives a crash at random, so 4 images of each
			// event find a root that keeps some renames and loses the others.
			takeMany := func() {
				for range 4 {
					c.take()
				}
			}
			_, closeDB = MustSucceed2(
				open(ctx, durableFS{FS: disk, onEvent: takeMany}, 0, 2),
			)
			c.mu.Lock()
			c.taking = false
			c.mu.Unlock()
			Expect(closeDB()).To(Succeed())
			Expect(c.images).To(BeNumerically(">", len(layout)))
			Expect(c.failures).To(BeEmpty())
		})

	It("Should convert again with every sample after a crash at each step of the "+
		"conversion, with no identity stored", func(ctx SpecContext) {
		// aside counts the images whose committed head 1 the reopen moved aside: the
		// crashes between the commit point and the write of the identity.
		var aside int
		c := &crashes{disk: disk, taking: true, check: func(image *fault.FS) error {
			db, closeDB, err := open(ctx, image, 0, 3)
			if err != nil {
				return err
			}
			if err = readsHeld(ctx, db, closeDB); err != nil {
				return err
			}
			undone, err := filesOf(image, isUndone)
			if len(undone) > 0 {
				aside++
			}
			return err
		}}
		_, closeDB := MustSucceed2(
			open(ctx, durableFS{FS: disk, onEvent: c.take}, 0, 2),
		)
		c.mu.Lock()
		c.taking = false
		c.mu.Unlock()
		Expect(closeDB()).To(Succeed())
		Expect(c.failures).To(BeEmpty())
		Expect(aside).To(BeNumerically(">", 0))
	})
})
