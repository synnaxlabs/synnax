// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/auth/token"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/version"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

// Service is the core authentication service for the Synnax API.
type Service struct {
	db            *gorp.DB
	authenticator auth.Authenticator
	token         *token.Service
	user          *user.Service
	cluster       cluster.Cluster
}

func NewService(cfg config.LayerConfig) *Service {
	return &Service{
		db:            cfg.Distribution.DB,
		authenticator: cfg.Service.Auth,
		token:         cfg.Service.Token,
		user:          cfg.Service.User,
		cluster:       cfg.Distribution.Cluster,
	}
}

// ClusterInfo is general information about the cluster and node that the request was
// sent to.
type ClusterInfo struct {
	// ClusterKey is the key of the cluster.
	ClusterKey string `json:"cluster_key" msgpack:"cluster_key"`
	// NodeVersion is the current version of the Synnax Core being used.
	NodeVersion string `json:"node_version" msgpack:"node_version"`
	// NodeKey is the key of the node in the cluster that the request was sent to.
	NodeKey cluster.NodeKey `json:"node_key" msgpack:"node_key"`
	// NodeTime is the time of the node that the request was sent to.
	NodeTime telem.TimeStamp `json:"node_time" msgpack:"node_time"`
}

type LoginResponse struct {
	// Token is the JWT.
	Token string `json:"token" msgpack:"token"`
	// ClusterInfo is the information about the cluster.
	ClusterInfo ClusterInfo `json:"cluster_info" msgpack:"cluster_info"`
	// User is the user the token is associated with.
	User user.User `json:"user" msgpack:"user"`
}

type LoginRequest struct {
	auth.InsecureCredentials
}

// Login attempts to authenticate a user with the provided credentials. If successful,
// returns a response containing a valid JWT along with the user's details.
func (s *Service) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	startTime := telem.Now()
	if err := s.authenticator.Authenticate(ctx, req.InsecureCredentials); err != nil {
		return LoginResponse{}, err
	}
	var u user.User
	if err := s.user.NewRetrieve().WhereUsernames(req.Username).Entry(&u).Exec(ctx, nil); err != nil {
		return LoginResponse{}, err
	}
	tk, err := s.token.New(u.Key)
	endTime := telem.Now()
	midPoint := startTime + (endTime-startTime)/2
	return LoginResponse{
		User:  u,
		Token: tk,
		ClusterInfo: ClusterInfo{
			ClusterKey:  s.cluster.Key().String(),
			NodeKey:     s.cluster.HostKey(),
			NodeVersion: version.Get(),
			NodeTime:    midPoint,
		},
	}, err
}

type ChangePasswordRequest struct {
	auth.InsecureCredentials
	NewPassword password.Raw `json:"new_password" msgpack:"new_password" validate:"required"`
}

// ChangePassword changes the password for the user with the provided credentials.
func (s *Service) ChangePassword(ctx context.Context, req ChangePasswordRequest) (types.Nil, error) {
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.authenticator.NewWriter(tx).
			UpdatePassword(ctx, req.InsecureCredentials, req.NewPassword)
	})
}
