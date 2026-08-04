// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, log, type Synnax } from "@synnaxlabs/client";

// addChannelsToActiveLog dispatches AddChannel actions for each channel that is
// not already in the target log. It runs outside of React (called from ontology
// handlers), so it bypasses Pluto's hook-based dispatch and goes through the
// client directly. The retrieve also warms the cache dispatch requires.
export const addChannelsToActiveLog = async (
  client: Synnax,
  key: log.Key,
  channels: channel.Key[],
): Promise<void> => {
  const existing = await client.logs.retrieve(key);
  const present = new Set(existing.channels.map((e) => e.channel));
  const additions = channels.filter((c) => !present.has(c));
  if (additions.length === 0) return;
  const actions = additions.map((c) => log.addChannel({ channel: c }));
  await client.logs.dispatch(key, actions);
};
