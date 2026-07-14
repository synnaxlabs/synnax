// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
)

// nopFreeIndexResolver is registered on every provisioned node so distribution-layer
// tests can write to free channels without a service layer. It resolves no indexes, so
// free frames carry zero alignments; tests that assert on alignments register their own
// resolver.
type nopFreeIndexResolver struct{}

func (nopFreeIndexResolver) ResolveFreeIndexes(
	context.Context, channel.Keys,
) (map[channel.Key]channel.Key, error) {
	return nil, nil
}
