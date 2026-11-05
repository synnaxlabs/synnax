// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package whip lets encoders publish to the WHIP read tasks of a Core. An encoder
// proves itself with an API key, and the key's subject must be allowed to write to the
// channel of the task.
package whip

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/whip"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the WHIP API service.
type ServiceConfig struct {
	// Factory runs the WHIP sessions of tasks.
	//
	// [REQUIRED]
	Factory *whip.Factory
	// APIKey checks the API keys of encoders.
	//
	// [REQUIRED]
	APIKey *apikey.Service
	// RBAC checks that a key may write to the channel of a task.
	//
	// [REQUIRED]
	RBAC *rbac.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the WHIP API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Factory = override.Nil(c.Factory, other.Factory)
	c.APIKey = override.Nil(c.APIKey, other.APIKey)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.whip")
	v.NotNil("factory", c.Factory)
	v.NotNil("api_key", c.APIKey)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

// Service publishes encoders to WHIP read tasks.
type Service struct {
	factory *whip.Factory
	apiKey  *apikey.Service
	access  *rbac.Service
}

// NewService validates the configuration and returns a Service.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{factory: cfg.Factory, apiKey: cfg.APIKey, access: cfg.RBAC}, nil
}

type (
	// PublishRequest asks to publish an encoder to a task.
	PublishRequest struct {
		// Task is the key of the WHIP read task.
		Task uuid.UUID
		// Key is the API key that the encoder sends.
		Key string
		// Offer is the SDP offer of the encoder.
		Offer string
	}
	// PublishResponse holds the session that a publish starts.
	PublishResponse struct {
		// Session is the key of the new session.
		Session uuid.UUID
		// Answer is the SDP answer to the offer.
		Answer string
	}
	// UnpublishRequest asks to end a session of a task.
	UnpublishRequest struct {
		// Task is the key of the WHIP read task.
		Task uuid.UUID
		// Session is the key of the session to end.
		Session uuid.UUID
		// Key is the API key that the encoder sends.
		Key string
	}
)

// Publish starts a session for an encoder and returns the SDP answer to its offer. It
// returns an error that matches auth.ErrInvalidCredentials for an unknown key,
// auth.ErrAccessDenied for a key that may not write to the channel of the task,
// whip.ErrNotRunning for a task that does not run on this Core, and
// validate.ErrValidation for an invalid offer.
func (s *Service) Publish(
	ctx context.Context,
	req PublishRequest,
) (PublishResponse, error) {
	if err := s.authorize(ctx, req.Key, req.Task); err != nil {
		return PublishResponse{}, err
	}
	answer, session, err := s.factory.Publish(ctx, req.Task, req.Offer)
	return PublishResponse{Session: session, Answer: answer}, err
}

// Unpublish ends a session of a task. It returns the key and task errors of Publish,
// and an error that matches query.ErrNotFound when the task has no such session.
func (s *Service) Unpublish(
	ctx context.Context,
	req UnpublishRequest,
) (struct{}, error) {
	if err := s.authorize(ctx, req.Key, req.Task); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.factory.Unpublish(req.Task, req.Session)
}

func (s *Service) authorize(ctx context.Context, key string, task uuid.UUID) error {
	identity, err := s.apiKey.Authenticate(ctx, auth.Proof{
		Variant: auth.APIKeyProof{APIKeyCredentials: auth.APIKeyCredentials{Key: key}},
	})
	if err != nil {
		return err
	}
	ch, err := s.factory.Channel(task)
	if err != nil {
		return err
	}
	return s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: identity.Subject,
		Action:  access.ActionCreate,
		Objects: []ontology.ID{channel.OntologyID(ch)},
	})
}
