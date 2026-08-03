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

	"github.com/cockroachdb/cmux"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
)

// SimpleHTTPBranch is a single handler Branch that serves HTTP requests.
type SimpleHTTPBranch struct {
	server  *http.Server
	handler http.Handler
	policy  RoutingPolicy
	ins     alamos.Instrumentation
}

func NewSimpleHTTPBranch(
	handler http.Handler,
	policy RoutingPolicy,
) *SimpleHTTPBranch {
	return &SimpleHTTPBranch{
		policy:  policy,
		handler: handler,
	}
}

// Key implements Branch.
func (*SimpleHTTPBranch) Key() string { return "http_redirect" }

// Routing implements Branch.
func (h *SimpleHTTPBranch) Routing() (i BranchRouting) {
	// Don't serve this branch if we're running in insecure mode.
	return BranchRouting{
		Policy:   h.policy,
		Matchers: []cmux.Matcher{cmux.HTTP1Fast()},
	}
}

// Init implements Branch.
func (h *SimpleHTTPBranch) Init(ctx BranchContext) {
	h.ins = ctx.Instrumentation
	h.server = &http.Server{Handler: h.handler}
}

// Serve implements Branch.
func (h *SimpleHTTPBranch) Serve(ctx BranchContext) error {
	if err := h.server.Serve(ctx.Lis); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Stop implements Branch.
func (h *SimpleHTTPBranch) Stop() {
	if h.server == nil {
		return
	}
	if err := h.server.Shutdown(context.TODO()); err != nil {
		h.ins.L.Error("failed to shut down http redirect server", zap.Error(err))
	}
}

func secureHTTPRedirect(w http.ResponseWriter, r *http.Request) {
	url := "https://" + r.Host + r.URL.String()
	http.Redirect(w, r, url, http.StatusMovedPermanently)
}
