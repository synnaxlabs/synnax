// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import (
	"context"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
	"go/types"
)

type Frame = framer.Frame

type FrameService struct {
	alamos.Instrumentation
	authProvider
	dbProvider
	Internal *framer.Service
}

func NewFrameService(p Provider) *FrameService {
	return &FrameService{
		Instrumentation: p.Instrumentation,
		Internal:        p.Config.Framer,
		authProvider:    p.auth,
		dbProvider:      p.db,
	}
}

type FrameDeleteRequest struct {
	Keys   channel.Keys    `json:"keys" msgpack:"keys" validate:"required"`
	Names  []string        `json:"names" msgpack:"names" validate:"names"`
	Bounds telem.TimeRange `json:"bounds" msgpack:"bounds" validate:"bounds"`
}

func (s *FrameService) FrameDelete(
	ctx context.Context,
	req FrameDeleteRequest,
) (types.Nil, error) {
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		c := errors.NewCatcher(errors.WithAggregation())
		w := s.Internal.NewDeleter()
		if len(req.Keys) > 0 {
			c.Exec(func() error {
				return w.DeleteTimeRangeMany(ctx, req.Keys, req.Bounds)
			})
		}
		if len(req.Names) > 0 {
			c.Exec(func() error {
				return w.DeleteTimeRangeManyByNames(ctx, req.Names, req.Bounds)
			})
		}
		return c.Error()
	})
}
