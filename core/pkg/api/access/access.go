// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package access

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
)

type Service struct {
	internal *rbac.Service
	db       *gorp.DB
}

func NewService(cfg config.Config) *Service {
	return &Service{internal: cfg.Service.RBAC, db: cfg.Distribution.DB}
}

const allowInternal = false

type (
	CreatePolicyRequest struct {
		Policies []policy.Policy `json:"policies" msgpack:"policies"`
	}
	CreatePolicyResponse = CreatePolicyRequest
)

func (s *Service) CreatePolicy(
	ctx context.Context,
	req CreatePolicyRequest,
) (CreatePolicyResponse, error) {
	results := make([]policy.Policy, len(req.Policies))
	if err := s.db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Objects: policy.OntologyIDsFromPolicies(req.Policies),
			Action:  access.ActionCreate,
		}); err != nil {
			return err
		}
		w := s.internal.Policy.NewWriter(tx, allowInternal)
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
		return CreatePolicyRequest{}, err
	}
	return CreatePolicyResponse{Policies: results}, nil
}

type RetrievePolicyRequest struct {
	Subjects []ontology.ID `json:"subjects" msgpack:"subjects"`
	Keys     []uuid.UUID   `json:"keys" msgpack:"keys"`
	Limit    int           `json:"limit" msgpack:"limit"`
	Offset   int           `json:"offset" msgpack:"offset"`
	Internal *bool         `json:"internal" msgpack:"internal"`
}

type RetrievePolicyResponse struct {
	Policies []policy.Policy `json:"policies" msgpack:"policies"`
}

func (s *Service) RetrievePolicy(
	ctx context.Context,
	req RetrievePolicyRequest,
) (res RetrievePolicyResponse, err error) {
	q := s.internal.Policy.NewRetrieve()
	if len(req.Subjects) > 0 {
		q = q.WhereSubjects(req.Subjects...)
	}
	if len(req.Keys) > 0 {
		q = q.WhereKeys(req.Keys...)
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}
	if req.Internal != nil {
		q = q.WhereInternal(*req.Internal)
	}
	if err = q.Entries(&res.Policies).Exec(ctx, nil); err != nil {
		return RetrievePolicyResponse{}, err
	}
	if err = s.internal.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: policy.OntologyIDsFromPolicies(res.Policies),
	}); err != nil {
		return RetrievePolicyResponse{}, err
	}
	return res, nil
}

type DeletePolicyRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *Service) DeletePolicy(ctx context.Context, req DeletePolicyRequest) (types.Nil, error) {
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Objects: policy.OntologyIDs(req.Keys),
			Action:  access.ActionDelete,
		}); err != nil {
			return err
		}
		return s.internal.Policy.NewWriter(tx, allowInternal).Delete(ctx, req.Keys...)
	})

}

type (
	CreateRoleRequest struct {
		Roles []role.Role `json:"roles" msgpack:"roles"`
	}
	CreateRoleResponse struct {
		Roles []role.Role `json:"roles" msgpack:"roles"`
	}
)

func (s *Service) CreateRole(
	ctx context.Context,
	req CreateRoleRequest,
) (CreateRoleResponse, error) {
	results := make([]role.Role, len(req.Roles))
	if err := s.db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Objects: []ontology.ID{role.OntologyID(uuid.Nil)},
			Action:  access.ActionCreate,
		}); err != nil {
			return err
		}
		w := s.internal.Role.NewWriter(tx, allowInternal)
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
		return CreateRoleResponse{}, err
	}
	return CreateRoleResponse{Roles: results}, nil
}

type (
	RetrieveRoleRequest struct {
		Keys     []uuid.UUID `json:"keys" msgpack:"keys"`
		Limit    int         `json:"limit" msgpack:"limit"`
		Offset   int         `json:"offset" msgpack:"offset"`
		Internal *bool       `json:"internal" msgpack:"internal"`
	}
	RetrieveRoleResponse struct {
		Roles []role.Role `json:"roles" msgpack:"roles"`
	}
)

func (s *Service) RetrieveRole(
	ctx context.Context,
	req RetrieveRoleRequest,
) (RetrieveRoleResponse, error) {
	var res RetrieveRoleResponse
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
	if req.Internal != nil {
		q = q.WhereInternal(*req.Internal)
	}
	if err := q.Entries(&res.Roles).Exec(ctx, nil); err != nil {
		return RetrieveRoleResponse{}, err
	}
	if err := s.internal.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: []ontology.ID{role.OntologyID(uuid.Nil)}, // Type-level check
	}); err != nil {
		return RetrieveRoleResponse{}, err
	}

	return res, nil
}

type DeleteRoleRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *Service) DeleteRole(ctx context.Context, req DeleteRoleRequest) (types.Nil, error) {
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		roleIDs := make([]ontology.ID, len(req.Keys))
		for i, key := range req.Keys {
			roleIDs[i] = role.OntologyID(key)
		}
		if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Objects: roleIDs,
			Action:  access.ActionDelete,
		}); err != nil {
			return err
		}
		w := s.internal.Role.NewWriter(tx, allowInternal)
		for _, key := range req.Keys {
			if err := w.Delete(ctx, key); err != nil {
				return err
			}
		}
		return nil
	})
}

type AssignRoleRequest struct {
	User uuid.UUID `json:"user" msgpack:"user"`
	Role uuid.UUID `json:"role" msgpack:"role"`
}

func (s *Service) AssignRole(
	ctx context.Context,
	req AssignRoleRequest,
) (types.Nil, error) {
	userID := user.OntologyID(req.User)
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Objects: []ontology.ID{userID},
			Action:  access.ActionUpdate,
		}); err != nil {
			return err
		}
		return s.internal.Role.NewWriter(tx, allowInternal).AssignRole(ctx, userID, req.Role)
	})
}

type UnassignRoleRequest struct {
	User uuid.UUID `json:"user" msgpack:"user"`
	Role uuid.UUID `json:"role" msgpack:"role"`
}

func (s *Service) UnassignRole(ctx context.Context, req UnassignRoleRequest) (types.Nil, error) {
	userID := user.OntologyID(req.User)
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.internal.NewEnforcer(tx).Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Objects: []ontology.ID{userID},
			Action:  access.ActionUpdate,
		}); err != nil {
			return err
		}
		return s.internal.Role.NewWriter(tx, true).UnassignRole(ctx, userID, req.Role)
	})
}
