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
		"Should take unreadable commits under lasting corruption",
		func(ctx SpecContext) {
			cfg := alteredConfig(false, func(r []crash.Record) []crash.Record {
				for i := range r {
					r[i].Unreadable, r[i].Payload = true, nil
				}
				return r
			})
			cfg.Profile = crash.OldCorruption
			Expect(crash.Run(ctx, cfg)).To(Succeed())
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
		)

		DescribeTable("Should hold with its corruption error",
			func(ctx SpecContext, profile crash.Profile) {
				cfg := refusing(errCorrupt)
				cfg.Profile = profile
				Expect(crash.Run(ctx, cfg)).To(Succeed())
			},
			Entry("under OldCorruption", crash.OldCorruption),
			Entry("under NewCorruption", crash.NewCorruption),
		)
	})
})

// spy is an engine that appends each payload to one file and syncs it. On a crash image
// it finds the byte that differs from what it wrote, and notes if the last Sync made
// that byte durable. It refuses an image with such a byte.
type spy struct {
	file xfs.File
	*spied
}

// spied is the state that the engines of one run share.
type spied struct {
	// written holds the bytes of each commit, in order.
	written []crash.Record
	// newest and older count the images with a flipped byte in the bytes of the last
	// synced commit, and in the bytes before them.
	newest, older int
}

func (s *spied) open(_ context.Context, fs xfs.FS) (crash.Engine, error) {
	file, err := fs.Open("log", os.O_CREATE|os.O_RDWR)
	if err != nil {
		return nil, err
	}
	got, err := xfs.ReadFile(fs, "log")
	if err != nil {
		return nil, err
	}
	end := 0
	for _, r := range s.written {
		start := end
		if end += len(r.Payload); end > len(got) {
			break
		}
		if !bytes.Equal(got[start:end], r.Payload) {
			if end == len(got) {
				s.newest++
			} else {
				s.older++
			}
			return nil, errors.Combine(errCorrupt, file.Close())
		}
	}
	return &spy{file: file, spied: s}, fs.SyncDir("")
}

func (s *spy) Commit(
	_ context.Context,
	r crash.Record,
	_ crash.Ack,
) (crash.Ack, error) {
	s.written = append(s.written, r)
	if _, err := s.file.Write(r.Payload); err != nil {
		return 0, err
	}
	return crash.Durable, s.file.Sync()
}

func (s *spy) Recovered(context.Context) ([]crash.Record, error) {
	info, err := s.file.Stat()
	if err != nil {
		return nil, err
	}
	var out []crash.Record
	for end, i := 0, 0; i < len(s.written); i++ {
		if end += len(s.written[i].Payload); int64(end) > info.Size() {
			break
		}
		out = append(out, s.written[i])
	}
	return out, nil
}

func (s *spy) Close() error { return s.file.Close() }

var _ = Describe("Lasting corruption", func() {
	DescribeTable(
		"Should flip a bit only in the bytes that the profile names",
		func(
			ctx SpecContext,
			profile crash.Profile,
			newest, older types.GomegaMatcher,
		) {
			s := &spied{}
			Expect(crash.Run(ctx, crash.Config{
				Open:    s.open,
				Corrupt: func(err error) bool { return errors.Is(err, errCorrupt) },
				Commits: 12,
				Stride:  1,
				Profile: profile,
			})).To(Succeed())
			Expect(s.newest).To(newest)
			Expect(s.older).To(older)
		},
		Entry("the newest synced bytes",
			crash.NewCorruption, BeNumerically(">", 0), BeZero()),
		Entry("the bytes that were durable before",
			crash.OldCorruption, BeZero(), BeNumerically(">", 0)),
		Entry("no byte under Strict", crash.Strict, BeZero(), BeZero()),
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
	DescribeTable("Should take one crash image for each stride of disk changes",
		func(ctx SpecContext, stride, images int) {
			var opens atomic.Int64
			Expect(crash.Run(ctx, crash.Config{
				Open:    openStepper(&opens),
				Commits: 5,
				Stride:  stride,
				Profile: crash.Strict,
			})).To(Succeed())
			Expect(opens.Load()).To(BeEquivalentTo(1 + images))
		},
		Entry("each change: a directory, a file, and five writes and syncs", 1, 12),
		Entry("each fifth change", 5, 2),
	)
})
