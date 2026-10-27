// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type SyntheticCore } from "@synnaxlabs/client/testutil";
import { MultiSeries } from "@synnaxlabs/x";

import { GL_TRANSFORM } from "@/telem/aether/convertSeries";
import { type Client } from "@/telem/aether/remote";

/**
 * @returns the client that the telemetry provider builds, served by the synthetic Core.
 * The caller closes it.
 */
export const createClient = (
  core: SyntheticCore,
): Client & { close: () => Promise<void> } => {
  const feed = core.openFeed({ transform: GL_TRANSFORM });
  const exactFeed = core.openFeed();
  return {
    feed,
    exactFeed,
    readLatest: async () => new MultiSeries([]),
    readEarliest: async () => new MultiSeries([]),
    channels: core.channels,
    now: core.now,
    close: async () => {
      await Promise.all([feed.close(), exactFeed.close()]);
    },
  };
};
