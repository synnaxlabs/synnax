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

	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

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
	// Auth reports the login methods the Core accepts.
	//
	// [REQUIRED]
	Auth *auth.Service
	// OIDC lists where to log in through each provider.
	//
	// [REQUIRED]
	OIDC *oidc.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the auth API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Auth = override.Nil(c.Auth, other.Auth)
	c.OIDC = override.Nil(c.OIDC, other.OIDC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.auth")
	v.NotNil("auth", c.Auth)
	v.NotNil("oidc", c.OIDC)
	return v.Error()
}

// Service lists the login methods the Core accepts.
type Service struct {
	auth *auth.Service
	oidc *oidc.Service
}

// NewService opens a new auth API service.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{auth: cfg.Auth, oidc: cfg.OIDC}, nil
}

// OIDCLogin is where a client sends a person to log in through a provider.
type OIDCLogin struct {
	// Provider is the name of the provider.
	Provider string `json:"provider" msgpack:"provider"`
	// AuthorizationEndpoint is the provider's login address.
	AuthorizationEndpoint string `json:"authorization_endpoint" msgpack:"authorization_endpoint"`
	// ClientID is the Core's client ID at the provider.
	ClientID string `json:"client_id" msgpack:"client_id"`
}

// MethodsResponse lists the login methods the Core accepts.
type MethodsResponse struct {
	// Methods are the accepted login methods, sorted.
	Methods []auth.Method `json:"methods" msgpack:"methods"`
	// Providers are where to log in through each OpenID Connect provider. It is empty
	// when the Core does not accept the oidc method.
	Providers []OIDCLogin `json:"providers" msgpack:"providers"`
}

// Methods returns the login methods the Core accepts. It needs no session.
func (s *Service) Methods(ctx context.Context, _ struct{}) (MethodsResponse, error) {
	res := MethodsResponse{Methods: s.auth.Methods(), Providers: []OIDCLogin{}}
	if s.auth.Accepts(auth.MethodOIDC) != nil {
		return res, nil
	}
	for _, l := range s.oidc.Logins(ctx) {
		res.Providers = append(res.Providers, OIDCLogin{
			Provider:              l.Provider,
			AuthorizationEndpoint: l.AuthorizationEndpoint,
			ClientID:              l.ClientID,
		})
	}
	return res, nil
}
