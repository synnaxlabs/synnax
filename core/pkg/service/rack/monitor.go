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

type monitor struct {
	alamos.Instrumentation
	interval telem.TimeSpan
	svc      *Service
	mu       struct {
		sync.Mutex
		lastUpdated map[Key]telem.TimeStamp
	}
	disconnectObserver observe.Disconnect
	shutdownRoutines   io.Closer
}

func (m *monitor) Close() error {
	m.disconnectObserver()
	return m.shutdownRoutines.Close()
}

func (m *monitor) checkAlive(ctx context.Context) error {
	m.L.Debug("checking health of racks")
	m.mu.Lock()
	defer m.mu.Unlock()
	now := telem.Now()
	var toCreate []Key
	for k, ts := range m.mu.lastUpdated {
		if telem.TimeSpan(now-ts) < m.interval {
			continue
		}
		toCreate = append(toCreate, k)
	}
	racks := make([]Rack, 0, len(toCreate))
	if err := m.svc.NewRetrieve().WhereKeys(toCreate...).Exec(ctx, nil); err != nil || !errors.Is(err, query.NotFound) {
		return err
	}
	statuses := make([]Status, len(racks))
	for i, r := range racks {
		lastUpdated := m.mu.lastUpdated[r.Key]
		timeSinceAlive := telem.TimeSpan(now - lastUpdated)
		stat := Status{
			Variant:     xstatus.WarningVariant,
			Time:        lastUpdated,
			Message:     fmt.Sprintf("Synnax driver on %s not running", r.Name),
			Description: fmt.Sprintf("Driver was last alive %s seconds ago", timeSinceAlive),
			Details:     StatusDetails{Rack: r.Key},
		}
		m.L.Warn(strings.ToLower(stat.Message), zap.Stringer("time_since_alive", timeSinceAlive))
		statuses[i] = stat
	}
	if err := status.NewWriter[StatusDetails](m.svc.Status, nil).
		SetMany(ctx, &statuses); err != nil {
		return err
	}
	return nil
}

func (m *monitor) handleChange(ctx context.Context, t gorp.TxReader[string, status.Status[any]]) {
	var (
		deletes []string
		updates []string
	)
	for {
		stat, ok := t.Next(ctx)
		if !ok {
			break
		}
		if !strings.HasPrefix(stat.Key, string(OntologyType)) {
			continue
		}
		if stat.Variant == xchange.Delete {
			deletes = append(deletes, stat.Key)
		} else {
			updates = append(updates, stat.Key)
		}
	}

	updateKeys, err := decodeKeysFromOntologyIDStrings(updates)
	if err != nil {
		m.L.Error("failed to decode status update as ontology id", zap.Error(err))
	}
	deleteKeys, err := decodeKeysFromOntologyIDStrings(deletes)
	if err != nil {
		m.L.Error("failed to decode status delete as ontology id", zap.Error(err))
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, key := range deleteKeys {
		delete(m.mu.lastUpdated, key)
	}
	for _, key := range updateKeys {
		m.mu.lastUpdated[key] = telem.Now()
	}
}

func decodeKeysFromOntologyIDStrings(strings []string) ([]Key, error) {
	ids, err := ontology.ParseIDs(strings)
	if err != nil {
		return nil, err
	}
	return KeysFromOntologyIds(ids)
}

func openMonitor(
	ins alamos.Instrumentation,
	svc *Service,
) (*monitor, error) {
	obs := gorp.Observe[string, status.Status[any]](svc.DB)
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(ins))
	s := &monitor{
		Instrumentation:  ins,
		svc:              svc,
		shutdownRoutines: signal.NewHardShutdown(sCtx, cancel),
	}
	s.mu.lastUpdated = make(map[Key]telem.TimeStamp)
	s.disconnectObserver = obs.OnChange(s.handleChange)
	signal.GoTick(sCtx, svc.HealthCheckInterval.Duration(), func(ctx context.Context, t time.Time) error {
		if err := s.checkAlive(ctx); err != nil {
			s.L.Error("failed to check alive status", zap.Error(err))
		}
		return nil
	})
	return s, nil
}
