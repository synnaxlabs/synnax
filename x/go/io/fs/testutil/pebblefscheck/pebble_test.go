// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pebblefscheck_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"os"
	"slices"
	"strings"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/io/fs/testutil/pebblefs"
	"github.com/synnaxlabs/x/kv/pebblekv"
	. "github.com/synnaxlabs/x/testutil"
)

// counts holds the calls that reach an xfs.FS and can change its disk.
type counts struct{ writes, syncs, dirSyncs int }

type countFS struct {
	*fault.FS
	c *counts
}

func (f countFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	return countFile{File: file, c: f.c}, nil
}

func (f countFS) SyncDir(name string) error {
	f.c.dirSyncs++
	return f.FS.SyncDir(name)
}

type countFile struct {
	xfs.File
	c *counts
}

func (f countFile) Write(p []byte) (int, error) {
	f.c.writes++
	return f.File.Write(p)
}

func (f countFile) WriteAt(p []byte, off int64) (int, error) {
	f.c.writes++
	return f.File.WriteAt(p, off)
}

func (f countFile) Sync() error {
	f.c.syncs++
	return f.File.Sync()
}

// createHook calls after behind each Create.
type createHook struct {
	vfs.FS
	after func(name string)
}

func (h createHook) Create(name string, c vfs.DiskWriteCategory) (vfs.File, error) {
	f, err := h.FS.Create(name, c)
	if err == nil {
		h.after(name)
	}
	return f, err
}

type pebbleEngine struct{ db *pebble.DB }

func openPebble(_ context.Context, fs xfs.FS) (crash.Engine, error) {
	db, err := pebble.Open("db", &pebble.Options{
		FS:     pebblefs.New(fs),
		Logger: pebblekv.NewNoopLogger(),
	})
	if err != nil {
		return nil, err
	}
	return &pebbleEngine{db: db}, nil
}

func (e *pebbleEngine) Commit(
	_ context.Context,
	r crash.Record,
	want crash.Ack,
) (crash.Ack, error) {
	opts := pebble.NoSync
	if want == crash.Durable {
		opts = pebble.Sync
	}
	return want, e.db.Set(binary.BigEndian.AppendUint64(nil, r.ID), r.Payload, opts)
}

func (e *pebbleEngine) Recovered(context.Context) ([]crash.Record, error) {
	it, err := e.db.NewIter(nil)
	if err != nil {
		return nil, err
	}
	var records []crash.Record
	for it.First(); it.Valid(); it.Next() {
		records = append(records, crash.Record{
			ID:      binary.BigEndian.Uint64(it.Key()),
			Payload: slices.Clone(it.Value()),
		})
	}
	return records, errors.Combine(it.Error(), it.Close())
}

func (e *pebbleEngine) Close() error { return e.db.Close() }

var _ = Describe("Pebble on pebblefs", func() {
	It(
		"Should send each write and sync to the xfs.FS once, as Pebble's FS does",
		func() {
			c := &counts{}
			fs := pebblefs.New(
				countFS{FS: MustSucceed(fault.New(fault.Config{})), c: c},
			)
			f := MustSucceed(fs.Create("a", vfs.WriteCategoryUnspecified))
			Expect(f.Write([]byte("taco"))).To(Equal(4))
			Expect(f.WriteAt([]byte("T"), 0)).To(Equal(1))
			Expect(f.Sync()).To(Succeed())
			Expect(f.SyncData()).To(Succeed())
			Expect(f.SyncTo(4)).To(BeTrue())
			Expect(f.Close()).To(Succeed())
			d := MustSucceed(fs.OpenDir(""))
			Expect(d.Sync()).To(Succeed())
			Expect(d.SyncData()).To(Succeed())
			MustSucceed(d.SyncTo(0))
			Expect(d.Close()).To(Succeed())
			Expect(*c).To(Equal(counts{writes: 2, syncs: 3, dirSyncs: 2}))
		},
	)

	It("Should leave the same files as Pebble on its memory FS", func() {
		listing := func(fs vfs.FS) string {
			db := MustSucceed(pebble.Open("db", &pebble.Options{
				FS:     fs,
				Logger: pebblekv.NewNoopLogger(),
			}))
			for i := range 200 {
				key := fmt.Appendf(nil, "key-%04d", i)
				Expect(db.Set(key, make([]byte, 512), pebble.Sync)).To(Succeed())
				if i%50 == 49 {
					Expect(db.Flush()).To(Succeed())
				}
			}
			Expect(db.Compact(
				context.Background(),
				[]byte("a"),
				[]byte("z"),
				true,
			)).To(Succeed())
			Expect(db.Close()).To(Succeed())
			db = MustSucceed(pebble.Open("db", &pebble.Options{
				FS:     fs,
				Logger: pebblekv.NewNoopLogger(),
			}))
			Expect(db.Close()).To(Succeed())
			names := MustSucceed(fs.List("db"))
			slices.Sort(names)
			return fmt.Sprint(names)
		}
		adapted := listing(pebblefs.New(MustSucceed(fault.New(fault.Config{}))))
		Expect(adapted).To(Equal(listing(vfs.NewMem())))
	})

	for _, profile := range []crash.Profile{crash.Strict} {
		It(
			fmt.Sprintf(
				"Should hold every rule of profile %d over 32 run keys",
				profile,
			),
			func(ctx SpecContext) {
				for key := range uint64(32) {
					_, err := crash.Run(ctx, crash.Config{
						Open:    openPebble,
						Corrupt: func(err error) bool { return errors.Is(err, pebble.ErrCorruption) },
						RunKey:  key,
						Commits: 6,
						Stride:  1,
						Profile: profile,
					})
					Expect(err).ToNot(HaveOccurred())
				}
			},
			SpecTimeout(10*60e9),
		)
	}

	It("Should fail as Pebble on its memory FS does on a lost manifest", func() {
		mem := vfs.NewCrashableMem()
		var image *vfs.MemFS
		hooked := createHook{FS: mem, after: func(name string) {
			if !strings.Contains(name, "marker.manifest") {
				return
			}
			for key := range uint64(64) {
				clone := mem.CrashClone(vfs.CrashCloneCfg{
					UnsyncedDataPercent: 50,
					RNG:                 rand.New(rand.NewPCG(key, 0)),
				})
				names, _ := clone.List("db")
				if slices.Contains(names, name[len("db/"):]) &&
					!slices.Contains(names, "MANIFEST-000001") {
					image = clone
					return
				}
			}
		}}
		db := MustSucceed(pebble.Open("db", &pebble.Options{
			FS:     hooked,
			Logger: pebblekv.NewNoopLogger(),
		}))
		Expect(db.Close()).To(Succeed())
		Expect(image).ToNot(BeNil())
		Expect(pebble.Open("db", &pebble.Options{
			FS:     image,
			Logger: pebblekv.NewNoopLogger(),
		})).Error().To(MatchError(ContainSubstring(
			`could not open manifest file "MANIFEST-000001"`,
		)))
	})

	It("Should open Pebble in a directory that does not exist", func() {
		disk := MustSucceed(fault.New(fault.Config{}))
		db := MustSucceed(pebble.Open("a/b/db", &pebble.Options{
			FS:     pebblefs.New(disk),
			Logger: pebblekv.NewNoopLogger(),
		}))
		Expect(db.Close()).To(Succeed())
		Expect(MustSucceed(disk.Exists("a/b/db/LOCK"))).To(BeTrue())
		Expect(MustSucceed(disk.Stat("a/b/db")).Mode() & os.ModeDir).ToNot(BeZero())
	})
})
