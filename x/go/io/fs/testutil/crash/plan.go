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
	"math/rand/v2"
	"os"
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
	// faulty is true when the workload disk has faults, so a call of the engine can
	// fail.
	faulty bool
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
	if cfg.Profile == OneTimeReadFaults || mixed {
		p.image.ReadCorruptionRate = rate(0.05, 0.2, 0.5)
		p.image.MisdirectedReadRate = rate(0.05, 0.2, 0.5)
	}
	p.workload.RunKey, p.image.RunKey = cfg.RunKey, cfg.RunKey
	p.faulty = p.workload.SyncErrorRate > 0 || p.workload.Capacity > 0
	return p
}

// crash returns the config of the crash image behind the event.
func (p plan) crash(event int) fault.CrashConfig {
	cfg := p.survival
	cfg.RunKey, cfg.Image = p.runKey^uint64(event)<<32, p.image
	return cfg
}

// spoil corrupts the files of image for good, as the profile asks, and returns the
// rules for the engine that recovers from it. The newest block of a file is the last 4
// KiB block that holds durable bytes of the file that the newest Sync made longer. No
// later block of that file can confirm it. An image that spoil did not change gets the
// strict rules.
func (p plan) spoil(image *fault.FS, c *counter, event int) (rules, error) {
	var (
		rng    = rand.New(rand.NewPCG(p.runKey, uint64(event)))
		newest = c.grown[c.last]
		r      = rules{
			exemptFrom: strictRules.exemptFrom,
			refusal:    true,
			unreadable: true,
		}
		changed bool
		err     error
	)
	if p.profile == OldCorruption {
		spans := make([]span, 0, len(c.grown))
		for _, name := range slices.Sorted(maps.Keys(c.grown)) {
			s := span{path: name, to: c.grown[name].size}
			if name == c.last {
				s.to = newest.before
			}
			spans = append(spans, s)
		}
		changed, err = flip(image, rng, spans)
	}
	if p.profile == NewCorruption {
		r.exemptFrom = newest.event
		stale := span{path: c.last, from: newest.block * blockSize, to: newest.size}
		changed, err = zero(image, stale)
	}
	if !changed {
		r = strictRules
	}
	return r, err
}

// span is a range of bytes in the file at a path.
type span struct {
	path     string
	from, to int64
}

// clip cuts each span to the bytes that its file holds in image, and returns the count
// of bytes that are left.
func clip(image *fault.FS, spans []span) (total int64, err error) {
	for i := range spans {
		info, err := image.Stat(spans[i].path)
		if errors.Is(err, os.ErrNotExist) {
			spans[i].to = spans[i].from
			continue
		}
		if err != nil {
			return 0, err
		}
		spans[i].to = max(min(spans[i].to, info.Size()), spans[i].from)
		total += spans[i].to - spans[i].from
	}
	return total, nil
}

// change opens the file of s in image, lets edit change the bytes of s, and makes the
// change durable.
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

// flip flips one bit in a byte of spans, and reports if image holds such a byte.
func flip(image *fault.FS, rng *rand.Rand, spans []span) (bool, error) {
	total, err := clip(image, spans)
	if err != nil || total == 0 {
		return false, err
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

// zero makes the bytes of s zeros, as a block that the disk never wrote is, and reports
// if image holds such a byte.
func zero(image *fault.FS, s span) (bool, error) {
	spans := []span{s}
	total, err := clip(image, spans)
	if err != nil || total == 0 {
		return false, err
	}
	return true, change(image, spans[0], func(b []byte) { clear(b) })
}
