// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rbac

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/access"
)

var _ access.Enforcer = (*Service)(nil)

// Enforce implements the access.Enforcer interface. It checks both direct user policies
// and policies from all roles assigned to the user.
func (s *Service) Enforce(ctx context.Context, req access.Request) error {
	var allPolicies []Policy

	// 1. Get direct user policies
	var userPolicies []Policy
	if err := s.NewRetrieve().Entries(&userPolicies).WhereSubjects(req.Subject).Exec(ctx, s.DB); err != nil {
		return err
	}
	allPolicies = append(allPolicies, userPolicies...)

	// 2. Get user's roles
	userKey, err := uuid.Parse(req.Subject.Key)
	if err != nil {
		// If subject key is not a valid UUID, skip role checking
		// (might be a type-level subject)
		if allowRequest(req, allPolicies) {
			return access.Granted
		}
		return access.Denied
	}

	roleKeys, err := s.UserRoleGetter.GetUserRoles(ctx, userKey)
	if err != nil {
		// If we can't get roles, just check with direct policies
		if allowRequest(req, allPolicies) {
			return access.Granted
		}
		return access.Denied
	}

	// 3. Get policies for each role the user has
	for _, roleKey := range roleKeys {
		var rolePolicies []Policy
		roleSubject := RoleOntologyID(roleKey)
		if err := s.NewRetrieve().Entries(&rolePolicies).WhereSubjects(roleSubject).Exec(ctx, s.DB); err != nil {
			// Skip this role if we can't retrieve its policies
			continue
		}
		allPolicies = append(allPolicies, rolePolicies...)
	}

	// 4. Check if combined policies allow the request
	if allowRequest(req, allPolicies) {
		return access.Granted
	}
	return access.Denied
}
