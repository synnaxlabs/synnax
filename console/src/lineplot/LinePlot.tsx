// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/lineplot/LinePlot.css";

import { type Dispatch, type PayloadAction } from "@reduxjs/toolkit";
import { type channel, type ranger } from "@synnaxlabs/client";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import {
  type axis,
  Channel,
  Icon,
  type Legend,
  LinePlot as Core,
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
  color,
  DataType,
  location,
  primitive,
  record,
  scale,
  TimeRange,
  unique,
} from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import {
  type ReactElement,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useDispatch } from "react-redux";

import { Menu } from "@/components";
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
import { NavControls } from "@/lineplot/NavControls";
import {
  select,
  useSelect,
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
import { useDownloadAsCSV } from "@/lineplot/useDownloadAsCSV";
import { Range } from "@/range";
import { Workspace } from "@/workspace";

interface SyncPayload {
  key?: string;
}

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

const CONTEXT_MENU_ERROR_MESSAGES: Record<string, string> = {
  iso: "Failed to copy ISO time range",
  python: "Failed to copy Python time range",
  typescript: "Failed to copy TypeScript time range",
  range: "Failed to create range from selection",
  download: "Failed to download region as CSV",
};

interface RangeAnnotationContextMenuProps {
  lines: Channel.LineProps[];
  range: ranger.Payload;
}

const RangeAnnotationContextMenu = ({
  lines,
  range,
}: RangeAnnotationContextMenuProps): ReactElement => {
  const downloadAsCSV = useDownloadAsCSV();
  const handleDownloadAsCSV = () =>
    downloadAsCSV({ timeRange: range.timeRange, lines, name: range.name });
  const addRangeToNewPlot = Range.useAddToNewPlot();
  const handleOpenInNewPlot = () => addRangeToNewPlot(range.key);
  const placeLayout = Layout.usePlacer();
  const handleViewDetails = () => {
    placeLayout({ ...Range.OVERVIEW_LAYOUT, name: range.name, key: range.key });
  };
  return (
    <PMenu.Menu level="small">
      <PMenu.Item
        itemKey="download"
        startIcon={<Icon.Download />}
        onClick={handleDownloadAsCSV}
      >
        Download as CSV
      </PMenu.Item>
      <PMenu.Item
        itemKey="line-plot"
        startIcon={<Icon.LinePlot />}
        onClick={handleOpenInNewPlot}
      >
        Open in New Plot
      </PMenu.Item>
      <PMenu.Item
        itemKey="metadata"
        startIcon={<Icon.Annotate />}
        onClick={handleViewDetails}
      >
        View Details
      </PMenu.Item>
    </PMenu.Menu>
  );
};

const Loaded: Layout.Renderer = ({ layoutKey, focused, visible }) => {
  const windowKey = useSelectWindowKey() as string;
  const { name } = Layout.useSelectRequired(layoutKey);
  const vis = useSelect(layoutKey);
  const prevVis = usePrevious(vis);
  const ranges = useSelectRanges(layoutKey);
  const client = Synnax.use();
  const dispatch = useDispatch();
  const syncDispatch = useSyncComponent(layoutKey);
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
      if (d.color != null) newLine.color = color.hex(d.color);
      syncDispatch(setLine({ key: layoutKey, line: [newLine] }));
    },
    [syncDispatch, layoutKey],
  );

  const handleRuleChange = useCallback<
    Exclude<Channel.LinePlotProps["onRuleChange"], undefined>
  >(
    (rule) => {
      syncDispatch(
        setRule({
          key: layoutKey,
          rule: {
            ...rule,
            color: rule.color != null ? color.hex(rule.color) : undefined,
            axis: rule.axis != null ? (rule.axis as AxisKey) : undefined,
          },
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
      if (!primitive.isZero(key)) {
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
    ({ box: b, slate, mode }) => {
      if (slate !== "end") return;
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

  const boundsQuerierRef = useRef<Core.GetBoundsFn>(null);

  const ContextMenuContent = ({ layoutKey }: ContextMenuContentProps): ReactElement => {
    const { box: selection } = useSelectSelection(layoutKey);
    const placeLayout = Layout.usePlacer();
    const handleError = Status.useErrorHandler();

    const getTimeRange = useCallback(async () => {
      const bounds = await boundsQuerierRef.current?.();
      if (bounds == null) return null;
      const s = scale.Scale.scale<number>(1).scale(bounds.x1);
      return new TimeRange(s.pos(box.left(selection)), s.pos(box.right(selection)));
    }, []);

    const downloadAsCSV = useDownloadAsCSV();

    const handleSelect = (key: string): void => {
      handleError(async () => {
        const tr = await getTimeRange();
        if (tr == null) return;
        switch (key) {
          case "iso":
            await navigator.clipboard.writeText(
              `${tr.start.fString("ISO")} - ${tr.end.fString("ISO")}`,
            );
            break;
          case "python":
            await navigator.clipboard.writeText(
              `sy.TimeRange(${tr.start.valueOf()}, ${tr.end.valueOf()})`,
            );
            break;
          case "typescript":
            await navigator.clipboard.writeText(
              `new TimeRange(${tr.start.valueOf()}, ${tr.end.valueOf()})`,
            );
            break;
          case "range":
            placeLayout(Range.createCreateLayout({ timeRange: tr.numeric }));
            break;
          case "download":
            if (client == null) return;
            downloadAsCSV({ timeRange: tr, lines, name });
            break;
        }
      }, `Failed to perform ${CONTEXT_MENU_ERROR_MESSAGES[key]}`);
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
              Download Region as CSV
            </PMenu.Item>
          </>
        )}
        <Menu.HardReloadItem />
      </PMenu.Menu>
    );
  };

  const rangeAnnotationProvider: Channel.LinePlotProps["rangeAnnotationProvider"] = {
    menu: (props) => <RangeAnnotationContextMenu lines={propsLines} range={props} />,
  };

  return (
    <div
      style={{ height: "100%", width: "100%", padding: "2rem" }}
      className={props.className}
    >
      <PMenu.ContextMenu
        {...props}
        menu={() => <ContextMenuContent layoutKey={layoutKey} />}
      >
        <Channel.LinePlot
          aetherKey={layoutKey}
          hold={hold}
          onContextMenu={props.open}
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
          rangeAnnotationProvider={rangeAnnotationProvider}
        >
          {!focused && <NavControls />}
          <Core.BoundsQuerier ref={boundsQuerierRef} />
        </Channel.LinePlot>
      </PMenu.ContextMenu>
      {focused && <NavControls />}
    </div>
  );
};

const buildAxes = (vis: State): Channel.AxisProps[] =>
  record
    .entries<AxesState["axes"]>(vis.axes.axes)
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

export const LinePlot: Layout.Renderer = ({ layoutKey, ...rest }) => {
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
