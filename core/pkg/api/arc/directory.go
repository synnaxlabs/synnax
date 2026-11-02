// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
)

type (
	// CreateDirectoryRequest creates Directory in the directory Parent.
	CreateDirectoryRequest struct {
		Parent    directory.Key       `json:"parent"    msgpack:"parent"`
		Directory directory.Directory `json:"directory" msgpack:"directory"`
	}
	// CreateDirectoryResponse carries the created directory with its key.
	CreateDirectoryResponse struct {
		Directory directory.Directory `json:"directory" msgpack:"directory"`
	}
)

// CreateDirectory creates a directory of Arcs. The caller must hold create access to
// directories.
func (s *Service) CreateDirectory(
	ctx context.Context,
	tx gorp.Tx,
	req CreateDirectoryRequest,
) (CreateDirectoryResponse, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{{Type: ontology.ResourceTypeArcDirectory}},
	}); err != nil {
		return CreateDirectoryResponse{}, err
	}
	if err := s.internal.NewWriter(tx).
		CreateDirectory(ctx, &req.Directory, req.Parent); err != nil {
		return CreateDirectoryResponse{}, err
	}
	return CreateDirectoryResponse{Directory: req.Directory}, nil
}

// RenameDirectoryRequest gives the directory with Key the name Name.
type RenameDirectoryRequest struct {
	Key  directory.Key `json:"key"  msgpack:"key"`
	Name string        `json:"name" msgpack:"name"`
}

// RenameDirectory renames a directory. The caller must hold update access to it.
func (s *Service) RenameDirectory(
	ctx context.Context,
	tx gorp.Tx,
	req RenameDirectoryRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{directory.OntologyID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).RenameDirectory(ctx, req.Key, req.Name)
}

// DeleteDirectoryRequest deletes the directories with Keys.
type DeleteDirectoryRequest struct {
	Keys []directory.Key `json:"keys" msgpack:"keys"`
}

// DeleteDirectory deletes empty directories. The caller must hold delete access to
// them.
func (s *Service) DeleteDirectory(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteDirectoryRequest,
) (struct{}, error) {
	ids := make([]ontology.ID, len(req.Keys))
	for i, k := range req.Keys {
		ids[i] = directory.OntologyID(k)
	}
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: ids,
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).DeleteDirectory(ctx, req.Keys...)
}

// MoveRequest moves the Arcs and directories with IDs into the directory To.
type MoveRequest struct {
	To  directory.Key `json:"to"  msgpack:"to"`
	IDs []ontology.ID `json:"ids" msgpack:"ids"`
}

// Move moves Arcs and directories into a directory. The caller must hold update access
// to each moved entry and to the destination.
func (s *Service) Move(
	ctx context.Context,
	tx gorp.Tx,
	req MoveRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: append(req.IDs, directory.OntologyID(req.To)),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Move(ctx, req.To, req.IDs...)
}
