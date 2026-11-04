// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the status API service.
type ServiceConfig struct {
	// Status stores and retrieves statuses.
	//
	// [REQUIRED]
	Status *status.Service
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the status API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Status = override.Nil(c.Status, other.Status)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.status")
	v.NotNil("status", c.Status)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

type Service struct {
	access   *rbac.Service
	internal *status.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		internal: cfg.Status,
		access:   cfg.RBAC,
	}, nil
}

func statusAccessResourceIDs(statuses []status.Status[any]) []resource.ID {
	ids := make([]resource.ID, 0, len(statuses))
	for _, s := range statuses {
		ids = append(ids, s.ResourceID())
		ids = append(ids, label.ResourceIDs(s.Labels)...)
	}
	return ids
}

// SetRequest is a request to set (create or update) statuses in the cluster.
type SetRequest struct {
	// Statuses are the statuses to set. A status with an owner goes to the status its
	// owner holds, whatever key it carries.
	Statuses []status.Status[any] `json:"statuses" msgpack:"statuses"`
}

// SetResponse is a response to a SetRequest.
type SetResponse struct {
	// Statuses are the statuses as stored, with their resolved keys and labels.
	Statuses []status.Status[any] `json:"statuses" msgpack:"statuses"`
}

// Set creates or updates statuses in the cluster.
func (s *Service) Set(
	ctx context.Context,
	tx gorp.Tx,
	req SetRequest,
) (SetResponse, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: statusAccessResourceIDs(req.Statuses),
	}); err != nil {
		return SetResponse{}, err
	}
	if err := s.internal.NewWriter(tx).SetMany(ctx, &req.Statuses); err != nil {
		return SetResponse{}, err
	}
	return SetResponse(req), nil
}

// SetByKeyOrNameRequest is a request to upsert a status by key or by name.
type SetByKeyOrNameRequest struct {
	// KeyOrName is either an existing status key or a status name to match.
	KeyOrName string `json:"key_or_name" msgpack:"key_or_name"`
	// Message is the new status message.
	Message string `json:"message" msgpack:"message"`
	// Variant is the new status variant.
	Variant status.Variant `json:"variant" msgpack:"variant"`
}

// SetByKeyOrNameResponse is a response to a SetByKeyOrNameRequest.
type SetByKeyOrNameResponse struct {
	// Key is the key of the upserted status.
	Key status.Key `json:"key" msgpack:"key"`
	// MultipleMatches reports whether multiple statuses matched by name.
	MultipleMatches bool `json:"multiple_matches" msgpack:"multiple_matches"`
}

// SetByKeyOrName upserts a status by key or by name.
func (s *Service) SetByKeyOrName(
	ctx context.Context,
	tx gorp.Tx,
	req SetByKeyOrNameRequest,
) (SetByKeyOrNameResponse, error) {
	if !req.Variant.IsValid() {
		return SetByKeyOrNameResponse{}, errors.Wrap(
			validate.ErrValidation,
			"invalid status variant",
		)
	}
	matches, err := s.internal.ResolveKeyOrName(ctx, tx, req.KeyOrName)
	if err != nil {
		return SetByKeyOrNameResponse{}, err
	}
	st := status.SetTarget(matches, req.KeyOrName, req.Message, string(req.Variant))
	action := access.ActionUpdate
	if len(matches) == 0 {
		action = access.ActionCreate
	}
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  action,
		Objects: []resource.ID{st.ResourceID()},
	}); err != nil {
		return SetByKeyOrNameResponse{}, err
	}
	if err := s.internal.NewWriter(tx).Set(ctx, &st); err != nil {
		return SetByKeyOrNameResponse{}, err
	}
	return SetByKeyOrNameResponse{
		Key:             st.Key,
		MultipleMatches: len(matches) > 1,
	}, nil
}

type RetrieveRequest struct {
	// SearchTerm is used for fuzzy searching statuses.
	SearchTerm string `json:"search_term" msgpack:"search_term"`
	// Keys are the keys of the statuses to retrieve.
	Keys []status.Key `json:"keys" msgpack:"keys"`
	// Owners retrieves the statuses that the resources with the given IDs own.
	Owners []resource.ID `json:"owners" msgpack:"owners"`
	// HasLabels retrieves statuses that are labeled by one or more labels with the
	// given keys.
	HasLabels []label.Key `json:"has_labels" msgpack:"has_labels"`
	// Variants filters for statuses with the given variants.
	Variants []status.Variant `json:"variants" msgpack:"variants"`
	// Limit is the maximum number of statuses to retrieve.
	Limit int `json:"limit" msgpack:"limit"`
	// Offset is the number of statuses to skip.
	Offset int `json:"offset" msgpack:"offset"`
	// IgnoreNotFoundError returns the statuses that do exist instead of an error when
	// one or more of Keys names a status that does not.
	IgnoreNotFoundError bool `json:"ignore_not_found_error" msgpack:"ignore_not_found_error"`
}

type RetrieveResponse struct {
	// Statuses are the statuses that were retrieved.
	Statuses []status.Status[any] `json:"statuses" msgpack:"statuses"`
}

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	q := s.internal.NewRetrieve[any]()
	resStatuses := make([]status.Status[any], 0, len(req.Keys))

	if req.SearchTerm != "" {
		q = q.Where(status.MatchSearch(req.SearchTerm))
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}
	if len(req.HasLabels) > 0 {
		q = q.Where(status.MatchLabels[any](req.HasLabels...))
	}
	if len(req.Variants) > 0 {
		q = q.Where(status.MatchVariants[any](req.Variants...))
	}
	if len(req.Keys) != 0 {
		q = q.Where(status.MatchKeys[any](req.Keys...))
	}
	if len(req.Owners) != 0 {
		q = q.Where(status.MatchOwners[any](req.Owners...))
	}
	err := q.Entries(&resStatuses).Exec(ctx, nil)
	if req.IgnoreNotFoundError && err != nil {
		err = errors.Skip(err, query.ErrNotFound)
	}
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: statusAccessResourceIDs(resStatuses),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Statuses: resStatuses}, nil
}

// RenameRequest names the status to rename and the name to give it.
type RenameRequest struct {
	// Name is the new name. It must not be empty.
	Name string `json:"name" msgpack:"name"`
	// Key selects the status to rename.
	Key status.Key `json:"key" msgpack:"key"`
}

// Rename renames the status. It requires update access on it.
func (s *Service) Rename(
	ctx context.Context,
	tx gorp.Tx,
	req RenameRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []resource.ID{status.ResourceID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
}

type DeleteRequest struct {
	// Keys are the keys of the statuses to delete.
	Keys []status.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: status.ResourceIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}
