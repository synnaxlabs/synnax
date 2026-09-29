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
)

// DecodeImExEnvelope materializes env's body as a current-version Library, keyless and
// named after the envelope. An unknown version is a path-scoped validation error.
func DecodeImExEnvelope(ctx context.Context, env imex.Envelope) (Library, error) {
	l, err := autoDecodeEnvelope(ctx, env)
	if err != nil {
		return Library{}, err
	}
	l.Key = uuid.Nil()
	l.Name = env.Name
	return l, nil
}
