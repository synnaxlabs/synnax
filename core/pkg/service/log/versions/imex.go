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
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/versions/legacy/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v3"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// DecodeImport materializes the envelope's body as a current-version Log. Envelopes
// stamped at or above Floor decode through the generated migration chain; older
// ones are legacy camelCase Console exports and are lifted forward. An envelope
// newer than Latest is rejected with a path-scoped validation error.
func DecodeImport(ctx context.Context, env imex.Envelope) (Log, error) {
	if env.Version >= Floor {
		return decodeMigrate(ctx, env)
	}
	// Console-era log wire formats cap at data version 1; versions between that and
	// Floor are rc-era formats that never shipped. The guard lives here because
	// MigrateLog swallows chain errors for boot-migration resilience.
	if env.Version > v1.Version {
		return Log{}, errors.Newf("unknown log data version %d", env.Version)
	}
	body, err := imex.Decode[msgpack.EncodedJSON](ctx, env)
	if err != nil {
		return Log{}, err
	}
	return v3.MigrateLog(ctx, v2.Log{Name: env.Name, Data: body})
}
