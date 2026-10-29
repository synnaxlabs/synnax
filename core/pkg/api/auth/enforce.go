// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
)

// EnforceOthers checks that caller may perform action on every subject other than
// itself. A caller may always act on itself.
func EnforceOthers(
	ctx context.Context,
	svc *rbac.Service,
	caller ontology.ID,
	action access.Action,
	subjects []ontology.ID,
) error {
	others := lo.Filter(subjects, func(id ontology.ID, _ int) bool {
		return id != caller
	})
	if len(others) == 0 {
		return nil
	}
	return svc.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: caller,
		Action:  action,
		Objects: others,
	})
}
