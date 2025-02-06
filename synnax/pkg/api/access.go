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
	"github.com/synnaxlabs/x/gorp"
)

type AccessService struct {
	internal *rbac.Service
	dbProvider
}

func NewAccessService(p Provider) *AccessService {
	return &AccessService{
		internal:   p.RBAC,
		dbProvider: p.db,
	}
}

type (
	AccessCreatePolicyRequest struct {
		Policies []rbac.Policy `json:"policies" msgpack:"policies"`
	}
	AccessCreatePolicyResponse = AccessCreatePolicyRequest
)

func (s *AccessService) CreatePolicy(ctx context.Context, req AccessCreatePolicyRequest) (AccessCreatePolicyResponse, error) {
	if err := s.internal.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Objects: rbac.PolicyOntologyIDsFromPolicies(req.Policies),
		Action:  access.Create,
	}); err != nil {
		return AccessCreatePolicyRequest{}, err
	}
	results := make([]rbac.Policy, len(req.Policies))
	if err := s.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.NewWriter(tx)
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

// AccessRetrievePolicyRequest is a request for retrieving a policy from the cluster.
type AccessRetrievePolicyRequest struct {
	// Subjects are the ontology IDs of subjects of the policy.
	Subjects []ontology.ID `json:"subjects" msgpack:"subjects"`
	// Keys is an optional parameter for the list of keys in the ontology.
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

// AccessRetrievePolicyResponse is the response containing the retrieved policies.
type AccessRetrievePolicyResponse struct {
	// Policies is a list of policies retrieved from the cluster.
	Policies []rbac.Policy `json:"policies" msgpack:"policies"`
}

// RetrievePolicy retrieves policies from the cluster based on the provided request.
// It filters policies by subjects and keys if they are provided in the request.
// It enforces access control to ensure the subject has permission to retrieve the policies.
//
// Parameters:
//   - ctx: The context for the request.
//   - req: The request containing the subjects and keys to filter the policies.
//
// Returns:
//   - res: The response containing the retrieved policies.
//   - err: An error if the retrieval or access control enforcement fails.
func (s *AccessService) RetrievePolicy(
	ctx context.Context,
	req AccessRetrievePolicyRequest,
) (res AccessRetrievePolicyResponse, err error) {
	var (
		hasSubjects = len(req.Subjects) > 0
		hasKeys     = len(req.Keys) > 0
	)
	q := s.internal.NewRetriever()
	if hasSubjects {
		q = q.WhereSubjects(req.Subjects...)
	}
	if hasKeys {
		q = q.WhereKeys(req.Keys...)
	}
	if err = q.Entries(&res.Policies).Exec(ctx, nil); err != nil {
		return AccessRetrievePolicyResponse{}, err
	}
	if err = s.internal.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: rbac.PolicyOntologyIDsFromPolicies(res.Policies),
	}); err != nil {
		return AccessRetrievePolicyResponse{}, err
	}
	return res, nil
}

type AccessDeletePolicyRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *AccessService) DeletePolicy(ctx context.Context, req AccessDeletePolicyRequest) (types.Nil, error) {
	if err := s.internal.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Objects: rbac.PolicyOntologyIDs(req.Keys),
		Action:  access.Delete,
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}
