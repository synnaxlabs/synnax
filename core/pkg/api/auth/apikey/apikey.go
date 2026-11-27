// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package apikey

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the API key API service.
type ServiceConfig struct {
	// Auth ends the sessions of deleted keys.
	//
	// [REQUIRED]
	Auth *auth.Service
	// APIKey stores API keys.
	//
	// [REQUIRED]
	APIKey *apikey.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Auth = override.Nil(c.Auth, other.Auth)
	c.APIKey = override.Nil(c.APIKey, other.APIKey)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.auth.apikey")
	v.NotNil("auth", c.Auth)
	v.NotNil("api_key", c.APIKey)
	return v.Error()
}

// Service creates, lists, rotates, and deletes API keys.
type Service struct {
	auth   *auth.Service
	apiKey *apikey.Service
}

// NewService opens a new API key API service.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{auth: cfg.Auth, apiKey: cfg.APIKey}, nil
}

func apiKey(k apikey.APIKey) APIKey {
	return APIKey{
		Key:     k.Key,
		Subject: k.Subject,
		Name:    k.Name,
		Created: k.Created,
		Expires: k.Expires,
	}
}

// CreateRequest creates an API key.
type CreateRequest struct {
	// Subject is the user or rack the key logs in as. Empty means the caller.
	Subject resource.ID `json:"subject" msgpack:"subject"`
	// Name is a label that tells the subject's keys apart.
	Name string `json:"name" msgpack:"name"`
	// Expires is when the key stops logging in. Zero or [telem.TimeStampMax] means
	// never.
	Expires telem.TimeStamp `json:"expires" msgpack:"expires"`
}

// CreateResponse is a new API key.
type CreateResponse struct {
	// Token is the API key to log in with. It is the only copy of its secret.
	Token string `json:"token" msgpack:"token"`
	// Key is the stored key.
	Key APIKey `json:"key" msgpack:"key"`
}

// Create creates an API key for the caller, or for another user or rack. It returns a
// validation error when the API key method is off, the subject is not a user or rack
// that exists, the expiry is not in the future, or the expiry is past the maximum key
// lifetime.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.auth.Accepts(auth.MethodAPIKey); err != nil {
		return CreateResponse{}, err
	}
	if err := s.apiKey.ValidateLifetime(req.Expires); err != nil {
		return CreateResponse{}, err
	}
	subject := lo.Ternary(req.Subject.IsZero(), apiauth.GetSubject(ctx), req.Subject)
	k, token, err := s.apiKey.NewWriter(tx).Create(ctx, subject, req.Name, req.Expires)
	if err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Token: token, Key: apiKey(k)}, nil
}

// Rotate replaces the API key that the caller's session logged in with by a new key for
// the same subject and name, and returns the new key with its token. The sessions of
// the old key move to the new key, so they stay open, but the old key cannot log in
// again. The new key lasts an hour short of the maximum key lifetime when one is set,
// and otherwise as long as the old key did. Rotate returns a validation
// error when the API key method is off or the session did not log in with an API key,
// and [auth.ErrInvalidCredentials] when the key has expired.
func (s *Service) Rotate(
	ctx context.Context,
	tx gorp.Tx,
	_ struct{},
) (CreateResponse, error) {
	if err := s.auth.Accepts(auth.MethodAPIKey); err != nil {
		return CreateResponse{}, err
	}
	old := apiauth.GetSession(ctx).Credential
	k, token, err := s.apiKey.NewWriter(tx).Rotate(ctx, old)
	if errors.Is(err, query.ErrNotFound) {
		return CreateResponse{}, errors.Wrap(
			validate.ErrValidation,
			"the session did not log in with an API key",
		)
	}
	if err != nil {
		return CreateResponse{}, err
	}
	if err = s.auth.NewWriter(tx).
		MoveCredential(ctx, old, k.Key, k.Created); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Token: token, Key: apiKey(k)}, nil
}

// RetrieveRequest selects API keys to list.
type RetrieveRequest struct {
	// Subjects limits the result to these subjects' keys. Empty means the caller.
	Subjects []resource.ID `json:"subjects" msgpack:"subjects"`
}

// RetrieveResponse lists API keys without their hashes.
type RetrieveResponse struct {
	// Keys are the keys that matched.
	Keys []APIKey `json:"keys" msgpack:"keys"`
}

// Holders implements bind.Held.
func (r RetrieveRequest) Holders() []resource.ID { return r.Subjects }

// Retrieve lists the API keys of the given subjects in snap, or the caller's keys when
// the request names no subject.
func (s *Service) Retrieve(
	ctx context.Context,
	snap gorp.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	subjects := req.Subjects
	if len(subjects) == 0 {
		subjects = []resource.ID{apiauth.GetSubject(ctx)}
	}
	var keys []apikey.APIKey
	if err := s.apiKey.NewRetrieve().
		Where(apikey.MatchSubjects(subjects...)).
		Entries(&keys).
		Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{
		Keys: lo.Map(keys, func(k apikey.APIKey, _ int) APIKey {
			return apiKey(k)
		}),
	}, nil
}

// DeleteRequest names API keys to delete.
type DeleteRequest struct {
	// Keys are the API keys to delete.
	Keys []apikey.Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the given API keys and ends the sessions they opened. Unknown keys are
// ignored.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	var keys []apikey.APIKey
	if err := s.apiKey.NewRetrieve().
		Where(apikey.MatchKeys(req.Keys...)).
		Entries(&keys).
		Exec(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return struct{}{}, err
	}
	stored := lo.Map(keys, func(k apikey.APIKey, _ int) apikey.Key { return k.Key })
	if err := s.apiKey.NewWriter(tx).Delete(ctx, stored...); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.auth.NewWriter(tx).RevokeCredential(ctx, stored...)
}
