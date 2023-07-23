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
  Color,
  RuleProps,
} from "@synnaxlabs/pluto";
import { TimeRange, unique } from "@synnaxlabs/x";
import { useDispatch } from "react-redux";

import { renameLayout, useSelectRequiredLayout } from "@/layout";
import { useSelectLinePlot, useSelectLinePlotRanges } from "@/line/store/selectors";
import {
  LinePlotState,
  setLinePlotLine,
  setLinePlotRule,
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

export const LinePlot = ({ layoutKey }: { layoutKey: string }): ReactElement => {
  const { name } = useSelectRequiredLayout(layoutKey);
  const vis = useSelectLinePlot(layoutKey);
  const ranges = useSelectLinePlotRanges(layoutKey);
  const client = Client.use();
  const dispatch = useDispatch();

  const lines = buildLines(vis, ranges);

  useAsyncEffect(async () => {
    if (client == null) return;
    const toFetch = lines.filter((line) => line.label == null);
    if (toFetch.length === 0) return;
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

  const handleTitleRename = (name: string): void => {
    dispatch(renameLayout({ key: layoutKey, name }));
  };

  const handleLineLabelChange = (key: string, label: string): void => {
    dispatch(setLinePlotLine({ key: layoutKey, line: [{ key, label }] }));
  };

  const handleLineColorChange = (key: string, color: Color): void => {
    dispatch(setLinePlotLine({ key: layoutKey, line: [{ key, color: color.hex }] }));
  };

  const handleRulePositionChange = (key: string, position: number): void => {
    dispatch(
      setLinePlotRule({
        key: layoutKey,
        rule: {
          key,
          position,
        },
      })
    );
  };

  const handleRuleLabelChange = (key: string, label: string): void => {
    dispatch(
      setLinePlotRule({
        key: layoutKey,
        rule: {
          key,
          label,
        },
      })
    );
  };

  const rules = buildRules(vis);
  const propsLines = buildLines(vis, ranges);
  const axes = buildAxes(vis);

  return (
    <PLinePlot
      title={name}
      style={{ padding: "2rem" }}
      axes={axes}
      lines={propsLines}
      rules={rules}
      clearOverscan={{ x: 5, y: 10 }}
      onTitleChange={handleTitleRename}
      titleLevel={vis.title.level}
      showTitle={vis.title.visible}
      showLegend={vis.legend.visible}
      onLineColorChange={handleLineColorChange}
      onLineLabelChange={handleLineLabelChange}
      onRulePositionChange={handleRulePositionChange}
      onRuleLabelChange={handleRuleLabelChange}
    />
  );
};

const buildRules = (vis: LinePlotState): RuleProps[] =>
  vis.rules?.map((rule) => ({
    id: rule.key,
    ...rule,
  }));

const buildAxes = (vis: LinePlotState): AxisProps[] =>
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

const buildLines = (
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
              id: key,
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
