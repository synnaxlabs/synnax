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
	"uuid"

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
	if r.Builtin, err = builtin.Provision(ctx, cfg.DB, r.Policy, r.Role); err != nil {
		return nil, err
	}
	return r, nil
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
	// User is the user service that holds the role of each user.
	//
	// [REQUIRED]
	User *user.Service
	// Rack is the rack service that holds the role of each rack.
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

// Service enforces access control on the role that each user and rack holds.
type Service struct {
	*Roles
	cfg ServiceConfig
}

// NewService creates a Service. It runs the migrations that give each stored user one
// role.
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
			User:    cfg.User,
			Role:    s.Role,
			Policy:  s.Policy,
			Builtin: s.Builtin,
		}),
	}); err != nil {
		return nil, err
	}
	return s, nil
}

// RolesOf returns the keys of the roles that the subjects hold, as seen by snap. A
// subject is a user or a rack, and holds one role. RolesOf returns a validation error
// for a subject of any other type, and query.ErrNotFound when a subject does not exist.
func (s *Service) RolesOf(
	ctx context.Context,
	snap gorp.Snapshot,
	subjects ...resource.ID,
) ([]role.Key, error) {
	roles := make([]role.Key, 0, len(subjects))
	for _, subject := range subjects {
		r, err := s.roleOf(ctx, snap, subject)
		if err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	return lo.Uniq(roles), nil
}

func (s *Service) roleOf(
	ctx context.Context,
	snap gorp.Snapshot,
	subject resource.ID,
) (role.Key, error) {
	switch subject.Type {
	case resource.TypeUser:
		key, err := user.KeyFromResourceID(subject)
		if err != nil {
			return uuid.Nil(), err
		}
		var u user.User
		if err = s.cfg.User.NewRetrieve().
			Where(user.MatchKeys(key)).
			Entry(&u).
			Exec(ctx, snap); err != nil {
			return uuid.Nil(), err
		}
		return u.Role, nil
	case resource.TypeRack:
		key, err := rack.KeyFromResourceID(subject)
		if err != nil {
			return uuid.Nil(), err
		}
		var r rack.Rack
		if err = s.cfg.Rack.NewRetrieve().
			Where(rack.MatchKeys(key)).
			Entry(&r).
			Exec(ctx, snap); err != nil {
			return uuid.Nil(), err
		}
		return r.Role, nil
	default:
		return uuid.Nil(), errors.Wrapf(
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

// Policies returns the policies of the subject's roles. A subject that does not exist
// holds none.
func (e *Enforcer) Policies(
	ctx context.Context,
	subject resource.ID,
) ([]policy.Policy, error) {
	roles, err := e.svc.RolesOf(ctx, e.snap, subject)
	if errors.Is(err, query.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var policies []policy.Policy
	if err := e.svc.Policy.NewRetrieve().
		Where(policy.MatchRoles(roles...)).
		Entries(&policies).
		Exec(ctx, e.snap); err != nil {
		return nil, err
	}
	return policies, nil
}

// Enforcer decides access from the policies of a subject's roles. A subject can always
// retrieve its own record and the policies of its roles.
type Enforcer struct {
	svc  *Service
	snap gorp.Snapshot
}

func (s *Service) NewEnforcer(snap gorp.Snapshot) *Enforcer {
	return &Enforcer{svc: s, snap: snap}
}

// Enforce allows the request when the policies of the subject's roles cover every
// object for the action, and returns an error wrapping access.ErrDenied otherwise. A
// subject that does not exist holds no policies.
func (e *Enforcer) Enforce(ctx context.Context, req access.Request) error {
	v, err := e.Policies(ctx, req.Subject)
	if err != nil {
		return err
	}
	denied, ok := firstDenied(req, v)
	if !ok {
		return nil
	}
	return access.Denied(req.Subject, req.Action, denied)
}

// Allowed returns a function that reports whether the policies of the subject's roles
// cover an object for the action. It reads the policies once, when Allowed is called. A
// subject that does not exist holds no policies.
func (e *Enforcer) Allowed(
	ctx context.Context,
	subject resource.ID,
	action access.Action,
) (func(resource.ID) bool, error) {
	policies, err := e.Policies(ctx, subject)
	if err != nil {
		return nil, err
	}
	return Allows(subject, policies, action), nil
}

// Allows returns a function that reports whether policies, which the subject holds,
// cover an object for the action.
func Allows(
	subject resource.ID,
	policies []policy.Policy,
	action access.Action,
) func(resource.ID) bool {
	return func(obj resource.ID) bool { return covers(subject, policies, action, obj) }
}

// firstDenied returns the first object in the request that no policy covers, and
// whether there was one. A request is allowed only when every object is covered.
func firstDenied(req access.Request, policies []policy.Policy) (resource.ID, bool) {
	for _, obj := range req.Objects {
		if !covers(req.Subject, policies, req.Action, obj) {
			return obj, true
		}
	}
	return resource.ID{}, false
}

// covers returns whether a policy allows the action on obj, by its ID or by its type,
// or whether the subject retrieves itself or one of its policies.
func covers(
	subject resource.ID,
	policies []policy.Policy,
	action access.Action,
	obj resource.ID,
) bool {
	if action == access.ActionRetrieve && obj == subject {
		return true
	}
	for _, p := range policies {
		if action == access.ActionRetrieve && obj == p.ResourceID() {
			return true
		}
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
