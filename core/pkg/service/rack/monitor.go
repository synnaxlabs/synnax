// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack

import (
	"context"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// SuspectHandler copies the outage of a rack onto what the rack runs. When it returns
// an error, the monitor calls it again with the same outage on the next health check.
type SuspectHandler = func(ctx context.Context, outage Status) error

type rackState struct {
	lastUpdated telem.TimeStamp
	// outage is set from the write of an outage until the rack reports healthy again,
	// so each outage is written once.
	outage *Status
	// pending holds the keys of the handlers that have not copied the outage.
	pending set.Set[int]
}

type monitor struct {
	shutdownRoutines         io.Closer
	svc                      *Service
	disconnectStatusObserver observe.Disconnect
	alamos.Instrumentation
	mu struct {
		racks       map[Key]rackState
		handlers    map[int]SuspectHandler
		nextHandler int
		sync.Mutex
	}
}

// onSuspect calls handler for each outage written after the call, until the returned
// function removes it.
func (m *monitor) onSuspect(handler SuspectHandler) observe.Disconnect {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := m.mu.nextHandler
	m.mu.nextHandler++
	m.mu.handlers[key] = handler
	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		delete(m.mu.handlers, key)
	}
}

func (m *monitor) Close() error {
	// Shut down background routines first so checkAlive stops calling suspect handlers,
	// then disconnect the status observer to avoid a deadlock.
	err := m.shutdownRoutines.Close()
	m.disconnectStatusObserver()
	return err
}

func (m *monitor) checkAlive(ctx context.Context) error {
	err := m.writeOutages(ctx)
	m.copyOutages(ctx)
	return err
}

func (m *monitor) writeOutages(ctx context.Context) error {
	m.L.Debug("checking health of racks")
	m.mu.Lock()
	now := m.svc.Now()
	var stale []Key
	for k, state := range m.mu.racks {
		if m.outageStarted(state, now) {
			stale = append(stale, k)
		}
	}
	m.mu.Unlock()

	if len(stale) == 0 {
		return nil
	}
	racks := make([]Rack, 0, len(stale))
	if err := m.svc.NewRetrieve().
		Where(MatchKeys(stale...)).
		Entries(&racks).
		Exec(ctx, nil); errors.Skip(err, query.ErrNotFound) != nil {
		return err
	}

	m.mu.Lock()
	now = m.svc.Now()
	var (
		statuses    []Status
		lastUpdated = make(map[Key]telem.TimeStamp, len(racks))
	)
	for _, r := range racks {
		state := m.mu.racks[r.Key]
		if !m.outageStarted(state, now) {
			continue
		}
		lastUpdated[r.Key] = state.lastUpdated
		statuses = append(statuses, Status{
			Key:     r.OntologyID().String(),
			Name:    r.Name,
			Variant: status.VariantWarning,
			Time:    state.lastUpdated,
			Message: "Driver not running",
			Description: fmt.Sprintf(
				"The Core has not heard from the Driver on %s. Check that the "+
					"Driver is running and can reach the Core.",
				r.Name,
			),
			Details: StatusDetails{Rack: r.Key},
		})
		m.L.Warn(
			"driver not running",
			zap.String("rack", r.Name),
			zap.Stringer("time_since_alive", telem.TimeSpan(now-state.lastUpdated)),
		)
	}
	m.mu.Unlock()

	if len(statuses) == 0 {
		return nil
	}
	if err := m.svc.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return m.svc.Status.NewWriter(tx).SetMany(ctx, &statuses)
	}); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range statuses {
		k := statuses[i].Details.Rack
		// A healthy report that landed during the write starts a new period.
		if state, ok := m.mu.racks[k]; ok && state.lastUpdated == lastUpdated[k] {
			state.outage = &statuses[i]
			state.pending = set.New(slices.Collect(maps.Keys(m.mu.handlers))...)
			m.mu.racks[k] = state
		}
	}
	return nil
}

// copyOutages calls each handler that has not copied the outage of a rack. A handler
// that succeeds is not called again for that outage.
func (m *monitor) copyOutages(ctx context.Context) {
	type copyJob struct {
		rack     Key
		outage   Status
		handlers map[int]SuspectHandler
	}
	m.mu.Lock()
	var jobs []copyJob
	for k, state := range m.mu.racks {
		if len(state.pending) == 0 {
			continue
		}
		job := copyJob{
			rack:     k,
			outage:   *state.outage,
			handlers: make(map[int]SuspectHandler),
		}
		for h := range state.pending {
			if handler, ok := m.mu.handlers[h]; ok {
				job.handlers[h] = handler
			} else {
				state.pending.Remove(h)
			}
		}
		jobs = append(jobs, job)
	}
	m.mu.Unlock()
	for _, job := range jobs {
		var copied []int
		for h, handler := range job.handlers {
			if err := handler(ctx, job.outage); err != nil {
				m.L.Error(
					"failed to copy rack outage",
					zap.String("rack", job.outage.Name),
					zap.Error(err),
				)
				continue
			}
			copied = append(copied, h)
		}
		m.mu.Lock()
		m.mu.racks[job.rack].pending.Remove(copied...)
		m.mu.Unlock()
	}
}

// outageStarted returns true when a rack has been silent for a health check interval
// and its outage is not yet written.
func (m *monitor) outageStarted(state rackState, now telem.TimeStamp) bool {
	return state.outage == nil &&
		telem.TimeSpan(now-state.lastUpdated) >= m.svc.HealthCheckInterval
}

func (m *monitor) handleChange(
	ctx context.Context,
	t gorp.TxReader[status.Key, status.Status[any]],
) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for ch := range t {
		if !strings.HasPrefix(ch.Key, string(ontology.ResourceTypeRack)) {
			continue
		}
		key, err := parseKeyFromOntologyIDString(ch.Key)
		if err != nil {
			m.L.Error("failed to decode status key", zap.Error(err))
			continue
		}
		if ch.Variant == xchange.VariantDelete {
			delete(m.mu.racks, key)
			continue
		}
		isHealthy := ch.Value.Variant == status.VariantSuccess ||
			ch.Value.Variant == status.VariantInfo
		if isHealthy || !lo.HasKey(m.mu.racks, key) {
			m.mu.racks[key] = rackState{lastUpdated: m.svc.Now()}
		}
	}
}

func parseKeyFromOntologyIDString(s status.Key) (Key, error) {
	id, err := ontology.ParseID(s)
	if err != nil {
		return 0, err
	}
	return KeyFromOntologyID(id)
}

// openMonitor watches the health of every stored rack. A rack that sends no status
// within one health check interval is marked as not running, even when its stored
// status is healthy.
func openMonitor(
	ctx context.Context,
	ins alamos.Instrumentation,
	svc *Service,
) (*monitor, error) {
	var racks []Rack
	if err := svc.NewRetrieve().Entries(&racks).Exec(ctx, nil); err != nil {
		return nil, err
	}
	obs := svc.Status.Observe()
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(ins))
	s := &monitor{
		Instrumentation:  ins,
		svc:              svc,
		shutdownRoutines: signal.NewHardShutdown(sCtx, cancel),
	}
	now := svc.Now()
	s.mu.racks = make(map[Key]rackState, len(racks))
	s.mu.handlers = make(map[int]SuspectHandler)
	for _, r := range racks {
		s.mu.racks[r.Key] = rackState{lastUpdated: now}
	}
	s.disconnectStatusObserver = obs.OnChange(s.handleChange)
	signal.GoTick(
		sCtx,
		svc.HealthCheckInterval.Duration(),
		func(ctx context.Context, t time.Time) error {
			if err := s.checkAlive(ctx); err != nil {
				s.L.Error("failed to check alive status", zap.Error(err))
			}
			return nil
		},
	)
	return s, nil
}
