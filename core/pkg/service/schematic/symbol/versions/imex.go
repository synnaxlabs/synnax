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
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/legacy"
	v2 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v2"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// DecodeImExEnvelope materializes env's body as a current-version Symbol, keyless and
// named after the envelope. An unknown version is a path-scoped validation error.
func DecodeImExEnvelope(ctx context.Context, env imex.Envelope) (Symbol, error) {
	var (
		sym Symbol
		err error
	)
	if env.Version > legacy.LastVersion {
		sym, err = autoDecodeEnvelope(ctx, env)
	} else {
		var body msgpack.EncodedJSON
		if body, err = imex.Decode[msgpack.EncodedJSON](ctx, env); err == nil {
			if err = imex.RequireFields(body, "symbol", "data"); err == nil {
				var d legacy.Data
				if d, err = imex.Decode[legacy.Data](ctx, env); err == nil {
					sym.Data = v2.SpecFromConsole(d.Spec)
				}
			}
		}
	}
	if err != nil {
		return Symbol{}, err
	}
	// Importing always materializes a new resource, so any key on the wire is dropped
	// and the importer mints a fresh one.
	sym.Key = uuid.Nil
	// The header is the resolved name: the body's name when present, or the file-name
	// fallback the imex service applies. Console-era decodes drop it, so it is stamped
	// here for every path.
	sym.Name = env.Name
	return sym, nil
}
