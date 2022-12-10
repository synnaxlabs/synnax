package server

import (
	"context"
	"github.com/cockroachdb/cmux"
	"github.com/cockroachdb/errors"
	"net/http"
)

// HTTPRedirectBranch is a Branch that redirects HTTP traffic to HTTPS
// under a secure Server.
type HTTPRedirectBranch struct {
	stopErr chan error
	server  *http.Server
}

// NewHTTPRedirectBranch returns a new HTTPRedirectBranch.
func NewHTTPRedirectBranch() *HTTPRedirectBranch {
	return &HTTPRedirectBranch{
		stopErr: make(chan error, 1),
	}
}

// Key implements Branch.
func (h *HTTPRedirectBranch) Key() string { return "http-redirect" }

// Routing implements Branch.
func (h *HTTPRedirectBranch) Routing() (i BranchRouting) {
	// Don't serve this branch if we're running in insecure mode.
	return BranchRouting{
		Policy:   ServeOnInsecureIfSecure,
		Matchers: []cmux.Matcher{cmux.HTTP1Fast()},
	}
}

// Serve implements Branch.
func (h *HTTPRedirectBranch) Serve(ctx BranchContext) error {
	h.server = &http.Server{Handler: http.HandlerFunc(secureHTTPRedirect)}
	err := h.server.Serve(ctx.Lis)
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return <-h.stopErr
}

// Stop implements Branch.
func (h *HTTPRedirectBranch) Stop() {
	// If the serve is nil, it means we never served this branch.
	if h.server == nil {
		return
	}
	h.stopErr <- h.server.Shutdown(context.TODO())
}

func secureHTTPRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
}
