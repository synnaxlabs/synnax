// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback, useMemo, useEffect } from "react";

import { type channel } from "@synnaxlabs/client";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import {
  useAsyncEffect,
  Viewport,
  useDebouncedCallback,
  Channel,
  Synnax,
  Color,
  Menu,
  usePrevious,
} from "@synnaxlabs/pluto";
import { type UnknownRecord, box, location, unique, getEntries } from "@synnaxlabs/x";
import { useDispatch } from "react-redux";

import { useSyncerDispatch, type Syncer } from "@/hooks/dispatchers";
import { Layout } from "@/layout";
import { ContextMenuContent } from "@/lineplot/ContextMenu";
import {
  useSelect,
  selectRanges,
  useSelectControlState,
  useSelectViewportMode,
  select,
} from "@/lineplot/selectors";
import {
  type State,
  setLine,
  setRanges,
  setRule,
  setXChannel,
  setYChannels,
  shouldDisplayAxis,
  storeViewport,
  typedLineKeyToString,
  setRemoteCreated,
  type StoreState,
  internalCreate,
  setSelection,
  setAxis,
  type AxisState,
  type LineState,
} from "@/lineplot/slice";
import { Range } from "@/range";
import { Vis } from "@/vis";
import { Workspace } from "@/workspace";

interface SyncPayload {
  key?: string;
}

const syncer: Syncer<
  Layout.StoreState & StoreState & Workspace.StoreState,
  SyncPayload
> = async (client, { key }, store) => {
  if (key == null) return;
  const s = store.getState();
  const ws = Workspace.selectActiveKey(s);
  if (ws == null) return;
  const data = select(s, key);
  const la = Layout.selectRequired(s, key);
  if (!data.remoteCreated) {
    store.dispatch(setRemoteCreated({ key }));
  }
  await client.workspaces.linePlot.create(ws, {
    key,
    name: la.name,
    data: data as unknown as UnknownRecord,
  });
};

const Loaded = ({ layoutKey }: { layoutKey: string }): ReactElement => {
  const windowKey = useSelectWindowKey() as string;
  const { name } = Layout.useSelectRequired(layoutKey);
  const vis = useSelect(layoutKey);
  const ranges = selectRanges(layoutKey);
  const client = Synnax.use();
  const syncDispatch = useSyncerDispatch<
    Layout.StoreState & Workspace.StoreState & StoreState,
    SyncPayload
  >(syncer, 500);
  const dispatch = useDispatch();

  const lines = buildLines(vis, ranges);

  const prevName = usePrevious(name);
  useEffect(() => {
    if (prevName !== name) syncDispatch(Layout.rename({ key: layoutKey, name }));
  }, [syncDispatch, name, prevName]);

  useAsyncEffect(async () => {
    if (client == null) return;
    const toFetch = lines.filter((line) => line.label == null);
    if (toFetch.length === 0) return;
    const fetched = await client.channels.retrieve(
      unique(toFetch.map((line) => line.channels.y)),
    );
    const update = toFetch.map((l) => ({
      key: l.key,
      label: fetched.find((f) => f.key === l.channels.y)?.name,
    }));
    syncDispatch(
      setLine({
        key: layoutKey,
        line: update,
      }),
    );
  }, [layoutKey, client, lines]);

  const handleTitleChange = (name: string): void => {
    syncDispatch(Layout.rename({ key: layoutKey, name }));
  };

  const handleLineChange = useCallback<
    Exclude<Channel.LinePlotProps["onLineChange"], undefined>
  >(
    (d): void => {
      const newLine = { ...d } as const as LineState;
      if (d.color != null) newLine.color = Color.toHex(d.color);
      syncDispatch(
        setLine({
          key: layoutKey,
          line: [newLine],
        }),
      );
    },
    [syncDispatch, layoutKey],
  );

  const handleRuleChange = useCallback<
    Exclude<Channel.LinePlotProps["onRuleChange"], undefined>
  >(
    (rule) =>
      syncDispatch(
        setRule({
          key: layoutKey,
          rule: {
            ...rule,
            axis: rule.axis as Vis.XAxisKey,
            color: Color.toHex(rule.color),
          },
        }),
      ),
    [syncDispatch, layoutKey],
  );

  const handleAxisChange = useCallback<
    Exclude<Channel.LinePlotProps["onAxisChange"], undefined>
  >(
    (axis) => {
      syncDispatch(
        setAxis({
          key: layoutKey,
          axisKey: axis.key as Vis.AxisKey,
          axis: axis as AxisState,
          triggerRender: false,
        }),
      );
    },
    [syncDispatch, layoutKey],
  );

  const propsLines = buildLines(vis, ranges);
  const axes = useMemo(() => buildAxes(vis), [vis.axes.renderTrigger]);
  const rng = Range.useSelect();

  const handleChannelAxisDrop = useCallback(
    (axis: string, channels: channel.Keys): void => {
      if (Vis.X_AXIS_KEYS.includes(axis as Vis.XAxisKey))
        syncDispatch(
          setXChannel({
            key: layoutKey,
            axisKey: axis as Vis.XAxisKey,
            channel: channels[0],
          }),
        );
      else
        syncDispatch(
          setYChannels({
            key: layoutKey,
            axisKey: axis as Vis.YAxisKey,
            channels,
            mode: "add",
          }),
        );
      if (propsLines.length === 0 && rng != null) {
        syncDispatch(
          setRanges({
            mode: "add",
            key: layoutKey,
            axisKey: "x1",
            ranges: [rng.key],
          }),
        );
      }
    },
    [syncDispatch, layoutKey, propsLines.length, rng],
  );

  const handleViewportChange: Viewport.UseHandler = useDebouncedCallback(
    ({ box: b, stage, mode }) => {
      if (stage !== "end") return;
      if (mode === "select") {
        syncDispatch(
          setSelection({
            key: layoutKey,
            box: b,
          }),
        );
      } else {
        syncDispatch(
          storeViewport({
            key: layoutKey,
            pan: box.bottomLeft(b),
            zoom: box.dims(b),
          }),
        );
      }
    },
    100,
    [syncDispatch, layoutKey],
  );

  const { enableTooltip, clickMode, hold } = useSelectControlState();
  const mode = useSelectViewportMode();
  const triggers = useMemo(() => Viewport.DEFAULT_TRIGGERS[mode], [mode]);

  const initialViewport = useMemo(() => {
    return box.reRoot(
      box.construct(vis.viewport.pan, vis.viewport.zoom),
      location.BOTTOM_LEFT,
    );
  }, [vis.viewport.renderTrigger]);

  const handleDoubleClick = useCallback(
    () =>
      dispatch(
        Layout.setNavdrawerVisible({ windowKey, key: "visualization", value: true }),
      ),
    [windowKey, dispatch],
  );

  const props = Menu.useContextMenu();

  return (
    <Menu.ContextMenu
      {...props}
      menu={() => <ContextMenuContent layoutKey={layoutKey} />}
    >
      <div style={{ height: "100%", width: "100%", padding: "2rem" }}>
        <Channel.LinePlot
          hold={hold}
          title={name}
          axes={axes}
          lines={propsLines}
          rules={vis.rules}
          clearOverscan={{ x: 5, y: 5 }}
          onTitleChange={handleTitleChange}
          titleLevel={vis.title.level}
          showTitle={vis.title.visible}
          showLegend={vis.legend.visible}
          onLineChange={handleLineChange}
          onRuleChange={handleRuleChange}
          onAxisChannelDrop={handleChannelAxisDrop}
          onAxisChange={handleAxisChange}
          onViewportChange={handleViewportChange}
          initialViewport={initialViewport}
          viewportTriggers={triggers}
          enableTooltip={enableTooltip}
          enableMeasure={clickMode === "measure"}
          onDoubleClick={handleDoubleClick}
        />
      </div>
    </Menu.ContextMenu>
  );
};

const buildAxes = (vis: State): Channel.AxisProps[] =>
  getEntries(vis.axes.axes)
    .filter(([key]) => shouldDisplayAxis(key, vis))
    .map(([key, axis]): Channel.AxisProps => {
      return {
        location: Vis.axisLocation(key),
        type: Vis.X_AXIS_KEYS.includes(key as Vis.XAxisKey) ? "time" : "linear",
        ...axis,
      };
    });

const buildLines = (
  vis: State,
  sug: Vis.MultiXAxisRecord<Range.Range>,
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
                  timeSpan: range.span,
                }
              : {
                  variant: "static",
                  timeRange: range.timeRange,
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
            const v: Channel.LineProps = {
              ...line,
              key,
              axes: {
                x: xAxis,
                y: yAxis,
              },
              channels: {
                x: xChannel,
                y: channel,
              },
              ...variantArg,
            } as unknown as Channel.LineProps;
            return v;
          });
        }),
    ),
  );

export const LinePlot: Layout.Renderer = ({
  layoutKey,
  ...props
}): ReactElement | null => {
  const linePlot = useSelect(layoutKey);
  const dispatch = useDispatch();
  const client = Synnax.use();
  useAsyncEffect(async () => {
    if (client == null || linePlot != null) return;
    const { data } = await client.workspaces.linePlot.retrieve(layoutKey);
    dispatch(internalCreate({ ...(data as unknown as State) }));
  }, [client, linePlot]);
  if (linePlot == null) return null;
  return <Loaded layoutKey={layoutKey} {...props} />;
};
