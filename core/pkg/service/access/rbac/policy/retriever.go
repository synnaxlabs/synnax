// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package policy

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

type Retrieve struct {
	baseTX  gorp.Tx
	gorp    gorp.Retrieve[Key, Policy]
	indexes indexes
	rolesOf RolesOf
}

// RolesOf returns the keys of the roles that the subjects hold, as seen by tx. It
// returns query.ErrNotFound when a subject does not exist.
type RolesOf func(
	ctx context.Context,
	tx gorp.Tx,
	subjects ...resource.ID,
) ([]role.Key, error)

// ResolveRolesWith returns r with the function that [MatchSubjects] reads the roles of
// each subject through.
func (r Retrieve) ResolveRolesWith(rolesOf RolesOf) Retrieve {
	r.rolesOf = rolesOf
	return r
}

// MatchSubjects returns a filter that matches the policies of the roles that any of the
// subjects hold. A subject that does not exist holds no roles. The filter fails on a
// retrieve opened without [Retrieve.ResolveRolesWith].
func MatchSubjects(subjects ...resource.ID) Filter {
	return func(r Retrieve) gorp.Filter[Key, Policy] {
		return gorp.MatchKeysFunc[Key, Policy](func(ctx gorp.Context) ([]Key, error) {
			if r.rolesOf == nil {
				return nil, errors.New("this policy retrieve cannot filter by subject")
			}
			var roles []role.Key
			for _, subject := range subjects {
				held, err := r.rolesOf(ctx, ctx.Tx, subject)
				if errors.Is(err, query.ErrNotFound) {
					continue
				}
				if err != nil {
					return nil, err
				}
				roles = append(roles, held...)
			}
			if len(roles) == 0 {
				return nil, nil
			}
			return r.indexes.role.Get(ctx, ctx.Tx, roles...)
		})
	}
}
