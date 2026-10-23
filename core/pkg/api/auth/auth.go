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

	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/cluster"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/version"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
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

// Service is the core authentication service for the Synnax API.
type Service struct {
	auth     *auth.Service
	password *password.Service
	apiKey   *apikey.Service
	ontology *ontology.Ontology
	user     *user.Service
	access   *rbac.Service
	cluster  cluster.Cluster
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		auth:     cfg.Service.Auth,
		password: cfg.Service.Password,
		apiKey:   cfg.Service.APIKey,
		ontology: cfg.Service.Ontology,
		user:     cfg.Service.User,
		access:   cfg.Service.RBAC,
		cluster:  cfg.Distribution.Cluster,
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
		ClusterKey:  s.cluster.Key().String(),
		NodeKey:     s.cluster.HostKey(),
		NodeVersion: version.Get(),
		NodeTime:    start + (end-start)/2,
	}
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
		[]auth.SessionKey{GetSession(ctx).Key},
		identity.Credential,
	)
}

func proofOf(creds Credentials) auth.Proof {
	return auth.Proof{Variant: auth.PasswordProof{Credentials: creds}}
}
