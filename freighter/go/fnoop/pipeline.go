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
	"io"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
)

type PipelineServer[RQ freighter.Payload] struct {
	freighter.Reporter
}

var _ freighter.PipelineServer[any] = (*PipelineServer[any])(nil)

func (ps PipelineServer[RQ]) Use(...freighter.Middleware) {}

func (ps PipelineServer[RQ]) BindHandler(func(context.Context, RQ) (io.Reader, error)) {}

type PipelineClient[RQ freighter.Payload] struct {
	freighter.Reporter
}

var _ freighter.PipelineClient[any] = (*PipelineClient[any])(nil)

func (pc PipelineClient[RQ]) Use(...freighter.Middleware) {}

func (pc PipelineClient[RQ]) Send(context.Context, address.Address, RQ) (io.ReadCloser, error) {
	return nil, nil
}
