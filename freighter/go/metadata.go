package freighter

import (
	"crypto/tls"
	"github.com/synnaxlabs/x/address"
)

// MD represents the metadata for a request that is passed to Middleware.
type MD struct {
	// Protocol is the protocol that the request is being sent over.
	Protocol string
	// Target is the address the request is being sent to.
	Target address.Address
	// Sec is the authentication information for the request.
	Sec SecurityInfo
	// Params is a set of arbitrary parameters that can be set by client side middleware,
	// and read by server side middleware.
	Params Params
}

type SecurityInfo struct {
	TLS struct {
		// Used is set to true if TLS is being used.
		Used bool
		tls.ConnectionState
	}
}

// Params is a set of arbitrary parameters that can be set by client side middleware, and
// read by server side middleware.
type Params map[string]interface{}

// GetRequired returns the value of the given key, or panics if the key is not set. It
// should only be used in contexts where the absence of a key represents a programming
// error.
func (p Params) GetRequired(k string) interface{} {
	v, ok := p[k]
	if !ok {
		panic("missing required param: " + k)
	}
	return v
}

// GetDefault returns the value of the given key, or the given default value if the key
// is not set.
func (p Params) GetDefault(k string, def interface{}) interface{} {
	v, ok := p[k]
	if !ok {
		return def
	}
	return v
}

// Get returns the value of the given key, or nil if the key is not set.
func (p Params) Get(k string) (interface{}, bool) {
	v, ok := p[k]
	return v, ok
}

// Set sets the value of the given key.
func (p Params) Set(k string, v interface{}) { p[k] = v }

// SetIfAbsent sets the value of the given key if it is not already set.
func (p Params) SetIfAbsent(k string, v interface{}) {
	if _, ok := p[k]; !ok {
		p[k] = v
	}
}
