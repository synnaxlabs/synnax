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
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
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
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.label")
	v.NotNil("label", c.Label)
	v.NotNil("ranger", c.Ranger)
	v.NotNil("status", c.Status)
	return v.Error()
}

type Service struct {
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

// Create creates the labels. It returns a validation error when a label with one of
// their keys exists.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
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
	tx gorp.Tx,
	req DispatchRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the labels that the top level of Where
// selects by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, label.ResourceID)
}

// Retrieve returns the labels that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.internal.NewRetrieve().Entries(&res.Labels))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err = q.Where(bind.Filter(snap, label.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type DeleteRequest struct {
	Keys []label.Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the labels with the given keys and removes them from every range and
// status that holds them. A key that matches no label is ignored.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}

// AddRequest adds labels to the range or status that ID names.
type AddRequest struct {
	ID      resource.ID `json:"id"      msgpack:"id"`
	Labels  []label.Key `json:"labels"  msgpack:"labels"`
	Replace bool        `json:"replace" msgpack:"replace"`
}

// Add adds the labels to the range or status that req.ID names, or replaces its labels
// when req.Replace is true. It returns a validation error when req.ID names a resource
// that cannot have labels.
func (s *Service) Add(
	ctx context.Context,
	tx gorp.Tx,
	req AddRequest,
) (struct{}, error) {
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

// Remove removes the labels from the range or status that req.ID names. It returns a
// validation error when req.ID names a resource that cannot have labels.
func (s *Service) Remove(
	ctx context.Context,
	tx gorp.Tx,
	req RemoveRequest,
) (struct{}, error) {
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
