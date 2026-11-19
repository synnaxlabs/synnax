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
	"encoding/binary"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
)

// script is the plan of an engine for one commit. The steps run on the disk of the
// workload, in order: begin at the open, commit, and end at the close. An engine on a
// crash image holds the commit when the file log holds one whole frame.
type script struct {
	begin  func(fs xfs.FS) error
	commit func(fs xfs.FS, frame []byte) error
	end    func() error
	// side names a file. An engine on a crash image loses the commit when the first
	// byte of that file is zero.
	side string
	// zeroed and mixed count the crash images whose log held only zeros, and whose log
	// held bytes that the steps never wrote.
	zeroed, mixed int
	began         bool
}

type scripted struct {
	fs xfs.FS
	*script
	record *crash.Record
	image  bool
}

// filler is the byte of each file that a script writes before the commit.
const filler = 'a'

// frame returns the bytes that hold the payload in the log: the length, the payload,
// and one end byte.
func frame(payload []byte) []byte {
	out := binary.LittleEndian.AppendUint32(nil, uint32(len(payload)))
	return append(append(out, payload...), 0xFF)
}

func (s *script) open(_ context.Context, fs xfs.FS) (crash.Engine, error) {
	e := &scripted{fs: fs, script: s, image: s.began}
	if !s.began {
		s.began = true
		return e, s.begin(fs)
	}
	exists, err := fs.Exists("log")
	if err != nil || !exists {
		return e, err
	}
	data, err := xfs.ReadFile(fs, "log")
	if err != nil {
		return nil, err
	}
	size := len(data) - 5
	switch {
	case len(data) == 0 || bytes.Equal(data, bytes.Repeat([]byte{filler}, len(data))):
	case bytes.Equal(data, make([]byte, len(data))):
		s.zeroed++
	case size >= 0 && int(binary.LittleEndian.Uint32(data)) == size:
		e.record = &crash.Record{ID: 1, Payload: data[4 : 4+size]}
	case size < 0 || int(binary.LittleEndian.Uint32(data)) != size+1:
		s.mixed++
	}
	if s.side == "" || e.record == nil {
		return e, nil
	}
	if side, err := xfs.ReadFile(fs, s.side); err != nil {
		return nil, err
	} else if len(side) > 0 && side[0] == 0 {
		e.record = nil
	}
	return e, nil
}

func (e *scripted) Commit(
	_ context.Context,
	r crash.Record,
	_ crash.Ack,
) (crash.Ack, error) {
	e.record = &r
	return crash.Durable, e.commit(e.fs, frame(r.Payload))
}

func (e *scripted) Recovered(context.Context) ([]crash.Record, error) {
	if e.record == nil {
		return nil, nil
	}
	return []crash.Record{*e.record}, nil
}

func (e *scripted) Close() error {
	if e.image || e.end == nil {
		return nil
	}
	return e.end()
}

// put writes data to the end of f and syncs f.
func put(f xfs.File, data []byte) error {
	if _, err := f.Write(data); err != nil {
		return err
	}
	return f.Sync()
}

var _ = Describe("Targets of lasting corruption", func() {
	run := func(ctx context.Context, s *script, profile crash.Profile) error {
		return crash.Run(ctx, crash.Config{
			Open:    s.open,
			Commits: 1,
			Stride:  1,
			Profile: profile,
		})
	}

	It("Should hold an ack from before the first sync into the newest block",
		func(ctx SpecContext) {
			var log, side xfs.File
			s := &script{side: "side"}
			// Three events: two new files and a directory sync.
			s.begin = func(fs xfs.FS) (err error) {
				if log, err = fs.Open("log", os.O_CREATE|os.O_RDWR); err != nil {
					return err
				}
				if side, err = fs.Open("side", os.O_CREATE|os.O_RDWR); err != nil {
					return err
				}
				return fs.SyncDir("")
			}
			// Three events. The ack follows event 6, the sync of the log.
			s.commit = func(_ xfs.FS, frame []byte) error {
				if _, err := side.Write([]byte{1}); err != nil {
					return err
				}
				return put(log, frame)
			}
			// Event 7 is the first sync into the newest block of the side file.
			s.end = func() error {
				return errors.Join(side.Sync(), side.Close(), log.Close())
			}
			err := run(ctx, s, crash.NewCorruption)
			var v *crash.Violation
			Expect(errors.As(err, &v)).To(BeTrue())
			Expect(v.Rule).To(Equal(crash.R1))
			Expect(v.Event).To(Equal(7))
		},
	)

	DescribeTable("Should not change a file that a sync made shorter",
		func(ctx SpecContext, profile crash.Profile) {
			var log xfs.File
			s := &script{}
			s.begin = func(fs xfs.FS) (err error) {
				if log, err = fs.Open("log", os.O_CREATE|os.O_RDWR); err != nil {
					return err
				}
				return errors.Join(
					put(log, bytes.Repeat([]byte{filler}, 8192)),
					fs.SyncDir(""),
					log.Truncate(0),
					log.Sync(),
				)
			}
			// Before the cut, the file is a target. The counts start at the commit.
			s.commit = func(_ xfs.FS, frame []byte) error {
				s.zeroed, s.mixed = 0, 0
				_, err := log.WriteAt(frame, 0)
				return errors.Join(err, log.Sync())
			}
			s.end = func() error { return log.Close() }
			Expect(run(ctx, s, profile)).To(Succeed())
			Expect(s.zeroed + s.mixed).To(BeZero())
		},
		Entry("under NewCorruption", crash.NewCorruption),
		Entry("under OldCorruption", crash.OldCorruption),
	)

	It("Should not change an older file that an image holds at the path of the newest",
		func(ctx SpecContext) {
			var log xfs.File
			s := &script{}
			s.begin = func(fs xfs.FS) error {
				old, err := fs.Open("log", os.O_CREATE|os.O_RDWR)
				if err != nil {
					return err
				}
				err = errors.Join(
					put(old, bytes.Repeat([]byte{filler}, 8192)),
					old.Close(),
					fs.SyncDir(""),
					fs.Remove("log"),
				)
				if err != nil {
					return err
				}
				log, err = fs.Open("log", os.O_CREATE|os.O_RDWR)
				return err
			}
			// Between the two syncs, a crash image holds the old file at the path. The
			// old file is a target until the sync of the new file, so the counts start
			// there.
			s.commit = func(fs xfs.FS, frame []byte) error {
				if err := put(log, frame); err != nil {
					return err
				}
				s.zeroed, s.mixed = 0, 0
				return fs.SyncDir("")
			}
			s.end = func() error { return log.Close() }
			Expect(run(ctx, s, crash.NewCorruption)).To(Succeed())
			Expect(s.mixed).To(BeZero())
			Expect(s.zeroed).To(BeNumerically(">", 0))
		},
	)

	It("Should follow a file through a rename", func(ctx SpecContext) {
		var log xfs.File
		s := &script{}
		s.begin = func(fs xfs.FS) (err error) {
			if log, err = fs.Open("tmp", os.O_CREATE|os.O_RDWR); err != nil {
				return err
			}
			return fs.SyncDir("")
		}
		// The end byte goes to the file after the rename, in the block of the frame.
		s.commit = func(fs xfs.FS, frame []byte) error {
			return errors.Join(
				put(log, frame[:len(frame)-1]),
				fs.Rename("tmp", "log"),
				fs.SyncDir(""),
				put(log, frame[len(frame)-1:]),
			)
		}
		s.end = func() error { return log.Close() }
		Expect(run(ctx, s, crash.NewCorruption)).To(Succeed())
		Expect(s.zeroed).To(BeNumerically(">", 0))
	})
})
