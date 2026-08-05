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

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/table/versions/legacy"
	v0 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v1"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// DecodeImExEnvelope materializes env's body as a current-version Table, keyless and
// named after the envelope. An unknown version is a path-scoped validation error.
func DecodeImExEnvelope(ctx context.Context, env imex.Envelope) (Table, error) {
	var (
		t   Table
		err error
	)
	if env.Version > legacy.LastVersion {
		// v1 is the shape both producers wrote: Core stamps it on export, and the
		// v0.56 Console export stamps the same number on the table it retrieved from
		// Core. Every Table field name is a single word, so the Console's camelCase and
		// Core's snake_case coincide.
		t, err = decodeMigrate(ctx, env)
	} else {
		// Console states embed the structural model inline: ride the storage lift,
		// which decodes the body through the legacy chain.
		var body msgpack.EncodedJSON
		if body, err = imex.Decode[msgpack.EncodedJSON](ctx, env); err == nil {
			t, err = v1.MigrateTable(ctx, v0.Table{Name: env.Name, Data: body})
		}
	}
	if err != nil {
		return Table{}, err
	}
	// Importing always materializes a new resource, so any key on the wire is dropped
	// and the importer mints a fresh one.
	t.Key = uuid.Nil
	// The header is the resolved name: the body's name when present, or the file-name
	// fallback the imex service applies. Console-era decodes drop it, so it is stamped
	// here for every path.
	t.Name = env.Name
	return t, nil
}
