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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the secret API service.
type ServiceConfig struct {
	// Secret stores secrets and resolves their values.
	//
	// [REQUIRED]
	Secret *secret.Service
	// Certificate finds the private key secrets of the certificates a grant names.
	//
	// [REQUIRED]
	Certificate *certificate.Service
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
	c.Certificate = override.Nil(c.Certificate, other.Certificate)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.secret")
	v.NotNil("secret", c.Secret)
	v.NotNil("certificate", c.Certificate)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

// Service is the API for secrets. No endpoint returns a value except Resolve, which
// only a rack may call.
type Service struct {
	access       *rbac.Service
	internal     *secret.Service
	certificates *certificate.Service
}

// NewService opens a Service with the given configuration.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		access:       cfg.RBAC,
		internal:     cfg.Secret,
		certificates: cfg.Certificate,
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
		v.Ternary("path", in.Path != "", "must be empty for a stored secret")
		return Secret{Variant: secret.StoredSecret{Base: base}}, v.Error()
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
// access on secrets.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{{Type: ontology.ResourceTypeSecret}},
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

// GrantRequest names a rack and the items it may read before any device on the rack
// uses them.
type GrantRequest struct {
	// Secrets are the secrets the rack may read.
	Secrets []Key `json:"secrets" msgpack:"secrets"`
	// Certificates are certificates whose private key secrets the rack may read.
	Certificates []certificate.Key `json:"certificates" msgpack:"certificates"`
	// Rack is the rack that may read the secrets.
	Rack rack.Key `json:"rack" msgpack:"rack"`
}

// GrantResponse holds a grant to send with a command to the rack.
type GrantResponse struct {
	Grant string `json:"grant" msgpack:"grant"`
}

// Grant returns a grant that lets the rack read the secrets and the private keys of the
// certificates for a short time. It requires create access on devices.
func (s *Service) Grant(
	ctx context.Context,
	tx gorp.Tx,
	req GrantRequest,
) (GrantResponse, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{{Type: ontology.ResourceTypeDevice}},
	}); err != nil {
		return GrantResponse{}, err
	}
	keys := req.Secrets
	if len(req.Certificates) != 0 {
		var certs []certificate.Certificate
		if err := s.certificates.NewRetrieve().
			Where(certificate.MatchKeys(req.Certificates...)).
			Entries(&certs).
			Exec(ctx, tx); err != nil {
			return GrantResponse{}, err
		}
		for _, c := range certs {
			if stored, ok := c.Variant.(certificate.StoredCertificate); ok &&
				stored.PrivateKey != nil {
				keys = append(keys, *stored.PrivateKey)
			}
		}
	}
	grant, err := s.internal.Grant(ctx, tx, req.Rack.OntologyID(), keys...)
	return GrantResponse{Grant: grant}, err
}

// ResolveRequest selects the secrets to resolve.
type ResolveRequest struct {
	// Grant is an optional grant for the calling rack.
	Grant string `json:"grant" msgpack:"grant"`
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
