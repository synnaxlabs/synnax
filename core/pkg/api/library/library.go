// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package library

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
)

// Service is the API for libraries.
type Service struct {
	access   *rbac.Service
	internal *library.Service
}

// NewService creates the library API service.
func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{internal: cfg.Service.Library, access: cfg.Service.RBAC}, nil
}

type (
	Key     = library.Key
	Library = library.Library
)

type CreateRequest struct {
	Libraries []Library `json:"libraries" msgpack:"libraries"`
}

type CreateResponse struct {
	Libraries []Library `json:"libraries" msgpack:"libraries"`
}

// Create creates or replaces the given libraries.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{{Type: ontology.ResourceTypeLibrary}},
	}); err != nil {
		return CreateResponse{}, err
	}
	if err := s.internal.NewWriter(tx).CreateMany(ctx, &req.Libraries); err != nil {
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
	Libraries []Library `json:"libraries" msgpack:"libraries"`
}

// Retrieve retrieves libraries by key or by a fuzzy search on their names.
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
		q = q.Where(library.MatchKeys(req.Keys...))
	}
	var libs []Library
	if err := q.Entries(&libs).Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: library.OntologyIDsFromLibraries(libs),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Libraries: libs}, nil
}

type RenameRequest struct {
	Name string `json:"name" msgpack:"name"`
	Key  Key    `json:"key"  msgpack:"key"`
}

// Rename renames a library.
func (s *Service) Rename(
	ctx context.Context,
	tx gorp.Tx,
	req RenameRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{library.OntologyID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
}

type DeleteRequest struct {
	Keys []Key `json:"keys" msgpack:"keys"`
}

// Delete deletes libraries that no task uses.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: library.OntologyIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}
