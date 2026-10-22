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

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
)

// Service is the API for interface control documents.
type Service struct {
	access   *rbac.Service
	internal *icd.Service
}

// NewService creates the ICD API service.
func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{internal: cfg.Service.ICD, access: cfg.Service.RBAC}, nil
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
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{{Type: ontology.ResourceTypeICD}},
	}); err != nil {
		return CreateResponse{}, err
	}
	if err := s.internal.NewWriter(tx).CreateMany(ctx, &req.ICDs); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse(req), nil
}

type RetrieveRequest struct {
	SearchTerm string `json:"search_term" msgpack:"search_term"`
	Keys       []Key  `json:"keys"        msgpack:"keys"`
	Limit      int    `json:"limit"       msgpack:"limit"`
	Offset     int    `json:"offset"      msgpack:"offset"`
}

type RetrieveResponse struct {
	ICDs []ICD `json:"icds" msgpack:"icds"`
}

// Retrieve retrieves ICDs by key or by a fuzzy search on their names.
func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	q := s.internal.NewRetrieve()
	if req.SearchTerm != "" {
		q = q.Search(req.SearchTerm)
	}
	if req.Limit != 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset != 0 {
		q = q.Offset(req.Offset)
	}
	if len(req.Keys) != 0 {
		q = q.Where(icd.MatchKeys(req.Keys...))
	}
	var ics []ICD
	if err := q.Entries(&ics).Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: icd.OntologyIDsFromICDs(ics),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{ICDs: ics}, nil
}

type CheckRequest struct {
	ICD ICD `json:"icd" msgpack:"icd"`
}

type CheckResponse struct {
	Problems library.Problems `json:"problems" msgpack:"problems"`
}

// Check returns every problem in an ICD without storing it.
func (s *Service) Check(ctx context.Context, req CheckRequest) (CheckResponse, error) {
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: []ontology.ID{{Type: ontology.ResourceTypeICD}},
	}); err != nil {
		return CheckResponse{}, err
	}
	return CheckResponse{Problems: icd.Check(req.ICD)}, nil
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
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{icd.OntologyID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
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
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{icd.OntologyID(req.Key)},
	}); err != nil {
		return ImportResponse{}, err
	}
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
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: icd.OntologyIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}
