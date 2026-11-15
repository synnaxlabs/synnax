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
	"strings"
	"sync"
	"time"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

type rackState struct {
	lastUpdated    telem.TimeStamp
	deadCheckCount int
}

type monitor struct {
	observe.Observer[Status]
	shutdownRoutines         io.Closer
	svc                      *Service
	disconnectStatusObserver observe.Disconnect
	alamos.Instrumentation
	mu struct {
		racks map[Key]rackState
		// taskTimes holds the time on the last status seen for each task.
		taskTimes map[status.Key]telem.TimeStamp
		sync.Mutex
	}
}

func (m *monitor) Close() error {
	// Shutdown background routines first to stop checkAlive from triggering
	// new observer notifications, then disconnect the observer to avoid deadlock
	err := m.shutdownRoutines.Close()
	m.disconnectStatusObserver()
	return err
}

func (m *monitor) checkAlive(ctx context.Context) error {
	m.L.Debug("checking health of racks")
	m.mu.Lock()
	now := m.svc.Now()
	var toAlert []Key
	for k, state := range m.mu.racks {
		if telem.TimeSpan(now-state.lastUpdated) < m.svc.HealthCheckInterval {
			continue
		}
		state.deadCheckCount++
		m.mu.racks[k] = state
		if state.deadCheckCount == 1 ||
			state.deadCheckCount%m.svc.AlertEveryNChecks == 0 {
			toAlert = append(toAlert, k)
		}
	}
	m.mu.Unlock()

	if len(toAlert) == 0 {
		return nil
	}
	racks := make([]Rack, 0, len(toAlert))
	if err := m.svc.NewRetrieve().
		Where(MatchKeys(toAlert...)).
		Entries(&racks).
		Exec(ctx, nil); errors.Skip(err, query.ErrNotFound) != nil {
		return err
	}

	m.mu.Lock()
	now = m.svc.Now()
	var statuses []Status
	for _, r := range racks {
		state := m.mu.racks[r.Key]
		if telem.TimeSpan(now-state.lastUpdated) < m.svc.HealthCheckInterval {
			continue
		}
		timeSinceAlive := telem.TimeSpan(now - state.lastUpdated)
		stat := Status{
			Key:     r.OntologyID().String(),
			Name:    r.Name,
			Variant: status.VariantWarning,
			Time:    state.lastUpdated,
			Message: fmt.Sprintf("Synnax Driver on %s not running", r.Name),
			Description: fmt.Sprintf(
				"Driver was last alive %s seconds ago",
				timeSinceAlive,
			),
			Details: StatusDetails{Rack: r.Key},
		}
		m.L.Warn(stat.Message, zap.Stringer("time_since_alive", timeSinceAlive))
		statuses = append(statuses, stat)
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
	// A report during the write ends the outage, so its tasks keep their statuses.
	statuses = lo.Filter(statuses, func(stat Status, _ int) bool {
		return m.mu.racks[stat.Details.Rack].lastUpdated == stat.Time
	})
	m.mu.Unlock()
	for _, stat := range statuses {
		m.Notify(ctx, stat)
	}
	return nil
}

func (m *monitor) handleChange(
	ctx context.Context,
	t gorp.TxReader[status.Key, status.Status[any]],
) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for ch := range t {
		if strings.HasPrefix(ch.Key, string(ontology.ResourceTypeTask)) {
			if ch.Variant == xchange.VariantDelete {
				delete(m.mu.taskTimes, ch.Key)
			} else {
				m.handleTaskStatus(ch.Key, ch.Value)
			}
			continue
		}
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
			m.mu.racks[key] = rackState{lastUpdated: m.svc.Now(), deadCheckCount: 0}
		}
	}
}

// handleTaskStatus counts a new running task status as a report from the rack that
// runs the task. A Driver that reconnects reports its tasks before its first heartbeat.
// A write that keeps the time of the last status seen is not a report.
func (m *monitor) handleTaskStatus(key status.Key, stat status.Status[any]) {
	prev, seen := m.mu.taskTimes[key]
	m.mu.taskTimes[key] = stat.Time
	if seen && prev == stat.Time {
		return
	}
	fields, _ := stat.Details.(map[string]any)
	if running, _ := fields["running"].(bool); !running {
		return
	}
	// Details are stored as JSON, so every number decodes as a float64.
	rack, ok := fields["rack"].(float64)
	rackKey := Key(rack)
	if !ok || !lo.HasKey(m.mu.racks, rackKey) {
		return
	}
	m.mu.racks[rackKey] = rackState{lastUpdated: m.svc.Now()}
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
		Observer:         observe.New[Status](),
		Instrumentation:  ins,
		svc:              svc,
		shutdownRoutines: signal.NewHardShutdown(sCtx, cancel),
	}
	now := svc.Now()
	s.mu.racks = make(map[Key]rackState, len(racks))
	s.mu.taskTimes = make(map[status.Key]telem.TimeStamp)
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
