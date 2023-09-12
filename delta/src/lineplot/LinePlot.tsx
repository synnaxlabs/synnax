// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback, useMemo } from "react";

import { type channel } from "@synnaxlabs/client";
import {
  useAsyncEffect,
  Viewport,
  useDebouncedCallback,
  Channel,
  Synnax,
  type Color,
} from "@synnaxlabs/pluto";
import { box, location, unique } from "@synnaxlabs/x";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";
import {
  useSelect,
  selectRanges,
  useSelectControlState,
  useSelectViewportMode,
} from "@/lineplot/selectors";
import {
  type State,
  type RuleState,
  setLine,
  setRanges,
  setRule,
  setXChannel,
  setYChannels,
  shouldDisplayAxis,
  storeViewport,
  typedLineKeyToString,
} from "@/lineplot/slice";
import { Vis } from "@/vis";
import { Workspace } from "@/workspace";

export const LinePlot = ({ layoutKey }: { layoutKey: string }): ReactElement => {
  const { name } = Layout.useSelectRequired(layoutKey);
  const vis = useSelect(layoutKey);
  const ranges = selectRanges(layoutKey);
  const client = Synnax.use();
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
      setLine({
        key: layoutKey,
        line: update,
      })
    );
  }, [client, lines]);

  const handleTitleRename = (name: string): void => {
    dispatch(Layout.rename({ key: layoutKey, name }));
  };

  const handleLineLabelChange = useCallback(
    (key: string, label: string): void => {
      dispatch(setLine({ key: layoutKey, line: [{ key, label }] }));
    },
    [dispatch, layoutKey]
  );

  const handleLineColorChange = useCallback(
    (key: string, color: Color.Color): void => {
      dispatch(setLine({ key: layoutKey, line: [{ key, color: color.hex }] }));
    },
    [dispatch, layoutKey]
  );

  const handleRulePositionChange = useCallback(
    (key: string, position: number): void => {
      dispatch(
        setRule({
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

  const rules = useMemo(() => buildRules(vis?.rules ?? []), [vis.rules]);
  const propsLines = buildLines(vis, ranges);
  const axes = buildAxes(vis);
  const rng = Workspace.useSelectRange();

  const handleChannelAxisDrop = useCallback(
    (axis: string, channels: channel.Keys): void => {
      if (Vis.X_AXIS_KEYS.includes(axis as Vis.XAxisKey))
        dispatch(
          setXChannel({
            key: layoutKey,
            axisKey: axis as Vis.XAxisKey,
            channel: channels[0],
          })
        );
      else
        dispatch(
          setYChannels({
            key: layoutKey,
            axisKey: axis as Vis.YAxisKey,
            channels,
            mode: "add",
          })
        );
      if (propsLines.length === 0 && rng != null) {
        dispatch(
          setRanges({
            mode: "add",
            key: layoutKey,
            axisKey: "x1",
            ranges: [rng.key],
          })
        );
      }
    },
    [dispatch, layoutKey, propsLines.length, rng]
  );

  const handleRuleLabelChange = useCallback(
    (key: string, label: string): void => {
      dispatch(
        setRule({
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

  const handleViewportChange: Viewport.UseHandler = useDebouncedCallback(
    ({ box: b, stage }) => {
      if (stage !== "end") return;
      dispatch(
        storeViewport({
          layoutKey,
          pan: box.bottomLeft(b),
          zoom: box.dims(b),
        })
      );
    },
    100,
    [dispatch, layoutKey]
  );

  const { enableTooltip, clickMode, hold } = useSelectControlState();
  const mode = useSelectViewportMode();
  const triggers = useMemo(() => Viewport.DEFAULT_TRIGGERS[mode], [mode]);

  const initialViewport = useMemo(() => {
    return box.reRoot(
      box.construct(vis.viewport.pan, vis.viewport.zoom),
      location.BOTTOM_LEFT
    );
  }, [vis.viewport.counter]);

  return (
    <div style={{ height: "100%", width: "100%", padding: "2rem" }}>
      <Channel.LinePlot
        hold={hold}
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
        onAxisChannelDrop={handleChannelAxisDrop}
        onViewportChange={handleViewportChange}
        initialViewport={initialViewport}
        viewportTriggers={triggers}
        enableTooltip={enableTooltip}
        enableMeasure={clickMode === "measure"}
      />
    </div>
  );
};

const buildRules = (rules: RuleState[]): Channel.RuleProps[] =>
  rules.map((rule) => ({
    id: rule.key,
    ...rule,
  }));

const buildAxes = (vis: State): Channel.AxisProps[] =>
  Object.entries(vis.axes)
    .filter(([key, axis]) => shouldDisplayAxis(key as Vis.AxisKey, vis))
    .map(([key, axis]): Channel.AxisProps => {
      return {
        id: key,
        location: Vis.axisLocation(key as Vis.AxisKey),
        label: axis.label,
        type: Vis.X_AXIS_KEYS.includes(key as Vis.XAxisKey) ? "time" : "linear",
        bounds: axis.bounds,
        labelDirection: axis.labelDirection,
      };
    });

const buildLines = (
  vis: State,
  sug: Vis.MultiXAxisRecord<Workspace.Range>
): Array<Channel.LineProps & { key: string }> =>
  Object.entries(sug).flatMap(([xAxis, ranges]) =>
    ranges.flatMap((range) =>
      Object.entries(vis.channels)
        .filter(([axis]) => !Vis.X_AXIS_KEYS.includes(axis as Vis.XAxisKey))
        .flatMap(([yAxis, yChannels]) => {
          const xChannel = vis.channels[xAxis as Vis.XAxisKey];
          const variantArg =
            range.variant === "dynamic"
              ? {
                  variant: "dynamic",
                  span: range.span,
                }
              : {
                  variant: "static",
                  range: range.timeRange,
                };

          return (yChannels as number[]).map((channel) => {
            const key = typedLineKeyToString({
              xAxis: xAxis as Vis.XAxisKey,
              yAxis: yAxis as Vis.YAxisKey,
              range: range.key,
              channels: {
                x: xChannel,
                y: channel,
              },
            });
            const line = vis.lines.find((l) => l.key === key);
            if (line == null) throw new Error("Line not found");
            const v: Channel.LineProps & { key: string } = {
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
              ...variantArg,
            };
            return v;
          });
        })
    )
  );
