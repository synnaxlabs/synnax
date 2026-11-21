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

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	taskconfig "github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
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
	// Task finds the rack of the task that a grant's command targets.
	//
	// [REQUIRED]
	Task *task.Service
	// Rack finds the devices and tasks on a rack.
	//
	// [REQUIRED]
	Rack *rack.Service
	// Certificate finds the certificates that a grant names.
	//
	// [REQUIRED]
	Certificate *certificate.Service
	// DB reads the records that a resolve without a grant checks.
	//
	// [REQUIRED]
	DB *gorp.DB
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
	c.Task = override.Nil(c.Task, other.Task)
	c.Rack = override.Nil(c.Rack, other.Rack)
	c.Certificate = override.Nil(c.Certificate, other.Certificate)
	c.DB = override.Nil(c.DB, other.DB)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.secret")
	v.NotNil("secret", c.Secret)
	v.NotNil("rbac", c.RBAC)
	v.NotNil("task", c.Task)
	v.NotNil("rack", c.Rack)
	v.NotNil("certificate", c.Certificate)
	v.NotNil("db", c.DB)
	return v.Error()
}

// Service is the API for secrets. No endpoint returns a value except Resolve, which
// only a rack may call.
type Service struct {
	access   *rbac.Service
	internal *secret.Service
	task     *task.Service
	racks    *rack.Service
	certs    *certificate.Service
	db       *gorp.DB
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
		task:     cfg.Task,
		racks:    cfg.Rack,
		certs:    cfg.Certificate,
		db:       cfg.DB,
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
	// Rack is the rack that reads the file of a file secret.
	Rack rack.Key `json:"rack" msgpack:"rack"`
	// Key selects the secret to replace. A zero key creates a new secret.
	Key Key `json:"key" msgpack:"key"`
}

func (in Input) secret() (Secret, error) {
	base := secret.Base{Key: in.Key, Name: in.Name}
	v := validate.New("secret")
	switch in.Source {
	case secret.SourceStored:
		v.Ternary("file_name", in.FileName != "", "must be empty for a stored secret")
		v.Ternary("rack", in.Rack != 0, "must be empty for a stored secret")
		if err := v.Error(); err != nil {
			return Secret{}, err
		}
		return Secret{Variant: secret.StoredSecret{Base: base}}, nil
	case secret.SourceFile:
		return Secret{Variant: secret.FileSecret{
			Base:     base,
			FileName: in.FileName,
			Rack:     in.Rack,
		}}, nil
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
// retrieve. A task config that uses a secret appears as its task. A secret with no
// readable uses gets empty uses, so the response tells it apart from a secret whose
// uses were not asked for.
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
		base.Uses = snap.Visible(uses)
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

// GrantCreateRequest names a subject, the command it may run once, and the secrets and
// certificates it reads to run it.
type GrantCreateRequest struct {
	// Subject is the subject that may run the command. It must be a rack.
	Subject resource.ID `json:"subject" msgpack:"subject"`
	// Secrets are the secrets the subject reads to run the command. The private key of
	// a stored certificate is a secret.
	Secrets []Key `json:"secrets" msgpack:"secrets"`
	// Certificates are the file certificates the subject reads to run the command.
	// Each must be a file of the subject rack.
	Certificates []certificate.Key `json:"certificates" msgpack:"certificates"`
	// Command is the command the subject runs. Its task must be on the subject rack.
	Command task.Command `json:"command" msgpack:"command"`
}

// GrantCreateResponse holds the key of a grant to send to the task of the command.
type GrantCreateResponse struct {
	Grant grant.Key `json:"grant" msgpack:"grant"`
}

// CreateGrant creates a grant that lets the subject run the command once with the
// secrets and certificates, within a minute. Only a user with create access on devices
// and retrieve access on each secret and certificate may call it. It returns
// query.ErrNotFound when the task of the command or a certificate does not exist, and a
// validation error when the task is not on the subject rack or a certificate is not a
// file of the subject rack.
func (s *Service) CreateGrant(
	ctx context.Context,
	tx gorp.Tx,
	req GrantCreateRequest,
) (GrantCreateResponse, error) {
	if err := s.enforceGrant(ctx, tx, req); err != nil {
		return GrantCreateResponse{}, err
	}
	if err := s.checkCertificates(ctx, tx, req); err != nil {
		return GrantCreateResponse{}, err
	}
	var t task.Task
	if err := s.task.NewRetrieve().
		Where(task.MatchKeys(req.Command.Task)).
		Entry(&t).
		Exec(ctx, tx); err != nil {
		return GrantCreateResponse{}, err
	}
	v := validate.New("grant")
	if v.Ternaryf(
		"command.task",
		t.Rack.ResourceID() != req.Subject,
		"%s is not on %s",
		t,
		req.Subject,
	) {
		return GrantCreateResponse{}, v.Error()
	}
	key, err := s.internal.Grant(ctx, tx, grant.Grant{
		Subject:      req.Subject,
		Secrets:      req.Secrets,
		Certificates: req.Certificates,
		Command:      req.Command,
	})
	if err != nil {
		return GrantCreateResponse{}, err
	}
	return GrantCreateResponse{Grant: key}, nil
}

// Vouched returns a predicate for bind.ServiceConfig.Vouched. It reports that a rack
// cannot add a use of a secret, or of a certificate with a private key, since the use
// would let the rack read it.
func Vouched(
	certs *certificate.Service,
) func(context.Context, gorp.Tx, resource.ID) (bool, error) {
	return func(ctx context.Context, tx gorp.Tx, target resource.ID) (bool, error) {
		if target.Type != resource.TypeCertificate {
			return target.Type == resource.TypeSecret, nil
		}
		key, err := resource.ParseUUIDKey(target)
		if err != nil {
			return false, err
		}
		return certs.HasPrivateKey(ctx, tx, key)
	}
}

var errGrantSubject = errors.Wrap(access.ErrDenied, "only a user can grant secrets")

func (s *Service) enforceGrant(
	ctx context.Context,
	tx gorp.Tx,
	req GrantCreateRequest,
) error {
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
		Objects: append(
			secret.ResourceIDs(req.Secrets),
			certificate.ResourceIDs(req.Certificates)...,
		),
	})
}

// checkCertificates returns a validation error when a certificate of req is not a file
// of the subject rack.
func (s *Service) checkCertificates(
	ctx context.Context,
	tx gorp.Tx,
	req GrantCreateRequest,
) error {
	if len(req.Certificates) == 0 {
		return nil
	}
	var certs []certificate.Certificate
	if err := s.certs.NewRetrieve().
		Where(certificate.MatchKeys(req.Certificates...)).
		Entries(&certs).
		Exec(ctx, tx); err != nil {
		return err
	}
	for _, c := range certs {
		file, ok := c.Variant.(certificate.FileCertificate)
		if !ok {
			return errors.Wrapf(
				validate.ErrValidation,
				"certificate %q is stored: name its private key in secrets",
				c.Base().Name,
			)
		}
		if file.Rack.ResourceID() != req.Subject {
			return errors.Wrapf(
				validate.ErrValidation,
				"certificate %q is a file of rack %d, not of %s",
				file.Name,
				file.Rack,
				req.Subject,
			)
		}
	}
	return nil
}

// ResolveRequest names a grant to spend and the task that runs its command, or, without
// a grant, the secrets to read.
type ResolveRequest struct {
	// Grant is the key of a grant for the calling rack.
	Grant grant.Key `json:"grant" msgpack:"grant"`
	// Task is the task that runs the command. It must be the task of the grant's
	// command, and empty without a grant.
	Task task.Key `json:"task" msgpack:"task"`
	// Secrets are the secrets to read without a grant. A device or task on the calling
	// rack must use each one. Empty with a grant.
	Secrets []Key `json:"secrets" msgpack:"secrets"`
}

func (r ResolveRequest) validate() error {
	v := validate.New("resolve")
	if r.Grant != uuid.Nil() {
		v.Ternary("task", r.Task == uuid.Nil(), "required with a grant")
		v.Ternary("secrets", len(r.Secrets) > 0, "must be empty with a grant")
	} else {
		v.Ternary("secrets", len(r.Secrets) == 0, "required without a grant")
		v.Ternary("task", r.Task != uuid.Nil(), "must be empty without a grant")
	}
	return v.Error()
}

// Value is the plain text value of a secret.
type Value struct {
	// Value is the plain text value.
	Value string `json:"value" msgpack:"value"`
	// Key is the key of the secret.
	Key Key `json:"key" msgpack:"key"`
}

// ResolveResponse holds the command of a grant, the secrets it names, and their values.
type ResolveResponse struct {
	// Command is the command to run, in place of the arguments of the command that
	// carried the grant.
	Command task.Command `json:"command" msgpack:"command"`
	// Secrets holds the keys of the secrets that the grant names, file secrets
	// included. Empty without a grant.
	Secrets []Key `json:"secrets" msgpack:"secrets"`
	// Certificates holds the keys of the file certificates that the grant names. Empty
	// without a grant.
	Certificates []certificate.Key `json:"certificates" msgpack:"certificates"`
	// Values holds one value for each secret, except a file secret of a calling rack
	// that is not the Core's, which the rack reads from its own secrets folder.
	Values []Value `json:"values" msgpack:"values"`
}

var errResolveSubject = errors.Wrap(access.ErrDenied, "only a rack can resolve secrets")

// Resolve returns the plain text values of secrets. Only a rack may call it. With a
// grant, it spends the grant and also returns its command: the grant must be to the
// rack for the task of the request. Without one, it returns the values of the secrets
// of the request, and an error that wraps access.ErrDenied when no device or task on
// the rack uses one. A wrong mix of grant, task, and secrets is a validation error.
func (s *Service) Resolve(
	ctx context.Context,
	req ResolveRequest,
) (ResolveResponse, error) {
	subject := auth.GetSubject(ctx)
	if subject.Type != resource.TypeRack {
		return ResolveResponse{}, errResolveSubject
	}
	if err := req.validate(); err != nil {
		return ResolveResponse{}, err
	}
	var (
		g      grant.Grant
		values map[Key][]byte
		err    error
	)
	if req.Grant == uuid.Nil() {
		values, err = s.resolveUsed(ctx, subject, req.Secrets)
	} else {
		g, values, err = s.internal.Resolve(ctx, subject, req.Task, req.Grant)
	}
	if err != nil {
		return ResolveResponse{}, err
	}
	res := ResolveResponse{
		Command:      g.Command,
		Secrets:      g.Secrets,
		Certificates: g.Certificates,
		Values:       make([]Value, 0, len(values)),
	}
	for k, v := range values {
		res.Values = append(res.Values, Value{Key: k, Value: string(v)})
	}
	return res, nil
}

// resolveUsed returns the values of the secrets with keys to reader, when a device or
// task on reader uses each one.
func (s *Service) resolveUsed(
	ctx context.Context,
	reader resource.ID,
	keys []Key,
) (values map[Key][]byte, err error) {
	rk, err := rack.KeyFromResourceID(reader)
	if err != nil {
		return nil, err
	}
	err = s.db.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		dependents, err := s.racks.Dependents(ctx, snap, rk)
		if err != nil {
			return err
		}
		// A file item references the rack that reads it, but is not on the rack.
		on := make(set.Set[resource.ID], len(dependents))
		for _, id := range dependents {
			if id.Type != resource.TypeSecret && id.Type != resource.TypeCertificate {
				on.Add(id)
			}
		}
		for _, key := range keys {
			used, err := s.usedOn(ctx, snap, on, key)
			if err != nil {
				return err
			}
			if !used {
				return errors.Wrapf(
					access.ErrDenied,
					"no device or task on %s uses secret %s",
					reader,
					key,
				)
			}
		}
		values, err = s.internal.RevealTo(ctx, snap, reader, keys...)
		return err
	})
	return values, err
}

// usedOn returns true when a record in on uses the secret with key, directly or through
// a certificate whose private key it holds.
func (s *Service) usedOn(
	ctx context.Context,
	snap gorp.Snapshot,
	on set.Set[resource.ID],
	key Key,
) (bool, error) {
	uses, err := s.internal.Uses(ctx, snap, key)
	if err != nil {
		return false, err
	}
	onRack := func(id resource.ID) bool { return on.Contains(taskOf(id)) }
	for _, id := range uses {
		if onRack(id) {
			return true, nil
		}
		if id.Type != resource.TypeCertificate {
			continue
		}
		cert, err := resource.ParseUUIDKey(id)
		if err != nil {
			return false, err
		}
		certUses, err := s.certs.Uses(ctx, snap, cert)
		if err != nil {
			return false, err
		}
		if lo.SomeBy(certUses, onRack) {
			return true, nil
		}
	}
	return false, nil
}

// taskOf returns the task of a config record, and any other ID as it is.
func taskOf(id resource.ID) resource.ID {
	if t, ok := taskconfig.TaskOf(
		gorp.Target{Table: string(id.Type), Key: id.Key},
	); ok {
		return t
	}
	return id
}
