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
	"sync"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
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
type SuspectHandler = func(ctx context.Context, outage Outage) error

// Outage is the status of a rack that stopped reporting, as a SuspectHandler gets it.
type Outage struct {
	Status
	// seen holds the time on the last status of each task that the monitor saw before
	// the copy started.
	seen map[status.Key]telem.TimeStamp
}

// Reported returns true when the task status with key and time at is not one that the
// monitor saw before the copy started. Such a status is a report from the Driver that
// ended the outage, so the copy must keep it.
func (o Outage) Reported(key status.Key, at telem.TimeStamp) bool {
	seen, ok := o.seen[key]
	return !ok || seen != at
}

type rackState struct {
	lastUpdated telem.TimeStamp
	// statusTime is the time on the last status seen since the Core started, or on the
	// outage being written. A write that keeps it, such as a rename, is not a report
	// from the rack.
	statusTime telem.TimeStamp
	// outage is set from the write of an outage until the Driver reports again, so each
	// outage is written once.
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
		racks map[Key]rackState
		// taskTimes holds the time on the last status seen for each task.
		taskTimes   map[status.Key]telem.TimeStamp
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
	if err := m.svc.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return m.svc.NewRetrieve().
			Where(MatchKeys(stale...)).
			Entries(&racks).
			Exec(ctx, snap)
	}); errors.Skip(err, query.ErrNotFound) != nil {
		return err
	}

	m.mu.Lock()
	// A rack that is gone needs no health check.
	for _, k := range stale {
		if !slices.ContainsFunc(racks, func(r Rack) bool { return r.Key == k }) {
			delete(m.mu.racks, k)
		}
	}
	now = m.svc.Now()
	var (
		statuses    []Status
		lastUpdated = make(map[Key]telem.TimeStamp, len(racks))
		owners      = make([]Key, 0, len(racks))
		statusTimes = make(map[Key]telem.TimeStamp, len(racks))
	)
	for _, r := range racks {
		state := m.mu.racks[r.Key]
		if !m.outageStarted(state, now) {
			continue
		}
		lastUpdated[r.Key] = state.lastUpdated
		owners = append(owners, r.Key)
		statusTimes[r.Key] = state.statusTime
		// A rename can commit after the outage and notify first. Its older status time
		// must already differ, so it counts as a report and the outage is written
		// again.
		state.statusTime = state.lastUpdated
		m.mu.racks[r.Key] = state
		statuses = append(statuses, m.outageStatus(r, state, now))
	}
	m.mu.Unlock()

	if len(statuses) == 0 {
		return nil
	}
	err := m.svc.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return m.svc.Status.NewWriter(tx).SetMany(ctx, &statuses)
	})
	m.mu.Lock()
	defer m.mu.Unlock()
	if err != nil {
		for k, statusTime := range statusTimes {
			if state, ok := m.mu.racks[k]; ok && state.lastUpdated == lastUpdated[k] {
				state.statusTime = statusTime
				m.mu.racks[k] = state
			}
		}
		return err
	}
	for i, k := range owners {
		// A Driver report that landed during the write starts a new period.
		if state, ok := m.mu.racks[k]; ok && state.lastUpdated == lastUpdated[k] {
			state.outage = &statuses[i]
			state.pending = set.New(slices.Collect(maps.Keys(m.mu.handlers))...)
			m.mu.racks[k] = state
		}
	}
	return nil
}

func (m *monitor) outageStatus(r Rack, state rackState, now telem.TimeStamp) Status {
	stat := Status{
		Owner: new(r.ResourceID()),
		Name:  r.Name,
		Time:  state.lastUpdated,
	}
	if m.svc.EmbeddedDriverDisabled && r.Key == m.svc.EmbeddedKey {
		stat.Variant = status.VariantDisabled
		stat.Message = "Driver disabled"
		stat.Description = "The Core runs without its embedded Driver, because it " +
			"started with --no-driver or was built without the Driver."
		return stat
	}
	m.L.Warn(
		"driver not running",
		zap.String("rack", r.Name),
		zap.Stringer("time_since_alive", telem.TimeSpan(now-state.lastUpdated)),
	)
	stat.Variant = status.VariantWarning
	stat.Message = "Driver not running"
	stat.Description = fmt.Sprintf(
		"The Core has not heard from the Driver on %s. Check that the Driver is "+
			"running and can reach the Core.",
		r.Name,
	)
	return stat
}

// copyOutages calls each handler that has not copied the outage of a rack. A handler
// that succeeds is not called again for that outage.
func (m *monitor) copyOutages(ctx context.Context) {
	type copyJob struct {
		rack     Key
		outage   *Status
		handlers map[int]SuspectHandler
	}
	m.mu.Lock()
	// A report can commit before a handler opens its transaction and reach the monitor
	// after it, so each handler compares the statuses it reads with this copy.
	seen := maps.Clone(m.mu.taskTimes)
	var jobs []copyJob
	for k, state := range m.mu.racks {
		if len(state.pending) == 0 {
			continue
		}
		job := copyJob{
			rack:     k,
			outage:   state.outage,
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
			outage := Outage{Status: *job.outage, seen: seen}
			if err := handler(ctx, outage); err != nil {
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
		// A report during the copy ends the outage. A later outage has its own copies.
		if state := m.mu.racks[job.rack]; state.outage == job.outage {
			state.pending.Remove(copied...)
		}
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
		if ch.Kind == gorp.ChangeDelete {
			delete(m.mu.taskTimes, ch.Key)
			continue
		}
		owner := ch.Value.Owner
		if owner == nil {
			continue
		}
		switch owner.Type {
		case resource.TypeRack:
			m.handleRackStatus(*owner, ch.Value)
		case resource.TypeTask:
			m.handleTaskStatus(ch.Key, ch.Value)
		case resource.TypeArc, resource.TypeBuiltin, resource.TypeChannel,
			resource.TypeDevice, resource.TypeFramer, resource.TypeGroup,
			resource.TypeLabel, resource.TypeLineplot, resource.TypeLog,
			resource.TypeNode, resource.TypePanel, resource.TypePolicy,
			resource.TypeProject, resource.TypeRange, resource.TypeRangeAlias,
			resource.TypeRole, resource.TypeSchematic, resource.TypeSchematicSymbol,
			resource.TypeStatus, resource.TypeTable, resource.TypeUser,
			resource.TypeView, resource.TypeEthercatMaster, resource.TypeEthercatSlave,
			resource.TypeHTTPServer, resource.TypeLabjackDevice,
			resource.TypeModbusServer, resource.TypeNiChassis, resource.TypeNiDevice,
			resource.TypeOpcuaServer, resource.TypeArcDirectory, resource.TypeSecret,
			resource.TypeCertificate, resource.TypeColor, resource.TypeICD,
			resource.TypeEnumeration,
			resource.TypeCalculation:
		}
	}
}

func (m *monitor) handleRackStatus(owner resource.ID, stat status.Status[any]) {
	key, err := KeyFromResourceID(owner)
	if err != nil {
		m.L.Error("failed to decode status owner", zap.Error(err))
		return
	}
	alive := storedDetails(stat.Details).Running
	state, known := m.mu.racks[key]
	if !known || (alive && stat.Time != state.statusTime) {
		state = rackState{lastUpdated: m.svc.Now()}
	}
	state.statusTime = stat.Time
	m.mu.racks[key] = state
}

// handleTaskStatus counts a new running task status as a report from the rack that
// runs the task, so it ends an outage of the rack. A Driver that reconnects reports its
// tasks before its first heartbeat. A write that keeps the time of the last status
// seen, such as a rename, is not a report.
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
	if !ok {
		return
	}
	state, known := m.mu.racks[Key(rack)]
	if !known {
		return
	}
	m.mu.racks[Key(rack)] = rackState{
		lastUpdated: m.svc.Now(),
		statusTime:  state.statusTime,
	}
}

// openMonitor watches the health of every stored rack. A rack that sends no status
// within one health check interval is marked as not running, even when its stored
// status is healthy.
func openMonitor(
	ctx context.Context,
	ins alamos.Instrumentation,
	svc *Service,
) (*monitor, error) {
	var (
		racks        []Rack
		taskStatuses []status.Status[any]
	)
	if err := svc.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		if err := svc.NewRetrieve().Entries(&racks).Exec(ctx, snap); err != nil {
			return err
		}
		return svc.Status.NewRetrieve[any]().
			Where(status.Match(func(
				_ gorp.Context,
				_ status.Retrieve[any],
				s *status.Status[any],
			) (bool, error) {
				return s.Owner != nil && s.Owner.Type == resource.TypeTask, nil
			})).
			Entries(&taskStatuses).
			Exec(ctx, snap)
	}); err != nil {
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
	// A task status stored before the Core started is one the monitor has seen, so an
	// outage copies it.
	s.mu.taskTimes = make(map[status.Key]telem.TimeStamp, len(taskStatuses))
	for _, stat := range taskStatuses {
		s.mu.taskTimes[stat.Key] = stat.Time
	}
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
