// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build !driver
// +build !driver

package embedded

import "context"

func OpenDriver(ctx context.Context, cfgs ...Config) (*Driver, error) {
	return &Driver{}, nil
}

func (d *Driver) Stop() error {
	return nil
}
