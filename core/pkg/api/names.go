// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import (
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
)

// channelNamesMiddleware rewrites the channel errors of a request so that they name
// each channel the caller can retrieve.
func channelNamesMiddleware(
	db *gorp.DB,
	channels *channel.Service,
	rbacSvc *rbac.Service,
) freighter.Middleware {
	return freighter.MiddlewareFunc(func(
		ctx freighter.Context,
		next freighter.Next,
	) (freighter.Context, error) {
		oCtx, err := next(ctx)
		if err == nil {
			return oCtx, nil
		}
		return oCtx, apichannel.NameErrors(ctx, db, channels, rbacSvc, err)
	})
}
