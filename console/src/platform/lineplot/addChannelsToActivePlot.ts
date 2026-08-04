// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, lineplot, type Synnax } from "@synnaxlabs/client";

// addChannelsToActivePlot dispatches AddChannel actions for each channel that
// is not already on the y1 axis of the target plot. It runs outside of React
// (called from ontology handlers), so it bypasses Pluto's hook-based dispatch
// and goes through the client directly. The server broadcast on
// sy_lineplot_set is what reconciles the local Pluto store afterwards.
export const addChannelsToActivePlot = async (
  client: Synnax,
  key: lineplot.Key,
  channels: channel.Key[],
): Promise<void> => {
  const existing = await client.lineplots.retrieve(key);
  const present = new Set(existing.channels.y1);
  const additions = channels.filter((c) => !present.has(c));
  if (additions.length === 0) return;
  const actions = additions.map((c) =>
    lineplot.addChannel({ axisKey: "y1", channel: c }),
  );
  await client.lineplots.dispatch(key, actions);
};
