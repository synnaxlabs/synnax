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
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/library"
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
}

type CreateResponse struct {
	ICDs []ICD `json:"icds" msgpack:"icds"`
}

// Create creates or replaces the given ICDs. A replacement must carry the hash of the
// ICD as the caller read it, or Create returns an error wrapping library.ErrStale.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.internal.NewWriter(tx).CreateMany(ctx, &req.ICDs); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse(req), nil
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

var _ bind.Named = CheckRequest{}

// Named implements bind.Named. It returns the stored ICD that the request replaces, if
// any.
func (r CheckRequest) Named() []resource.ID {
	if r.ICD.Key == uuid.Nil() {
		return nil
	}
	return []resource.ID{r.ICD.ResourceID()}
}

type CheckResponse struct {
	Problems library.Problems `json:"problems" msgpack:"problems"`
}

// Check returns every problem that would stop an ICD from saving, without storing it.
func (s *Service) Check(
	ctx context.Context,
	snap bind.Snapshot,
	req CheckRequest,
) (CheckResponse, error) {
	problems, err := s.internal.Check(ctx, snap, req.ICD)
	if err != nil {
		return CheckResponse{}, err
	}
	return CheckResponse{Problems: problems}, nil
}

type RenameRequest struct {
	Name string `json:"name" msgpack:"name"`
	Key  Key    `json:"key"  msgpack:"key"`
}

// Rename renames an ICD.
func (s *Service) Rename(
	ctx context.Context,
	tx gorp.Tx,
	req RenameRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
}

type ImportRequest struct {
	Key    Key        `json:"key"    msgpack:"key"`
	Format icd.Format `json:"format" msgpack:"format"`
	Data   []byte     `json:"data"   msgpack:"data"`
}

type ImportResponse struct {
	ICD ICD `json:"icd" msgpack:"icd"`
}

// Import replaces the entries of an ICD with those parsed from a DBC file or a field
// table.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	req ImportRequest,
) (ImportResponse, error) {
	ic, err := s.internal.NewWriter(tx).Import(ctx, req.Key, req.Format, req.Data)
	if err != nil {
		return ImportResponse{}, err
	}
	return ImportResponse{ICD: ic}, nil
}

type DeleteRequest struct {
	Keys []Key `json:"keys" msgpack:"keys"`
}

// Delete deletes ICDs that no task uses.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}
