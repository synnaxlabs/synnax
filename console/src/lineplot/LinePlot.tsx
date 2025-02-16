// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch, type PayloadAction } from "@reduxjs/toolkit";
import { type channel } from "@synnaxlabs/client";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Icon } from "@synnaxlabs/media";
import {
  type axis,
  Channel,
  Color,
  type Legend,
  Menu as PMenu,
  Status,
  Synnax,
  useAsyncEffect,
  useDebouncedCallback,
  usePrevious,
  Viewport,
} from "@synnaxlabs/pluto";
import {
  box,
  DataType,
  deep,
  getEntries,
  location,
  primitiveIsZero,
  scale,
  TimeRange,
  unique,
} from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement, useCallback, useEffect, useMemo, useState } from "react";
import { useDispatch } from "react-redux";
import { v4 as uuid } from "uuid";

import { Menu } from "@/components/menu";
import { useLoadRemote } from "@/hooks/useLoadRemote";
import { Layout } from "@/layout";
import {
  type AxisKey,
  axisLocation,
  type MultiXAxisRecord,
  X_AXIS_KEYS,
  type XAxisKey,
  type YAxisKey,
} from "@/lineplot/axis";
import { download } from "@/lineplot/download";
import {
  select,
  useSelect,
  useSelectAxisBounds,
  useSelectControlState,
  useSelectRanges,
  useSelectSelection,
  useSelectVersion,
  useSelectViewportMode,
} from "@/lineplot/selectors";
import {
  type AxesState,
  type AxisState,
  internalCreate,
  type LineState,
  selectRule,
  setActiveToolbarTab,
  setAxis,
  setControlState,
  setLegend,
  setLine,
  setRanges,
  setRemoteCreated,
  setRule,
  setSelection,
  setXChannel,
  setYChannels,
  shouldDisplayAxis,
  type State,
  storeViewport,
  typedLineKeyToString,
  ZERO_STATE,
} from "@/lineplot/slice";
import { Range } from "@/range";
import { Workspace } from "@/workspace";

interface SyncPayload {
  key?: string;
}

export const ContextMenu: Layout.ContextMenuRenderer = ({ layoutKey }) => (
  <PMenu.Menu level="small" iconSpacing="small">
    <Layout.MenuItems layoutKey={layoutKey} />
  </PMenu.Menu>
);

const useSyncComponent = (layoutKey: string): Dispatch<PayloadAction<SyncPayload>> =>
  Workspace.useSyncComponent<SyncPayload>(
    "Line Plot",
    layoutKey,
    async (ws, store, client) => {
      const s = store.getState();
      const data = select(s, layoutKey);
      if (data == null) return;
      const la = Layout.selectRequired(s, layoutKey);
      if (!data.remoteCreated) store.dispatch(setRemoteCreated({ key: layoutKey }));
      await client.workspaces.linePlot.create(ws, {
        key: layoutKey,
        name: la.name,
        data,
      });
    },
  );

const Loaded: Layout.Renderer = ({ layoutKey, focused, visible }): ReactElement => {
  const windowKey = useSelectWindowKey() as string;
  const { name } = Layout.useSelectRequired(layoutKey);
  const place = Layout.usePlacer();
  const vis = useSelect(layoutKey);
  const prevVis = usePrevious(vis);
  const ranges = useSelectRanges(layoutKey);
  const client = Synnax.use();
  const dispatch = useDispatch();
  const syncDispatch = useSyncComponent(layoutKey);
  const lines = buildLines(vis, ranges);
  const prevName = usePrevious(name);
  const handleException = Status.useExceptionHandler();

  useEffect(() => {
    if (prevName !== name) syncDispatch(Layout.rename({ key: layoutKey, name }));
  }, [syncDispatch, name, prevName]);

  useAsyncEffect(async () => {
    if (client == null) return;
    const toFetch = lines.filter((line) => line.label == null);
    if (toFetch.length === 0) return;
    const fetched = await client.channels.retrieve(
      unique.unique(toFetch.map((line) => line.channels.y)) as channel.KeysOrNames,
    );
    const update = toFetch.map((l) => ({
      key: l.key,
      label: fetched.find((f) => f.key === l.channels.y)?.name,
    }));
    syncDispatch(setLine({ key: layoutKey, line: update }));
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
      syncDispatch(setLine({ key: layoutKey, line: [newLine] }));
    },
    [syncDispatch, layoutKey],
  );

  const handleRuleChange = useCallback<
    Exclude<Channel.LinePlotProps["onRuleChange"], undefined>
  >(
    (rule) => {
      if (rule.color != null) rule.color = Color.toHex(rule.color);
      syncDispatch(
        setRule({
          key: layoutKey,
          // @ts-expect-error rule.color was reassigned to be a string or undefined
          rule,
        }),
      );
    },
    [syncDispatch, layoutKey],
  );

  const handleAxisChange = useCallback(
    (axis: Partial<Channel.AxisProps> & { key: string }) => {
      syncDispatch(
        setAxis({
          key: layoutKey,
          axisKey: axis.key as AxisKey,
          axis: axis as AxisState,
          triggerRender: true,
        }),
      );
    },
    [syncDispatch, layoutKey],
  );

  const xAxisChannelChange = useMutation<
    void,
    Error,
    Omit<Channel.AxisProps, "location">
  >({
    mutationFn: async (axis) => {
      const key = vis.channels[axis.key as XAxisKey];
      const prevKey = prevVis?.channels[axis.key as XAxisKey];
      if (client == null || key === prevKey) return;
      let newType: axis.TickType = "time";
      if (!primitiveIsZero(key)) {
        const ch = await client.channels.retrieve(key);
        if (!ch.dataType.equals(DataType.TIMESTAMP)) newType = "linear";
      }
      if (axis.type === newType) return;
      syncDispatch(
        setAxis({
          key: layoutKey,
          axisKey: axis.key as AxisKey,
          axis: { ...(axis as AxisState), type: newType },
          triggerRender: true,
        }),
      );
    },
  });
  useEffect(() => {
    xAxisChannelChange.mutate(vis.axes.axes.x1);
  }, [vis.channels.x1]);

  const propsLines = buildLines(vis, ranges);
  const axes = useMemo(() => buildAxes(vis), [vis.axes.renderTrigger]);
  const rng = Range.useSelect();

  const handleChannelAxisDrop = useCallback(
    (axis: string, channels: channel.Keys): void => {
      if (X_AXIS_KEYS.includes(axis as XAxisKey))
        syncDispatch(
          setXChannel({
            key: layoutKey,
            axisKey: axis as XAxisKey,
            channel: channels[0],
          }),
        );
      else
        syncDispatch(
          setYChannels({
            key: layoutKey,
            axisKey: axis as YAxisKey,
            channels,
            mode: "add",
          }),
        );
      if (propsLines.length === 0 && rng != null)
        syncDispatch(
          setRanges({ mode: "add", key: layoutKey, axisKey: "x1", ranges: [rng.key] }),
        );
    },
    [syncDispatch, layoutKey, propsLines.length, rng],
  );

  const handleViewportChange: Viewport.UseHandler = useDebouncedCallback(
    ({ box: b, stage, mode }) => {
      if (stage !== "end") return;
      if (mode === "select") syncDispatch(setSelection({ key: layoutKey, box: b }));
      else
        syncDispatch(
          storeViewport({ key: layoutKey, pan: box.bottomLeft(b), zoom: box.dims(b) }),
        );
    },
    100,
    [syncDispatch, layoutKey],
  );

  const [legendPosition, setLegendPosition] = useState(vis.legend.position);

  const storeLegendPosition = useDebouncedCallback(
    (position: Legend.StickyXY) =>
      syncDispatch(setLegend({ key: layoutKey, legend: { position } })),
    100,
    [syncDispatch, layoutKey],
  );

  const handleLegendPositionChange = useCallback(
    (position: Legend.StickyXY) => {
      setLegendPosition(position);
      storeLegendPosition(position);
    },
    [storeLegendPosition],
  );

  const { enableTooltip, clickMode, hold } = useSelectControlState();
  const mode = useSelectViewportMode();
  const triggers = useMemo(() => Viewport.DEFAULT_TRIGGERS[mode], [mode]);

  const initialViewport = useMemo(
    () =>
      box.reRoot(
        box.construct(vis.viewport.pan, vis.viewport.zoom),
        location.BOTTOM_LEFT,
      ),
    [vis.viewport.renderTrigger],
  );

  const handleDoubleClick = useCallback(() => {
    dispatch(
      Layout.setNavDrawerVisible({ windowKey, key: "visualization", value: true }),
    );
    dispatch(setActiveToolbarTab({ tab: "data" }));
  }, [windowKey, dispatch]);

  const props = PMenu.useContextMenu();

  interface ContextMenuContentProps {
    layoutKey: string;
  }

  const ContextMenuContent = ({ layoutKey }: ContextMenuContentProps): ReactElement => {
    const { box: selection } = useSelectSelection(layoutKey);
    const bounds = useSelectAxisBounds(layoutKey, "x1");
    const s = scale.Scale.scale<number>(1).scale(bounds);
    const place = Layout.usePlacer();

    const timeRange = new TimeRange(
      s.pos(box.left(selection)),
      s.pos(box.right(selection)),
    );

    const handleSelect = (key: string): void => {
      switch (key) {
        case "iso":
          void navigator.clipboard.writeText(
            `${timeRange.start.fString("ISO")} - ${timeRange.end.fString("ISO")}`,
          );
          break;
        case "python":
          void navigator.clipboard.writeText(
            `sy.TimeRange(${timeRange.start.valueOf()}, ${timeRange.end.valueOf()})`,
          );
          break;
        case "typescript":
          void navigator.clipboard.writeText(
            `new TimeRange(${timeRange.start.valueOf()}, ${timeRange.end.valueOf()})`,
          );
          break;
        case "range":
          place(
            Range.createCreateLayout({
              timeRange: {
                start: Number(timeRange.start.valueOf()),
                end: Number(timeRange.end.valueOf()),
              },
            }),
          );
          break;
        case "download":
          if (client == null) return;
          download({ timeRange, lines, client, name: `${name}-data`, handleException });
          break;
      }
    };

    return (
      <PMenu.Menu onChange={handleSelect} iconSpacing="small" level="small">
        {!box.areaIsZero(selection) && (
          <>
            <PMenu.Item itemKey="iso" startIcon={<Icon.Range />}>
              Copy ISO Time Range
            </PMenu.Item>
            <PMenu.Item itemKey="python" startIcon={<Icon.Python />}>
              Copy Python Time Range
            </PMenu.Item>
            <PMenu.Item itemKey="typescript" startIcon={<Icon.TypeScript />}>
              Copy TypeScript Time Range
            </PMenu.Item>
            <PMenu.Divider />
            <PMenu.Item itemKey="range" startIcon={<Icon.Add />}>
              Create Range from Selection
            </PMenu.Item>
            <PMenu.Divider />
            <PMenu.Item itemKey="download" startIcon={<Icon.Download />}>
              Download as CSV
            </PMenu.Item>
          </>
        )}
        <Menu.HardReloadItem />
      </PMenu.Menu>
    );
  };
  const addRangeToNewPlot = Range.useAddToNewPlot();
  return (
    <PMenu.ContextMenu
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
          clearOverScan={{ x: 5, y: 5 }}
          onTitleChange={handleTitleChange}
          visible={visible}
          titleLevel={vis.title.level}
          showTitle={vis.title.visible}
          showLegend={vis.legend.visible}
          onLineChange={handleLineChange}
          onRuleChange={handleRuleChange}
          onAxisChannelDrop={handleChannelAxisDrop}
          onAxisChange={handleAxisChange}
          onViewportChange={handleViewportChange}
          initialViewport={initialViewport}
          onLegendPositionChange={handleLegendPositionChange}
          legendPosition={legendPosition}
          viewportTriggers={triggers}
          enableTooltip={enableTooltip}
          legendVariant={focused ? "fixed" : "floating"}
          enableMeasure={clickMode === "measure"}
          onDoubleClick={handleDoubleClick}
          onSelectRule={(ruleKey) => dispatch(selectRule({ key: layoutKey, ruleKey }))}
          onHold={(hold) => dispatch(setControlState({ state: { hold } }))}
          annotationProvider={{
            menu: ({ key, timeRange, name }) => {
              const handleSelect = (itemKey: string) => {
                switch (itemKey) {
                  case "download":
                    if (client == null) return;
                    download({ client, lines, timeRange, name, handleException });
                    break;
                  case "metadata":
                    place({ ...Range.OVERVIEW_LAYOUT, name, key });
                    break;
                  case "line-plot":
                    addRangeToNewPlot(key);
                    break;
                  default:
                    break;
                }
              };
              return (
                <PMenu.Menu level="small" key={key} onChange={handleSelect}>
                  <PMenu.Item itemKey="download" startIcon={<Icon.Download />}>
                    Download as CSV
                  </PMenu.Item>
                  <PMenu.Item itemKey="line-plot" startIcon={<Icon.LinePlot />}>
                    Open in New Plot
                  </PMenu.Item>
                  <PMenu.Item itemKey="metadata" startIcon={<Icon.Annotate />}>
                    View Details
                  </PMenu.Item>
                </PMenu.Menu>
              );
            },
          }}
        />
      </div>
    </PMenu.ContextMenu>
  );
};

const buildAxes = (vis: State): Channel.AxisProps[] =>
  getEntries<AxesState["axes"]>(vis.axes.axes)
    .filter(([key]) => shouldDisplayAxis(key, vis))
    .map(
      ([key, axis]): Channel.AxisProps => ({
        location: axisLocation(key),
        ...axis,
      }),
    );

const buildLines = (
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

export const LAYOUT_TYPE = "lineplot";
export type LayoutType = typeof LAYOUT_TYPE;

export const create =
  (initial: Partial<State> & Omit<Partial<Layout.BaseState>, "type">): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "Line Plot", location = "mosaic", window, tab, ...rest } = initial;
    const key: string = primitiveIsZero(initial.key) ? uuid() : (initial.key as string);
    dispatch(internalCreate({ ...deep.copy(ZERO_STATE), ...rest, key }));
    return { key, name, location, type: LAYOUT_TYPE, icon: "Visualize", window, tab };
  };

export const LinePlot: Layout.Renderer = ({
  layoutKey,
  ...rest
}): ReactElement | null => {
  const linePlot = useLoadRemote({
    name: "Line Plot",
    targetVersion: ZERO_STATE.version,
    layoutKey,
    useSelectVersion,
    fetcher: async (client, layoutKey) => {
      const { data } = await client.workspaces.linePlot.retrieve(layoutKey);
      return data as State;
    },
    actionCreator: internalCreate,
  });
  if (linePlot == null) return null;
  return <Loaded layoutKey={layoutKey} {...rest} />;
};

export const SELECTABLE: Layout.Selectable = {
  key: LAYOUT_TYPE,
  title: "Line Plot",
  icon: <Icon.LinePlot />,
  create: async ({ layoutKey }) => create({ key: layoutKey }),
};
