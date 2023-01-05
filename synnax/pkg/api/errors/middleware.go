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
	"context"
	roacherrors "github.com/cockroachdb/errors"
	"github.com/synnaxlabs/freighter"
)

func Middleware() freighter.Middleware {
	return freighter.MiddlewareFunc(func(
		ctx context.Context,
		md freighter.MD,
		next freighter.Next,
	) (freighter.MD, error) {
		oMd, err := next(ctx, md)
		var t Typed
		if roacherrors.As(err, &t) {
			if !t.Occurred() {
				return oMd, nil
			}
		}
		return oMd, err
	})
}
