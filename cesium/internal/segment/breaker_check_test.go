// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package segment_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/segment"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

// crashBytes returns the bytes of 3.seg that a power loss on disk keeps.
func crashBytes(disk *fault.FS) []byte {
	GinkgoHelper()
	img := MustSucceed(disk.Crash(fault.CrashConfig{}))
	f := MustSucceed(img.Open("3.seg", os.O_RDONLY))
	defer func() { Expect(f.Close()).To(Succeed()) }()
	return MustSucceed(io.ReadAll(f))
}

// liveBytes returns the bytes of 3.seg that a read on disk returns now.
func liveBytes(disk *fault.FS) []byte {
	GinkgoHelper()
	f := MustSucceed(disk.Open("3.seg", os.O_RDONLY))
	defer func() { Expect(f.Close()).To(Succeed()) }()
	return MustSucceed(io.ReadAll(f))
}

// discardFS opens files whose writes are dropped, so a spec can write a long stream.
type discardFS struct{ xfs.FS }

type discardFile struct{ xfs.File }

func (f discardFile) WriteAt(p []byte, _ int64) (int, error) { return len(p), nil }

func (f discardFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	return discardFile{File: file}, nil
}

var _ = Describe("Breaker check", func() {
	It("Should not return a committed length that wraps past 2^32 stream bytes",
		func() {
			h := MustSucceed(
				MustSucceed(
					segment.NewBudget(8),
				).Create(discardFS{xfs.NewMem()}, "3.seg"),
			)
			w := segment.NewWriter(h, cfg)
			chunk := make([]byte, 16<<20)
			for range 256 {
				if w.Write(chunk) != nil {
					break
				}
			}
			if err := w.Write(make([]byte, 8)); err == nil {
				length, _ := w.Commit()
				Expect(int64(length)).To(Equal(w.Len()))
			}
			Expect(h.Close()).To(Succeed())
		})

	It("Should leave the written bytes durable when Sync returns nil after a "+
		"failed sync", func() {
		disk := MustSucceed(fault.New(fault.Config{SyncErrorRate: 1}))
		h := MustSucceed(MustSucceed(segment.NewBudget(8)).Create(disk, "3.seg"))
		w := segment.NewWriter(h, cfg)
		disk.Writeback()
		data := pattern(100)
		Expect(w.Write(data)).To(Succeed())
		w.Commit()
		Expect(h.Sync()).To(MatchError(ContainSubstring("input/output error")))
		if err := h.Sync(); err == nil {
			Expect(crashBytes(disk)).To(Equal(data))
		}
		Expect(h.Close()).To(Succeed())
	})

	It("Should keep every byte that a nil Sync covered, over 64 run keys", func() {
		var broken []string
		for key := range uint64(64) {
			disk := MustSucceed(fault.New(fault.Config{
				RunKey:        key,
				SyncErrorRate: 0.5,
			}))
			h := MustSucceed(MustSucceed(segment.NewBudget(8)).Create(disk, "3.seg"))
			w := segment.NewWriter(h, cfg)
			disk.Writeback()
			data := pattern(3*segment.BlockSize + 77)
			var durable int64
			for at := 0; at < len(data); {
				n := min(len(data)-at, 20_000)
				// A failed sync marks the handle failed, and it refuses every write.
				if w.Write(data[at:at+n]) != nil {
					break
				}
				at += n
				w.Commit()
				if h.Sync() == nil {
					durable = w.Len()
				}
			}
			live := liveBytes(disk)
			got := crashBytes(disk)
			want := cfg.FileOffset(durable)
			if int64(len(got)) < want || !bytes.Equal(got[:want], live[:want]) {
				broken = append(broken, fmt.Sprintf(
					"key %d: a nil Sync covered stream length %d, but a crash "+
						"loses bytes before it",
					key,
					durable,
				))
			}
			Expect(h.Close()).To(Succeed())
		}
		Expect(len(broken)).To(
			BeZero(),
			"%s",
			strings.Join(broken[:min(3, len(broken))], "\n"),
		)
	})

	It("Should keep a new segment file after a Sync returns nil", func() {
		disk := MustSucceed(fault.New(fault.Config{}))
		h := MustSucceed(MustSucceed(segment.NewBudget(8)).Create(disk, "3.seg"))
		w := segment.NewWriter(h, cfg)
		data := pattern(100)
		Expect(w.Write(data)).To(Succeed())
		w.Commit()
		Expect(h.Sync()).To(Succeed())
		img := MustSucceed(disk.Crash(fault.CrashConfig{}))
		Expect(img.Exists("3.seg")).To(BeTrue())
		Expect(h.Close()).To(Succeed())
	})
})
