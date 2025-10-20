// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// UserRoleGetter is an interface for getting user roles. This allows the RBAC
// service to check role-based policies without depending on the full user service.
type UserRoleGetter interface {
	GetUserRoles(ctx context.Context, userKey uuid.UUID) ([]uuid.UUID, error)
}

type Config struct {
	DB             *gorp.DB
	UserRoleGetter UserRoleGetter
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Override implements [config.Config].
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.UserRoleGetter = override.Nil(c.UserRoleGetter, other.UserRoleGetter)
	return c
}

// Validate implements [config.Config].
func (c Config) Validate() error {
	v := validate.New("policy")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "user_role_getter", c.UserRoleGetter)
	return v.Error()
}

type Service struct {
	Config
}

func NewService(ctx context.Context, configs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	s := &Service{Config: cfg}

	// Bootstrap the builtin Administrator role (idempotent)
	if err := s.bootstrapAdminRole(ctx); err != nil {
		return nil, err
	}

	return s, nil
}

// bootstrapAdminRole creates the builtin Administrator role if it doesn't exist.
// This is idempotent - safe to call multiple times.
func (s *Service) bootstrapAdminRole(ctx context.Context) error {
	// Check if admin role already exists by name
	var existing []Role
	if err := s.NewRoleRetriever().WhereName("Administrator").Entries(&existing).Exec(ctx, s.DB); err != nil {
		return err
	}

	if len(existing) > 0 {
		// Admin role already exists, nothing to do
		return nil
	}

	// Create the admin role
	adminRole := Role{
		Key:         uuid.New(),
		Name:        "Administrator",
		Description: "Full system access",
		Builtin:     true,
		Policies:    []uuid.UUID{}, // Will be populated with AllowAll policy
	}

	// Create the AllowAll policy for the admin role
	adminPolicy := Policy{
		Key:      uuid.New(),
		Subjects: []ontology.ID{adminRole.OntologyID()},
		Objects:  []ontology.ID{AllowAllOntologyID},
		Actions:  []access.Action{access.All},
	}

	// Add policy UUID to role
	adminRole.Policies = []uuid.UUID{adminPolicy.Key}

	// Write both role and policy in a transaction
	w := s.NewWriter(nil)
	if err := w.CreateRole(ctx, &adminRole); err != nil {
		return err
	}
	if err := w.Create(ctx, &adminPolicy); err != nil {
		return err
	}

	return nil
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{tx: gorp.OverrideTx(s.DB, tx)}
}

func (s *Service) NewRetrieve() Retriever {
	return Retriever{
		baseTx: s.DB,
		gorp:   gorp.NewRetrieve[uuid.UUID, Policy](),
	}
}

func (s *Service) NewRoleRetriever() RoleRetriever {
	return RoleRetriever{
		baseTx: s.DB,
		gorp:   gorp.NewRetrieve[uuid.UUID, Role](),
	}
}
