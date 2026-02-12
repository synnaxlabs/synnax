// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build !driver

package driver

import (
	"context"

	"github.com/synnaxlabs/x/config"
)

// Driver is a no-op stub used when the server is built without the "driver" build tag.
// Open returns a valid Driver whose Close method is a no-op.
type Driver struct{}

func Open(_ context.Context, cfgs ...Config) (*Driver, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	cfg.L.Info("server built without embedded driver")
	return &Driver{}, nil
}

func (d *Driver) Close() error { return nil }
