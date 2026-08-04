// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v1 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v2"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// lastStateVersion is the final Console state version ("1.0.0" files). A v1
// state parks the structural model under pendingUpload; v0 states embed it
// inline and ride the storage lift's legacy chain.
const lastStateVersion imex.Version = 1

// consoleDocument mirrors the typed Table body fields as they appear inside a v1
// Console state's pendingUpload.
type consoleDocument struct {
	Rows    []Row           `json:"rows"`
	Columns []Column        `json:"columns"`
	Cells   map[string]Cell `json:"cells"`
}

// DecodeImport materializes the envelope's body as a current-version Table.
// Envelopes stamped at or above Floor decode through the generated migration
// chain; older ones are Console-era files — camelCase typed exports or console
// states — and are lifted forward. An envelope newer than Latest is rejected
// with a path-scoped validation error.
func DecodeImport(ctx context.Context, env imex.Envelope) (Table, error) {
	if env.Version >= Floor {
		return decodeMigrate(ctx, env)
	}
	// Console-era typed exports ("1.0.0"-stamped or versionless) carry the current
	// shape with camelCase keys; every Table field key is a single word, so the
	// standard decoder's case-insensitive matching covers them. Console states
	// never carry a name.
	if env.BodyNamed() {
		return imex.Decode[Table](ctx, env)
	}
	if env.Version == lastStateVersion {
		doc, err := imex.DecodePendingUpload[consoleDocument](ctx, env, "table")
		if err != nil {
			return Table{}, err
		}
		return Table{Rows: doc.Rows, Columns: doc.Columns, Cells: doc.Cells}, nil
	}
	// v0 console states embed the structural model inline: ride the storage lift,
	// which decodes the body through the legacy chain.
	body, err := imex.Decode[msgpack.EncodedJSON](ctx, env)
	if err != nil {
		return Table{}, err
	}
	return v2.MigrateTable(ctx, v1.Table{Name: env.Name, Data: body})
}
