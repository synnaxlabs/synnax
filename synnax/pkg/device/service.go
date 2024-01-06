/*
 * Copyright 2024 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

package device

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/device/module"
	"github.com/synnaxlabs/synnax/pkg/device/rack"
	"github.com/synnaxlabs/x/config"
)

type Config = rack.Config

var DefaultConfig = rack.DefaultConfig

type Service struct {
	Rack   *rack.Service
	Module *module.Service
}

func OpenService(ctx context.Context, configs ...Config) (*Service, error) {
	cfg, err := config.New[Config](DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	rackService, err := rack.OpenService(ctx, cfg)
	if err != nil {
		return nil, err
	}

	moduleService, err := module.OpenService(ctx, module.Config{
		DB:       cfg.DB,
		Ontology: cfg.Ontology,
		Group:    cfg.Group,
		Rack:     rackService,
	})
	return &Service{Rack: rackService, Module: moduleService}, err
}
