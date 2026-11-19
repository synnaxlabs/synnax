// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pebblekv_test

import (
	"context"
	"encoding/binary"
	"os"
	"strings"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

// walImage commits n durable records of size bytes on a fault disk, and returns a
// crash image that keeps only durable state, and the name of its WAL.
func walImage(ctx context.Context, n, size int) (*fault.FS, string) {
	GinkgoHelper()
	disk := MustSucceed(fault.New(fault.Config{}))
	e := MustSucceed(openCrashEngine(ctx, disk))
	for id := 1; id <= n; id++ {
		r := crash.Record{ID: uint64(id), Payload: make([]byte, size)}
		Expect(e.Commit(ctx, r, crash.Durable)).To(Equal(crash.Durable))
	}
	image := MustSucceed(disk.Crash(fault.CrashConfig{}))
	Expect(e.Close()).To(Succeed())
	for _, info := range MustSucceed(image.List("kv")) {
		if strings.HasSuffix(info.Name(), ".log") {
			return image, "kv/" + info.Name()
		}
	}
	Fail("no WAL in the image")
	return nil, ""
}

func flip(image *fault.FS, name string, at int64) {
	GinkgoHelper()
	f := MustSucceed(image.Open(name, os.O_RDWR))
	b := make([]byte, 1)
	Expect(f.ReadAt(b, at)).To(Equal(1))
	b[0] ^= 1
	Expect(f.WriteAt(b, at)).To(Equal(1))
	Expect(errors.Join(f.Sync(), f.Close())).To(Succeed())
}

var _ = Describe(
	"Pebble WAL replay after one flipped bit in the first WAL of a new store",
	func() {
		It(
			"Should drop the record and all later ones in the last 32 KiB block, with no error",
			func(ctx SpecContext) {
				image, wal := walImage(ctx, 10, 500)
				flip(image, wal, 1100)
				e := MustSucceed(openCrashEngine(ctx, image))
				got := MustSucceed(e.Recovered(ctx))
				Expect(e.Close()).To(Succeed())
				Expect(len(got)).To(BeNumerically("<", 10))
				Expect(len(got)).To(BeNumerically(">", 0))
			},
		)

		It(
			"Should drop later records in later blocks too, as on an OS disk",
			func(ctx SpecContext) {
				image, wal := walImage(ctx, 100, 1000)
				flip(image, wal, 40000)
				e := MustSucceed(openCrashEngine(ctx, image))
				onFault := len(MustSucceed(e.Recovered(ctx)))
				Expect(e.Close()).To(Succeed())
				Expect(onFault).To(BeNumerically("<", 50))
				Expect(osRecovered(40000)).To(Equal(onFault))
			},
		)
	},
)

// osRecovered commits 100 durable records of 1000 bytes to a new store with Pebble on
// the OS file system, flips one bit of its WAL at offset at, and returns how many
// records the next open holds. It fails when the open returns an error.
func osRecovered(at int64) int {
	GinkgoHelper()
	dir := GinkgoT().TempDir()
	opts := func() *pebble.Options {
		return &pebble.Options{
			FS:                 vfs.Default,
			FormatMajorVersion: pebble.FormatNewest,
			Logger:             crashLogger{},
		}
	}
	db := MustSucceed(pebble.Open(dir, opts()))
	for id := uint64(1); id <= 100; id++ {
		key := binary.BigEndian.AppendUint64(nil, id)
		Expect(db.Set(key, make([]byte, 1000), pebble.Sync)).To(Succeed())
	}
	Expect(db.Close()).To(Succeed())
	var wal string
	for _, n := range MustSucceed(vfs.Default.List(dir)) {
		if strings.HasSuffix(n, ".log") {
			wal = vfs.Default.PathJoin(dir, n)
		}
	}
	f := MustSucceed(os.OpenFile(wal, os.O_RDWR, 0))
	b := make([]byte, 1)
	MustSucceed(f.ReadAt(b, at))
	b[0] ^= 1
	MustSucceed(f.WriteAt(b, at))
	Expect(errors.Join(f.Sync(), f.Close())).To(Succeed())
	db = MustSucceed(pebble.Open(dir, opts()))
	iter := MustSucceed(db.NewIter(nil))
	n := 0
	for ok := iter.First(); ok; ok = iter.Next() {
		n++
	}
	Expect(errors.Join(iter.Close(), db.Close())).To(Succeed())
	return n
}

var _ = Describe("Crash adapter reads", func() {
	It("Should return each payload whole when values come from sstables",
		func(ctx SpecContext) {
			disk := MustSucceed(fault.New(fault.Config{}))
			e := MustSucceed(openCrashEngine(ctx, disk))
			want := map[uint64][]byte{}
			for id := uint64(1); id <= 200; id++ {
				p := make([]byte, 300)
				for i := range p {
					p[i] = byte(id + uint64(i))
				}
				want[id] = p
				r := crash.Record{ID: id, Payload: p}
				Expect(e.Commit(ctx, r, crash.Durable)).To(Equal(crash.Durable))
			}
			Expect(e.(*crashEngine).pebble.Flush()).To(Succeed())
			got := MustSucceed(e.Recovered(ctx))
			Expect(e.Close()).To(Succeed())
			Expect(got).To(HaveLen(200))
			for _, r := range got {
				Expect(r.Payload).To(Equal(want[r.ID]), "record %d", r.ID)
			}
		})
})
