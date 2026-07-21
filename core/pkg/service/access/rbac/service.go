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
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/builtin"
	v0 "github.com/synnaxlabs/synnax/pkg/service/access/rbac/migrations/v0"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/override"
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
	// Ontology is the ontology used to define role and policy resources and the
	// relationships (role -> subject, role -> policy) that drive enforcement.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Signals propagates role and policy changes through the Synnax signals system so
	// subscribers see RBAC updates without polling.
	//
	// [OPTIONAL] - Defaults to nil.
	Signals *signals.Provider
	// Group is the top-level group service used to create the built-in "Roles" and
	// "Policies" groups that own the corresponding resources in the ontology.
	//
	// [REQUIRED]
	Group *group.Service
	// Search is the search index used by the role and policy sub-services for name
	// lookups.
	//
	// [REQUIRED]
	Search *search.Index
	// User is the user service used to load existing root users so the Owner role can
	// be assigned to them during startup.
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
	c.Group = override.Nil(c.Group, other.Group)
	c.Search = override.Nil(c.Search, other.Search)
	c.User = override.Nil(c.User, other.User)
	return c
}

// Validate implements [config.Config].
func (c ServiceConfig) Validate() error {
	v := validate.New("rbac")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	validate.NotNil(v, "search", c.Search)
	validate.NotNil(v, "user", c.User)
	return v.Error()
}

type Service struct {
	Policy *policy.Service
	Role   *role.Service
	closer io.MultiCloser
	cfg    ServiceConfig
}

// Close shuts down the RBAC service and its sub-services.
func (s *Service) Close() error { return s.closer.Close() }

// RetrievePoliciesForSubject retrieves all policies that apply to the given subject.
// This includes all policies from roles assigned to the subject via ontology
// relationships.
func (s *Service) RetrievePoliciesForSubject(
	ctx context.Context,
	subject ontology.ID,
	tx gorp.Tx,
) ([]policy.Policy, error) {
	return s.NewEnforcer(tx).retrievePolicies(ctx, subject)
}

// OpenService creates a new RBAC service with both Policy and Role sub-services. It
// also provisions built-in roles/policies and runs legacy permission migrations.
func OpenService(ctx context.Context, configs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, configs...)
	if err != nil {
		return nil, err
	}
	s = &Service{cfg: cfg}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.Policy, err = policy.OpenService(ctx, policy.ServiceConfig{
		Instrumentation: cfg.Child("policy"),
		DB:              cfg.DB,
		Signals:         cfg.Signals,
		Ontology:        cfg.Ontology,
		Search:          cfg.Search,
	}); !ok(err, s.Policy) {
		return nil, err
	}
	if s.Role, err = role.OpenService(ctx, role.ServiceConfig{
		Instrumentation: cfg.Child("role"),
		DB:              cfg.DB,
		Ontology:        cfg.Ontology,
		Signals:         cfg.Signals,
		Group:           cfg.Group,
		Search:          cfg.Search,
	}); !ok(err, s.Role) {
		return nil, err
	}
	// Provision built-in roles and policies. This is idempotent and runs every startup
	// to ensure policy definitions stay up to date.
	builtinRoles, err := builtin.Provision(ctx, cfg.DB, s.Policy, s.Role)
	if !ok(err, nil) {
		return nil, err
	}
	// Phase 2 migration assigns users to roles based on the legacy mapping extracted by
	// Phase 1 in the policy package. This runs after provisioning so the built-in roles
	// exist in the ontology.
	if err = gorp.Migrate(ctx, gorp.MigrateConfig{
		Instrumentation: cfg.Instrumentation,
		DB:              cfg.DB,
		Namespace:       "RBAC",
		Migrations: []migrate.Migration{v0.NewMigration(v0.MigrationConfig{
			User:     cfg.User,
			Ontology: cfg.Ontology,
			Role:     s.Role,
			Roles:    builtinRoles,
		})},
	}); !ok(err, nil) {
		return nil, err
	}
	// assignOwnerToRoots ensures every user with RootUser=true holds the Owner role.
	// Reconciliation of which users are flagged as root happens earlier in
	// user.OpenService; this step only wires the role assignment.
	if err = s.assignOwnerToRoots(ctx, builtinRoles.OwnerKey); !ok(err, nil) {
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
		w := s.Role.NewWriter(tx, false)
		for _, r := range roots {
			if err := w.AssignRole(
				ctx,
				user.OntologyID(r.Key),
				ownerKey,
			); err != nil {
				return errors.Wrapf(
					err,
					"assign Owner role to root user %q",
					r.Username,
				)
			}
		}
		return nil
	})
}

func (e *Enforcer) retrievePolicies(
	ctx context.Context,
	subject ontology.ID,
) ([]policy.Policy, error) {
	keys, err := e.policy.ResolveSubjects(ctx, e.tx, subject)
	if err != nil {
		return nil, err
	}
	var policies []policy.Policy
	if err := e.policy.NewRetrieve().
		Where(policy.MatchKeys(keys...)).
		Entries(&policies).
		Exec(ctx, e.tx); err != nil {
		return nil, err
	}
	return policies, nil
}

type Enforcer struct {
	policy *policy.Service
	role   *role.Service
	cfg    ServiceConfig
	tx     gorp.Tx
}

func (s *Service) NewEnforcer(tx gorp.Tx) *Enforcer {
	return &Enforcer{
		role:   s.Role,
		policy: s.Policy,
		cfg:    s.cfg,
		tx:     gorp.OverrideTx(s.cfg.DB, tx),
	}
}

// Enforce checks both direct user policies and policies from all roles assigned to the
// user.
func (e *Enforcer) Enforce(ctx context.Context, req access.Request) error {
	v, err := e.retrievePolicies(ctx, req.Subject)
	if err != nil {
		return err
	}
	if allowRequest(req, v) {
		return nil
	}
	return access.ErrDenied
}

func allowRequest(req access.Request, policies []policy.Policy) bool {
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
			return false
		}
	}
	return true
}
