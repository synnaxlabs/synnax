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

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
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
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.secret")
	v.NotNil("secret", c.Secret)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

// Service is the API for secrets. No endpoint returns a value except Resolve, which
// only a rack may call.
type Service struct {
	access   *rbac.Service
	internal *secret.Service
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
	// FileName is the name of a file secret's file in the secrets folder of the rack.
	FileName string `json:"file_name" msgpack:"file_name"`
	// Key selects the secret to replace. A zero key creates a new secret.
	Key Key `json:"key" msgpack:"key"`
}

func (in Input) secret() (Secret, error) {
	base := secret.Base{Key: in.Key, Name: in.Name}
	v := validate.New("secret")
	switch in.Source {
	case secret.SourceStored:
		if v.Ternary(
			"file_name",
			in.FileName != "",
			"must be empty for a stored secret",
		) {
			return Secret{}, v.Error()
		}
		return Secret{Variant: secret.StoredSecret{Base: base}}, nil
	case secret.SourceFile:
		return Secret{
			Variant: secret.FileSecret{Base: base, FileName: in.FileName},
		}, nil
	}
	v.Ternaryf("source", true, "must be stored or file, not %q", in.Source)
	return Secret{}, v.Error()
}

// SetRequest holds the secrets to create or replace.
type SetRequest struct {
	Secrets []Input `json:"secrets" msgpack:"secrets"`
}

// SetResponse holds the set secrets, without their values.
type SetResponse struct {
	Secrets []Secret `json:"secrets" msgpack:"secrets"`
}

// Set creates the secrets, or replaces those whose key exists.
func (s *Service) Set(
	ctx context.Context,
	tx gorp.Tx,
	req SetRequest,
) (SetResponse, error) {
	w := s.internal.NewWriter(tx)
	res := SetResponse{Secrets: make([]Secret, len(req.Secrets))}
	for i, in := range req.Secrets {
		sec, err := in.secret()
		if err != nil {
			return SetResponse{}, err
		}
		if err = w.Set(ctx, &sec, []byte(in.Value)); err != nil {
			return SetResponse{}, err
		}
		res.Secrets[i] = sec
	}
	return res, nil
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the secrets that the top level of Where
// selects by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, secret.ResourceID)
}

// Retrieve returns the secrets that the request selects and that the subject can
// retrieve, with their uses when the request asks for them.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.internal.NewRetrieve().Entries(&res.Secrets))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err = q.Where(bind.Filter(snap, secret.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	if req.IncludeUses {
		if err = s.fillUses(ctx, snap, res.Secrets); err != nil {
			return RetrieveResponse{}, err
		}
	}
	return res, nil
}

// fillUses sets the uses of each secret, leaving out the resources the subject cannot
// retrieve. A secret with no readable uses gets empty uses, so the response tells it
// apart from a secret whose uses were not asked for.
func (s *Service) fillUses(
	ctx context.Context,
	snap bind.Snapshot,
	secrets []Secret,
) error {
	for i := range secrets {
		base := secrets[i].Base()
		uses, err := s.internal.Uses(ctx, snap, base.Key)
		if err != nil {
			return err
		}
		base.Uses = lo.Filter(
			uses,
			func(id resource.ID, _ int) bool { return snap.Allowed(id) },
		)
		secrets[i].SetBase(base)
	}
	return nil
}

// DispatchRequest carries an action sequence to apply to a single secret. DispatchKey
// is a client-generated identifier for the batch, echoed on the broadcast so the sender
// recognizes its own echo.
type DispatchRequest = actions.DispatchRequest[secret.Key, secret.Action]

// Dispatch applies the action sequence to the target secret atomically. Subscribers to
// the secret action signals receive the sequence after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
}

// DeleteRequest selects the secrets to delete.
type DeleteRequest struct {
	Keys []Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the secrets. It refuses while anything uses one.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}

// GrantCreateRequest names a subject and the secrets it may read once.
type GrantCreateRequest struct {
	// Subject is the subject that may read the secrets. It must be a rack.
	Subject resource.ID `json:"subject" msgpack:"subject"`
	// Secrets are the secrets the subject may read.
	Secrets []Key `json:"secrets" msgpack:"secrets"`
}

// GrantCreateResponse holds the key of a grant to send with a command to the subject.
type GrantCreateResponse struct {
	Grant grant.Key `json:"grant" msgpack:"grant"`
}

// CreateGrant creates a grant that lets the subject read the secrets once, within a
// minute. Only a user with create access on devices and retrieve access on each secret
// may call it.
func (s *Service) CreateGrant(
	ctx context.Context,
	tx gorp.Tx,
	req GrantCreateRequest,
) (GrantCreateResponse, error) {
	if err := s.enforceGrant(ctx, tx, req.Secrets); err != nil {
		return GrantCreateResponse{}, err
	}
	key, err := s.internal.Grant(ctx, tx, req.Subject, req.Secrets...)
	if err != nil {
		return GrantCreateResponse{}, err
	}
	return GrantCreateResponse{Grant: key}, nil
}

var errGrantSubject = errors.Wrap(access.ErrDenied, "only a user can grant secrets")

func (s *Service) enforceGrant(ctx context.Context, tx gorp.Tx, keys []Key) error {
	subject := auth.GetSubject(ctx)
	if subject.Type != resource.TypeUser {
		return errGrantSubject
	}
	enforcer := s.access.NewEnforcer(tx)
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionCreate,
		Objects: []resource.ID{{Type: resource.TypeDevice}},
	}); err != nil {
		return err
	}
	return enforcer.Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionRetrieve,
		Objects: secret.ResourceIDs(keys),
	})
}

// ResolveRequest selects the secrets to resolve.
type ResolveRequest struct {
	// Grant is the key of a grant for the calling rack. The request reads only the
	// secrets that the grant names, and spends the grant.
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
// only for secrets that the grant names.
func (s *Service) Resolve(
	ctx context.Context,
	req ResolveRequest,
) (ResolveResponse, error) {
	subject := auth.GetSubject(ctx)
	if subject.Type != resource.TypeRack {
		return ResolveResponse{}, errResolveSubject
	}
	values, err := s.internal.Resolve(ctx, subject, req.Grant, req.Keys...)
	if err != nil {
		return ResolveResponse{}, err
	}
	res := ResolveResponse{Values: make([]Value, len(req.Keys))}
	for i, k := range req.Keys {
		res.Values[i] = Value{Key: k, Value: string(values[k])}
	}
	return res, nil
}
