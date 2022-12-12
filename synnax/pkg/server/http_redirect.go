package server

import "net/http"

// NewHTTPRedirectBranch returns a new SimpleHTTPBranch.
func NewHTTPRedirectBranch() *SimpleHTTPBranch {
	return NewSimpleHTTPBranch(http.HandlerFunc(secureHTTPRedirect), ServeOnInsecureIfSecure)
}
