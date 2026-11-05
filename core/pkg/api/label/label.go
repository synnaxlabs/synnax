// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package label

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the label API service.
type ServiceConfig struct {
	// Label stores and retrieves labels.
	//
	// [REQUIRED]
	Label *label.Service
	// Ranger stores the labels of ranges.
	//
	// [REQUIRED]
	Ranger *ranger.Service
	// Status stores the labels of statuses.
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
	// DefaultServiceConfig is the default configuration for the label API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Label = override.Nil(c.Label, other.Label)
	c.Ranger = override.Nil(c.Ranger, other.Ranger)
	c.Status = override.Nil(c.Status, other.Status)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.label")
	v.NotNil("label", c.Label)
	v.NotNil("ranger", c.Ranger)
	v.NotNil("status", c.Status)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

type Service struct {
	access   *rbac.Service
	internal *label.Service
	ranger   *ranger.Service
	status   *status.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		internal: cfg.Label,
		ranger:   cfg.Ranger,
		status:   cfg.Status,
		access:   cfg.RBAC,
	}, nil
}

type Label = label.Label

// CreateRequest is a request to create a Label in the cluster.
type CreateRequest struct {
	// Labels are the labels to create.
	Labels []Label `json:"labels" msgpack:"labels"`
	// DispatchKey is stamped onto the create broadcasts so the sender skips its echo.
	DispatchKey string `json:"dispatch_key" msgpack:"dispatch_key"`
}

// CreateResponse is a response to a CreateRequest.
type CreateResponse struct {
	// Labels are the labels that were created.
	Labels []Label `json:"labels" msgpack:"labels"`
}

// Create creates the labels in the cluster.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []resource.ID{{Type: resource.TypeLabel}},
	}); err != nil {
		return CreateResponse{}, err
	}
	if err := s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		CreateMany(ctx, &req.Labels); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Labels: req.Labels}, nil
}

// DispatchRequest carries an action sequence to apply to a single label. DispatchKey is
// a client-generated identifier for the batch, echoed on the broadcast so the sender
// recognizes its own echo.
type DispatchRequest = actions.DispatchRequest[label.Key, label.Action]

// Dispatch applies the action sequence to the target label atomically. Subscribers to
// the label action signals receive the sequence after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	req DispatchRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []resource.ID{label.ResourceID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.Dispatch(ctx, req.Key, req.DispatchKey, req.Actions)
}

type RetrieveRequest struct {
	SearchTerm          string      `json:"search_term"            msgpack:"search_term"`
	Keys                []label.Key `json:"keys"                   msgpack:"keys"`
	Names               []string    `json:"names"                  msgpack:"names"`
	Limit               int         `json:"limit"                  msgpack:"limit"`
	Offset              int         `json:"offset"                 msgpack:"offset"`
	IgnoreNotFoundError bool        `json:"ignore_not_found_error" msgpack:"ignore_not_found_error"`
}

type RetrieveResponse struct {
	// Labels are the labels that were retrieved.
	Labels []Label `json:"labels" msgpack:"labels"`
}

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q := s.internal.NewRetrieve()
	if req.SearchTerm != "" {
		q = q.Search(req.SearchTerm)
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}
	if len(req.Keys) != 0 {
		q = q.Where(label.MatchKeys(req.Keys...))
	}
	if len(req.Names) != 0 {
		q = q.Where(label.MatchNames(req.Names...))
	}
	err := q.Entries(&res.Labels).Exec(ctx, nil)
	if req.IgnoreNotFoundError && err != nil {
		err = errors.Skip(err, query.ErrNotFound)
	}
	if err != nil {
		return RetrieveResponse{}, err
	}
	objects := make([]resource.ID, len(res.Labels))
	for i, l := range res.Labels {
		objects[i] = l.ResourceID()
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: objects,
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type DeleteRequest struct {
	Keys []label.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: label.ResourceIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}

// AddRequest adds labels to the range or status that ID names.
type AddRequest struct {
	ID      resource.ID `json:"id"      msgpack:"id"`
	Labels  []label.Key `json:"labels"  msgpack:"labels"`
	Replace bool        `json:"replace" msgpack:"replace"`
}

func (s *Service) Add(
	ctx context.Context,
	tx gorp.Tx,
	req AddRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: append(label.ResourceIDs(req.Labels), req.ID),
	}); err != nil {
		return struct{}{}, err
	}
	switch req.ID.Type {
	case resource.TypeRange:
		key, err := ranger.KeyFromResourceID(req.ID)
		if err != nil {
			return struct{}{}, err
		}
		w := s.ranger.NewWriter(tx)
		if req.Replace {
			return struct{}{}, w.ReplaceLabels(ctx, key, req.Labels)
		}
		return struct{}{}, w.AddLabels(ctx, key, req.Labels)
	case resource.TypeStatus:
		w := s.status.NewWriter(tx)
		if req.Replace {
			return struct{}{}, w.ReplaceLabels(ctx, req.ID.Key, req.Labels)
		}
		return struct{}{}, w.AddLabels(ctx, req.ID.Key, req.Labels)
	}
	return struct{}{}, unlabeled(req.ID)
}

// RemoveRequest removes labels from the range or status that ID names.
type RemoveRequest struct {
	ID     resource.ID `json:"id"     msgpack:"id"`
	Labels []label.Key `json:"labels" msgpack:"labels"`
}

func (s *Service) Remove(
	ctx context.Context,
	tx gorp.Tx,
	req RemoveRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: append(label.ResourceIDs(req.Labels), req.ID),
	}); err != nil {
		return struct{}{}, err
	}
	switch req.ID.Type {
	case resource.TypeRange:
		key, err := ranger.KeyFromResourceID(req.ID)
		if err != nil {
			return struct{}{}, err
		}
		return struct{}{}, s.ranger.NewWriter(tx).RemoveLabels(ctx, key, req.Labels)
	case resource.TypeStatus:
		return struct{}{}, s.status.NewWriter(tx).
			RemoveLabels(ctx, req.ID.Key, req.Labels)
	}
	return struct{}{}, unlabeled(req.ID)
}

func unlabeled(id resource.ID) error {
	return validate.PathedError(
		errors.Wrapf(validate.ErrValidation, "a %s cannot have labels", id.Type),
		"id",
	)
}
