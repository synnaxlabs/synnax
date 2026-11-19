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
	if cfg.Profile == OneTimeReadFaults || mixed {
		p.image.ReadCorruptionRate = rate(0.05, 0.2, 0.5)
		p.image.MisdirectedReadRate = rate(0.05, 0.2, 0.5)
	}
	p.workload.RunKey, p.image.RunKey = cfg.RunKey, cfg.RunKey
	return p
}

// crash returns the config of the crash image behind the event.
func (p plan) crash(event int) fault.CrashConfig {
	cfg := p.survival
	cfg.RunKey, cfg.Image = p.runKey^uint64(event)<<32, p.image
	return cfg
}

// spoil corrupts the files of image for good, as the profile asks, and returns the
// rules for the engine that recovers from it. The newest block is the last 4 KiB block
// that holds durable bytes of the file that the newest Sync made longer. No later block
// of that file can confirm it. An image that spoil did not change gets the strict
// rules. So does an image that does not hold that file: there the engine cannot see
// what the newest Sync made durable.
func (p plan) spoil(image *fault.FS, c *counter, event int) (rules, error) {
	newest := c.grown[c.last]
	if p.profile != OldCorruption && p.profile != NewCorruption || newest.cut {
		return strictRules, nil
	}
	paths := map[fault.ID]string{}
	if err := files(image, "", paths); err != nil {
		return rules{}, err
	}
	last, ok := paths[c.last]
	if !ok {
		return strictRules, nil
	}
	var (
		r = rules{
			exemptFrom:        math.MaxInt,
			refusalAllowed:    true,
			unreadableAllowed: true,
		}
		changed bool
		err     error
	)
	if p.profile == NewCorruption {
		r.exemptFrom = newest.event
		stale := span{path: last, from: newest.block * blockSize, to: newest.size}
		// A block that the disk never wrote holds zeros.
		changed, err = true, change(image, stale, func(b []byte) { clear(b) })
	} else {
		var spans []span
		for _, file := range slices.Sorted(maps.Keys(paths)) {
			s := span{path: paths[file], to: c.grown[file].size}
			if file == c.last {
				s.to = newest.before
			}
			spans = append(spans, s)
		}
		rng := rand.New(rand.NewPCG(p.runKey, uint64(event)))
		changed, err = flip(image, rng, spans)
	}
	if !changed {
		r = strictRules
	}
	return r, err
}

// files puts the path of each file under dir in image into paths, by the ID of the
// file.
func files(image *fault.FS, dir string, paths map[fault.ID]string) error {
	infos, err := image.List(dir)
	if err != nil {
		return err
	}
	for _, info := range infos {
		p := path.Join(dir, info.Name())
		if info.IsDir() {
			if err = files(image, p, paths); err != nil {
				return err
			}
			continue
		}
		id, err := fileID(info)
		if err != nil {
			return err
		}
		paths[id] = p
	}
	return nil
}

// span is a range of bytes in the file at a path.
type span struct {
	path     string
	from, to int64
}

// change opens the file of s in image and lets edit change the bytes of s.
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
	_, err = f.WriteAt(b, s.from)
	return err
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
