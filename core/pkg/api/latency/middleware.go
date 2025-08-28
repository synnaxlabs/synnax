// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package latency

import (
	"github.com/synnaxlabs/freighter"
	"time"
)

func Middleware(delay time.Duration) freighter.Middleware {
	return freighter.MiddlewareFunc(func(
		ctx freighter.Context,
		next freighter.Next,
	) (oMD freighter.Context, err error) {
		time.Sleep(delay)
		return next(ctx)
	})
}
