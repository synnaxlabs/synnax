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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

type rackState struct {
	lastUpdated telem.TimeStamp
	// dead is true from the write of an outage until the rack reports healthy again, so
	// each outage is written once.
	dead bool
}

type monitor struct {
	observe.Observer[Status]
	shutdownRoutines         io.Closer
	svc                      *Service
	disconnectStatusObserver observe.Disconnect
	alamos.Instrumentation
	mu struct {
		racks map[Key]rackState
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
	for k, at := range lastUpdated {
		// A healthy report that landed during the write starts a new period.
		if state, ok := m.mu.racks[k]; ok && state.lastUpdated == at {
			state.dead = true
			m.mu.racks[k] = state
		}
	}
	m.mu.Unlock()
	for _, stat := range statuses {
		m.Notify(ctx, stat)
	}
	return nil
}

// outageStarted returns true when a rack has been silent for a health check interval
// and its outage is not yet written.
func (m *monitor) outageStarted(state rackState, now telem.TimeStamp) bool {
	return !state.dead &&
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
		Observer:         observe.New[Status](),
		Instrumentation:  ins,
		svc:              svc,
		shutdownRoutines: signal.NewHardShutdown(sCtx, cancel),
	}
	now := svc.Now()
	s.mu.racks = make(map[Key]rackState, len(racks))
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
