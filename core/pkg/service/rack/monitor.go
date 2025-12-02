// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

type rackState struct {
	lastUpdated    telem.TimeStamp
	deadCheckCount int
}

type monitor struct {
	observe.Observer[Status]
	alamos.Instrumentation
	svc *Service
	mu  struct {
		sync.Mutex
		racks map[Key]rackState
	}
	disconnectStatusObserver observe.Disconnect
	shutdownRoutines         io.Closer
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
	now := telem.Now()
	var toAlert []Key
	for k, state := range m.mu.racks {
		if telem.TimeSpan(now-state.lastUpdated) < m.svc.HealthCheckInterval {
			continue
		}
		state.deadCheckCount++
		m.mu.racks[k] = state
		if state.deadCheckCount == 1 || state.deadCheckCount%m.svc.AlertEveryNChecks == 0 {
			toAlert = append(toAlert, k)
		}
	}
	m.mu.Unlock()

	if len(toAlert) == 0 {
		return nil
	}
	racks := make([]Rack, 0, len(toAlert))
	if err := m.svc.NewRetrieve().
		WhereKeys(toAlert...).
		Entries(&racks).
		Exec(ctx, nil); errors.Skip(err, query.NotFound) != nil {
		return err
	}

	m.mu.Lock()
	statuses := make([]Status, len(racks))
	for i, r := range racks {
		state := m.mu.racks[r.Key]
		timeSinceAlive := telem.TimeSpan(now - state.lastUpdated)
		stat := Status{
			Key:         OntologyID(r.Key).String(),
			Name:        r.Name,
			Variant:     xstatus.WarningVariant,
			Time:        state.lastUpdated,
			Message:     fmt.Sprintf("Synnax Driver on %s not running", r.Name),
			Description: fmt.Sprintf("Driver was last alive %s seconds ago", timeSinceAlive),
			Details:     StatusDetails{Rack: r.Key},
		}
		m.L.Warn(strings.ToLower(stat.Message), zap.Stringer("time_since_alive", timeSinceAlive))
		statuses[i] = stat
	}
	m.mu.Unlock()

	if err := status.NewWriter[StatusDetails](m.svc.Status, nil).
		SetMany(ctx, &statuses); err != nil {
		return err
	}
	for _, stat := range statuses {
		m.Notify(ctx, stat)
	}
	return nil
}

func (m *monitor) handleChange(ctx context.Context, t gorp.TxReader[string, status.Status[any]]) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for {
		change, ok := t.Next(ctx)
		if !ok {
			break
		}
		if !strings.HasPrefix(change.Key, string(OntologyType)) {
			continue
		}
		key, err := decodeKeyFromOntologyIDString(change.Key)
		if err != nil {
			m.L.Error("failed to decode status key", zap.Error(err))
			continue
		}
		if change.Variant == xchange.Delete {
			delete(m.mu.racks, key)
			continue
		}
		isHealthy := change.Value.Variant == xstatus.SuccessVariant ||
			change.Value.Variant == xstatus.InfoVariant
		if isHealthy {
			m.mu.racks[key] = rackState{lastUpdated: telem.Now(), deadCheckCount: 0}
		} else if _, exists := m.mu.racks[key]; !exists {
			m.mu.racks[key] = rackState{lastUpdated: telem.Now(), deadCheckCount: 0}
		}
	}
}

func decodeKeyFromOntologyIDString(s string) (Key, error) {
	id, err := ontology.ParseID(s)
	if err != nil {
		return 0, err
	}
	keys, err := KeysFromOntologyIds([]ontology.ID{id})
	if err != nil || len(keys) == 0 {
		return 0, err
	}
	return keys[0], nil
}

func openMonitor(
	ins alamos.Instrumentation,
	svc *Service,
) (*monitor, error) {
	obs := gorp.Observe[string, status.Status[any]](svc.DB)
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(ins))
	s := &monitor{
		Observer:         observe.New[Status](),
		Instrumentation:  ins,
		svc:              svc,
		shutdownRoutines: signal.NewHardShutdown(sCtx, cancel),
	}
	s.mu.racks = make(map[Key]rackState)
	s.disconnectStatusObserver = obs.OnChange(s.handleChange)
	signal.GoTick(sCtx, svc.HealthCheckInterval.Duration(), func(ctx context.Context, t time.Time) error {
		if err := s.checkAlive(ctx); err != nil {
			s.L.Error("failed to check alive status", zap.Error(err))
		}
		return nil
	})
	return s, nil
}
