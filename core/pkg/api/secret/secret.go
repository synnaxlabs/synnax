// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the secret API service.
type ServiceConfig struct {
	// Secret stores secrets and resolves their values.
	//
	// [REQUIRED]
	Secret *secret.Service
	// Grant stores the grants that let a rack read secrets that nothing on it uses.
	//
	// [REQUIRED]
	Grant *grant.Service
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the secret API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Secret = override.Nil(c.Secret, other.Secret)
	c.Grant = override.Nil(c.Grant, other.Grant)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.secret")
	v.NotNil("secret", c.Secret)
	v.NotNil("grant", c.Grant)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

// Service is the API for secrets. No endpoint returns a value except Resolve, which
// only a rack may call.
type Service struct {
	access   *rbac.Service
	internal *secret.Service
	grants   *grant.Service
}

// NewService opens a Service with the given configuration.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		access:   cfg.RBAC,
		internal: cfg.Secret,
		grants:   cfg.Grant,
	}, nil
}

type (
	// Key is the key of a secret.
	Key = secret.Key
	// Secret is a secret as the API returns it, never with its value.
	Secret = secret.Secret
)

// Input is a secret to create or replace.
type Input struct {
	// Source is where the value lives: stored or file.
	Source secret.Source `json:"source" msgpack:"source"`
	// Name is the name of the secret.
	Name string `json:"name" msgpack:"name"`
	// Value is the plain text value of a stored secret.
	Value string `json:"value" msgpack:"value"`
	// Path is the path of a file secret on the computer that uses it.
	Path string `json:"path" msgpack:"path"`
	// Key selects the secret to replace. A zero key creates a new secret.
	Key Key `json:"key" msgpack:"key"`
}

func (in Input) secret() (Secret, error) {
	base := secret.Base{Key: in.Key, Name: in.Name}
	v := validate.New("secret")
	switch in.Source {
	case secret.SourceStored:
		if v.Ternary("path", in.Path != "", "must be empty for a stored secret") {
			return Secret{}, v.Error()
		}
		return Secret{Variant: secret.StoredSecret{Base: base}}, nil
	case secret.SourceFile:
		return Secret{Variant: secret.FileSecret{Base: base, Path: in.Path}}, nil
	}
	v.Ternaryf("source", true, "must be stored or file, not %q", in.Source)
	return Secret{}, v.Error()
}

// CreateRequest holds the secrets to create or replace.
type CreateRequest struct {
	Secrets []Input `json:"secrets" msgpack:"secrets"`
}

// CreateResponse holds the created secrets, without their values.
type CreateResponse struct {
	Secrets []Secret `json:"secrets" msgpack:"secrets"`
}

// Create creates the secrets, or replaces those whose key exists. It requires create
// access on secrets, and update access on each secret it replaces.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	enforcer := s.access.NewEnforcer(tx)
	subject := auth.GetSubject(ctx)
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionCreate,
		Objects: []ontology.ID{{Type: ontology.ResourceTypeSecret}},
	}); err != nil {
		return CreateResponse{}, err
	}
	replaced, err := s.retrieveReplaced(ctx, tx, req.Secrets)
	if err != nil {
		return CreateResponse{}, err
	}
	if err = enforcer.Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionUpdate,
		Objects: secret.OntologyIDsFromSecrets(replaced),
	}); err != nil {
		return CreateResponse{}, err
	}
	w := s.internal.NewWriter(tx)
	res := CreateResponse{Secrets: make([]Secret, len(req.Secrets))}
	for i, in := range req.Secrets {
		sec, err := in.secret()
		if err != nil {
			return CreateResponse{}, err
		}
		if err = w.Create(ctx, &sec, []byte(in.Value)); err != nil {
			return CreateResponse{}, err
		}
		res.Secrets[i] = sec
	}
	return res, nil
}

// retrieveReplaced returns the existing secrets whose keys the inputs give. A key that
// no secret has creates a new secret.
func (s *Service) retrieveReplaced(
	ctx context.Context,
	tx gorp.Tx,
	inputs []Input,
) ([]Secret, error) {
	var keys []Key
	for _, in := range inputs {
		if in.Key != uuid.Nil() {
			keys = append(keys, in.Key)
		}
	}
	if len(keys) == 0 {
		return nil, nil
	}
	var replaced []Secret
	if err := s.internal.NewRetrieve().
		Where(secret.MatchKeys(keys...)).
		Entries(&replaced).
		Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
		return nil, err
	}
	return replaced, nil
}

// RetrieveRequest selects the secrets to retrieve.
type RetrieveRequest struct {
	// SearchTerm filters the secrets by name.
	SearchTerm string `json:"search_term" msgpack:"search_term"`
	// Keys selects secrets by key.
	Keys []Key `json:"keys" msgpack:"keys"`
	// Limit is the maximum number of secrets to return.
	Limit int `json:"limit" msgpack:"limit"`
	// Offset is the number of secrets to skip.
	Offset int `json:"offset" msgpack:"offset"`
}

// RetrieveResponse holds the retrieved secrets, without their values.
type RetrieveResponse struct {
	Secrets []Secret `json:"secrets" msgpack:"secrets"`
}

// Retrieve returns the secrets the request selects. It requires retrieve access on each
// of them.
func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	q := s.internal.NewRetrieve().Search(req.SearchTerm)
	if req.Limit != 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset != 0 {
		q = q.Offset(req.Offset)
	}
	if len(req.Keys) != 0 {
		q = q.Where(secret.MatchKeys(req.Keys...))
	}
	var secrets []Secret
	if err := q.Entries(&secrets).Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: secret.OntologyIDsFromSecrets(secrets),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Secrets: secrets}, nil
}

// RenameRequest names the secret to rename and its new name.
type RenameRequest struct {
	// Name is the new name. It must not be empty.
	Name string `json:"name" msgpack:"name"`
	// Key selects the secret to rename.
	Key Key `json:"key" msgpack:"key"`
}

// Rename renames the secret. It requires update access on it.
func (s *Service) Rename(
	ctx context.Context,
	tx gorp.Tx,
	req RenameRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{secret.OntologyID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
}

// DeleteRequest selects the secrets to delete.
type DeleteRequest struct {
	Keys []Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the secrets. It requires delete access on each of them, and refuses
// while anything uses one.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: secret.OntologyIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}

// GrantCreateRequest names a rack and the secrets it may read before any device on the
// rack uses them.
type GrantCreateRequest struct {
	// Secrets are the secrets the rack may read.
	Secrets []Key `json:"secrets" msgpack:"secrets"`
	// Rack is the rack that may read the secrets.
	Rack rack.Key `json:"rack" msgpack:"rack"`
}

// GrantCreateResponse holds the key of a grant to send with a command to the rack.
type GrantCreateResponse struct {
	Grant grant.Key `json:"grant" msgpack:"grant"`
}

// CreateGrant creates a grant that lets the rack read the secrets until the caller's
// session ends or the grant is deleted. Only a user with create access on devices may
// call it.
func (s *Service) CreateGrant(
	ctx context.Context,
	tx gorp.Tx,
	req GrantCreateRequest,
) (GrantCreateResponse, error) {
	if err := s.enforceGrant(ctx, tx); err != nil {
		return GrantCreateResponse{}, err
	}
	key, err := s.internal.Grant(
		ctx,
		tx,
		req.Rack.OntologyID(),
		auth.GetSessionKey(ctx),
		req.Secrets...,
	)
	if err != nil {
		return GrantCreateResponse{}, err
	}
	return GrantCreateResponse{Grant: key}, nil
}

// GrantDeleteRequest selects the grants to delete.
type GrantDeleteRequest struct {
	Grants []grant.Key `json:"grants" msgpack:"grants"`
}

// DeleteGrants deletes the grants. Unknown keys are ignored. Only a user with create
// access on devices may call it.
func (s *Service) DeleteGrants(
	ctx context.Context,
	tx gorp.Tx,
	req GrantDeleteRequest,
) (struct{}, error) {
	if err := s.enforceGrant(ctx, tx); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.grants.NewWriter(tx).Delete(ctx, req.Grants...)
}

var errGrantSubject = errors.Wrap(access.ErrDenied, "only a user can grant secrets")

func (s *Service) enforceGrant(ctx context.Context, tx gorp.Tx) error {
	subject := auth.GetSubject(ctx)
	if subject.Type != ontology.ResourceTypeUser {
		return errGrantSubject
	}
	return s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionCreate,
		Objects: []ontology.ID{{Type: ontology.ResourceTypeDevice}},
	})
}

// ResolveRequest selects the secrets to resolve.
type ResolveRequest struct {
	// Grant is the key of an optional grant for the calling rack. A zero key names no
	// grant.
	Grant grant.Key `json:"grant" msgpack:"grant"`
	// Keys selects the secrets to resolve.
	Keys []Key `json:"keys" msgpack:"keys"`
}

// Value is the plain text value of a secret.
type Value struct {
	// Value is the plain text value.
	Value string `json:"value" msgpack:"value"`
	// Key is the key of the secret.
	Key Key `json:"key" msgpack:"key"`
}

// ResolveResponse holds the values of the secrets, in the order of the request.
type ResolveResponse struct {
	Values []Value `json:"values" msgpack:"values"`
}

var errResolveSubject = errors.Wrap(access.ErrDenied, "only a rack can resolve secrets")

// Resolve returns the plain text values of the secrets. Only a rack may call it, and
// only for secrets that something on the rack uses or that the grant names.
func (s *Service) Resolve(
	ctx context.Context,
	req ResolveRequest,
) (ResolveResponse, error) {
	subject := auth.GetSubject(ctx)
	if subject.Type != ontology.ResourceTypeRack {
		return ResolveResponse{}, errResolveSubject
	}
	values, err := s.internal.Resolve(ctx, nil, subject, req.Grant, req.Keys...)
	if err != nil {
		return ResolveResponse{}, err
	}
	res := ResolveResponse{Values: make([]Value, len(req.Keys))}
	for i, k := range req.Keys {
		res.Values[i] = Value{Key: k, Value: string(values[k])}
	}
	return res, nil
}
