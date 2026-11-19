// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crash_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
)

// altered is a correct reference engine whose Recovered goes through alter.
type altered struct {
	crash.Engine
	alter func([]crash.Record) []crash.Record
}

func (a altered) Recovered(ctx context.Context) ([]crash.Record, error) {
	records, err := a.Engine.Recovered(ctx)
	return a.alter(records), err
}

// alteredConfig returns a config whose engines are correct, but for the Recovered of
// the workload engine (live) or of each engine that recovers from a crash image.
func alteredConfig(
	live bool,
	alter func([]crash.Record) []crash.Record,
) crash.Config {
	var opens atomic.Int64
	return crash.Config{
		Commits: 12,
		Stride:  1,
		Profile: crash.Strict,
		Corrupt: func(err error) bool { return errors.Is(err, errCorrupt) },
		Open: func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
			// The workload engine opens first. Its open changes the disk, so the
			// engines of the first crash images open before it returns.
			first := opens.Add(1) == 1
			engine, err := openRefLog(bugs{})(ctx, fs)
			if err != nil || first != live {
				return engine, err
			}
			return altered{Engine: engine, alter: alter}, nil
		},
	}
}

var _ = Describe("Oracle", func() {
	It(
		"Should catch a live engine that does not show an acked commit",
		func(ctx SpecContext) {
			cfg := alteredConfig(true, func(r []crash.Record) []crash.Record {
				return r[:len(r)-1]
			})
			Expect(broke(crash.Run(ctx, cfg))).To(Equal(crash.R0))
		},
	)

	DescribeTable("Should catch a recovered list that",
		func(ctx SpecContext, alter func([]crash.Record) []crash.Record) {
			Expect(
				broke(crash.Run(ctx, alteredConfig(false, alter))),
			).To(Equal(crash.R2))
		},
		Entry("holds a commit twice", func(r []crash.Record) []crash.Record {
			return append(r, r...)
		}),
		Entry("holds the last commit twice", func(r []crash.Record) []crash.Record {
			return append(r, r[max(len(r)-1, 0):]...)
		}),
		Entry("holds a commit that is not in the workload",
			func(r []crash.Record) []crash.Record {
				return append(r, crash.Record{ID: 1000})
			}),
		Entry("holds a wrong payload", func(r []crash.Record) []crash.Record {
			for i := range r {
				r[i].Payload = append([]byte("x"), r[i].Payload...)
			}
			return r
		}),
		Entry("holds an unreadable commit", func(r []crash.Record) []crash.Record {
			for i := range r {
				r[i].Unreadable = true
			}
			return r
		}),
		Entry("is out of commit order", func(r []crash.Record) []crash.Record {
			if len(r) > 1 {
				r[0], r[1] = r[1], r[0]
			}
			return r
		}),
	)

	It(
		"Should not take an unreadable commit on an image with no corruption",
		func(ctx SpecContext) {
			cfg := alteredConfig(false, func(r []crash.Record) []crash.Record {
				for i := range r {
					r[i].Unreadable, r[i].Payload = true, nil
				}
				return r
			})
			cfg.Profile = crash.OldCorruption
			Expect(broke(crash.Run(ctx, cfg))).To(Equal(crash.R2))
		},
	)

	Describe("An engine that refuses each crash image", func() {
		refusing := func(err error) crash.Config {
			cfg := alteredConfig(false, nil)
			open := openRefLog(bugs{})
			var opens atomic.Int64
			cfg.Open = func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
				if opens.Add(1) == 1 {
					return open(ctx, fs)
				}
				return nil, err
			}
			return cfg
		}

		DescribeTable("Should break R4",
			func(ctx SpecContext, profile crash.Profile, err error) {
				cfg := refusing(err)
				cfg.Profile = profile
				Expect(broke(crash.Run(ctx, cfg))).To(Equal(crash.R4))
			},
			Entry("under Strict, with its corruption error", crash.Strict, errCorrupt),
			Entry("under OneTimeReadFaults", crash.OneTimeReadFaults, errCorrupt),
			Entry("under OldCorruption, with another error",
				crash.OldCorruption, os.ErrPermission),
			Entry("under OldCorruption, on an image with no corruption",
				crash.OldCorruption, errCorrupt),
			Entry("under NewCorruption, on an image with no corruption",
				crash.NewCorruption, errCorrupt),
		)
	})
})

// spy is an engine that appends each payload to a log file and syncs it. It also holds
// a meta file that never grows, and syncs it after each commit. On a crash image it
// finds the bytes that differ from what it wrote. It refuses an image with such a byte,
// or it returns the commits of such bytes as unreadable.
type spy struct {
	file, meta xfs.File
	*spied
}

// spied is the state that the engines of one run share.
type spied struct {
	// refusal is the error of an engine that refuses a changed image. Nil is the
	// corruption error.
	refusal error
	// written holds the bytes of each commit, in order.
	written []byte
	records []crash.Record
	// newest and older count the images with a changed byte in the newest block of the
	// log, and before that block. other counts the images with a changed byte in the
	// meta file.
	newest, older, other int
	// tolerant makes an engine open a changed image and return the commits with changed
	// bytes as unreadable.
	tolerant bool
}

const meta = "metadata"

func (s *spied) open(_ context.Context, fs xfs.FS) (crash.Engine, error) {
	exists, err := fs.Exists("meta")
	if err != nil {
		return nil, err
	}
	fresh := !exists && len(s.written) == 0
	if fresh {
		if err = xfs.WriteFile(fs, "meta", []byte(meta)); err != nil {
			return nil, err
		}
	}
	metaFile, err := fs.Open("meta", os.O_RDWR)
	if err != nil {
		return nil, err
	}
	file, err := fs.Open("log", os.O_CREATE|os.O_RDWR)
	if err != nil {
		return nil, errors.Combine(err, metaFile.Close())
	}
	engine := &spy{file: file, meta: metaFile, spied: s}
	got, err := xfs.ReadFile(fs, "log")
	if err != nil {
		return nil, errors.Combine(err, engine.Close())
	}
	gotMeta, err := xfs.ReadFile(fs, "meta")
	if err != nil {
		return nil, errors.Combine(err, engine.Close())
	}
	block := max(len(got)-1, 0) / 4096 * 4096
	if !bytes.Equal(got[:block], s.written[:block]) {
		s.older++
	}
	if !bytes.Equal(got[block:], s.written[block:len(got)]) {
		s.newest++
	}
	if string(gotMeta) != meta {
		s.other++
	}
	changed := string(gotMeta) != meta || !bytes.Equal(got, s.written[:len(got)])
	if changed && !s.tolerant {
		refusal := s.refusal
		if refusal == nil {
			refusal = errCorrupt
		}
		return nil, errors.Combine(refusal, engine.Close())
	}
	if fresh {
		err = errors.Combine(metaFile.Sync(), fs.SyncDir(""))
	}
	return engine, err
}

func (s *spy) Commit(
	_ context.Context,
	r crash.Record,
	_ crash.Ack,
) (crash.Ack, error) {
	s.written, s.records = append(s.written, r.Payload...), append(s.records, r)
	if _, err := s.file.Write(r.Payload); err != nil {
		return 0, err
	}
	if err := s.file.Sync(); err != nil {
		return 0, err
	}
	return crash.Durable, s.meta.Sync()
}

func (s *spy) Recovered(context.Context) ([]crash.Record, error) {
	info, err := s.file.Stat()
	if err != nil {
		return nil, err
	}
	got := make([]byte, info.Size())
	if _, err = s.file.ReadAt(got, 0); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	var out []crash.Record
	for start, i := 0, 0; i < len(s.records); i++ {
		r := s.records[i]
		end := start + len(r.Payload)
		if end > len(got) {
			break
		}
		if !bytes.Equal(got[start:end], r.Payload) {
			r = crash.Record{ID: r.ID, Unreadable: true}
		}
		out, start = append(out, r), end
	}
	return out, nil
}

func (s *spy) Close() error { return errors.Combine(s.file.Close(), s.meta.Close()) }

var _ = Describe("Lasting corruption", func() {
	run := func(ctx context.Context, s *spied, profile crash.Profile) error {
		return crash.Run(ctx, crash.Config{
			Open:    s.open,
			Corrupt: func(err error) bool { return errors.Is(err, errCorrupt) },
			Commits: 12,
			Stride:  1,
			Profile: profile,
		})
	}

	DescribeTable("Should change only the bytes that the profile names",
		func(
			ctx SpecContext,
			profile crash.Profile,
			newest, older, other types.GomegaMatcher,
		) {
			s := &spied{}
			Expect(run(ctx, s, profile)).To(Succeed())
			Expect(s.newest).To(newest)
			Expect(s.older).To(older)
			Expect(s.other).To(other)
		},
		Entry(
			"the newest block of the file that grew last",
			crash.NewCorruption,
			BeNumerically(">", 0),
			BeZero(),
			BeNumerically("<", 5),
		),
		Entry(
			"a durable byte of any file, but for that newest block",
			crash.OldCorruption,
			BeZero(),
			BeNumerically(">", 0),
			BeNumerically(">", 0),
		),
		Entry("no byte under Strict", crash.Strict, BeZero(), BeZero(), BeZero()),
	)

	It("Should break R4 when an engine refuses a changed image with another error",
		func(ctx SpecContext) {
			s := &spied{refusal: os.ErrPermission}
			Expect(broke(run(ctx, s, crash.NewCorruption))).To(Equal(crash.R4))
			Expect(s.newest + s.other).To(Equal(1))
		},
	)

	DescribeTable("Should take the commits of changed bytes as unreadable",
		func(ctx SpecContext, profile crash.Profile) {
			s := &spied{tolerant: true}
			Expect(run(ctx, s, profile)).To(Succeed())
			Expect(s.newest + s.older).To(BeNumerically(">", 0))
		},
		Entry("under OldCorruption", crash.OldCorruption),
		Entry("under NewCorruption", crash.NewCorruption),
	)
})

// stepper is an engine that makes a known count of disk changes: a directory and a file
// at the open of an empty disk, and a write and a sync for each commit.
type stepper struct {
	file    xfs.File
	records []crash.Record
}

func openStepper(
	opens *atomic.Int64,
) func(context.Context, xfs.FS) (crash.Engine, error) {
	return func(_ context.Context, fs xfs.FS) (crash.Engine, error) {
		opens.Add(1)
		sub, err := fs.Sub("data")
		if err != nil {
			return nil, err
		}
		file, err := sub.Open("log", os.O_CREATE|os.O_RDWR)
		if err != nil {
			return nil, err
		}
		if _, err = sub.Stat("log"); err != nil {
			return nil, err
		}
		return &stepper{file: file}, nil
	}
}

func (s *stepper) Commit(
	_ context.Context,
	r crash.Record,
	_ crash.Ack,
) (crash.Ack, error) {
	if _, err := s.file.Write(r.Payload); err != nil {
		return 0, err
	}
	if _, err := s.file.ReadAt(make([]byte, 1), 0); err != nil {
		return 0, err
	}
	s.records = append(s.records, r)
	return crash.Visible, s.file.Sync()
}

func (s *stepper) Recovered(context.Context) ([]crash.Record, error) {
	return s.records, nil
}

func (s *stepper) Close() error { return s.file.Close() }

var _ = Describe("Events", func() {
	DescribeTable(
		"Should open each crash image two times: the empty disk, each stride, and the end",
		func(ctx SpecContext, stride, images int) {
			var opens atomic.Int64
			Expect(crash.Run(ctx, crash.Config{
				Open:    openStepper(&opens),
				Commits: 5,
				Stride:  stride,
				Profile: crash.Strict,
			})).To(Succeed())
			Expect(opens.Load()).To(BeEquivalentTo(1 + 2*images))
		},
		Entry("each change: a directory, a file, and five writes and syncs", 1, 13),
		Entry("each fifth change", 5, 4),
	)
})
