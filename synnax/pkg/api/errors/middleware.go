// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package errors

import (
	roacherrors "github.com/cockroachdb/errors"
	"github.com/sirupsen/logrus"
	"github.com/synnaxlabs/freighter"
)

func Middleware() freighter.Middleware {
	return freighter.MiddlewareFunc(func(
		ctx freighter.Context,
		next freighter.Next,
	) (freighter.Context, error) {
		oCtx, err := next(ctx)
		logrus.Info(err)
		var t Typed
		if roacherrors.As(err, &t) {
			if !t.Occurred() {
				return oCtx, nil
			}
		}
		return oCtx, err
	})
}
