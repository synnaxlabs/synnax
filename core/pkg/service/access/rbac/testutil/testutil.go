// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil opens RBAC and grants access through it for specs.
package testutil

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/service"
)

// Config is the configuration for [Open].
type Config struct {
	// DB holds the roles, policies, users, and racks.
	DB *gorp.DB
	// User is the user service that holds the roles of each user.
	User *user.Service
	// Status is the status service that racks write to.
	Status *status.Service
}

// Services holds the services that [Open] opens.
type Services struct {
	// RBAC enforces access by the roles of users and racks.
	RBAC *rbac.Service
	// Rack holds the roles of racks.
	Rack   *rack.Service
	closer xio.MultiCloser
}

// Close closes the services that [Open] opened.
func (s *Services) Close() error { return s.closer.Close() }

// Open opens the roles, a rack service on one static host, and an RBAC service over
// them.
func Open(ctx context.Context, cfg Config) (s *Services, err error) {
	s = &Services{}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	roles, err := rbac.OpenRoles(ctx, rbac.RolesConfig{
		DB: cfg.DB,
	})
	if !ok(err, roles) {
		return nil, err
	}
	if s.Rack, err = rack.OpenService(ctx, rack.ServiceConfig{
		DB:         cfg.DB,
		HostKey:    1,
		Status:     cfg.Status,
		DriverRole: roles.Builtin.DriverKey,
	}); !ok(err, s.Rack) {
		return nil, err
	}
	if s.RBAC, err = rbac.NewService(ctx, rbac.ServiceConfig{
		DB:    cfg.DB,
		Roles: roles,
		User:  cfg.User,
		Rack:  s.Rack,
	}); err != nil {
		return nil, err
	}
	return s, nil
}

// CreateUser creates u holding a new role of its own with no policies, so the user can
// do nothing until Grant allows it. It assigns u a key when u.Key is zero.
func CreateUser(
	ctx context.Context,
	db *gorp.DB,
	rbacSvc *rbac.Service,
	users *user.Service,
	u user.User,
) (user.User, error) {
	if u.Key == uuid.Nil() {
		u.Key = uuid.New()
	}
	err := db.WithTx(ctx, func(tx gorp.Tx) (err error) {
		r := role.Role{Name: ownRoleName(u.Key)}
		if err = rbacSvc.Role.NewWriter(tx, false).Create(ctx, &r); err != nil {
			return err
		}
		u.Role = r.Key
		u, err = users.NewWriter(tx).Create(ctx, u)
		return err
	})
	if err != nil {
		return user.User{}, err
	}
	return u, nil
}

// Grant allows the user with the given key the actions on the objects through a new
// policy on the user's own role, and returns the policy. A user that does not hold a
// role of its own first gets one in place of its role, so grants to one user add up
// without changing what other users can do.
func Grant(
	ctx context.Context,
	db *gorp.DB,
	rbacSvc *rbac.Service,
	users *user.Service,
	key user.Key,
	actions []access.Action,
	objects ...resource.ID,
) (policy.Policy, error) {
	var p policy.Policy
	err := db.WithTx(ctx, func(tx gorp.Tx) error {
		r, err := ownRole(ctx, tx, rbacSvc, users, key)
		if err != nil {
			return err
		}
		p = policy.Policy{
			Role:    r,
			Name:    ownRoleName(key),
			Objects: objects,
			Actions: actions,
		}
		return rbacSvc.Policy.NewWriter(tx, false).Create(ctx, &p)
	})
	return p, err
}

func ownRoleName(key user.Key) string { return "grant-" + key.String() }

func ownRole(
	ctx context.Context,
	tx gorp.Tx,
	rbacSvc *rbac.Service,
	users *user.Service,
	key user.Key,
) (role.Key, error) {
	var u user.User
	if err := users.NewRetrieve().
		Where(user.MatchKeys(key)).
		Entry(&u).
		Exec(ctx, nil); err != nil {
		return uuid.Nil(), err
	}
	var held role.Role
	if err := rbacSvc.Role.NewRetrieve().
		Where(role.MatchKeys(u.Role)).
		Entry(&held).
		Exec(ctx, nil); err != nil {
		return uuid.Nil(), err
	}
	if held.Name == ownRoleName(key) {
		return held.Key, nil
	}
	r := role.Role{Name: ownRoleName(key)}
	if err := rbacSvc.Role.NewWriter(tx, false).Create(ctx, &r); err != nil {
		return uuid.Nil(), err
	}
	return r.Key, users.NewWriter(tx).SetRole(ctx, r.Key, nil, key)
}
