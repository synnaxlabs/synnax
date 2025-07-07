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

type UnaryServer[RQ, RS freighter.Payload] struct {
	freighter.Reporter
}

var _ freighter.UnaryServer[any, any] = (*UnaryServer[any, any])(nil)

func (us UnaryServer[RQ, RS]) Use(...freighter.Middleware) {}

func (us UnaryServer[RQ, RS]) BindHandler(func(context.Context, RQ) (RS, error)) {}

type UnaryClient[RQ, RS freighter.Payload] struct {
	freighter.Reporter
}

var _ freighter.UnaryClient[any, any] = (*UnaryClient[any, any])(nil)

func (uc UnaryClient[RQ, RS]) Use(...freighter.Middleware) {}

func (uc UnaryClient[RQ, RS]) Send(context.Context, address.Address, RQ) (RS, error) {
	var res RS
	return res, nil
}
