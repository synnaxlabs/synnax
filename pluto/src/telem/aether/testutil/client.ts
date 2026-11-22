// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type framer } from "@synnaxlabs/client";
import { type SyntheticCore } from "@synnaxlabs/client/testutil";

import { type Client } from "@/telem/aether/remote";

/**
 * @returns the client that the telemetry provider builds, served by the synthetic Core
 * through a feed opened with the props. The caller closes the feed. The synthetic Core
 * holds no calculations.
 */
export const createClient = (
  core: SyntheticCore,
  props: Partial<framer.FeedProps> = {},
): Client & { feed: framer.Feed } => ({
  feed: core.openFeed(props),
  channels: core.channels,
  calculations: {
    outputs: {
      retrieve: async (keys) =>
        keys.map((key) => ({ key, hidden: false, calculation: null })),
    },
  },
  now: core.now,
});
