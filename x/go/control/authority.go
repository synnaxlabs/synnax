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
	"github.com/synnaxlabs/x/errors"
	"math"
)

type Authority uint8

const (
	errorPrefix            = "sy.control"
	unauthorized           = errorPrefix + ".unauthorized"
	Absolute     Authority = math.MaxUint8
)

var (
	Error        = errors.New("control")
	Unauthorized = errors.Wrap(Error, "unauthorized")
)

type Concurrency uint8

const (
	Exclusive Concurrency = iota
	Shared
)

func encode(_ context.Context, err error) (errors.Payload, bool) {
	if errors.Is(err, Unauthorized) {
		return errors.Payload{Type: unauthorized, Data: err.Error()}, true
	}
	return errors.Payload{}, false
}

func decode(_ context.Context, p errors.Payload) (error, bool) {
	switch p.Type {
	case "sy.control.unauthorized":
		return errors.Wrap(Unauthorized, p.Data), true
	default:
		return nil, false
	}
}

func init() {
	errors.Register(encode, decode)
}
