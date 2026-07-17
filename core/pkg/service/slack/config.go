// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package slack

import (
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// FactoryConfig is the configuration for the Slack factory.
type FactoryConfig struct {
	// Status observes the status changes that drive Slack posts.
	//
	// [REQUIRED]
	Status *status.Service
	// Device resolves the Slack device holding the workspace bot token.
	//
	// [REQUIRED]
	Device *device.Service
	// Sender performs the outbound Slack call.
	//
	// [OPTIONAL] - Defaults to the real chat.postMessage API.
	Sender Sender
	alamos.Instrumentation
}

var _ config.Config[FactoryConfig] = FactoryConfig{}

// Override overrides the factory configuration with the given other configuration.
func (c FactoryConfig) Override(other FactoryConfig) FactoryConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Status = override.Nil(c.Status, other.Status)
	c.Device = override.Nil(c.Device, other.Device)
	c.Sender = override.Nil(c.Sender, other.Sender)
	return c
}

// Validate validates the factory configuration.
func (c FactoryConfig) Validate() error {
	v := validate.New("slack.factory")
	validate.NotNil(v, "status", c.Status)
	validate.NotNil(v, "device", c.Device)
	validate.NotNil(v, "sender", c.Sender)
	return v.Error()
}
