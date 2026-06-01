// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type lineplot } from "@synnaxlabs/client";
import { LinePlot as PLinePlot } from "@synnaxlabs/pluto";
import { useMemo } from "react";

import { X_AXIS_KEYS, type XAxisKey, type YAxisKey } from "@/lineplot/axis";
import { typedLineKeyToString } from "@/lineplot/slice";

// DEFAULT_LINE is the placeholder returned for a (channel x range) combo that
// has no stored styling override yet. A null color signals to the renderer
// that it should pick a palette color from the active theme.
const DEFAULT_LINE: Omit<lineplot.Line, "key"> = {
  label: undefined,
  color: undefined,
  strokeWidth: 2,
  downsample: 1,
  downsampleMode: "decimate",
};

export interface DerivedLine extends lineplot.Line {
  xAxis: XAxisKey;
  yAxis: YAxisKey;
  range: string;
  xChannel: channel.Key;
  yChannel: channel.Key;
}

// useDerivedLines returns the list of lines that should currently be plotted,
// derived from the cartesian product of channels and ranges, merged with any
// stored styling overrides. Each entry has a stable `key` that the SetLine
// action targets so overrides persist across re-derivations.
export const useDerivedLines = (key: lineplot.Key): DerivedLine[] => {
  const channels = PLinePlot.useSelectChannels({ key });
  const ranges = PLinePlot.useSelectRanges({ key });
  const lines = PLinePlot.useSelectLines({ key });
  return useMemo(() => {
    const out: DerivedLine[] = [];
    const overrides = new Map(lines.map((l) => [l.key, l]));
    for (const xAxis of X_AXIS_KEYS) {
      const xChannel = channels[xAxis];
      for (const range of ranges[xAxis])
        for (const yAxis of ["y1", "y2", "y3", "y4"] as const)
          for (const yChannel of channels[yAxis]) {
            const lineKey = typedLineKeyToString({
              xAxis,
              yAxis,
              range,
              channels: { x: xChannel, y: yChannel },
            });
            const override = overrides.get(lineKey);
            out.push({
              ...DEFAULT_LINE,
              ...override,
              key: lineKey,
              xAxis,
              yAxis,
              range,
              xChannel,
              yChannel,
            });
          }
    }
    return out;
  }, [channels, ranges, lines]);
};

export const useDerivedLineKeys = (key: lineplot.Key): string[] => {
  const channels = PLinePlot.useSelectChannels({ key });
  const ranges = PLinePlot.useSelectRanges({ key });
  return useMemo(() => {
    const out: string[] = [];
    for (const xAxis of X_AXIS_KEYS) {
      const xChannel = channels[xAxis];
      for (const range of ranges[xAxis])
        for (const yAxis of ["y1", "y2", "y3", "y4"] as const)
          for (const yChannel of channels[yAxis])
            out.push(
              typedLineKeyToString({
                xAxis,
                yAxis,
                range,
                channels: { x: xChannel, y: yChannel },
              }),
            );
    }
    return out;
  }, [channels, ranges]);
};

export const lookupDerivedLine = (
  lines: lineplot.Line[],
  key: string,
): lineplot.Line => ({ ...DEFAULT_LINE, ...lines.find((l) => l.key === key), key });
