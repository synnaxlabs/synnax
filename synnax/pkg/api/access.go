// Copyright 2024 Synnax Labs, Inc.
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
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/access"
	"github.com/synnaxlabs/synnax/pkg/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
	"go/types"
)

type AccessService struct {
	internal *rbac.Service
	accessProvider
	dbProvider
}

type (
	AccessCreatePolicyRequest struct {
		Policies []rbac.Policy `json:"policies" msgpack:"policies"`
	}
	AccessCreatePolicyResponse = AccessCreatePolicyRequest
)

func (a *AccessService) CreatePolicy(ctx context.Context, req AccessCreatePolicyRequest) (AccessCreatePolicyResponse, error) {
	if err := a.enforcer.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Objects: []ontology.ID{{Type: rbac.OntologyType}},
		Action:  access.Create,
	}); err != nil {
		return AccessCreatePolicyRequest{}, err
	}
	results := make([]rbac.Policy, len(req.Policies))
	if err := a.WithTx(ctx, func(tx gorp.Tx) error {
		w := a.internal.NewWriter(tx)
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

type AccessDeletePolicyRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (a *AccessService) DeletePolicy(ctx context.Context, req AccessDeletePolicyRequest) (types.Nil, error) {
	if err := a.enforcer.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Objects: rbac.OntologyIDs(req.Keys),
		Action:  access.Delete,
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, a.WithTx(ctx, func(tx gorp.Tx) error {
		w := a.internal.NewWriter(tx)
		for _, key := range req.Keys {
			if err := w.Delete(ctx, key); err != nil {
				return err
			}
		}
		return nil
	})
}

type (
	AccessRetrievePolicyRequest struct {
		Subject ontology.ID `json:"subject" msgpack:"subject""`
	}
	AccessRetrievePolicyResponse struct {
		Policies []rbac.Policy `json:"policies" msgpack:"policies"`
	}
)

func (a *AccessService) RetrievePolicy(
	ctx context.Context,
	req AccessRetrievePolicyRequest,
) (res AccessRetrievePolicyResponse, err error) {
	if err = a.internal.NewRetrieve().
		WhereSubject(req.Subject).
		Entries(&res.Policies).
		Exec(ctx, nil); err != nil {
		return AccessRetrievePolicyResponse{}, err
	}
	if err = a.enforcer.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: rbac.OntologyIDsFromPolicies(res.Policies),
	}); err != nil {
		return AccessRetrievePolicyResponse{}, err
	}
	return res, nil
}
