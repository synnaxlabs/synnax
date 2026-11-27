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
	"context"
	"encoding/binary"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
)

const (
	// probeBlocks is the size of the log of a prober, in blocks.
	probeBlocks = 32
	// probeHeader is the size of the ID and the length that start each record.
	probeHeader = 12
)

// probed holds what the probers of one run saw.
type probed struct {
	// broke is true once a prober returned a wrong payload.
	broke bool
	// faults are the read faults of the recovery that returned it.
	faults []fault.Fault
}

// prober keeps its commits in a synced log. Each recovery also reads chains of spans
// that overlap and, last, a span apart. The first recovery of a run whose disk gives
// a read fault returns a wrong payload, and keeps the faults in seen.
type prober struct {
	fs      xfs.FS
	f       xfs.File
	off     int64
	records []crash.Record
	seen    *probed
}

func openProber(seen *probed) func(context.Context, xfs.FS) (crash.Engine, error) {
	return func(_ context.Context, fs xfs.FS) (crash.Engine, error) {
		f, err := fs.Open("log", os.O_CREATE|os.O_RDWR)
		if err != nil {
			return nil, err
		}
		p := &prober{fs: fs, f: f, seen: seen}
		info, err := f.Stat()
		if err == nil && info.Size() < probeBlocks*blockSize {
			_, err = f.WriteAt(make([]byte, probeBlocks*blockSize), 0)
			err = errors.Join(err, f.Sync(), fs.SyncDir(""))
		}
		if err == nil {
			err = p.scan()
		}
		if err != nil {
			return nil, errors.Combine(err, f.Close())
		}
		return p, nil
	}
}

// scan reads the records of the log, up to the first ID of zero.
func (p *prober) scan() error {
	header := make([]byte, probeHeader)
	for {
		if _, err := p.f.ReadAt(header, p.off); err != nil {
			return err
		}
		id := binary.LittleEndian.Uint64(header)
		size := int64(binary.LittleEndian.Uint32(header[8:]))
		if id == 0 || p.off+probeHeader+size > 8*blockSize {
			return nil
		}
		payload := make([]byte, size)
		if _, err := p.f.ReadAt(payload, p.off+probeHeader); err != nil {
			return err
		}
		p.records = append(p.records, crash.Record{ID: id, Payload: payload})
		p.off += probeHeader + size
	}
}

func (p *prober) Commit(
	_ context.Context,
	r crash.Record,
	_ crash.Ack,
) (crash.Ack, error) {
	b := make([]byte, probeHeader+len(r.Payload))
	binary.LittleEndian.PutUint64(b, r.ID)
	binary.LittleEndian.PutUint32(b[8:], uint32(len(r.Payload)))
	copy(b[probeHeader:], r.Payload)
	if _, err := p.f.WriteAt(b, p.off); err != nil {
		return 0, err
	}
	if err := p.f.Sync(); err != nil {
		return 0, err
	}
	p.off += int64(len(b))
	p.records = append(p.records, r)
	return crash.Durable, nil
}

func (p *prober) Recovered(context.Context) ([]crash.Record, error) {
	// From block 8 on, past the records, each block holds a chain: the third span
	// overlaps the first two, which do not overlap each other.
	for k := range int64(20) {
		at := (8 + k) * blockSize
		for _, s := range [][2]int64{{0, 100}, {200, 300}, {50, 250}} {
			if _, err := p.f.ReadAt(make([]byte, s[1]-s[0]), at+s[0]); err != nil {
				return nil, err
			}
		}
	}
	if _, err := p.f.ReadAt(make([]byte, 64), 30*blockSize); err != nil {
		return nil, err
	}
	disk, ok := p.fs.(*fault.FS)
	if !ok || p.seen.broke {
		return p.records, nil
	}
	if faults := disk.ReadFaults(); len(faults) > 0 {
		p.seen.broke, p.seen.faults = true, faults
		return []crash.Record{{ID: 1, Payload: []byte("wrong")}}, nil
	}
	return p.records, nil
}

func (p *prober) Close() error { return p.f.Close() }

// largestCluster returns the size of the largest group of faults whose reads overlap
// in one file, directly or through other faults of the group.
func largestCluster(faults []fault.Fault) int {
	overlap := func(a, b fault.Fault) bool {
		return a.File == b.File && a.Read.From < b.Read.To && b.Read.From < a.Read.To
	}
	visited := make([]bool, len(faults))
	most := 0
	for i := range faults {
		if visited[i] {
			continue
		}
		visited[i] = true
		group, size := []int{i}, 0
		for len(group) > 0 {
			at := group[len(group)-1]
			group = group[:len(group)-1]
			size++
			for j := range faults {
				if !visited[j] && overlap(faults[at], faults[j]) {
					visited[j] = true
					group = append(group, j)
				}
			}
		}
		most = max(most, size)
	}
	return most
}

var _ = Describe("Read fault clusters", func() {
	It("Should count the largest cluster of the faults of the recovery that broke a "+
		"rule", func(ctx SpecContext) {
		for runKey := range uint64(32) {
			seen := &probed{}
			var v *crash.Violation
			Expect(errors.As(violation(ctx, crash.Config{
				RunKey:  runKey,
				Open:    openProber(seen),
				Commits: 4,
				Stride:  1,
				Profile: crash.RepeatedReadFaults,
			}), &v)).To(BeTrue(), "run key %d", runKey)
			Expect(v.ReadFaults).To(Equal(len(seen.faults)), "run key %d", runKey)
			Expect(v.Clustered).
				To(Equal(largestCluster(seen.faults)), "run key %d", runKey)
		}
	})
})
