// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package verification

import (
	"github.com/synnaxlabs/freighter"
	svcverification "github.com/synnaxlabs/synnax/pkg/service/channel/verification"
)

// Middleware rejects every request while the Core has no covering grant, with the
// error the service reports. The roster it is applied to is the allowlist: the
// middleware never inspects the request target.
func Middleware(svc *svcverification.Service) freighter.Middleware {
	return freighter.MiddlewareFunc(func(
		ctx freighter.Context,
		next freighter.Next,
	) (freighter.Context, error) {
		if err := svc.Check(); err != nil {
			return ctx, err
		}
		return next(ctx)
	})
}
