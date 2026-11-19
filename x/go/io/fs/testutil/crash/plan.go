// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crash

import (
	"maps"
	"math"
	"math/rand/v2"
	"os"
	"path"
	"slices"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
)

// plan holds the faults that a profile and a run key give to one run.
type plan struct {
	// workload sets the faults of the disk that the workload runs on.
	workload fault.Config
	// image sets the faults of each crash image.
	image fault.Config
	// survival sets what a crash keeps of the state that is not durable.
	survival fault.CrashConfig
	runKey   uint64
	profile  Profile
}

func newPlan(cfg Config) plan {
	rng := rand.New(rand.NewPCG(cfg.RunKey, uint64(cfg.Profile)))
	rate := func(rates ...float64) float64 { return rates[rng.IntN(len(rates))] }
	p := plan{runKey: cfg.RunKey, profile: cfg.Profile}
	mixed := cfg.Profile == Mixed
	if cfg.Profile == Partial || mixed {
		p.survival = fault.CrashConfig{
			DataSurvivalRate:  rate(0.25, 0.5, 0.75),
			TornRate:          rate(0.25, 0.5, 0.75),
			EntrySurvivalRate: rate(0.25, 0.5, 0.75),
		}
	}
	if cfg.Profile == SyncErrors || mixed {
		p.workload.SyncErrorRate = rate(0.02, 0.1, 0.3)
		p.workload.EvictionRate = rate(0, 0.5, 1)
	}
	if cfg.Profile == FullDisk || mixed {
		p.workload.Capacity = 4096 + rng.Int64N(60<<10)
	}
	if cfg.Profile == OneTimeReadFaults || cfg.Profile == RepeatedReadFaults || mixed {
		p.image.ReadCorruptionRate = rate(0.05, 0.2, 0.5)
		p.image.MisdirectedReadRate = rate(0.05, 0.2, 0.5)
	}
	p.image.ReadFaultsRepeated = cfg.Profile == RepeatedReadFaults
	p.workload.RunKey, p.image.RunKey = cfg.RunKey, cfg.RunKey
	return p
}

// crash returns the config of the crash image behind the event, and of the crash in the
// recovery from that image.
func (p plan) crash(event int) fault.CrashConfig {
	cfg := p.survival
	cfg.RunKey, cfg.Image = p.runKey^uint64(event)<<32, p.image
	return cfg
}

// spoil corrupts the files of image for good, as the profile asks, and returns the
// rules for the engine that recovers from it. The unit is the ranges that the newest
// Sync with a changed block made durable. No later Sync confirms them. An image that
// spoil did not change gets the strict rules. So does an image that does not hold the
// file of the unit: there the engine cannot see what that Sync made durable.
func (p plan) spoil(image *fault.FS, c *counter, event int) (rules, error) {
	if p.profile != OldCorruption && p.profile != NewCorruption {
		return strictRules, nil
	}
	held := map[fault.ID]span{}
	if err := files(image, "", held); err != nil {
		return rules{}, err
	}
	last, ok := held[c.last]
	if !ok {
		return strictRules, nil
	}
	var (
		rng = rand.New(rand.NewPCG(p.runKey, uint64(event)))
		r   = rules{
			exemptFrom:        math.MaxInt,
			refusalAllowed:    true,
			unreadableAllowed: true,
		}
		spans []span
	)
	if p.profile == NewCorruption {
		// One block of the unit is zeros, as a block that the disk never wrote is. The
		// engine can lose each commit from the first Sync that changed the first block
		// of the range that holds it.
		i := rng.IntN(len(c.unit))
		s := c.unit[i]
		block := s.From/blockSize + rng.Int64N((s.To-1)/blockSize-s.From/blockSize+1)
		stale := span{
			path: last.path,
			from: block * blockSize,
			to:   min((block+1)*blockSize, last.to),
		}
		if stale.to <= stale.from {
			return strictRules, nil
		}
		r.exemptFrom = c.lossFrom[i]
		return r, change(image, stale, func(b []byte) { clear(b) })
	}
	for _, file := range slices.Sorted(maps.Keys(held)) {
		s := held[file]
		if file != c.last {
			spans = append(spans, s)
			continue
		}
		for _, u := range c.unit {
			spans = append(
				spans,
				span{path: s.path, from: s.from, to: min(u.From, s.to)},
			)
			s.from = min(u.To, s.to)
		}
		spans = append(spans, s)
	}
	changed, err := flip(image, rng, spans)
	if !changed {
		r = strictRules
	}
	return r, err
}

// files puts each file under dir in image into held, by the ID of the file: its path
// and its bytes.
func files(image *fault.FS, dir string, held map[fault.ID]span) error {
	infos, err := image.List(dir)
	if err != nil {
		return err
	}
	for _, info := range infos {
		p := path.Join(dir, info.Name())
		if info.IsDir() {
			if err = files(image, p, held); err != nil {
				return err
			}
			continue
		}
		stat, err := statOf(info)
		if err != nil {
			return err
		}
		held[stat.ID] = span{path: p, to: info.Size()}
	}
	return nil
}

// span is a range of bytes in the file at a path.
type span struct {
	path     string
	from, to int64
}

// change opens the file of s in image, lets edit change the bytes of s, and makes them
// durable.
func change(image *fault.FS, s span, edit func(b []byte)) (err error) {
	f, err := image.Open(s.path, os.O_RDWR)
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, f.Close()) }()
	b := make([]byte, s.to-s.from)
	if _, err = f.ReadAt(b, s.from); err != nil {
		return err
	}
	edit(b)
	if _, err = f.WriteAt(b, s.from); err != nil {
		return err
	}
	return f.Sync()
}

// flip flips one bit in a byte of spans, and reports if spans hold a byte.
func flip(image *fault.FS, rng *rand.Rand, spans []span) (bool, error) {
	var total int64
	for _, s := range spans {
		total += s.to - s.from
	}
	if total == 0 {
		return false, nil
	}
	at := rng.Int64N(total)
	for _, s := range spans {
		if size := s.to - s.from; at >= size {
			at -= size
			continue
		}
		s.from, s.to = s.from+at, s.from+at+1
		return true, change(image, s, func(b []byte) { b[0] ^= 1 << rng.IntN(8) })
	}
	return false, nil
}
