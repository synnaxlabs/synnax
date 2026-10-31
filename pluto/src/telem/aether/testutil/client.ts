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
 * through a feed opened with the props. The caller closes the feed.
 */
export const createClient = (
  core: SyntheticCore,
  props: Partial<framer.FeedProps> = {},
): Client & { feed: framer.Feed } => ({
  feed: core.openFeed(props),
  channels: core.channels,
  now: core.now,
});

/** @returns the client with its channel retrievals run through retrieve. */
export const withRetrieve = (
  client: Client,
  retrieve: Client["channels"]["retrieve"],
): Client => ({ ...client, channels: { retrieve } });
