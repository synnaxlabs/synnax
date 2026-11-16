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
	apidirectory "github.com/synnaxlabs/synnax/pkg/api/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
)

// RetrieveDirectories returns the Arc directories that req selects and that the subject
// can retrieve.
func (s *Service) RetrieveDirectories(
	ctx context.Context,
	snap bind.Snapshot,
	req apidirectory.RetrieveRequest,
) (apidirectory.RetrieveResponse, error) {
	var res apidirectory.RetrieveResponse
	q, err := apidirectory.ApplyRetrieve(
		req,
		s.internal.NewDirectoryRetrieve().Entries(&res.Directories),
	)
	if err != nil {
		return apidirectory.RetrieveResponse{}, err
	}
	if err = q.Where(bind.Filter(snap, directory.Match)).Exec(ctx, snap); err != nil {
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

// DispatchDirectoryRequest carries a sequence of actions to apply to one directory.
type DispatchDirectoryRequest = actions.DispatchRequest[directory.Key, directory.Action]

// DispatchDirectory applies the action sequence to the directory in tx.
func (s *Service) DispatchDirectory(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchDirectoryRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		ApplyDirectory(ctx, req.Key, req.Actions...)
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
