// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package control

import (
	"context"
	"math"
	"strings"

	"github.com/synnaxlabs/x/errors"
)

// Authority is an 8-bit unsigned integer that represents the authority that a
// particular controlling subject has over a resource. A higher authority means higher
// precedence over the resource. AuthorityAbsolute authority (255) maintains exclusive control
// over the resource.
type Authority uint8

const (
	errorPrefix  = "sy.control"
	unauthorized = errorPrefix + ".unauthorized"
	// AuthorityAbsolute control authority is the higher control authority possible.
	AuthorityAbsolute Authority = math.MaxUint8
)

var (
	// Error is a general classification of control error.
	Error = errors.New("control")
	// ErrUnauthorized is returned when a subject does not have authority to perform
	// actions on a resource.
	ErrUnauthorized = errors.Wrap(Error, "unauthorized")
)

// Concurrency defines whether a resource can have multiple subjects acting on it at once.
type Concurrency uint8

const (
	// Exclusive means that only a single subject has control over a resource at once.
	Exclusive Concurrency = iota
	// Shared means that multiple subjects can share control over a resource.
	Shared
)

func encode(_ context.Context, err error) (errors.Payload, bool) {
	if errors.Is(err, ErrUnauthorized) {
		return errors.Payload{Type: unauthorized, Data: err.Error()}, true
	}
	return errors.Payload{}, false
}

func decode(_ context.Context, p errors.Payload) (error, bool) {
	if !strings.HasPrefix(p.Type, errorPrefix) {
		return nil, false
	}
	if p.Type == unauthorized {
		return errors.Wrap(ErrUnauthorized, p.Data), true
	}
	return errors.Wrap(Error, p.Data), true
}

func init() {
	errors.Register(encode, decode)
}
