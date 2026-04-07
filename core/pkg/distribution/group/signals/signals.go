// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signals

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
)

// Publish publishes group changes to the signals provider.
func Publish(
	ctx context.Context,
	prov *signals.Provider,
	obs observe.Observable[gorp.TxReader[uuid.UUID, group.Group]],
) (io.Closer, error) {
	return signals.PublishFromGorp(
		ctx,
		prov,
		signals.GorpPublisherConfigUUID[group.Group](obs),
	)
}
