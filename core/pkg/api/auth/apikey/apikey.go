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
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	svcauth "github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the API key API service.
type ServiceConfig struct {
	// Auth checks that the Core accepts API keys and ends the sessions they opened.
	//
	// [REQUIRED]
	Auth *svcauth.Service
	// APIKey stores API keys.
	//
	// [REQUIRED]
	APIKey *apikey.Service
	// Ontology resolves the resources that API keys belong to.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the API key API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Auth = override.Nil(c.Auth, other.Auth)
	c.APIKey = override.Nil(c.APIKey, other.APIKey)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.auth.apikey")
	v.NotNil("auth", c.Auth)
	v.NotNil("api_key", c.APIKey)
	v.NotNil("ontology", c.Ontology)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

// Service creates, lists, and deletes API keys.
type Service struct {
	auth     *svcauth.Service
	apiKey   *apikey.Service
	ontology *ontology.Ontology
	rbac     *rbac.Service
}

// NewService opens a new API key API service.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		auth:     cfg.Auth,
		apiKey:   cfg.APIKey,
		ontology: cfg.Ontology,
		rbac:     cfg.RBAC,
	}, nil
}

func apiKey(k apikey.APIKey) APIKey {
	return APIKey{Key: k.Key, Subject: k.Subject, Name: k.Name, Created: k.Created}
}

// CreateRequest creates an API key.
type CreateRequest struct {
	// Subject is the user or rack the key logs in as. Empty means the caller.
	Subject ontology.ID `json:"subject" msgpack:"subject"`
	// Name is a label that tells the subject's keys apart.
	Name string `json:"name" msgpack:"name"`
}

// CreateResponse is a new API key.
type CreateResponse struct {
	// Token is the API key to log in with. It is the only copy of its secret.
	Token string `json:"token" msgpack:"token"`
	// Key is the stored key.
	Key APIKey `json:"key" msgpack:"key"`
}

// Create creates an API key for the caller, or for another user or rack when the caller
// may create that subject. It returns a validation error when the subject is not a user
// or rack, and an error wrapping query.ErrNotFound when it does not exist.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.auth.Accepts(svcauth.MethodAPIKey); err != nil {
		return CreateResponse{}, err
	}
	caller := auth.GetSubject(ctx)
	subject := lo.Ternary(req.Subject.IsZero(), caller, req.Subject)
	if subject.Type != ontology.ResourceTypeUser &&
		subject.Type != ontology.ResourceTypeRack {
		return CreateResponse{}, errors.Wrapf(
			validate.ErrValidation,
			"an API key cannot log in as a %s",
			subject.Type,
		)
	}
	if err := auth.EnforceOthers(
		ctx,
		s.rbac,
		caller,
		access.ActionCreate,
		[]ontology.ID{subject},
	); err != nil {
		return CreateResponse{}, err
	}
	exists, err := s.ontology.NewRetrieve().WhereIDs(subject).Exists(ctx, tx)
	if err != nil {
		return CreateResponse{}, err
	}
	if !exists {
		return CreateResponse{}, errors.Wrapf(query.ErrNotFound, "%s", subject)
	}
	k, token, err := s.apiKey.NewWriter(tx).Create(ctx, subject, req.Name)
	if err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Token: token, Key: apiKey(k)}, nil
}

// RetrieveRequest selects API keys to list.
type RetrieveRequest struct {
	// Subjects limits the result to these subjects' keys. Empty means the caller.
	Subjects []ontology.ID `json:"subjects" msgpack:"subjects"`
}

// RetrieveResponse lists API keys without their hashes.
type RetrieveResponse struct {
	// Keys are the keys that matched.
	Keys []APIKey `json:"keys" msgpack:"keys"`
}

// Retrieve lists the caller's API keys, or the keys of the given subjects when the
// caller may retrieve those subjects.
func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	caller := auth.GetSubject(ctx)
	subjects := req.Subjects
	if len(subjects) == 0 {
		subjects = []ontology.ID{caller}
	}
	if err := auth.EnforceOthers(
		ctx,
		s.rbac,
		caller,
		access.ActionRetrieve,
		subjects,
	); err != nil {
		return RetrieveResponse{}, err
	}
	var keys []apikey.APIKey
	if err := s.apiKey.NewRetrieve().
		Where(apikey.MatchSubjects(subjects...)).
		Entries(&keys).
		Exec(ctx, nil); err != nil {
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

// Delete deletes the given API keys and ends the sessions they opened. A caller may
// delete its own keys; deleting another subject's needs permission to delete that
// subject. Unknown keys are ignored.
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
	subjects := lo.Uniq(lo.Map(keys, func(k apikey.APIKey, _ int) ontology.ID {
		return k.Subject
	}))
	if err := auth.EnforceOthers(
		ctx,
		s.rbac,
		auth.GetSubject(ctx),
		access.ActionDelete,
		subjects,
	); err != nil {
		return struct{}{}, err
	}
	stored := lo.Map(keys, func(k apikey.APIKey, _ int) apikey.Key { return k.Key })
	if err := s.apiKey.NewWriter(tx).Delete(ctx, stored...); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.auth.NewWriter(tx).RevokeCredential(ctx, stored...)
}
