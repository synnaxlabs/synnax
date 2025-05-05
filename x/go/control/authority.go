// Copyright 2025 Synnax Labs, Inc.
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

// Authority is an 8-bit unsigned integer that represents the authority that a particular
// controlling subject has over a resource. A higher authority means higher precedence
// over the resource. Absolute authority (255) maintains exclusive control over the
// resource.
type Authority uint8

const (
	errorPrefix  = "sy.control"
	unauthorized = errorPrefix + ".unauthorized"
	// Absolute control authority is the higher control authority possible.
	Absolute Authority = math.MaxUint8
)

var (
	// Error is a general classification of control error.
	Error = errors.New("control")
	// Unauthorized is returned when a subject does not have authority to perform
	// actions on a resource.
	Unauthorized = errors.Wrap(Error, "unauthorized")
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
	if errors.Is(err, Unauthorized) {
		return errors.Payload{Type: unauthorized, Data: err.Error()}, true
	}
	return errors.Payload{}, false
}

func decode(_ context.Context, p errors.Payload) (error, bool) {
	if !strings.HasPrefix(p.Type, "sy.control") {
		return nil, false
	}
	if p.Type == unauthorized {
		return errors.Wrap(Unauthorized, p.Data), true
	}
	return errors.Wrap(Error, p.Data), true
}

func init() {
	errors.Register(encode, decode)
}
