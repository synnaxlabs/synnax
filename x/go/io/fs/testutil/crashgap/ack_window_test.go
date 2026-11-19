// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crashgap_test

import (
	"context"
	"encoding/binary"
	"hash/crc32"
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
)

// memory holds every commit in memory, never touches the disk, and acks each commit
// at the level asked for.
type memory struct{ records []crash.Record }

func (m *memory) Commit(
	_ context.Context,
	r crash.Record,
	want crash.Ack,
) (crash.Ack, error) {
	m.records = append(m.records, r)
	return want, nil
}

func (m *memory) Recovered(
	context.Context,
) ([]crash.Record, error) {
	return m.records, nil
}

func (m *memory) Close() error { return nil }

// lazy is a log that acks a Durable commit before its sync. It syncs the log at the
// start of the next commit, so the newest Durable ack is never durable.
type lazy struct {
	f       xfs.File
	off     int64
	pending bool
	// eager syncs before the ack, which makes the log correct.
	eager bool
}

func openLazy(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
	return open(ctx, fs, false)
}

func openEager(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
	return open(ctx, fs, true)
}

func open(_ context.Context, fs xfs.FS, eager bool) (crash.Engine, error) {
	f, err := fs.Open("log", os.O_CREATE|os.O_RDWR)
	if err != nil {
		return nil, err
	}
	if err := f.Sync(); err != nil {
		return nil, err
	}
	if err := fs.SyncDir(""); err != nil {
		return nil, err
	}
	l := &lazy{f: f, eager: eager}
	recs, err := l.Recovered(context.Background())
	if err != nil {
		return nil, err
	}
	for _, r := range recs {
		l.off += int64(16 + len(r.Payload))
	}
	return l, nil
}

func (l *lazy) Commit(
	_ context.Context,
	r crash.Record,
	want crash.Ack,
) (crash.Ack, error) {
	if l.pending {
		if err := l.f.Sync(); err != nil {
			return 0, err
		}
	}
	buf := make([]byte, 16+len(r.Payload))
	binary.LittleEndian.PutUint64(buf, r.ID)
	binary.LittleEndian.PutUint32(buf[8:], uint32(len(r.Payload)))
	copy(buf[12:], r.Payload)
	binary.LittleEndian.PutUint32(
		buf[12+len(r.Payload):],
		crc32.ChecksumIEEE(buf[:12+len(r.Payload)]),
	)
	n, err := l.f.WriteAt(buf, l.off)
	l.off += int64(n)
	if err != nil {
		return 0, err
	}
	l.pending = true
	if l.eager {
		if err := l.f.Sync(); err != nil {
			return 0, err
		}
	}
	return want, nil
}

func (l *lazy) Recovered(context.Context) ([]crash.Record, error) {
	var out []crash.Record
	head := make([]byte, 12)
	for off := int64(0); ; {
		if _, err := l.f.ReadAt(head, off); err != nil {
			if err == io.EOF {
				return out, nil
			}
			return nil, err
		}
		size := int(binary.LittleEndian.Uint32(head[8:]))
		body := make([]byte, size+4)
		if _, err := l.f.ReadAt(body, off+12); err != nil {
			return out, nil
		}
		sum := crc32.ChecksumIEEE(append(append([]byte{}, head...), body[:size]...))
		if sum != binary.LittleEndian.Uint32(body[size:]) {
			return out, nil
		}
		out = append(out, crash.Record{
			ID:      binary.LittleEndian.Uint64(head),
			Payload: body[:size],
		})
		off += int64(16 + size)
	}
}

func (l *lazy) Close() error { return l.f.Close() }

var _ = Describe("Durable acks", func() {
	It(
		"Should fail an engine that acks Durable and never writes",
		func(ctx SpecContext) {
			err := crash.Run(ctx, crash.Config{
				RunKey:  1,
				Open:    func(context.Context, xfs.FS) (crash.Engine, error) { return &memory{}, nil },
				Commits: 20,
				Stride:  1,
				Profile: crash.Strict,
			})
			var v *crash.Violation
			Expect(err).To(BeAssignableToTypeOf(v))
			Expect(err.(*crash.Violation).Rule).To(Equal(crash.R1))
		},
	)

	It(
		"Should fail an engine that syncs a Durable commit only after its ack",
		func(ctx SpecContext) {
			err := crash.Run(ctx, crash.Config{
				RunKey:  1,
				Open:    openLazy,
				Commits: 20,
				Stride:  1,
				Profile: crash.Strict,
			})
			var v *crash.Violation
			Expect(err).To(BeAssignableToTypeOf(v))
			Expect(err.(*crash.Violation).Rule).To(Equal(crash.R1))
		},
	)

	It("Should pass the same log when it syncs before the ack", func(ctx SpecContext) {
		Expect(crash.Run(ctx, crash.Config{
			RunKey:  1,
			Open:    openEager,
			Commits: 20,
			Stride:  1,
			Profile: crash.Strict,
		})).To(Succeed())
	})
})
