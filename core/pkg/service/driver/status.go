// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver

import (
	"context"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// StatusHandler is the authority on a live task instance's status. It retains the
// last status the instance sent so a command that needs no work can be answered by
// re-sending it, without reading the server. Safe for concurrent use.
type StatusHandler struct {
	db  *gorp.DB
	svc *status.Service
	mu  sync.Mutex
	// stat is the last status sent, seeded at construction before any send.
	stat task.Status
	// prior is the variant and message that the active warning replaced.
	prior struct {
		variant status.Variant
		message string
	}
}

// NewStatusHandler seeds a handler for a fresh instance of t: success variant, not
// running, "Task configured" message.
func NewStatusHandler(db *gorp.DB, svc *status.Service, t task.Task) *StatusHandler {
	return &StatusHandler{db: db, svc: svc, stat: task.Status{
		Key:     t.OntologyID().String(),
		Name:    t.Name,
		Variant: status.VariantSuccess,
		Message: "Task configured",
		Details: task.NewStatusDetails(t, false),
	}}
}

// Send records a status transition and writes it, answering cmdKey (NoCommand for
// unsolicited updates). It clears any description a prior transition set.
func (h *StatusHandler) Send(
	ctx context.Context,
	cmdKey string,
	variant status.Variant,
	running bool,
	message string,
) error {
	h.mu.Lock()
	h.stat.Variant = variant
	h.stat.Message = message
	h.stat.Description = ""
	h.stat.Details.Running = running
	h.stat.Details.Cmd = cmdKey
	stat := h.stamp()
	h.mu.Unlock()
	return h.write(ctx, stat)
}

// Warn records an unsolicited warning with a supporting description, leaving the
// running state as it was. A warning equal to the active one writes nothing, so a
// retry loop can call Warn on every attempt.
func (h *StatusHandler) Warn(ctx context.Context, message, description string) error {
	h.mu.Lock()
	active := h.stat.Variant == status.VariantWarning
	if active && h.stat.Message == message && h.stat.Description == description {
		h.mu.Unlock()
		return nil
	}
	if !active {
		h.prior.variant = h.stat.Variant
		h.prior.message = h.stat.Message
	}
	h.stat.Variant = status.VariantWarning
	h.stat.Message = message
	h.stat.Description = description
	h.stat.Details.Cmd = NoCommand
	stat := h.stamp()
	h.mu.Unlock()
	return h.write(ctx, stat)
}

// ClearWarning restores the status that the active warning replaced. It writes
// nothing when no warning is active.
func (h *StatusHandler) ClearWarning(ctx context.Context) error {
	h.mu.Lock()
	if h.stat.Variant != status.VariantWarning {
		h.mu.Unlock()
		return nil
	}
	h.stat.Variant = h.prior.variant
	h.stat.Message = h.prior.message
	h.stat.Description = ""
	h.stat.Details.Cmd = NoCommand
	stat := h.stamp()
	h.mu.Unlock()
	return h.write(ctx, stat)
}

// Ack answers cmdKey by re-sending the current status, re-asserting the running
// state, for a command that needs no work.
func (h *StatusHandler) Ack(ctx context.Context, cmdKey string, running bool) error {
	h.mu.Lock()
	h.stat.Details.Running = running
	h.stat.Details.Cmd = cmdKey
	stat := h.stamp()
	h.mu.Unlock()
	return h.write(ctx, stat)
}

// Reply answers cmdKey with the result of a command that leaves the task state as it
// was, such as a connection test. The handler does not retain the reply, so later
// statuses carry neither its message nor its data.
func (h *StatusHandler) Reply(
	ctx context.Context,
	cmdKey string,
	variant status.Variant,
	message string,
	data msgpack.EncodedJSON,
) error {
	h.mu.Lock()
	stat := h.stamp()
	h.mu.Unlock()
	stat.Variant = variant
	stat.Message = message
	stat.Description = ""
	stat.Details.Cmd = cmdKey
	stat.Details.Data = data
	return h.write(ctx, stat)
}

// stamp refreshes the status time and returns a copy to write outside the lock.
func (h *StatusHandler) stamp() task.Status {
	h.stat.Time = telem.Now()
	return h.stat
}

// write keeps the stored status name, since a rename updates it while the instance
// runs.
func (h *StatusHandler) write(ctx context.Context, stat task.Status) error {
	return h.db.WithTx(ctx, func(tx gorp.Tx) error {
		var prev task.Status
		if err := h.svc.NewRetrieve[task.StatusDetails]().
			Where(status.MatchKeys[task.StatusDetails](stat.Key)).
			Entry(&prev).
			Exec(ctx, tx); err == nil {
			stat.Name = prev.Name
		} else if !errors.Is(err, query.ErrNotFound) {
			return err
		}
		return h.svc.NewWriter(tx).Set(ctx, &stat)
	})
}

// ConfigReporter reports the errors of tasks that fail to configure. A factory builds
// one and uses it for every task it configures.
type ConfigReporter struct {
	// DB opens the transactions that status writes run in.
	DB *gorp.DB
	// Status writes the error statuses.
	Status *status.Service
	alamos.Instrumentation
}

// Report reports err for t, which failed to configure. It writes an error status that
// answers cmdKey when a caller waits on the command or the task starts automatically,
// and logs a warning otherwise.
func (r ConfigReporter) Report(
	ctx context.Context,
	t task.Task,
	cmdKey string,
	autoStart bool,
	err error,
) {
	if cmdKey == NoCommand && !autoStart {
		r.L.Warn("failed to configure task", zap.Stringer("task", t), zap.Error(err))
		return
	}
	details := task.NewStatusDetails(t, false)
	details.Cmd = cmdKey
	stat := task.Status{
		Key:     t.OntologyID().String(),
		Name:    t.Name,
		Variant: status.VariantError,
		Message: err.Error(),
		Time:    telem.Now(),
		Details: details,
	}
	if sErr := r.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return r.Status.NewWriter(tx).Set(ctx, &stat)
	}); sErr != nil {
		r.L.Error(
			"failed to set configuration status",
			zap.Stringer("task", t),
			zap.Error(sErr),
		)
	}
}
