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
	v2 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v3"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// typedVersion is the first envelope version whose body is the typed Log rather
// than a legacy Console state.
const typedVersion imex.Version = 2

// DecodeImport materializes the envelope's body as a current-version Log. Envelopes
// older than Latest are legacy camelCase Console exports and are lifted forward
// through the migration chain; an envelope newer than Latest is rejected with a
// path-scoped validation error.
func DecodeImport(ctx context.Context, env imex.Envelope) (Log, error) {
	switch {
	case env.Version > Latest:
		return Log{}, imex.NewErrUnsupportedVersion(env.Type, env.Version, Latest)
	case env.Version >= typedVersion:
		return imex.Decode[Log](ctx, env)
	default:
		body, err := imex.Decode[msgpack.EncodedJSON](ctx, env)
		if err != nil {
			return Log{}, err
		}
		return v3.MigrateLog(ctx, v2.Log{Name: env.Name, Data: body})
	}
}
