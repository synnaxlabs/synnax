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
	"github.com/synnaxlabs/x/errors"
)

// SimpleHTTPBranch is a single handler Branch that serves HTTP requests.
type SimpleHTTPBranch struct {
	stopErr chan error
	server  *http.Server
	handler http.Handler
	policy  RoutingPolicy
}

func NewSimpleHTTPBranch(
	handler http.Handler,
	policy RoutingPolicy,
) *SimpleHTTPBranch {
	return &SimpleHTTPBranch{
		policy:  policy,
		stopErr: make(chan error, 1),
		handler: handler,
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

// Serve implements Branch.
func (h *SimpleHTTPBranch) Serve(ctx BranchContext) error {
	h.server = &http.Server{Handler: h.handler}
	err := h.server.Serve(ctx.Lis)
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return <-h.stopErr
}

// Stop implements Branch.
func (h *SimpleHTTPBranch) Stop() {
	// If the serve is nil, it means we never served this branch.
	if h.server == nil {
		return
	}
	h.stopErr <- h.server.Shutdown(context.TODO())
}

func secureHTTPRedirect(w http.ResponseWriter, r *http.Request) {
	url := "https://" + r.Host + r.URL.String()
	http.Redirect(w, r, url, http.StatusMovedPermanently)
}
