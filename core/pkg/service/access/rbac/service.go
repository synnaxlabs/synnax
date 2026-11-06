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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
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

// ServiceConfig is the configuration for opening a [Service].
type ServiceConfig struct {
	// Instrumentation is the Alamos instrumentation used for logging, tracing, and
	// metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// DB is the Gorp database used to persist roles, policies, and the legacy
	// permission migration namespace.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is the ontology used to define role and policy resources.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Signals propagates role and policy changes through the Synnax signals system so
	// subscribers see RBAC updates without polling.
	//
	// [OPTIONAL] - Defaults to nil.
	Signals *signals.Provider
	// User is the user service that holds the roles of each user. It assigns the Owner
	// role to root users at startup.
	//
	// [REQUIRED]
	User *user.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements [config.Config].
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.User = override.Nil(c.User, other.User)
	return c
}

// Validate implements [config.Config].
func (c ServiceConfig) Validate() error {
	v := validate.New("rbac")
	v.NotNil("db", c.DB)
	v.NotNil("ontology", c.Ontology)
	v.NotNil("user", c.User)
	return v.Error()
}

type Service struct {
	Policy *policy.Service
	Role   *role.Service
	// Builtin holds the keys of the built-in roles.
	Builtin builtin.ProvisionResult
	closer  io.MultiCloser
	cfg     ServiceConfig
}

// Close shuts down the RBAC service and its sub-services.
func (s *Service) Close() error { return s.closer.Close() }

// OpenService creates a new RBAC service with both Policy and Role sub-services. It
// also provisions built-in roles/policies and runs legacy permission migrations.
func OpenService(
	ctx context.Context,
	configs ...ServiceConfig,
) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, configs...)
	if err != nil {
		return nil, err
	}
	s = &Service{cfg: cfg}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.Role, err = role.OpenService(ctx, role.ServiceConfig{
		Instrumentation: cfg.Child("role"),
		DB:              cfg.DB,
		Ontology:        cfg.Ontology,
		Signals:         cfg.Signals,
	}); !ok(err, s.Role) {
		return nil, err
	}
	// Policies reference roles, so the role table opens and migrates first.
	if s.Policy, err = policy.OpenService(ctx, policy.ServiceConfig{
		Instrumentation: cfg.Child("policy"),
		DB:              cfg.DB,
		Signals:         cfg.Signals,
		Ontology:        cfg.Ontology,
		Role:            s.Role,
	}); !ok(err, s.Policy) {
		return nil, err
	}
	// Provision built-in roles and policies. This is idempotent and runs every startup
	// to ensure policy definitions stay up to date.
	s.Builtin, err = builtin.Provision(ctx, cfg.DB, s.Policy, s.Role)
	if !ok(err, nil) {
		return nil, err
	}
	// The migrations assign roles to users, so they run after provisioning creates the
	// built-in roles.
	if err = gorp.Migrate(ctx, gorp.MigrateConfig{
		Instrumentation: cfg.Instrumentation,
		DB:              cfg.DB,
		Namespace:       "RBAC",
		Migrations: migrations.New(migrations.Config{
			User:     cfg.User,
			Ontology: cfg.Ontology,
			Roles:    s.Builtin,
		}),
	}); !ok(err, nil) {
		return nil, err
	}
	// assignOwnerToRoots ensures every user with RootUser=true holds the Owner role.
	// Reconciliation of which users are flagged as root happens earlier in
	// user.OpenService; this step only wires the role assignment.
	if err = s.assignOwnerToRoots(ctx, s.Builtin.OwnerKey); !ok(err, nil) {
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
		// The rack service takes the Driver role from this service, so it opens later
		// and racks are read from their table.
		var r rack.Rack
		err = gorp.NewRetrieve[rack.Key, rack.Rack]().
			Where(gorp.MatchKeys[rack.Key, rack.Rack](key)).
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

// firstDenied returns the first object in the request that no policy covers, and
// whether there was one. A request is allowed only when every object is covered.
func firstDenied(req access.Request, policies []policy.Policy) (resource.ID, bool) {
	for _, requestedObj := range req.Objects {
		found := false
		for _, p := range policies {
			hasAction := lo.Contains(p.Actions, req.Action)
			if !hasAction {
				continue
			}
			for _, policyObj := range p.Objects {
				if policyObj.IsType() {
					if policyObj.Type == requestedObj.Type {
						found = true
						break
					}
				} else if policyObj.Type == requestedObj.Type &&
					policyObj.Key == requestedObj.Key {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return requestedObj, true
		}
	}
	return resource.ID{}, false
}
