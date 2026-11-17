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
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/cluster"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/version"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

type Credentials = auth.Credentials

// HeldTables returns the tables of the records that a user or rack holds: its API keys,
// password, and sessions. Pass them as bind.ServiceConfig.Held.
func HeldTables() []string {
	return []string{
		types.Name[apikey.APIKey](),
		types.Name[password.Password](),
		types.Name[auth.Session](),
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
	NodeKey node.Key `json:"node_key" msgpack:"node_key"`
	// NodeTime is the time of the node that the request was sent to.
	NodeTime telem.TimeStamp `json:"node_time" msgpack:"node_time"`
}

// ServiceConfig is the configuration for the auth API service.
type ServiceConfig struct {
	// DB opens the snapshots that access checks and user lookups read.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Auth logs subjects in and checks their session tokens.
	//
	// [REQUIRED]
	Auth *auth.Service
	// Password stores passwords and checks proofs of them.
	//
	// [REQUIRED]
	Password *password.Service
	// APIKey stores API keys and checks proofs of them.
	//
	// [REQUIRED]
	APIKey *apikey.Service
	// User stores and retrieves users.
	//
	// [REQUIRED]
	User *user.Service
	// RBAC checks update on the user whose password another caller sets.
	//
	// [REQUIRED]
	RBAC *rbac.Service
	// Identity names the Core in responses.
	//
	// [REQUIRED]
	Identity cluster.Identity
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the auth API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Auth = override.Nil(c.Auth, other.Auth)
	c.Password = override.Nil(c.Password, other.Password)
	c.APIKey = override.Nil(c.APIKey, other.APIKey)
	c.User = override.Nil(c.User, other.User)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.Identity = override.If(
		c.Identity,
		other.Identity,
		other.Identity != cluster.Identity{},
	)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.auth")
	v.NotNil("db", c.DB)
	v.NotNil("auth", c.Auth)
	v.NotNil("password", c.Password)
	v.NotNil("api_key", c.APIKey)
	v.NotNil("user", c.User)
	v.NotNil("rbac", c.RBAC)
	v.Ternary("identity.key", c.Identity.Key == uuid.Nil(), "must be set")
	v.NonZero("identity.host_key", c.Identity.HostKey)
	return v.Error()
}

// Service is the core authentication service for the Synnax API.
type Service struct {
	db       *gorp.DB
	auth     *auth.Service
	password *password.Service
	apiKey   *apikey.Service
	user     *user.Service
	access   *rbac.Service
	identity cluster.Identity
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:       cfg.DB,
		auth:     cfg.Auth,
		password: cfg.Password,
		apiKey:   cfg.APIKey,
		user:     cfg.User,
		access:   cfg.RBAC,
		identity: cfg.Identity,
	}, nil
}

type LoginResponse struct {
	// User is the user the token is associated with.
	User User `json:"user" msgpack:"user"`
	// Token is the session token to send with every later request.
	Token string `json:"token" msgpack:"token"`
	// ClusterInfo is the information about the cluster.
	ClusterInfo ClusterInfo `json:"cluster_info" msgpack:"cluster_info"`
}

type LoginRequest struct{ Credentials }

// Login opens a session for the user with the provided credentials. If successful,
// returns a response containing the session token along with the user's details. It
// is kept for clients built before CreateSession existed.
func (s *Service) Login(
	ctx context.Context,
	req LoginRequest,
) (LoginResponse, error) {
	start := telem.Now()
	res, err := s.CreateSession(
		ctx,
		SessionCreateRequest{Proof: proofOf(req.Credentials)},
	)
	if err != nil {
		return LoginResponse{}, err
	}
	return LoginResponse{
		User:        *res.User,
		Token:       res.Token,
		ClusterInfo: s.clusterInfo(start),
	}, nil
}

// clusterInfo stamps the node time at the midpoint since start.
func (s *Service) clusterInfo(start telem.TimeStamp) ClusterInfo {
	end := telem.Now()
	return ClusterInfo{
		ClusterKey:  s.identity.Key.String(),
		NodeKey:     s.identity.HostKey,
		NodeVersion: version.Get(),
		NodeTime:    start + (end-start)/2,
	}
}

type ChangePasswordRequest struct {
	Credentials
	NewPassword string `json:"new_password" msgpack:"new_password"`
}

// PrepareChangePassword checks the credentials of req and hashes its new password. It
// returns [auth.ErrInvalidCredentials] when the credentials match no password, and a
// validation error when the new password is empty.
func (s *Service) PrepareChangePassword(
	ctx context.Context,
	req ChangePasswordRequest,
) (password.Hash, error) {
	return s.password.HashProven(
		ctx,
		auth.PasswordProof{Credentials: req.Credentials},
		req.NewPassword,
	)
}

// ChangePassword stores the password that PrepareChangePassword hashed and ends every
// session the old password opened, the caller's included. It returns
// [auth.ErrInvalidCredentials] when the password changed after its proof.
func (s *Service) ChangePassword(
	ctx context.Context,
	tx gorp.Tx,
	_ ChangePasswordRequest,
	hash password.Hash,
) (struct{}, error) {
	identity, _ := hash.Proven()
	if err := s.password.NewWriter(tx).
		Apply(ctx, identity.Subject, password.NewSetHashAction(hash)); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.auth.NewWriter(tx).RevokeCredential(ctx, identity.Credential)
}

func proofOf(creds Credentials) auth.Proof {
	return auth.Proof{Variant: auth.PasswordProof{Credentials: creds}}
}
