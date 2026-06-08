// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@/channel";
import {
  type Channels,
  type Line,
  lineZ,
  type Ranges,
  X_AXIS_KEYS,
  type XAxisKey,
  Y_AXIS_KEYS,
  type YAxisKey,
} from "@/lineplot/types.gen";

// LineKeyParts is the structured identity of a plotted line: a y-channel on a
// y-axis plotted against an x-channel/x-axis over a range. It is the decoded
// form of Line.key.
export interface LineKeyParts {
  yAxis: YAxisKey;
  xAxis: XAxisKey;
  range: string;
  xChannel: channel.Key;
  yChannel: channel.Key;
}

const SEPARATOR = "---";

// lineKey encodes the identity of a line into the stable string stored as
// Line.key. This is the only place the key format is defined.
export const lineKey = (parts: LineKeyParts): string =>
  [parts.yAxis, parts.xAxis, parts.range, parts.xChannel, parts.yChannel].join(
    SEPARATOR,
  );

// parseLineKey decodes a Line.key produced by lineKey back into its identity.
export const parseLineKey = (key: string): LineKeyParts => {
  const [yAxis, xAxis, range, xChannel, yChannel] = key.split(SEPARATOR);
  return {
    yAxis: yAxis as YAxisKey,
    xAxis: xAxis as XAxisKey,
    range,
    xChannel: Number(xChannel),
    yChannel: Number(yChannel),
  };
};

// reconcileLines rebuilds the complete set of lines implied by the channel and
// range bindings: one line per (x-axis, range, y-axis, y-channel) combination.
// Existing lines are preserved by key so user styling survives, missing ones
// are created with default styling, and lines whose combination no longer
// exists are returned in `dropped` (so callers can restore them on undo).
export const reconcileLines = (
  channels: Channels,
  ranges: Ranges,
  existing: Line[],
): { lines: Line[]; dropped: Line[] } => {
  const byKey = new Map(existing.map((l) => [l.key, l]));
  const kept = new Set<string>();
  const lines: Line[] = [];
  for (const xAxis of X_AXIS_KEYS) {
    const xChannel = channels[xAxis];
    for (const range of ranges[xAxis])
      for (const yAxis of Y_AXIS_KEYS)
        for (const yChannel of channels[yAxis]) {
          const key = lineKey({ yAxis, xAxis, range, xChannel, yChannel });
          if (kept.has(key)) continue;
          kept.add(key);
          // A new line takes Oracle schema defaults (stroke width, downsample);
          // label and color stay unset so they resolve at render time.
          lines.push(byKey.get(key) ?? lineZ.parse({ key }));
        }
  }
  const dropped = existing.filter((l) => !kept.has(l.key));
  return { lines, dropped };
};
