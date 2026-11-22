// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package enumeration

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the enumeration API service.
type ServiceConfig struct {
	// Enumeration stores and retrieves enumerations.
	//
	// [REQUIRED]
	Enumeration *enumeration.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Enumeration = override.Nil(c.Enumeration, other.Enumeration)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.enumeration")
	v.NotNil("enumeration", c.Enumeration)
	return v.Error()
}

// Service is the API for enumerations.
type Service struct{ internal *enumeration.Service }

// NewService opens a Service with the given configuration.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{internal: cfg.Enumeration}, nil
}

// CreateRequest holds the enumerations to create.
type CreateRequest struct {
	// Enumerations are the enumerations to create.
	Enumerations []enumeration.Enumeration `json:"enumerations" msgpack:"enumerations"`
	// DispatchKey is stamped onto the create broadcasts so the sender skips its echo.
	DispatchKey string `json:"dispatch_key" msgpack:"dispatch_key"`
}

// CreateResponse holds the created enumerations.
type CreateResponse struct {
	// Enumerations are the created enumerations, each with its key.
	Enumerations []enumeration.Enumeration `json:"enumerations" msgpack:"enumerations"`
}

// Create creates the enumerations. It returns a validation error when an enumeration is
// not valid, two of its entries share a key or a value, or an enumeration with its key
// exists.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		CreateMany(ctx, req.Enumerations); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Enumerations: req.Enumerations}, nil
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the enumerations that the top level of Where
// selects by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, enumeration.ResourceID)
}

// Retrieve returns the enumerations that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.internal.NewRetrieve().Entries(&res.Enumerations))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err = q.Where(bind.Filter(snap, enumeration.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

// RetrieveBuiltinsResponse holds the built-in enumerations.
type RetrieveBuiltinsResponse struct {
	// Enumerations are the built-in enumerations with their shipped values.
	Enumerations []enumeration.Enumeration `json:"enumerations" msgpack:"enumerations"`
}

// RetrieveBuiltins returns the enumerations that the Core ships, with their shipped
// values. Any signed-in subject can read them.
func (s *Service) RetrieveBuiltins(
	context.Context,
	struct{},
) (RetrieveBuiltinsResponse, error) {
	return RetrieveBuiltinsResponse{Enumerations: enumeration.Builtins()}, nil
}

// RetrieveDependentsRequest names the enumeration whose users to list.
type RetrieveDependentsRequest struct {
	// Key is the key of the enumeration.
	Key enumeration.Key `json:"key" msgpack:"key"`
}

var _ bind.Named = RetrieveDependentsRequest{}

// Named implements bind.Named. It returns the enumeration that the request names.
func (r RetrieveDependentsRequest) Named() []resource.ID {
	return []resource.ID{enumeration.ResourceID(r.Key)}
}

// RetrieveDependentsResponse holds the records that use an enumeration.
type RetrieveDependentsResponse struct {
	// Dependents are the IDs of the records that use the enumeration.
	Dependents []resource.ID `json:"dependents" msgpack:"dependents"`
}

// RetrieveDependents returns the ID of each record that uses the enumeration and that
// the subject can retrieve. Each record that uses the enumeration refuses its delete.
func (s *Service) RetrieveDependents(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveDependentsRequest,
) (RetrieveDependentsResponse, error) {
	deps, err := s.internal.Dependents(ctx, snap, req.Key)
	if err != nil {
		return RetrieveDependentsResponse{}, err
	}
	return RetrieveDependentsResponse{Dependents: lo.Filter(
		deps,
		func(id resource.ID, _ int) bool { return snap.Allowed(id) },
	)}, nil
}

// DispatchRequest carries an action sequence to apply to one enumeration. DispatchKey
// is echoed on the broadcast so the sender recognizes its own echo.
type DispatchRequest = actions.DispatchRequest[enumeration.Key, enumeration.Action]

// Dispatch applies the actions to the enumeration atomically. Subscribers to the
// enumeration action signals receive them after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
}

// ReplaceRequest names the enumeration to replace and the one that replaces it.
type ReplaceRequest struct {
	// From is the key of the enumeration to delete.
	From enumeration.Key `json:"from" msgpack:"from"`
	// To is the key of the enumeration that each user of From moves to.
	To enumeration.Key `json:"to" msgpack:"to"`
}

var _ bind.Copy = ReplaceRequest{}

// Sources implements bind.Copy. The subject needs retrieve on To even when nothing uses
// From.
func (r ReplaceRequest) Sources() []resource.ID {
	return []resource.ID{enumeration.ResourceID(r.To)}
}

// Replace moves each record that uses the enumeration From to the enumeration To, then
// deletes From. It returns a validation error when From is built in or equals To, and a
// not found error when From or To does not exist.
func (s *Service) Replace(
	ctx context.Context,
	tx gorp.Tx,
	req ReplaceRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).Replace(ctx, req.From, req.To)
}

// DeleteRequest selects the enumerations to delete.
type DeleteRequest struct {
	// Keys are the keys of the enumerations to delete.
	Keys []enumeration.Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the enumerations. A key that matches no enumeration is ignored. It
// returns a validation error for a built-in enumeration or one that a record uses, and
// then deletes none of them.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}
