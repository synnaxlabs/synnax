// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package video

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/video"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the video view API service.
type ServiceConfig struct {
	// Video stores and retrieves video views.
	//
	// [REQUIRED]
	Video *video.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the video view API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Video = override.Nil(c.Video, other.Video)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.video")
	v.NotNil("video", c.Video)
	return v.Error()
}

type Service struct {
	internal *video.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		internal: cfg.Video,
	}, nil
}

type (
	CreateRequest struct {
		Videos []video.Video `json:"videos" msgpack:"videos"`
		// DispatchKey is stamped onto the create broadcasts so the sender skips its
		// echo.
		DispatchKey string `json:"dispatch_key" msgpack:"dispatch_key"`
	}
	CreateResponse struct {
		Videos []video.Video `json:"videos" msgpack:"videos"`
	}
)

// Create creates the video views, and overwrites each one whose key exists.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		CreateMany(ctx, &req.Videos); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Videos: req.Videos}, nil
}

// DispatchRequest carries the actions to apply to one video view. DispatchKey
// identifies the batch of the sender, so the sender can skip its own broadcast.
type DispatchRequest = actions.DispatchRequest[video.Key, video.Action]

// Dispatch applies the actions to the video view in tx. Subscribers to the video
// action signals receive them after the transaction commits.
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

// Named implements bind.Named. It returns the video views that the top level of Where
// selects by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, video.ResourceID)
}

// Retrieve returns the video views that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.internal.NewRetrieve().Entries(&res.Videos))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err = q.Where(bind.Filter(snap, video.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type DeleteRequest struct {
	Keys []video.Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the video views with the given keys. A key that matches no view is
// ignored.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}

// SetGroupRequest moves video views into Group, or out of their group when Group is
// nil.
type SetGroupRequest struct {
	Group *group.Key  `json:"group" msgpack:"group"`
	Keys  []video.Key `json:"keys"  msgpack:"keys"`
}

// SetGroup moves the video views with the given keys into a group. A move into a group
// under another project moves the views into that project.
func (s *Service) SetGroup(
	ctx context.Context,
	tx gorp.Tx,
	req SetGroupRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).SetGroup(ctx, req.Group, req.Keys...)
}

// SetProjectRequest moves video views into Project.
type SetProjectRequest struct {
	Project project.Key `json:"project" msgpack:"project"`
	Keys    []video.Key `json:"keys"    msgpack:"keys"`
}

// SetProject moves the video views with the given keys out of their group and into a
// project.
func (s *Service) SetProject(
	ctx context.Context,
	tx gorp.Tx,
	req SetProjectRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).
		SetProject(ctx, req.Project, req.Keys...)
}
