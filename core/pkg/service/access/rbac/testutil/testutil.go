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
	disttestutil "github.com/synnaxlabs/synnax/pkg/distribution/testutil"
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
	DB   *gorp.DB
	User *user.Service
	// Status is the status service that racks write to.
	Status *status.Service
}

// Services holds the services that [Open] opens.
type Services struct {
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
		DB:           cfg.DB,
		HostProvider: disttestutil.NewStaticHostProvider(1),
		Status:       cfg.Status,
		DriverRole:   roles.Builtin.DriverKey,
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

// Grant gives the user with the given key a new role whose one policy allows the
// actions on the objects, and returns the key of the role.
func Grant(
	ctx context.Context,
	rbacSvc *rbac.Service,
	users *user.Service,
	key user.Key,
	actions []access.Action,
	objects ...resource.ID,
) (role.Key, error) {
	r := role.Role{Name: "grant-" + uuid.New().String()}
	if err := rbacSvc.Role.NewWriter(nil, false).Create(ctx, &r); err != nil {
		return uuid.Nil(), err
	}
	if err := rbacSvc.Policy.NewWriter(nil, false).Create(ctx, &policy.Policy{
		Role:    r.Key,
		Name:    r.Name,
		Objects: objects,
		Actions: actions,
	}); err != nil {
		return uuid.Nil(), err
	}
	if err := users.NewWriter(nil).AssignRole(ctx, r.Key, key); err != nil {
		return uuid.Nil(), err
	}
	return r.Key, nil
}
