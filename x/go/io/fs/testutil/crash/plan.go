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

// spoil flips bits that stay flipped in the files of image, as the profile asks, and
// returns the rules for the engine that recovers from it.
func (p plan) spoil(image *fault.FS, c *counter, event int) (rules, error) {
	rng := rand.New(rand.NewPCG(p.runKey, uint64(event)))
	if p.profile == OldCorruption {
		spans := make([]span, 0, len(c.synced))
		for _, name := range slices.Sorted(maps.Keys(c.synced)) {
			size := c.synced[name]
			if name == c.last.path {
				size = c.last.before
			}
			spans = append(spans, span{path: name, to: size})
		}
		r := rules{exemptFrom: strictRules.exemptFrom, refusal: true, unreadable: true}
		return r, flip(image, rng, spans)
	}
	if p.profile == NewCorruption {
		newest := span{path: c.last.path, from: c.last.before, to: c.last.after}
		r := rules{exemptFrom: c.last.event, refusal: true, unreadable: true}
		return r, flip(image, rng, []span{newest})
	}
	return strictRules, nil
}

// span is a range of bytes in the file at a path.
type span struct {
	path     string
	from, to int64
}

// flip flips one bit in a byte of spans that is in image, and makes it durable.
func flip(image *fault.FS, rng *rand.Rand, spans []span) (err error) {
	total := int64(0)
	for i := range spans {
		info, statErr := image.Stat(spans[i].path)
		if errors.Is(statErr, os.ErrNotExist) {
			spans[i].to = spans[i].from
			continue
		}
		if statErr != nil {
			return statErr
		}
		spans[i].to = min(spans[i].to, info.Size())
		total += max(spans[i].to-spans[i].from, 0)
	}
	if total == 0 {
		return nil
	}
	at := rng.Int64N(total)
	for _, s := range spans {
		if size := max(s.to-s.from, 0); at >= size {
			at -= size
			continue
		}
		f, err := image.Open(s.path, os.O_RDWR)
		if err != nil {
			return err
		}
		defer func() { err = errors.Combine(err, f.Close()) }()
		b := make([]byte, 1)
		if _, err = f.ReadAt(b, s.from+at); err != nil {
			return err
		}
		b[0] ^= 1 << rng.IntN(8)
		if _, err = f.WriteAt(b, s.from+at); err != nil {
			return err
		}
		return f.Sync()
	}
	return nil
}
