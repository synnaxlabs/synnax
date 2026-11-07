// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rbac

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/builtin"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/migrations"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// RolesConfig is the configuration for opening [Roles].
type RolesConfig struct {
	// Instrumentation is the Alamos instrumentation used for logging, tracing, and
	// metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// DB is the Gorp database used to persist roles and policies.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Signals propagates role and policy changes through the Synnax signals system so
	// subscribers see RBAC updates without polling.
	//
	// [OPTIONAL] - Defaults to nil.
	Signals *signals.Provider
}

var _ config.Config[RolesConfig] = RolesConfig{}

// Override implements [config.Config].
func (c RolesConfig) Override(other RolesConfig) RolesConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate implements [config.Config].
func (c RolesConfig) Validate() error {
	v := validate.New("rbac.roles")
	v.NotNil("db", c.DB)
	return v.Error()
}

// Roles holds the role and policy services and the keys of the built-in roles.
type Roles struct {
	Policy *policy.Service
	Role   *role.Service
	// Builtin holds the keys of the built-in roles.
	Builtin builtin.ProvisionResult
	closer  io.MultiCloser
}

// Close shuts down the role and policy services.
func (r *Roles) Close() error { return r.closer.Close() }

// OpenRoles opens the role and policy services, and provisions the built-in roles and
// their policies.
func OpenRoles(ctx context.Context, configs ...RolesConfig) (r *Roles, err error) {
	cfg, err := config.New(RolesConfig{}, configs...)
	if err != nil {
		return nil, err
	}
	r = &Roles{}
	cleanup, ok := service.NewOpener(ctx, &r.closer)
	defer func() { err = cleanup(err) }()
	if r.Role, err = role.OpenService(ctx, role.ServiceConfig{
		Instrumentation: cfg.Child("role"),
		DB:              cfg.DB,
		Signals:         cfg.Signals,
	}); !ok(err, r.Role) {
		return nil, err
	}
	// Policies reference roles, so the role table opens and migrates first.
	if r.Policy, err = policy.OpenService(ctx, policy.ServiceConfig{
		Instrumentation: cfg.Child("policy"),
		DB:              cfg.DB,
		Signals:         cfg.Signals,
		Role:            r.Role,
	}); !ok(err, r.Policy) {
		return nil, err
	}
	r.Builtin, err = builtin.Provision(ctx, cfg.DB, r.Policy, r.Role)
	return r, err
}

// ServiceConfig is the configuration for creating a [Service].
type ServiceConfig struct {
	// Instrumentation is the Alamos instrumentation used for logging, tracing, and
	// metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// DB is the Gorp database that holds the legacy permission migration namespace.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Roles holds the roles and policies that the service enforces.
	//
	// [REQUIRED]
	Roles *Roles
	// User is the user service that holds the roles of each user. It assigns the Owner
	// role to root users at startup.
	//
	// [REQUIRED]
	User *user.Service
	// Rack is the rack service that holds the roles of each rack.
	//
	// [REQUIRED]
	Rack *rack.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements [config.Config].
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Roles = override.Nil(c.Roles, other.Roles)
	c.User = override.Nil(c.User, other.User)
	c.Rack = override.Nil(c.Rack, other.Rack)
	return c
}

// Validate implements [config.Config].
func (c ServiceConfig) Validate() error {
	v := validate.New("rbac")
	v.NotNil("db", c.DB)
	v.NotNil("roles", c.Roles)
	v.NotNil("user", c.User)
	v.NotNil("rack", c.Rack)
	return v.Error()
}

// Service enforces access control on the roles that users and racks hold.
type Service struct {
	*Roles
	cfg ServiceConfig
}

// NewService creates a Service. It runs the legacy permission migrations and gives the
// Owner role to every root user.
func NewService(ctx context.Context, configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, configs...)
	if err != nil {
		return nil, err
	}
	s := &Service{Roles: cfg.Roles, cfg: cfg}
	if err = gorp.Migrate(ctx, gorp.MigrateConfig{
		Instrumentation: cfg.Instrumentation,
		DB:              cfg.DB,
		Namespace:       "RBAC",
		Migrations: migrations.New(migrations.Config{
			User:  cfg.User,
			Roles: s.Builtin,
		}),
	}); err != nil {
		return nil, err
	}
	if err = s.assignOwnerToRoots(ctx, s.Builtin.OwnerKey); err != nil {
		return nil, err
	}
	return s, nil
}

// assignOwnerToRoots assigns the role identified by ownerKey to every user with
// RootUser=true. The assignment is idempotent, so it is safe to run on every startup.
func (s *Service) assignOwnerToRoots(ctx context.Context, ownerKey role.Key) error {
	return s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		var roots []user.User
		if err := s.cfg.User.NewRetrieve().
			Where(user.MatchRootUser(true)).
			Entries(&roots).
			Exec(ctx, tx); err != nil {
			return errors.Wrap(err, "load root users")
		}
		if len(roots) == 0 {
			return nil
		}
		keys := lo.Map(roots, func(u user.User, _ int) user.Key { return u.Key })
		return errors.Wrap(
			s.cfg.User.NewWriter(tx).AssignRole(ctx, ownerKey, keys...),
			"assign Owner role to root users",
		)
	})
}

// RolesOf returns the keys of the roles that the subjects hold, as seen by tx. A
// subject is a user or a rack. RolesOf returns a validation error for a subject of any
// other type, and query.ErrNotFound when a subject does not exist.
func (s *Service) RolesOf(
	ctx context.Context,
	tx gorp.Tx,
	subjects ...resource.ID,
) ([]role.Key, error) {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	var roles []role.Key
	for _, subject := range subjects {
		held, err := s.rolesOf(ctx, tx, subject)
		if err != nil {
			return nil, err
		}
		roles = append(roles, held...)
	}
	return lo.Uniq(roles), nil
}

func (s *Service) rolesOf(
	ctx context.Context,
	tx gorp.Tx,
	subject resource.ID,
) ([]role.Key, error) {
	switch subject.Type {
	case resource.TypeUser:
		key, err := user.KeyFromResourceID(subject)
		if err != nil {
			return nil, err
		}
		var u user.User
		err = s.cfg.User.NewRetrieve().
			Where(user.MatchKeys(key)).
			Entry(&u).
			Exec(ctx, tx)
		return u.Roles, err
	case resource.TypeRack:
		key, err := rack.KeyFromResourceID(subject)
		if err != nil {
			return nil, err
		}
		var r rack.Rack
		err = s.cfg.Rack.NewRetrieve().
			Where(rack.MatchKeys(key)).
			Entry(&r).
			Exec(ctx, tx)
		return r.Roles, err
	default:
		return nil, errors.Wrapf(
			validate.ErrValidation,
			"%s cannot hold roles",
			subject,
		)
	}
}

// NewPolicyRetrieve opens a policy retrieve whose [policy.MatchSubjects] filters read
// the roles of each subject through [Service.RolesOf].
func (s *Service) NewPolicyRetrieve() policy.Retrieve {
	return s.Policy.NewRetrieve().ResolveRolesWith(s.RolesOf)
}

func (e *Enforcer) retrievePolicies(
	ctx context.Context,
	subject resource.ID,
) ([]policy.Policy, error) {
	roles, err := e.svc.RolesOf(ctx, e.tx, subject)
	// A subject that does not exist, such as a deleted user, holds no roles.
	if errors.Is(err, query.ErrNotFound) {
		return nil, nil
	}
	if err != nil || len(roles) == 0 {
		return nil, err
	}
	var policies []policy.Policy
	if err := e.svc.Policy.NewRetrieve().
		Where(policy.MatchRoles(roles...)).
		Entries(&policies).
		Exec(ctx, e.tx); err != nil {
		return nil, err
	}
	return policies, nil
}

type Enforcer struct {
	svc *Service
	tx  gorp.Tx
}

func (s *Service) NewEnforcer(tx gorp.Tx) *Enforcer {
	return &Enforcer{svc: s, tx: gorp.OverrideTx(s.cfg.DB, tx)}
}

// Enforce allows the request when the policies of the subject's roles cover every
// object for the action, and returns an error wrapping access.ErrDenied otherwise. A
// subject that does not exist holds no policies.
func (e *Enforcer) Enforce(ctx context.Context, req access.Request) error {
	v, err := e.retrievePolicies(ctx, req.Subject)
	if err != nil {
		return err
	}
	denied, ok := firstDenied(req, v)
	if !ok {
		return nil
	}
	return errors.Wrapf(
		access.ErrDenied,
		"%s cannot %s %s",
		req.Subject,
		req.Action,
		denied,
	)
}

// Allowed returns a function that reports whether the policies of the subject's roles
// cover an object for the action. It reads the policies once, when Allowed is called. A
// subject that does not exist holds no policies.
func (e *Enforcer) Allowed(
	ctx context.Context,
	subject resource.ID,
	action access.Action,
) (func(resource.ID) bool, error) {
	policies, err := e.retrievePolicies(ctx, subject)
	if err != nil {
		return nil, err
	}
	return func(obj resource.ID) bool { return covers(policies, action, obj) }, nil
}

// firstDenied returns the first object in the request that no policy covers, and
// whether there was one. A request is allowed only when every object is covered.
func firstDenied(req access.Request, policies []policy.Policy) (resource.ID, bool) {
	for _, obj := range req.Objects {
		if !covers(policies, req.Action, obj) {
			return obj, true
		}
	}
	return resource.ID{}, false
}

// covers returns whether a policy allows the action on obj, by its ID or by its type.
func covers(policies []policy.Policy, action access.Action, obj resource.ID) bool {
	for _, p := range policies {
		if !lo.Contains(p.Actions, action) {
			continue
		}
		for _, o := range p.Objects {
			if o.Type == obj.Type && (o.IsType() || o.Key == obj.Key) {
				return true
			}
		}
	}
	return false
}
