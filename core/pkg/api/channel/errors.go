// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
)

// NameErrors returns err with each channel error in it naming its channel by record,
// for the channels that the subject of ctx can retrieve. A channel the subject cannot
// retrieve keeps its key. When the access check fails, NameErrors returns err.
func NameErrors(
	ctx context.Context,
	db *gorp.DB,
	channels *channel.Service,
	rbacSvc *rbac.Service,
	err error,
) error {
	if err == nil {
		return nil
	}
	named := err
	if sErr := db.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		allowed, aErr := rbacSvc.NewEnforcer(snap).Allowed(
			ctx,
			auth.GetSubject(ctx),
			access.ActionRetrieve,
		)
		if aErr != nil {
			return aErr
		}
		named = channels.NameErrors(ctx, err, func(k channel.Key) bool {
			return allowed(channel.ResourceID(k))
		})
		return nil
	}); sErr != nil {
		return err
	}
	return named
}
