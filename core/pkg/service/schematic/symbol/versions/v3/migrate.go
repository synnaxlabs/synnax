// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	"context"

	"github.com/synnaxlabs/gorp/migrate"
	v2 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v2"
)

// MigrateSymbol lifts a v2 symbol to v3, which stores every color alpha as a 0-255
// byte.
func MigrateSymbol(ctx context.Context, old v2.Symbol) (Symbol, error) {
	return autoMigrateSymbol(ctx, old)
}

// Migration lifts stored symbols from v2 to v3.
var Migration = migrate.NewEntry("v61_color_alpha_byte", MigrateSymbol)

// MigrateSpec lifts a v2 spec to v3.
func MigrateSpec(ctx context.Context, old v2.Spec) (Spec, error) {
	return autoMigrateSpec(ctx, old)
}

// MigrateState lifts a v2 state to v3.
func MigrateState(ctx context.Context, old v2.State) (State, error) {
	return autoMigrateState(ctx, old)
}

// MigrateRegion lifts a v2 region to v3.
func MigrateRegion(ctx context.Context, old v2.Region) (Region, error) {
	return autoMigrateRegion(ctx, old)
}
