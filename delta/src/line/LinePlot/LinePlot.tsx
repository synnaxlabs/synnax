// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import {
  LinePlot as PLinePlot,
  AxisProps,
  LineProps,
  Client,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { TimeRange, TimeSpan, unique } from "@synnaxlabs/x";
import { useDispatch } from "react-redux";

import { useSelectLinePlot, useSelectLinePlotRanges } from "@/line/store/selectors";
import {
  LinePlotState,
  setLinePlotLine,
  shouldDisplayAxis,
  typedLineKeyToString,
} from "@/line/store/slice";
import {
  AxisKey,
  axisLocation,
  MultiXAxisRecord,
  X_AXIS_KEYS,
  XAxisKey,
} from "@/vis/axis";
import { Range } from "@/workspace";

import "./LinePlot.css";

export const LinePlot = ({ layoutKey }: { layoutKey: string }): ReactElement => {
  const vis = useSelectLinePlot(layoutKey);
  const ranges = useSelectLinePlotRanges(layoutKey);
  const client = Client.use();
  const dispatch = useDispatch();

  const lines = buildLines(vis, ranges);

  useAsyncEffect(async () => {
    if (client == null) return;
    const toFetch = lines.filter((line) => line.label == null);
    const fetched = await client.channels.retrieve(
      unique(toFetch.map((line) => line.channels.y))
    );
    const update = toFetch.map((l) => ({
      key: l.key,
      label: fetched.find((f) => f.key === l.channels.y)?.name,
    }));
    dispatch(
      setLinePlotLine({
        key: layoutKey,
        line: update,
      })
    );
  }, [client, lines]);

  return (
    <PLinePlot
      style={{ padding: "2rem" }}
      axes={buildAxes(vis)}
      lines={buildLines(vis, ranges)}
      clearOverscan={{ x: 5, y: 10 }}
    />
  );
};

export const buildAxes = (vis: LinePlotState): AxisProps[] =>
  Object.entries(vis.axes)
    .filter(([key, axis]) => shouldDisplayAxis(key as AxisKey, vis))
    .map(([key, axis]): AxisProps => {
      return {
        id: key,
        location: axisLocation(key as AxisKey),
        label: axis.label,
        type: "time",
        bounds: axis.bounds,
      };
    });

export const buildLines = (
  vis: LinePlotState,
  sug: MultiXAxisRecord<Range>
): Array<LineProps & { key: string }> =>
  Object.entries(sug).flatMap(([xAxis, ranges]) =>
    ranges.flatMap((range) =>
      Object.entries(vis.channels)
        .filter(([axis]) => !X_AXIS_KEYS.includes(axis as XAxisKey))
        .flatMap(([yAxis, yChannels]) => {
          const xChannel = vis.channels[xAxis as XAxisKey];
          return (yChannels as number[]).map((channel) => {
            const key = typedLineKeyToString({
              xAxis: xAxis as XAxisKey,
              yAxis: yAxis as AxisKey,
              range: range.key,
              channels: {
                x: xChannel,
                y: channel,
              },
            });
            const line = vis.lines.find((l) => l.key === key);
            if (line == null) throw new Error("Line not found");
            const v: LineProps & { key: string } = {
              ...line,
              downsample:
                isNaN(line.downsample) || line.downsample == null ? 1 : line.downsample,
              strokeWidth:
                isNaN(line.strokeWidth) ||
                line.strokeWidth == null ||
                line.strokeWidth === 0
                  ? 1
                  : line.strokeWidth,
              variant: "static",
              key,
              color: line.color === "" ? "#000000" : line.color,
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
