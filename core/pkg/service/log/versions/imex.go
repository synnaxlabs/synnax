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
	"github.com/synnaxlabs/synnax/pkg/service/log/versions/legacy"
	v0 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v0"
	v2 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v2"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// DecodeImExEnvelope materializes env's body as a current-version Log, keyless and
// named after the envelope. An unknown version is a path-scoped validation error.
func DecodeImExEnvelope(ctx context.Context, env imex.Envelope) (Log, error) {
	var (
		l   Log
		err error
	)
	if env.Version > legacy.LastVersion {
		l, err = autoDecodeEnvelope(ctx, env)
	} else {
		l, err = decodeConsoleEnvelope(ctx, env)
	}
	if err != nil {
		return Log{}, err
	}
	// A Console state lifts through Go code that cannot tell an absent field from a
	// zero one, so its zero fields take their defaults.
	if env.Version <= legacy.LastVersion {
		l = defaults.Fill(l)
	}
	// Importing always materializes a new resource, so any key on the wire is dropped
	// and the importer mints a fresh one.
	l.Key = uuid.Nil()
	// The header is the resolved name: the body's name when present, or the file-name
	// fallback the imex service applies. Console-era decodes drop it, so it is stamped
	// here for every path.
	l.Name = env.Name
	return l, nil
}

// decodeConsoleEnvelope lifts a Console-written log envelope to a current-version Log.
// Console states embed the body inline, so it rides the storage lift, which decodes the
// body through the legacy chain.
func decodeConsoleEnvelope(ctx context.Context, env imex.Envelope) (Log, error) {
	body, err := env.Decode[msgpack.EncodedJSON]()
	if err != nil {
		return Log{}, err
	}
	if err = imex.RequireFields(body, "a log", "channels"); err != nil {
		return Log{}, err
	}
	l2, err := v2.MigrateLog(ctx, v0.Log{Name: env.Name, Data: body})
	if err != nil {
		return Log{}, err
	}
	return liftFirst(ctx, l2)
}
