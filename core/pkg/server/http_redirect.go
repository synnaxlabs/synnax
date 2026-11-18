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
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
)

// NewHTTPRedirectBranch returns a Branch that redirects HTTP requests to HTTPS, keeping
// the path and query.
func NewHTTPRedirectBranch() *SimpleHTTPBranch {
	return &SimpleHTTPBranch{
		handler: redirectToListener,
		policy:  RoutingPolicyServeOnInsecureIfSecure,
	}
}

// redirectToListener redirects to the host and port of the request when the Listener's
// certificate covers the host, or when the Listener has no host that remote clients can
// reach. Otherwise it redirects to the configured host and port of the Listener, so a
// forged Host header cannot send a client to a host the Core does not serve.
func redirectToListener(ctx BranchContext) (http.Handler, error) {
	_, listenerPort, err := net.SplitHostPort(ctx.Lis.Addr().String())
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, port := ctx.ServerName, listenerPort
		reqHost, reqPort := splitRequestHost(r)
		if !reachable(ctx.ServerName) || certCovers(ctx.TLS, reqHost) {
			host = reqHost
			if reqPort != "" {
				port = reqPort
			}
		}
		target := "https://" + net.JoinHostPort(host, port) + r.URL.RequestURI()
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	}), nil
}

// reachable returns true when host names one machine that remote clients can reach. A
// loopback or unspecified host does not name the machine to a remote client.
func reachable(host string) bool {
	if host == "" || host == "localhost" {
		return false
	}
	ip := net.ParseIP(host)
	return ip == nil || (!ip.IsLoopback() && !ip.IsUnspecified())
}

// splitRequestHost returns the host and port of the Host header of r. The port is empty
// when the header has none.
func splitRequestHost(r *http.Request) (host, port string) {
	if host, port, err := net.SplitHostPort(r.Host); err == nil {
		return host, port
	}
	return r.Host, ""
}

func certCovers(cfg *tls.Config, host string) bool {
	if cfg == nil || cfg.GetCertificate == nil {
		return false
	}
	c, err := cfg.GetCertificate(&tls.ClientHelloInfo{ServerName: host})
	if err != nil || len(c.Certificate) == 0 {
		return false
	}
	leaf := c.Leaf
	if leaf == nil {
		if leaf, err = x509.ParseCertificate(c.Certificate[0]); err != nil {
			return false
		}
	}
	return leaf.VerifyHostname(host) == nil
}
