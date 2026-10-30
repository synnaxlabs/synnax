// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";

import { type channel } from "@/channel";
import {
  type Channels,
  type Key,
  type Line,
  lineZ,
  Y_AXIS_KEYS,
  type YAxisKey,
} from "@/lineplot/types.gen";

// LINE_NAMESPACE is the UUID namespace line keys are derived in. It matches the
// Core's, so both derive the same key for the same line.
const LINE_NAMESPACE = "a708aa30-81f8-4cf6-82fd-146c35663a6b";

/** LineIdentity is what a line styles: a y-channel on a y-axis. */
export interface LineIdentity {
  yAxis: YAxisKey;
  yChannel: channel.Key;
  xChannel: channel.Key;
}

/**
 * Returns the key of the plot's line with the given identity: a name-based UUID, so
 * the Core and the client derive the same key for the same line.
 */
export const lineKey = (
  plot: Key,
  { yAxis, yChannel, xChannel }: LineIdentity,
): string =>
  uuid.fromName(`${plot}---${yAxis}---${yChannel}---${xChannel}`, LINE_NAMESPACE);

const identityOf = ({ yAxis, yChannel, xChannel }: LineIdentity): string =>
  `${yAxis}---${yChannel}---${xChannel}`;

// reconcileLines rebuilds the plot's line set: one line per y-channel bound to a
// y-axis, plotted against the x1 channel. A line styles its channel across every
// range, so the set does not depend on the plot's ranges. An existing line with the
// same identity keeps its styling and is re-keyed to the plot; missing ones are
// created with default styling, and lines whose channel is no longer bound are
// returned in `dropped` (so callers can restore them on undo).
export const reconcileLines = (
  plot: Key,
  channels: Channels,
  existing: Line[],
): { lines: Line[]; dropped: Line[] } => {
  const byIdentity = new Map<string, Line>();
  for (const l of existing)
    if (!byIdentity.has(identityOf(l))) byIdentity.set(identityOf(l), l);
  const kept = new Set<string>();
  const lines: Line[] = [];
  const xChannel = channels.x1;
  for (const yAxis of Y_AXIS_KEYS)
    for (const yChannel of channels[yAxis]) {
      const identity = { yAxis, yChannel, xChannel };
      const id = identityOf(identity);
      if (kept.has(id)) continue;
      kept.add(id);
      const key = lineKey(plot, identity);
      const prev = byIdentity.get(id);
      // A new line takes Oracle schema defaults (stroke width, aggregation); label
      // and color stay unset so they resolve at render time.
      lines.push(prev != null ? { ...prev, key } : lineZ.parse({ key, ...identity }));
    }
  const dropped = existing.filter((l) => !kept.has(identityOf(l)));
  return { lines, dropped };
};
