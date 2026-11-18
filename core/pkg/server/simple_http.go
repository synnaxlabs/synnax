// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/cockroachdb/cmux"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
)

// SimpleHTTPBranch is a single handler Branch that serves HTTP requests.
type SimpleHTTPBranch struct {
	handler func(BranchContext) (http.Handler, error)
	policy  RoutingPolicy
	ins     alamos.Instrumentation
	mu      struct {
		sync.Mutex
		servers []*http.Server
		stopped bool
	}
}

func NewSimpleHTTPBranch(
	handler http.Handler,
	policy RoutingPolicy,
) *SimpleHTTPBranch {
	return &SimpleHTTPBranch{
		policy:  policy,
		handler: func(BranchContext) (http.Handler, error) { return handler, nil },
	}
}

// Key implements Branch.
func (h *SimpleHTTPBranch) Key() string { return "http_redirect" }

// Routing implements Branch.
func (h *SimpleHTTPBranch) Routing() (i BranchRouting) {
	// Don't serve this branch if we're running in insecure mode.
	return BranchRouting{
		Policy:   h.policy,
		Matchers: []cmux.Matcher{cmux.HTTP1Fast()},
	}
}

// Init implements Branch.
func (h *SimpleHTTPBranch) Init(ctx BranchContext) { h.ins = ctx.Instrumentation }

// Serve implements Branch.
func (h *SimpleHTTPBranch) Serve(ctx BranchContext) error {
	handler, err := h.handler(ctx)
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	h.mu.Lock()
	if h.mu.stopped {
		h.mu.Unlock()
		return nil
	}
	h.mu.servers = append(h.mu.servers, srv)
	h.mu.Unlock()
	if err := srv.Serve(ctx.Lis); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Stop implements Branch.
func (h *SimpleHTTPBranch) Stop() {
	h.mu.Lock()
	h.mu.stopped = true
	servers := h.mu.servers
	h.mu.Unlock()
	for _, srv := range servers {
		if err := srv.Shutdown(context.TODO()); err != nil {
			h.ins.L.Error("failed to shut down http redirect server", zap.Error(err))
		}
	}
}
