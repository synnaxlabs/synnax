// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { LinePlot as PLinePlot, AxisProps, LineProps } from "@synnaxlabs/pluto";
import { TimeRange } from "@synnaxlabs/x";

import {
  LineVis,
  SugaredRangesState,
  useSelectLinevis,
  useSelectLineVisRanges,
} from "./core";
import "./LinePlot.css";

import { AxisKey, axisLocation, X_AXIS_KEYS, XAxisKey } from "@/vis/axis";

export const LinePlot = ({ layoutKey }: { layoutKey: string }): ReactElement => {
  const vis = useSelectLinevis(layoutKey);
  const ranges = useSelectLineVisRanges(layoutKey);

  // return <PLinePlot lines={buildLines(vis, ranges)} axes={buildAxes(vis)} />;
  return null;
};

export const buildAxes = (vis: LineVis): AxisProps[] =>
  Object.entries(vis.axes)
    .map(([key, axis]): AxisProps | null => {
      const channels = vis.channels[key as AxisKey];
      if ((Array.isArray(channels) && channels.length > 0) || channels === 0)
        return null;
      return {
        id: key,
        location: axisLocation(key as AxisKey),
      };
    })
    .filter((axis: AxisProps | null): boolean => axis !== null) as AxisProps[];

export const buildLines = (vis: LineVis, sug: SugaredRangesState): LineProps[] =>
  Object.entries(sug).map(([xAxis, ranges]) =>
    ranges.map((range) =>
      Object.entries(vis.channels)
        .filter(([axis]) => !X_AXIS_KEYS.includes(axis as XAxisKey))
        .map(([yAxis, yChannels]) => {
          const xChannel = vis.channels[xAxis as XAxisKey];
          return (yChannels as number[]).map((channel) => {
            const v: LineProps = {
              variant: "static",
              axes: {
                x: xAxis,
                y: yAxis,
              },
              channels: {
                x: xChannel,
                y: channel,
              },
              range: new TimeRange(range.start, range.end),
            };
            return v;
          });
        })
    )
  );
