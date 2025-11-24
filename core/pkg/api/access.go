// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/gorp"
)

type AccessService struct {
	internal *rbac.Service
	dbProvider
}

func NewAccessService(p Provider) *AccessService {
	return &AccessService{internal: p.Service.RBAC, dbProvider: p.db}
}

type (
	AccessCreatePolicyRequest struct {
		Policies []policy.Policy `json:"policies" msgpack:"policies"`
	}
	AccessCreatePolicyResponse = AccessCreatePolicyRequest
)

func (s *AccessService) CreatePolicy(
	ctx context.Context,
	req AccessCreatePolicyRequest,
) (AccessCreatePolicyResponse, error) {
	results := make([]policy.Policy, len(req.Policies))
	if err := s.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: getSubject(ctx),
			Objects: policy.OntologyIDsFromPolicies(req.Policies),
			Action:  access.ActionCreate,
		}); err != nil {
			return err
		}
		w := s.internal.Policy.NewWriter(tx)
		for i, p := range req.Policies {
			if p.Key == uuid.Nil {
				p.Key = uuid.New()
			}
			if err := w.Create(ctx, &p); err != nil {
				return err
			}
			results[i] = p
		}
		return nil
	}); err != nil {
		return AccessCreatePolicyRequest{}, err
	}
	return AccessCreatePolicyResponse{Policies: results}, nil
}

type AccessRetrievePolicyRequest struct {
	Subjects []ontology.ID `json:"subjects" msgpack:"subjects"`
	Keys     []uuid.UUID   `json:"keys" msgpack:"keys"`
	Limit    int           `json:"limit" msgpack:"limit"`
	Offset   int           `json:"offset" msgpack:"offset"`
}

type AccessRetrievePolicyResponse struct {
	Policies []policy.Policy `json:"policies" msgpack:"policies"`
}

func (s *AccessService) RetrievePolicy(
	ctx context.Context,
	req AccessRetrievePolicyRequest,
) (res AccessRetrievePolicyResponse, err error) {
	// If subjects are provided, retrieve policies for those subjects (via roles)
	if len(req.Subjects) > 0 {
		// Retrieve policies for each subject and deduplicate
		policyMap := make(map[uuid.UUID]policy.Policy)
		for _, subject := range req.Subjects {
			policies, err := s.internal.RetrievePoliciesForSubject(ctx, subject, nil)
			if err != nil {
				return AccessRetrievePolicyResponse{}, err
			}
			for _, p := range policies {
				policyMap[p.Key] = p
			}
		}
		// Convert map to slice
		res.Policies = make([]policy.Policy, 0, len(policyMap))
		for _, p := range policyMap {
			res.Policies = append(res.Policies, p)
		}
		// Apply pagination to the result
		if req.Offset > 0 {
			if req.Offset >= len(res.Policies) {
				res.Policies = []policy.Policy{}
			} else {
				res.Policies = res.Policies[req.Offset:]
			}
		}
		if req.Limit > 0 && len(res.Policies) > req.Limit {
			res.Policies = res.Policies[:req.Limit]
		}
	} else if len(req.Keys) > 0 {
		// Retrieve by keys (existing behavior)
		q := s.internal.Policy.NewRetrieve()
		q = q.WhereKeys(req.Keys...)
		if req.Limit > 0 {
			q = q.Limit(req.Limit)
		}
		if req.Offset > 0 {
			q = q.Offset(req.Offset)
		}
		if err = q.Entries(&res.Policies).Exec(ctx, nil); err != nil {
			return AccessRetrievePolicyResponse{}, err
		}
	} else {
		// No filters provided, retrieve all policies
		q := s.internal.Policy.NewRetrieve()
		if req.Limit > 0 {
			q = q.Limit(req.Limit)
		}
		if req.Offset > 0 {
			q = q.Offset(req.Offset)
		}
		if err = q.Entries(&res.Policies).Exec(ctx, nil); err != nil {
			return AccessRetrievePolicyResponse{}, err
		}
	}

	// Enforce access control check
	if err = s.internal.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: policy.OntologyIDsFromPolicies(res.Policies),
	}); err != nil {
		return AccessRetrievePolicyResponse{}, err
	}
	return res, nil
}

type AccessDeletePolicyRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *AccessService) DeletePolicy(ctx context.Context, req AccessDeletePolicyRequest) (types.Nil, error) {
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: getSubject(ctx),
			Objects: policy.OntologyIDs(req.Keys),
			Action:  access.ActionDelete,
		}); err != nil {
			return err
		}
		return s.internal.Policy.NewWriter(tx).Delete(ctx, req.Keys...)
	})

}

// Role API endpoints

type (
	AccessCreateRoleRequest struct {
		Roles []role.Role `json:"roles" msgpack:"roles"`
	}
	AccessCreateRoleResponse struct {
		Roles []role.Role `json:"roles" msgpack:"roles"`
	}
)

func (s *AccessService) CreateRole(
	ctx context.Context,
	req AccessCreateRoleRequest,
) (AccessCreateRoleResponse, error) {
	results := make([]role.Role, len(req.Roles))
	if err := s.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: getSubject(ctx),
			Objects: []ontology.ID{role.OntologyID(uuid.Nil)},
			Action:  access.ActionCreate,
		}); err != nil {
			return err
		}
		w := s.internal.Role.NewWriter(tx)
		for i, r := range req.Roles {
			if r.Key == uuid.Nil {
				r.Key = uuid.New()
			}
			if err := w.Create(ctx, &r); err != nil {
				return err
			}
			results[i] = r
		}
		return nil
	}); err != nil {
		return AccessCreateRoleResponse{}, err
	}
	return AccessCreateRoleResponse{Roles: results}, nil
}

type (
	AccessRetrieveRoleRequest struct {
		Keys   []uuid.UUID `json:"keys" msgpack:"keys"`
		Limit  int         `json:"limit" msgpack:"limit"`
		Offset int         `json:"offset" msgpack:"offset"`
	}
	AccessRetrieveRoleResponse struct {
		Roles []role.Role `json:"roles" msgpack:"roles"`
	}
)

func (s *AccessService) RetrieveRole(
	ctx context.Context,
	req AccessRetrieveRoleRequest,
) (AccessRetrieveRoleResponse, error) {
	var res AccessRetrieveRoleResponse
	q := s.internal.Role.NewRetrieve()
	if len(req.Keys) > 0 {
		q = q.WhereKeys(req.Keys...)
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}
	if err := q.Entries(&res.Roles).Exec(ctx, nil); err != nil {
		return AccessRetrieveRoleResponse{}, err
	}
	if err := s.internal.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: []ontology.ID{role.OntologyID(uuid.Nil)}, // Type-level check
	}); err != nil {
		return AccessRetrieveRoleResponse{}, err
	}

	return res, nil
}

type AccessDeleteRoleRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *AccessService) DeleteRole(ctx context.Context, req AccessDeleteRoleRequest) (types.Nil, error) {
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		roleIDs := make([]ontology.ID, len(req.Keys))
		for i, key := range req.Keys {
			roleIDs[i] = role.OntologyID(key)
		}
		if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: getSubject(ctx),
			Objects: roleIDs,
			Action:  access.ActionDelete,
		}); err != nil {
			return err
		}
		w := s.internal.Role.NewWriter(tx)
		for _, key := range req.Keys {
			if err := w.Delete(ctx, key); err != nil {
				return err
			}
		}
		return nil
	})
}

type AccessAssignRoleRequest struct {
	User ontology.ID `json:"user" msgpack:"user"`
	Role uuid.UUID   `json:"role" msgpack:"role"`
}

func (s *AccessService) AssignRole(
	ctx context.Context,
	req AccessAssignRoleRequest,
) (types.Nil, error) {
	return types.Nil{}, s.DB.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: getSubject(ctx),
			Objects: []ontology.ID{req.User},
			Action:  access.ActionUpdate,
		}); err != nil {
			return err
		}
		return s.internal.Role.NewWriter(tx).AssignRole(ctx, req.User, req.Role)
	})
}

type AccessUnassignRoleRequest struct {
	User ontology.ID `json:"user" msgpack:"user"`
	Role uuid.UUID   `json:"role" msgpack:"role"`
}

func (s *AccessService) UnassignRole(ctx context.Context, req AccessUnassignRoleRequest) (types.Nil, error) {
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: getSubject(ctx),
			Objects: []ontology.ID{req.User},
			Action:  access.ActionUpdate,
		}); err != nil {
			return err
		}
		return s.internal.Role.NewWriter(tx).UnassignRole(ctx, req.User, req.Role)
	})
}
