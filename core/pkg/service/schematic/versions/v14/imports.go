// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v14

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	symbol "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v5"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
)

// ImportCustomNodes gives each custom node of old with state overrides, in an import of
// an old file on tx, the enumeration or the copy of its symbol that NewMigration gives
// a stored node. Each custom actuator whose symbol does not actuate gets a copy that
// does. s must be old lifted to v14. A node keeps its symbol when retrievable rejects
// the symbol or an enumeration that its enumeration or copy needs. Each enumeration
// comes from intern, and create creates each copy.
func ImportCustomNodes(
	ctx context.Context,
	tx gorp.Tx,
	s *Schematic,
	old v13.Schematic,
	retrievable func(resource.ID) bool,
	intern symbol.Intern,
	create func(ctx context.Context, sym *symbol.Symbol) error,
) error {
	o := &overrides{
		tx:          tx,
		intern:      intern,
		create:      create,
		retrievable: retrievable,
		copies:      map[string]symbol.Key{},
	}
	return o.apply(ctx, s, old)
}
