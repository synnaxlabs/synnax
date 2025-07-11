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
	"github.com/synnaxlabs/x/address"
)

type StreamServer[RQ, RS freighter.Payload] struct {
	freighter.Reporter
}

var _ freighter.StreamServer[any, any] = (*StreamServer[any, any])(nil)

func (ss StreamServer[RQ, RS]) Use(...freighter.Middleware) {}

func (ss StreamServer[RQ, RS]) BindHandler(func(context.Context, freighter.ServerStream[RQ, RS]) error) {
}

type StreamClient[RQ, RS freighter.Payload] struct{ freighter.Reporter }

var _ freighter.StreamClient[any, any] = (*StreamClient[any, any])(nil)

func (sc StreamClient[RQ, RS]) Use(...freighter.Middleware) {}

func (sc StreamClient[RQ, RS]) Stream(context.Context, address.Address) (freighter.ClientStream[RQ, RS], error) {
	return nil, nil
}
