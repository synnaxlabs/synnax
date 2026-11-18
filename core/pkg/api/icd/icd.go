// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the ICD API service.
type ServiceConfig struct {
	// ICD stores, checks, and retrieves interface control documents.
	//
	// [REQUIRED]
	ICD *icd.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the ICD API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.ICD = override.Nil(c.ICD, other.ICD)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.icd")
	v.NotNil("icd", c.ICD)
	return v.Error()
}

// Service is the API for interface control documents.
type Service struct {
	internal *icd.Service
}

// NewService creates the ICD API service.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{internal: cfg.ICD}, nil
}

type (
	Key = icd.Key
	ICD = icd.ICD
)

type CreateRequest struct {
	ICDs []ICD `json:"icds" msgpack:"icds"`
	// DispatchKey is stamped onto the create broadcasts so the sender skips its echo.
	DispatchKey string `json:"dispatch_key" msgpack:"dispatch_key"`
}

type CreateResponse struct {
	ICDs []ICD `json:"icds" msgpack:"icds"`
}

// Create creates the given ICDs in tx. It refuses an ICD whose key exists.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		CreateMany(ctx, &req.ICDs); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{ICDs: req.ICDs}, nil
}

// DispatchRequest carries an action sequence to apply to a single ICD. DispatchKey is a
// client-generated identifier for the batch, echoed on the broadcast so the sender
// recognizes its own echo.
type DispatchRequest = actions.DispatchRequest[icd.Key, icd.Action]

// Dispatch applies the action sequence to the target ICD in tx. Subscribers to the ICD
// action channel receive the sequence after tx commits.
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

// Named implements bind.Named. It returns the ICDs that the top level of Where selects
// by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, icd.ResourceID)
}

// Retrieve returns the ICDs that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.internal.NewRetrieve().Entries(&res.ICDs))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err = q.Where(bind.Filter(snap, icd.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type CheckRequest struct {
	ICD ICD `json:"icd" msgpack:"icd"`
}

type CheckResponse struct {
	Problems validate.Problems `json:"problems" msgpack:"problems"`
}

// Check returns every problem in an ICD without storing it. The subject needs retrieve
// on that ICD.
func (s *Service) Check(
	_ context.Context,
	snap bind.Snapshot,
	req CheckRequest,
) (CheckResponse, error) {
	if err := snap.Enforce(req.ICD.ResourceID()); err != nil {
		return CheckResponse{}, err
	}
	return CheckResponse{Problems: icd.Check(req.ICD)}, nil
}

type ImportRequest struct {
	Key    Key        `json:"key"    msgpack:"key"`
	Format icd.Format `json:"format" msgpack:"format"`
	Data   []byte     `json:"data"   msgpack:"data"`
	// DispatchKey is stamped onto the create broadcast so the sender skips its echo.
	DispatchKey string `json:"dispatch_key" msgpack:"dispatch_key"`
}

type ImportResponse struct {
	ICD ICD `json:"icd" msgpack:"icd"`
}

// Import replaces the messages of an ICD in tx with those parsed from a DBC file or a
// field table.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	req ImportRequest,
) (ImportResponse, error) {
	ic, err := s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Import(ctx, req.Key, req.Format, req.Data)
	if err != nil {
		return ImportResponse{}, err
	}
	return ImportResponse{ICD: ic}, nil
}

type DeleteRequest struct {
	Keys []Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the ICDs with the given keys in tx. A key that matches no ICD is
// ignored.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}
