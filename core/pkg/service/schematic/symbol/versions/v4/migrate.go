// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4

import (
	"context"

	"github.com/synnaxlabs/gorp/migrate"
	v3 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v3"
)

// MigrateSymbol lifts a v3 symbol to v4, which stores every color alpha as a 0-255
// byte.
func MigrateSymbol(ctx context.Context, old v3.Symbol) (Symbol, error) {
	return autoMigrateSymbol(ctx, old)
}

// Migration lifts stored symbols from v3 to v4.
var Migration = migrate.NewEntry("v62_color_alpha_byte", MigrateSymbol)

// MigrateSpec lifts a v3 spec to v4.
func MigrateSpec(ctx context.Context, old v3.Spec) (Spec, error) {
	return autoMigrateSpec(ctx, old)
}

// MigrateState lifts a v3 state to v4.
func MigrateState(ctx context.Context, old v3.State) (State, error) {
	return autoMigrateState(ctx, old)
}

// MigrateRegion lifts a v3 region to v4.
func MigrateRegion(ctx context.Context, old v3.Region) (Region, error) {
	return autoMigrateRegion(ctx, old)
}
