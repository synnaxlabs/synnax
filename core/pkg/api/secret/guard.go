// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Guard returns a guard for a write by the subject of ctx. The guard returns
// access.ErrDenied when the resources a record references include a secret that the
// subject cannot retrieve, so a user cannot point a task at a secret they cannot read.
func Guard(ctx context.Context, enforcer *rbac.Enforcer) func([]resource.ID) error {
	subject := auth.GetSubject(ctx)
	return func(ids []resource.ID) error {
		secrets := lo.Filter(ids, func(id resource.ID, _ int) bool {
			return id.Type == resource.TypeSecret
		})
		if len(secrets) == 0 {
			return nil
		}
		return enforcer.Enforce(ctx, access.Request{
			Subject: subject,
			Action:  access.ActionRetrieve,
			Objects: secrets,
		})
	}
}
