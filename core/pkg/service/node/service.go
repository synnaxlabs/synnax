// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package node serves the host node of a Core as a resource.
package node

import (
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig configures a node Service.
type ServiceConfig struct {
	alamos.Instrumentation
	// Host is the host node of the Core.
	//
	// [REQUIRED]
	Host Node
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Host = override.If(c.Host, other.Host, other.Host != Node{})
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("node")
	v.NonZero("host.key", c.Host.Key)
	return v.Error()
}

// Service serves the host node as a resource.
type Service struct{ cfg ServiceConfig }

// NewService creates a Service.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}
