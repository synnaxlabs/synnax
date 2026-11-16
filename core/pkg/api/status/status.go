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
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
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
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the status API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Status = override.Nil(c.Status, other.Status)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.status")
	v.NotNil("status", c.Status)
	return v.Error()
}

type Service struct {
	internal *status.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		internal: cfg.Status,
	}, nil
}

func decodeDetails[D any](details any) (D, error) {
	var decoded D
	b, err := json.Marshal(details)
	if err != nil {
		return decoded, errors.Wrap(validate.ErrValidation, err.Error())
	}
	if err = json.Unmarshal(b, &decoded); err != nil {
		var zero D
		return zero, errors.Wrap(validate.ErrValidation, err.Error())
	}
	return decoded, nil
}

// resolveDetails returns an error if the owner of s cannot decode its details, since
// such a status fails every read. A rack status takes its decoded details, so a status
// sent without them reads as not running in every client.
func resolveDetails(s *status.Status[any]) error {
	if s.Owner == nil {
		return nil
	}
	var err error
	switch s.Owner.Type {
	case resource.TypeDevice:
		_, err = decodeDetails[device.StatusDetails](s.Details)
	case resource.TypeTask:
		_, err = decodeDetails[task.StatusDetails](s.Details)
	case resource.TypeRack:
		var details rack.StatusDetails
		if details, err = decodeDetails[rack.StatusDetails](s.Details); err == nil {
			s.Details = details
		}
	case resource.TypeChannel:
		_, err = decodeDetails[struct{}](s.Details)
	}
	return err
}

// SetRequest is a request to set (create or update) statuses in the cluster.
type SetRequest struct {
	// Statuses are the statuses to set. A status with an owner goes to the status its
	// owner holds, whatever key it carries. An update keeps the stored owner.
	Statuses []status.Status[any] `json:"statuses" msgpack:"statuses"`
}

// SetResponse is a response to a SetRequest.
type SetResponse struct {
	// Statuses are the statuses as stored, with their resolved keys and labels.
	Statuses []status.Status[any] `json:"statuses" msgpack:"statuses"`
}

// Set creates or updates the statuses in order. It returns a validation error if a
// status carries details that its resolved owner cannot decode. On error, tx keeps the
// writes of earlier statuses.
func (s *Service) Set(
	ctx context.Context,
	tx gorp.Tx,
	req SetRequest,
) (SetResponse, error) {
	w := s.internal.NewWriter(tx)
	for i := range req.Statuses {
		st := &req.Statuses[i]
		if err := w.Resolve(ctx, st); err != nil {
			return SetResponse{}, err
		}
		if err := resolveDetails(st); err != nil {
			return SetResponse{}, validate.PathedError(
				err, "statuses", strconv.Itoa(i), "details",
			)
		}
		if err := w.Set(ctx, st); err != nil {
			return SetResponse{}, err
		}
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
	if err := s.internal.NewWriter(tx).Set(ctx, &st); err != nil {
		return SetByKeyOrNameResponse{}, err
	}
	return SetByKeyOrNameResponse{
		Key:             st.Key,
		MultipleMatches: len(matches) > 1,
	}, nil
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the statuses that the top level of Where
// selects by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, status.ResourceID)
}

// Retrieve returns the statuses that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	res := RetrieveResponse{Statuses: []status.Status[any]{}}
	q, err := ApplyRetrieve(req, s.internal.NewRetrieve[any]().Entries(&res.Statuses))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err = q.Where(bind.Filter(snap, status.Match[any])).Exec(ctx, snap); err != nil {
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

// Rename renames the status.
func (s *Service) Rename(
	ctx context.Context,
	tx gorp.Tx,
	req RenameRequest,
) (struct{}, error) {
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
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}
