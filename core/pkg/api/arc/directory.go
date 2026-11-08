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

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	apidirectory "github.com/synnaxlabs/synnax/pkg/api/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// RetrieveDirectories retrieves Arc directories. The caller must hold retrieve access
// to each directory it returns.
func (s *Service) RetrieveDirectories(
	ctx context.Context,
	req apidirectory.RetrieveRequest,
) (apidirectory.RetrieveResponse, error) {
	var res apidirectory.RetrieveResponse
	q, err := req.Apply(s.internal.NewDirectoryRetrieve().Entries(&res.Directories))
	if err != nil {
		return apidirectory.RetrieveResponse{}, err
	}
	if err = q.Exec(ctx, nil); err != nil {
		return apidirectory.RetrieveResponse{}, err
	}
	if err = s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: lo.Map(
			res.Directories,
			func(d directory.Directory, _ int) resource.ID {
				return d.ResourceID()
			},
		),
	}); err != nil {
		return apidirectory.RetrieveResponse{}, err
	}
	return res, nil
}

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

// CreateDirectory creates a directory of Arcs.
func (s *Service) CreateDirectory(
	ctx context.Context,
	tx gorp.Tx,
	req CreateDirectoryRequest,
) (CreateDirectoryResponse, error) {
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

// RenameDirectory renames a directory.
func (s *Service) RenameDirectory(
	ctx context.Context,
	tx gorp.Tx,
	req RenameDirectoryRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).RenameDirectory(ctx, req.Key, req.Name)
}

// DeleteDirectoryRequest deletes the directories with Keys.
type DeleteDirectoryRequest struct {
	Keys []directory.Key `json:"keys" msgpack:"keys"`
}

// DeleteDirectory deletes empty directories.
func (s *Service) DeleteDirectory(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteDirectoryRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).DeleteDirectory(ctx, req.Keys...)
}

// MoveRequest moves the Arcs and directories with IDs from the directory From into the
// directory To.
type MoveRequest struct {
	From directory.Key `json:"from" msgpack:"from"`
	To   directory.Key `json:"to"   msgpack:"to"`
	IDs  []resource.ID `json:"ids"  msgpack:"ids"`
}

// Move moves Arcs and directories between directories in one transaction.
func (s *Service) Move(
	ctx context.Context,
	tx gorp.Tx,
	req MoveRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).Move(ctx, req.From, req.To, req.IDs...)
}
