// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Channel } from "@synnaxlabs/pluto";

import {
  type MultiXAxisRecord,
  X_AXIS_KEYS,
  type XAxisKey,
  type YAxisKey,
} from "@/lineplot/axis";
import { type State, typedLineKeyToString } from "@/lineplot/slice";
import { type Range } from "@/range";

export const buildLines = (
  vis: State,
  sug: MultiXAxisRecord<Range.Range>,
): Array<Channel.LineProps & { key: string }> =>
  Object.entries(sug).flatMap(([xAxis, ranges]) =>
    ranges.flatMap((range) =>
      Object.entries(vis.channels)
        .filter(([axis]) => !X_AXIS_KEYS.includes(axis as XAxisKey))
        .flatMap(([yAxis, yChannels]) => {
          const xChannel = vis.channels[xAxis as XAxisKey];
          const variantArg =
            range.variant === "dynamic"
              ? { variant: "dynamic", timeSpan: range.span }
              : { variant: "static", timeRange: range.timeRange };
          return (yChannels as number[]).map((channel) => {
            const key = typedLineKeyToString({
              xAxis: xAxis as XAxisKey,
              yAxis: yAxis as YAxisKey,
              range: range.key,
              channels: { x: xChannel, y: channel },
            });
            const line = vis.lines.find((l) => l.key === key);
            if (line == null) throw new Error("Line not found");
            const v: Channel.LineProps = {
              ...line,
              key,
              axes: { x: xAxis, y: yAxis },
              channels: { x: xChannel, y: channel },
              ...variantArg,
            } as unknown as Channel.LineProps;
            return v;
          });
        }),
    ),
  );
