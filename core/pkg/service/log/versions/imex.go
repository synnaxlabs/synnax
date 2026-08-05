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
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/versions/legacy/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v3"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// DecodeImExEnvelope materializes env's body as a current-version Log, keyless and
// named after the envelope. An unknown version is a path-scoped validation error.
func DecodeImExEnvelope(ctx context.Context, env imex.Envelope) (Log, error) {
	var (
		l   Log
		err error
	)
	switch {
	case env.Version >= Floor:
		l, err = decodeMigrate(ctx, env)
	// Console-era log wire formats cap at data version 1; versions between that and
	// Floor never shipped. The guard lives here because MigrateLog swallows chain
	// errors for boot-migration resilience.
	case env.Version > v1.Version:
		err = errors.Newf("unknown log data version %d", env.Version)
	default:
		var body msgpack.EncodedJSON
		if body, err = imex.Decode[msgpack.EncodedJSON](ctx, env); err == nil {
			l, err = v3.MigrateLog(ctx, v2.Log{Name: env.Name, Data: body})
		}
	}
	if err != nil {
		return Log{}, err
	}
	// Importing always materializes a new resource, so any key on the wire is dropped
	// and the importer mints a fresh one.
	l.Key = uuid.Nil
	// The header is the resolved name: the body's name when present, or the file-name
	// fallback the imex service applies. Console-era decodes drop it, so it is stamped
	// here for every path.
	l.Name = env.Name
	return l, nil
}
