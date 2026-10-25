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
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/cluster"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/version"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type Credentials = auth.Credentials

// ClusterInfo is general information about the cluster and node that the request was
// sent to.
type ClusterInfo struct {
	// ClusterKey is the key of the cluster.
	ClusterKey string `json:"cluster_key" msgpack:"cluster_key"`
	// NodeVersion is the current version of the Synnax Core being used.
	NodeVersion string `json:"node_version" msgpack:"node_version"`
	// NodeKey is the key of the node in the cluster that the request was sent to.
	NodeKey node.Key `json:"node_key" msgpack:"node_key"`
	// NodeTime is the time of the node that the request was sent to.
	NodeTime telem.TimeStamp `json:"node_time" msgpack:"node_time"`
}

// ServiceConfig is the configuration for the auth API service.
type ServiceConfig struct {
	// Auth logs subjects in and checks their session tokens.
	//
	// [REQUIRED]
	Auth *auth.Service
	// Password stores passwords and checks proofs of them.
	//
	// [REQUIRED]
	Password *password.Service
	// User stores and retrieves users.
	//
	// [REQUIRED]
	User *user.Service
	// Cluster provides information about the cluster topology.
	//
	// [REQUIRED]
	Cluster cluster.Cluster
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the auth API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Auth = override.Nil(c.Auth, other.Auth)
	c.Password = override.Nil(c.Password, other.Password)
	c.User = override.Nil(c.User, other.User)
	c.Cluster = override.Nil(c.Cluster, other.Cluster)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.auth")
	v.NotNil("auth", c.Auth)
	v.NotNil("password", c.Password)
	v.NotNil("user", c.User)
	v.NotNil("cluster", c.Cluster)
	return v.Error()
}

// Service is the core authentication service for the Synnax API.
type Service struct {
	auth     *auth.Service
	password *password.Service
	user     *user.Service
	cluster  cluster.Cluster
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		auth:     cfg.Auth,
		password: cfg.Password,
		user:     cfg.User,
		cluster:  cfg.Cluster,
	}, nil
}

type LoginResponse struct {
	// User is the user the token is associated with.
	User user.User `json:"user" msgpack:"user"`
	// Token is the session token to send with every later request.
	Token string `json:"token" msgpack:"token"`
	// ClusterInfo is the information about the cluster.
	ClusterInfo ClusterInfo `json:"cluster_info" msgpack:"cluster_info"`
}

type LoginRequest struct{ Credentials }

// Login opens a session for the user with the provided credentials. If successful,
// returns a response containing the session token along with the user's details.
func (s *Service) Login(
	ctx context.Context,
	req LoginRequest,
) (LoginResponse, error) {
	startTime := telem.Now()
	session, tk, err := s.auth.Login(ctx, proofOf(req.Credentials))
	if err != nil {
		return LoginResponse{}, err
	}
	key, err := uuid.Parse(session.Subject.Key)
	if err != nil {
		return LoginResponse{}, err
	}
	var u user.User
	if err := s.user.NewRetrieve().
		Where(user.MatchKeys(key)).
		Entry(&u).
		Exec(ctx, nil); err != nil {
		return LoginResponse{}, err
	}
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
	}, nil
}

type ChangePasswordRequest struct {
	Credentials
	NewPassword string `json:"new_password" msgpack:"new_password"`
}

// ChangePassword changes the password for the user with the provided credentials and
// ends every other session that password opened.
func (s *Service) ChangePassword(
	ctx context.Context,
	tx gorp.Tx,
	req ChangePasswordRequest,
) (struct{}, error) {
	identity, err := s.password.Authenticate(ctx, proofOf(req.Credentials))
	if err != nil {
		return struct{}{}, err
	}
	if err = s.password.NewWriter(tx).
		ChangePassword(ctx, identity.Subject, req.NewPassword); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.auth.NewWriter(tx).RevokeCredential(
		ctx,
		[]auth.SessionKey{getSession(ctx)},
		identity.Credential,
	)
}

func proofOf(creds Credentials) auth.Proof {
	return auth.Proof{Variant: auth.PasswordProof{Credentials: creds}}
}
