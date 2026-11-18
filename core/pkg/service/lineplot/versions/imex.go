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
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/legacy"
	v0 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v0"
	v5 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v5"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// DecodeImExEnvelope materializes env's body as a current-version LinePlot, keyless and
// named after the envelope. An unknown version is a path-scoped validation error.
func DecodeImExEnvelope(ctx context.Context, env imex.Envelope) (LinePlot, error) {
	var (
		lp  LinePlot
		err error
	)
	if env.Version > legacy.LastVersion {
		lp, err = autoDecodeEnvelope(ctx, env)
	} else {
		lp, err = decodeConsoleEnvelope(ctx, env)
	}
	if err != nil {
		return LinePlot{}, err
	}
	// A Console state lifts through Go code that cannot tell an absent field from a
	// zero one, so its zero fields take their defaults. A Core export already decodes
	// over them.
	if env.Version <= legacy.LastVersion {
		lp = defaults.Fill(lp)
	}
	// Importing always materializes a new resource, so any key on the wire is dropped
	// and the importer mints a fresh one.
	lp.Key = uuid.Nil()
	// The header is the resolved name: the body's name when present, or the file-name
	// fallback the imex service applies. Console-era decodes drop it, so it is stamped
	// here for every path.
	lp.Name = env.Name
	return lp, nil
}

// decodeConsoleEnvelope lifts a Console-written line plot envelope to a current-version
// LinePlot. Every Console line plot file is a state: the exporter writes the slice
// entry, which embeds the body inline under a stamped version. It rides the storage
// lift, which dispatches on that version.
func decodeConsoleEnvelope(ctx context.Context, env imex.Envelope) (LinePlot, error) {
	body, err := env.Decode[msgpack.EncodedJSON]()
	if err != nil {
		return LinePlot{}, err
	}
	if err = imex.RequireFields(body, "a line plot", "axes", "channels"); err != nil {
		return LinePlot{}, err
	}
	lp5, err := v5.MigrateLinePlot(ctx, v0.LinePlot{Name: env.Name, Data: body})
	if err != nil {
		return LinePlot{}, err
	}
	return liftFirst(ctx, lp5)
}
