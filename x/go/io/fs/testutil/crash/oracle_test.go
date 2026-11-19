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
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
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

	DescribeTable(
		"Should catch a recovered list that",
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
		Entry(
			"holds a payload with one flipped bit",
			func(r []crash.Record) []crash.Record {
				for i := range r {
					r[i].Payload = append(
						[]byte{r[i].Payload[0] ^ 1},
						r[i].Payload[1:]...,
					)
				}
				return r
			},
		),
		Entry("holds two payloads in the place of each other",
			func(r []crash.Record) []crash.Record {
				if len(r) > 1 {
					r[0].Payload, r[1].Payload = r[1].Payload, r[0].Payload
				}
				return r
			}),
	)

	It("Should catch a recovered list with a hole", func(ctx SpecContext) {
		cfg := alteredConfig(false, func(r []crash.Record) []crash.Record {
			if len(r) > 1 {
				return r[1:]
			}
			return r
		})
		Expect(broke(crash.Run(ctx, cfg))).To(Equal(crash.R3))
	})

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
	// These count the images with a changed byte: newest in the last commit of the log,
	// shared before that commit in its first block, older before that block, and other
	// in the meta file.
	newest, shared, older, other int
	// tolerant makes an engine open a changed image and return the commits with changed
	// bytes as unreadable.
	tolerant bool
	// lasting holds, for each open of a crash image, if it found a changed byte.
	lasting []bool
}

const meta = "metadata"

func (s *spied) open(_ context.Context, root xfs.FS) (crash.Engine, error) {
	fs, err := root.Sub("data")
	if err != nil {
		return nil, err
	}
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
	changed := s.note(got, string(gotMeta))
	if _, image := root.(*fault.FS); image {
		s.lasting = append(s.lasting, changed)
	}
	if changed && !s.tolerant {
		refusal := s.refusal
		if refusal == nil {
			refusal = errCorrupt
		}
		return nil, errors.Combine(refusal, engine.Close())
	}
	if fresh {
		err = errors.Join(metaFile.Sync(), fs.SyncDir(""), root.SyncDir(""))
	}
	return engine, err
}

// note counts the places where the log and the meta file of a crash image differ from
// what the run wrote, and reports if they differ.
func (s *spied) note(got []byte, gotMeta string) bool {
	// The newest bytes are the last whole commit of the image.
	newest := 0
	for start, i := 0, 0; i < len(s.records); i++ {
		end := start + len(s.records[i].Payload)
		if end > len(got) {
			break
		}
		newest, start = start, end
	}
	block := newest / 4096 * 4096
	if !bytes.Equal(got[:block], s.written[:block]) {
		s.older++
	}
	if !bytes.Equal(got[block:newest], s.written[block:newest]) {
		s.shared++
	}
	if !bytes.Equal(got[newest:], s.written[newest:len(got)]) {
		s.newest++
	}
	if gotMeta != meta {
		s.other++
	}
	return gotMeta != meta || !bytes.Equal(got, s.written[:len(got)])
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
			newest, shared, older, other types.GomegaMatcher,
		) {
			s := &spied{}
			Expect(run(ctx, s, profile)).To(Succeed())
			Expect(s.newest).To(newest)
			Expect(s.shared).To(shared)
			Expect(s.older).To(older)
			Expect(s.other).To(other)
		},
		Entry(
			"one block of the newest commit",
			crash.NewCorruption,
			BeNumerically(">", 0),
			BeNumerically(">", 0),
			BeZero(),
			BeNumerically("<", 5),
		),
		Entry(
			"a durable byte of any file, but for that newest commit",
			crash.OldCorruption,
			BeZero(),
			BeNumerically(">", 0),
			BeNumerically(">", 0),
			BeNumerically(">", 0),
		),
		Entry(
			"no byte under Strict",
			crash.Strict,
			BeZero(),
			BeZero(),
			BeZero(),
			BeZero(),
		),
	)

	DescribeTable("Should keep a change through the crash in the recovery",
		func(ctx SpecContext, profile crash.Profile) {
			s := &spied{tolerant: true}
			Expect(run(ctx, s, profile)).To(Succeed())
			Expect(len(s.lasting) % 2).To(BeZero())
			for i := 0; i < len(s.lasting); i += 2 {
				Expect(s.lasting[i+1]).To(Equal(s.lasting[i]), "image %d", i/2)
			}
			Expect(s.lasting).To(ContainElement(true))
		},
		Entry("under OldCorruption", crash.OldCorruption),
		Entry("under NewCorruption", crash.NewCorruption),
	)

	It("Should break R4 when an engine refuses a changed image with another error",
		func(ctx SpecContext) {
			s := &spied{refusal: os.ErrPermission}
			Expect(broke(run(ctx, s, crash.NewCorruption))).To(Equal(crash.R4))
			Expect(s.newest + s.shared + s.other).To(Equal(1))
		},
	)

	DescribeTable("Should take the commits of changed bytes as unreadable",
		func(ctx SpecContext, profile crash.Profile) {
			s := &spied{tolerant: true}
			Expect(run(ctx, s, profile)).To(Succeed())
			Expect(s.newest + s.shared + s.older).To(BeNumerically(">", 0))
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

	It(
		"Should count each call that changes the disk, and no other call",
		func(ctx SpecContext) {
			var opens atomic.Int64
			open := func(_ context.Context, fs xfs.FS) (crash.Engine, error) {
				opens.Add(1)
				if _, image := fs.(*fault.FS); image {
					return &visible{}, nil
				}
				f, err := fs.Open("a", os.O_CREATE|os.O_RDWR)
				if err != nil {
					return nil, err
				}
				for _, flag := range []int{os.O_RDWR, os.O_RDWR | os.O_TRUNC} {
					again, err := fs.Open("a", flag)
					if err != nil {
						return nil, err
					}
					if err = again.Close(); err != nil {
						return nil, err
					}
				}
				if _, err = f.Write([]byte("tacocat")); err != nil {
					return nil, err
				}
				if _, err = f.ReadAt(make([]byte, 1), 0); err != nil {
					return nil, err
				}
				if _, err = fs.List(""); err != nil {
					return nil, err
				}
				sub, err := fs.Sub("d")
				if err != nil {
					return nil, err
				}
				if _, err = fs.Sub("d"); err != nil {
					return nil, err
				}
				return &visible{}, errors.Join(
					f.Truncate(1),
					f.Sync(),
					sub.SyncDir(""),
					fs.Rename("a", "b"),
					fs.Remove("b"),
					f.Close(),
				)
			}
			Expect(crash.Run(ctx, crash.Config{
				Open:    open,
				Commits: 1,
				Stride:  1,
				Profile: crash.Strict,
			})).To(Succeed())
			// Nine changes: a create, a truncating open, a write, a new directory, a
			// truncate, a sync, a directory sync, a rename, and a remove.
			Expect(opens.Load()).To(BeEquivalentTo(1 + 2*10))
		},
	)
})

// visible holds each commit in memory and acks it as Visible.
type visible struct{ records []crash.Record }

func (v *visible) Commit(
	_ context.Context,
	r crash.Record,
	_ crash.Ack,
) (crash.Ack, error) {
	v.records = append(v.records, r)
	return crash.Visible, nil
}

func (v *visible) Recovered(
	context.Context,
) ([]crash.Record, error) {
	return v.records, nil
}

func (v *visible) Close() error { return nil }
