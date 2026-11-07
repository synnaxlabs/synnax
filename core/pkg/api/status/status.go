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
	"encoding/json/v2"
	"strconv"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
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
	ids := make([]resource.ID, len(statuses))
	for i, s := range statuses {
		ids[i] = s.ResourceID()
	}
	return ids
}

func labelAccessResourceIDs(statuses []status.Status[any]) []resource.ID {
	var ids []resource.ID
	for _, s := range statuses {
		ids = append(ids, label.ResourceIDs(s.Labels)...)
	}
	return ids
}

func decodeDetails[D any](details any) error {
	b, err := json.Marshal(details)
	if err != nil {
		return errors.Wrap(validate.ErrValidation, err.Error())
	}
	var decoded D
	if err = json.Unmarshal(b, &decoded); err != nil {
		return errors.Wrap(validate.ErrValidation, err.Error())
	}
	return nil
}

func validateDetails(s status.Status[any]) error {
	// A status whose details its owner cannot decode fails every read of that status.
	if s.Owner == nil {
		return nil
	}
	switch s.Owner.Type {
	case resource.TypeDevice:
		return decodeDetails[device.StatusDetails](s.Details)
	case resource.TypeTask:
		return decodeDetails[task.StatusDetails](s.Details)
	case resource.TypeRack, resource.TypeChannel:
		return decodeDetails[struct{}](s.Details)
	default:
		return nil
	}
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

// Set creates or updates statuses in the cluster. It returns a validation error and
// writes nothing if a status carries details that its resolved owner cannot decode.
func (s *Service) Set(
	ctx context.Context,
	tx gorp.Tx,
	req SetRequest,
) (SetResponse, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: append(
			statusAccessResourceIDs(req.Statuses),
			labelAccessResourceIDs(req.Statuses)...,
		),
	}); err != nil {
		return SetResponse{}, err
	}
	w := s.internal.NewWriter(tx)
	for i := range req.Statuses {
		if err := w.Resolve(ctx, &req.Statuses[i]); err != nil {
			return SetResponse{}, err
		}
		if err := validateDetails(req.Statuses[i]); err != nil {
			return SetResponse{}, validate.PathedError(
				err, "statuses", strconv.Itoa(i), "details",
			)
		}
	}
	if err := w.SetMany(ctx, &req.Statuses); err != nil {
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

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	res := RetrieveResponse{Statuses: []status.Status[any]{}}
	q, err := req.Apply(s.internal.NewRetrieve[any]().Entries(&res.Statuses))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	// A status holds only the keys of its labels, so a read needs no label access.
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: statusAccessResourceIDs(res.Statuses),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
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
