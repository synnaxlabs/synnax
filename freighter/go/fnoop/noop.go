// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fnoop

import (
	"context"

	"github.com/synnaxlabs/freighter"
)

type UnaryServer[RQ, RS freighter.Payload] struct {
	freighter.Reporter
}

var _ freighter.UnaryServer[any, any] = (*UnaryServer[any, any])(nil)

func (s UnaryServer[RQ, RS]) Use(middleware ...freighter.Middleware) {
}

func (s UnaryServer[RQ, RS]) BindHandler(handle func(ctx context.Context, req RQ) (res RS, err error)) {
}

type StreamServer[RQ, RS freighter.Payload] struct {
	freighter.Reporter
}

var _ freighter.StreamServer[any, any] = (*StreamServer[any, any])(nil)

func (s StreamServer[RQ, RS]) Use(middleware ...freighter.Middleware) {
}

func (s StreamServer[RQ, RS]) BindHandler(handle func(ctx context.Context, stream freighter.ServerStream[RQ, RS]) (err error)) {
}
