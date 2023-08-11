// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback, useMemo } from "react";

import {
  LinePlot as PLinePlot,
  AxisProps,
  LineProps,
  Client,
  useAsyncEffect,
  Color,
  RuleProps,
  Viewport,
  UseViewportHandler,
  useDebouncedCallback,
} from "@synnaxlabs/pluto";
import { Box, TimeRange, unique } from "@synnaxlabs/x";
import { useDispatch } from "react-redux";

import { renameLayout, useSelectRequiredLayout } from "@/layout";
import {
  useSelectLinePlot,
  useSelectLinePlotRanges,
  useSelectLineControlState,
} from "@/line/store/selectors";
import {
  LinePlotState,
  setLinePlotLine,
  setLinePlotRule,
  shouldDisplayAxis,
  storeLinePlotViewport,
  typedLineKeyToString,
} from "@/line/store/slice";
import {
  AxisKey,
  axisLocation,
  MultiXAxisRecord,
  X_AXIS_KEYS,
  XAxisKey,
  YAxisKey,
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

  const handleLineLabelChange = useCallback(
    (key: string, label: string): void => {
      dispatch(setLinePlotLine({ key: layoutKey, line: [{ key, label }] }));
    },
    [dispatch, layoutKey]
  );

  const handleLineColorChange = useCallback(
    (key: string, color: Color): void => {
      dispatch(setLinePlotLine({ key: layoutKey, line: [{ key, color: color.hex }] }));
    },
    [dispatch, layoutKey]
  );

  const handleRulePositionChange = useCallback(
    (key: string, position: number): void => {
      dispatch(
        setLinePlotRule({
          key: layoutKey,
          rule: {
            key,
            position,
          },
        })
      );
    },
    [dispatch, layoutKey]
  );

  const handleRuleLabelChange = useCallback(
    (key: string, label: string): void => {
      dispatch(
        setLinePlotRule({
          key: layoutKey,
          rule: {
            key,
            label,
          },
        })
      );
    },
    [dispatch, layoutKey]
  );

  const handleViewportChange: UseViewportHandler = useDebouncedCallback(
    ({ box, stage }) => {
      if (stage !== "end") return;
      dispatch(
        storeLinePlotViewport({
          layoutKey,
          pan: box.bottomLeft.crude,
          zoom: box.dims.crude,
        })
      );
    },
    100,
    [dispatch, layoutKey]
  );

  const rules = buildRules(vis);
  const propsLines = buildLines(vis, ranges);
  const axes = buildAxes(vis);

  const { mode, enableTooltip, clickMode } = useSelectLineControlState();
  const triggers = useMemo(() => Viewport.DEFAULT_TRIGGERS[mode], [mode]);

  const initialViewport = useMemo(
    () =>
      new Box(vis.viewport.pan, vis.viewport.zoom).reRoot({ x: "left", y: "bottom" }),
    [vis.viewport.counter]
  );

  return (
    <div style={{ height: "100%", width: "100%", padding: "2rem" }}>
      <PLinePlot
        title={name}
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
        initialViewport={initialViewport}
        onViewportChange={handleViewportChange}
        viewportTriggers={triggers}
        enableTooltip={enableTooltip}
        enableMeasure={clickMode === "measure"}
      />
    </div>
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
        type: X_AXIS_KEYS.includes(key as XAxisKey) ? "time" : "linear",
        bounds: axis.bounds,
        labelDirection: axis.labelDirection,
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
          const variantArg = range.variant == "dynamic" ? {
            variant: range.variant,
            span: range.span,
          } : {
            variant: range.variant,
            timeRange: range.timeRange
          };


          return (yChannels as number[]).map((channel) => {
            const key = typedLineKeyToString({
              xAxis: xAxis as XAxisKey,
              yAxis: yAxis as YAxisKey,
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
              ...variantArg
            };
            return v;
          });
        })
    )
  );
