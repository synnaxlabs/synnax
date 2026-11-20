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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// NewReferenceFeedAppend returns the append of a reference feed, as
// PublishReferenceChanges builds it. The append returns the frame of a commit and the
// changes that the feed delivers for it.
func NewReferenceFeedAppend(
	holderOf func(gorp.Target) resource.ID,
) func(gorp.Commit) (framer.Frame, []ReferenceChange) {
	f := newReferenceFeed(holderOf)
	var delivered []ReferenceChange
	record := func(_ context.Context, changes []ReferenceChange) { delivered = changes }
	ReferenceChanges{feed: f}.OnChange(record)
	return func(c gorp.Commit) (framer.Frame, []ReferenceChange) {
		delivered = nil
		fr := f.append(c, framer.Frame{})
		return fr, delivered
	}
}
