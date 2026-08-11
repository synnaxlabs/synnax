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

	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/telem"
)

// StatusHandler is the authority on a live task instance's status. It retains the
// last status the instance sent so a command that needs no work can be answered by
// re-sending it, without reading the server. Safe for concurrent use.
type StatusHandler struct {
	svc *status.Service
	mu  sync.Mutex
	// stat is the last status sent, seeded at construction before any send.
	stat task.Status
}

// NewStatusHandler seeds a handler for a fresh instance of t: success variant, not
// running, "Task configured" message.
func NewStatusHandler(svc *status.Service, t task.Task) *StatusHandler {
	return &StatusHandler{svc: svc, stat: task.Status{
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
// running state as it was.
func (h *StatusHandler) Warn(ctx context.Context, message, description string) error {
	h.mu.Lock()
	h.stat.Variant = status.VariantWarning
	h.stat.Message = message
	h.stat.Description = description
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

// stamp refreshes the status time and returns a copy to write outside the lock.
func (h *StatusHandler) stamp() task.Status {
	h.stat.Time = telem.Now()
	return h.stat
}

func (h *StatusHandler) write(ctx context.Context, stat task.Status) error {
	return status.NewWriter[task.StatusDetails](h.svc, nil).Set(ctx, &stat)
}
