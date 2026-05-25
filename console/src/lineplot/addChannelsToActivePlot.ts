// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, lineplot, type Synnax } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";

const ROLLING_30S_RANGE_KEY = "recent";

// addChannelsToActivePlot dispatches AddChannel actions for each channel that
// is not already on the y1 axis of the target plot. It runs outside of React
// (called from ontology handlers), so it bypasses Pluto's hook-based dispatch
// and goes through the client directly. The server broadcast on
// sy_lineplot_set is what reconciles the local Pluto store afterwards.
//
// When the target plot has no x1 ranges yet, the batch is prefixed with an
// AddRange for the supplied activeRange (defaulting to the rolling 30s key)
// so the canvas has something to plot the new channels against. The
// useAssignedDispatch React hook performs the equivalent prepend on the
// in-app dispatch path; mirroring it here keeps the "drag a channel from
// Resources onto an empty plot" UX symmetric.
export const addChannelsToActivePlot = async (
  client: Synnax,
  key: lineplot.Key,
  channels: channel.Key[],
  activeRange: string = ROLLING_30S_RANGE_KEY,
): Promise<void> => {
  const existing = await client.lineplots.retrieve({ key });
  const present = new Set(existing.channels.y1);
  const additions = channels.filter((c) => !present.has(c));
  if (additions.length === 0) return;
  const actions: lineplot.Action[] = [];
  if (existing.ranges.x1.length === 0)
    actions.push(lineplot.addRange({ axisKey: "x1", range: activeRange }));
  for (const c of additions)
    actions.push(lineplot.addChannel({ axisKey: "y1", channel: c }));
  await client.lineplots.dispatch(key, id.create(), actions);
};
