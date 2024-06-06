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

type TimeRangeDeleteRequest struct {
	Keys      channel.Keys    `json:"keys" msgpack:"keys" validate:"required"`
	Names     []string        `json:"names" msgpack:"names" validate:"required"`
	TimeRange telem.TimeRange `json:"timerange" msgpack:"timerange" validate:"required"`
}

func (s *FrameService) TimeRangeDelete(
	ctx context.Context,
	req TimeRangeDeleteRequest,
) (types.Nil, error) {
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		c := errors.NewCatcher(errors.WithAggregation())
		w := s.internal.NewWriter(tx)
		if len(req.Keys) > 0 {
			c.Exec(func() error {
				return w.DeleteMany(ctx, req.Keys, false)
			})
		}
		if len(req.Names) > 0 {
			c.Exec(func() error {
				return w.DeleteManyByNames(ctx, req.Names, false)
			})
		}
		return c.Error()
	})
}
