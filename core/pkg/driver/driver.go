// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/driver/cpp"
	godriver2 "github.com/synnaxlabs/synnax/pkg/driver/go"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
)

type Config struct {
	Go  godriver2.Config
	CPP cpp.Config
}

func (c Config) Validate() error {
	if err := c.Go.Validate(); err != nil {
		return err
	}
	return c.CPP.Validate()
}

func (c Config) Override(other Config) Config {
	c.Go = c.Go.Override(other.Go)
	c.CPP = c.CPP.Override(other.CPP)
	return c
}

type Driver struct {
	go_ *godriver2.Driver
	cpp *cpp.Driver
}

func (d *Driver) Close() error {
	e := errors.NewCatcher(errors.WithAggregation())
	e.Exec(d.go_.Close)
	e.Exec(d.cpp.Close)
	return e.Error()

}

func Open(ctx context.Context, cfgs ...Config) (*Driver, error) {
	cfg, err := config.New(Config{}, cfgs...)
	if err != nil {
		return nil, err
	}
	d := &Driver{}

	if d.go_, err = godriver2.Open(ctx, cfg.Go); err != nil {
		return nil, err
	}
	if d.cpp, err = cpp.Open(ctx, cfg.CPP); err != nil {
		return nil, err
	}
	return d, nil
}
