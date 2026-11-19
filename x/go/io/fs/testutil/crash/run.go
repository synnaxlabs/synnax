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
	"context"
	"fmt"
	"math/rand/v2"
	"sync"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
)

// run is the state of one Run.
type run struct {
	ctx     context.Context
	disk    *fault.FS
	counter *counter
	// broken is the first rule that the engine broke, at the event brokenAt.
	broken   *broken
	plan     plan
	attempts []attempt
	cfg      Config
	brokenAt int
	// mu guards attempts, broken, and brokenAt.
	mu sync.Mutex
}

// Run runs the workload of cfg on an engine, takes a crash image after each Stride
// calls that change the disk, opens the engine on each image, and applies the oracle.
// It returns a *Violation for the first rule that the engine breaks, an error that
// matches validate.ErrValidation for a bad cfg, and nil when the engine holds.
func Run(ctx context.Context, cfg Config) error {
	if err := cfg.validate(); err != nil {
		return err
	}
	r := &run{ctx: ctx, cfg: cfg, plan: newPlan(cfg)}
	disk, err := fault.New(r.plan.workload)
	if err != nil {
		return err
	}
	r.disk = disk
	r.counter = &counter{grown: map[string]growth{}, after: r.afterEvent}
	engine, err := cfg.Open(ctx, &countedFS{inner: disk, c: r.counter})
	if err != nil && r.plan.faulty {
		return nil
	}
	if err != nil {
		return r.violation(&broken{R4, "open on an empty disk: " + err.Error()}, 0)
	}
	rng := rand.New(rand.NewPCG(cfg.RunKey, 0))
	for id := uint64(1); id <= uint64(cfg.Commits) && r.first() == nil; id++ {
		rec, want := Record{ID: id, Payload: payload(cfg.RunKey, id)}, Durable
		if rng.IntN(5) < 2 {
			want = Visible
		}
		r.mu.Lock()
		r.attempts = append(r.attempts, attempt{record: rec})
		r.mu.Unlock()
		ack, err := engine.Commit(ctx, rec, want)
		if err != nil {
			continue
		}
		r.counter.mu.Lock()
		events := r.counter.events
		r.counter.mu.Unlock()
		r.mu.Lock()
		r.attempts[id-1].ack, r.attempts[id-1].ackedAt = ack, events
		r.mu.Unlock()
	}
	if r.first() == nil {
		records, err := engine.Recovered(ctx)
		if err != nil {
			r.fail(&broken{R0, fmt.Sprintf("read before a crash: %v", err)}, 0)
		} else if b := checkLive(r.attempts, records); b != nil {
			r.fail(b, 0)
		}
	}
	if err := engine.Close(); err != nil && !r.plan.faulty && r.first() == nil {
		return errors.Wrap(err, "crash: close the engine")
	}
	if b := r.first(); b != nil {
		return r.violation(b, r.brokenAt)
	}
	return nil
}

func (r *run) violation(b *broken, event int) *Violation {
	return &Violation{
		Rule:    b.rule,
		Detail:  b.detail,
		RunKey:  r.cfg.RunKey,
		Profile: r.cfg.Profile,
		Event:   event,
	}
}

func (r *run) first() *broken {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.broken
}

func (r *run) fail(b *broken, event int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.broken == nil {
		r.broken, r.brokenAt = b, event
	}
}

// afterEvent takes the crash image behind the event, and checks what the engine
// recovers from it. The counter holds its mutex, so the disk does not change.
func (r *run) afterEvent(event int) {
	if event%r.cfg.Stride != 0 || r.first() != nil {
		return
	}
	r.mu.Lock()
	attempts := append([]attempt(nil), r.attempts...)
	r.mu.Unlock()
	image, err := r.disk.Crash(r.plan.crash(event))
	if err != nil {
		r.fail(&broken{R4, fmt.Sprintf("crash image: %v", err)}, event)
		return
	}
	rules, err := r.plan.spoil(image, r.counter, event)
	if err != nil {
		r.fail(&broken{R4, fmt.Sprintf("corrupt the image: %v", err)}, event)
		return
	}
	records, err := r.recover(image)
	switch {
	case err != nil && rules.refusal && r.cfg.Corrupt != nil && r.cfg.Corrupt(err):
	case err != nil:
		r.fail(&broken{R4, fmt.Sprintf("the engine did not recover: %v", err)}, event)
	default:
		if b := checkRecovered(attempts, records, rules); b != nil {
			r.fail(b, event)
		}
	}
}

// recover opens the engine on image and returns the commits that it holds.
func (r *run) recover(image *fault.FS) (records []Record, err error) {
	engine, err := r.cfg.Open(r.ctx, image)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = errors.Combine(err, engine.Close()); err != nil {
			records = nil
		}
	}()
	return engine.Recovered(r.ctx)
}

// payload returns the bytes of the commit id in the run of runKey.
func payload(runKey, id uint64) []byte {
	rng := rand.New(rand.NewPCG(runKey, id))
	out := make([]byte, 1+rng.IntN(1400))
	for i := range out {
		out[i] = byte(rng.Uint32())
	}
	return out
}
