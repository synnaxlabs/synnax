// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package bind runs the outside writes of the API layer in one Gorp transaction and
// checks access on what each write changes.
package bind

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for creating a Service.
type ServiceConfig struct {
	// DB opens the transaction of each write and the snapshot of each read.
	//
	// [REQUIRED]
	DB *gorp.DB
	// RBAC checks the access of each read and write.
	//
	// [REQUIRED]
	RBAC *rbac.Service
	// Held names the tables of the records that a user or rack holds, such as its API
	// keys. A change to a held record needs its own action on its holder, or nothing
	// when the holder makes it. Any other record with no resource type needs update on
	// the record that it is placed in.
	//
	// [REQUIRED]
	Held []string
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.Held = override.Slice(c.Held, other.Held)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.bind")
	v.NotNil("db", c.DB)
	v.NotNil("rbac", c.RBAC)
	v.NotNil("held", c.Held)
	return v.Error()
}

// Service holds what Write needs to run a write.
type Service struct{ cfg ServiceConfig }

// NewService creates a Service. It returns an error when a required field of the
// configuration is missing.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}
