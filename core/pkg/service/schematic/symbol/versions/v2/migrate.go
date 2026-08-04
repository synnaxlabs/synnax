// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"

	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v1"
	"github.com/synnaxlabs/x/gorp"
)

// Migration drops the persisted version field: envelope versioning now lives on the
// imex wire format alone.
var Migration = gorp.NewEntryMigration("v2_drop_symbol_version", migrateSymbol)

func migrateSymbol(_ context.Context, old v1.Symbol) (Symbol, error) {
	return Symbol{Key: old.Key, Name: old.Name, Data: old.Data}, nil
}
