// Copyright 2022 Synnax Labs, Inc.
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
	"time"

	"github.com/synnaxlabs/freighter"
	"go.uber.org/zap"
)

func logMiddleware(z *zap.SugaredLogger) freighter.Middleware {
	return freighter.MiddlewareFunc(func(
		ctx context.Context,
		md freighter.MD,
		next freighter.Next,
	) (freighter.MD, error) {
		t0 := time.Now()
		oMd, err := next(ctx, md)
		logFunc(z, err)(
			"api request",
			"protocol", md.Protocol,
			"target", md.Target,
			"duration", time.Since(t0),
			"err",
			err,
		)
		return oMd, err
	})
}

func logFunc(z *zap.SugaredLogger, err error) func(msg string, args ...interface{}) {
	if err != nil {
		return z.Errorw
	}
	return z.Infow
}
