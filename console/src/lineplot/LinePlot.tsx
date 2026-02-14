// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, lineplot, type ranger } from "@synnaxlabs/client";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import {
  Access,
  type axis,
  Channel,
  Icon,
  LinePlot as Base,
  Menu as PMenu,
  Ranger,
  Status,
  Synnax,
  useAsyncEffect,
  useDebouncedCallback,
  usePrevious,
  Viewport,
} from "@synnaxlabs/pluto";
import { type measure } from "@synnaxlabs/pluto/ether";
import {
  box,
  color,
  DataType,
  location,
  primitive,
  record,
  scale,
  type sticky,
  TimeRange,
  unique,
} from "@synnaxlabs/x";
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
import { createLoadRemote } from "@/hooks/useLoadRemote";
import { Layout } from "@/layout";
import {
  type AxisKey,
  axisLocation,
  X_AXIS_KEYS,
  type XAxisKey,
  type YAxisKey,
} from "@/lineplot/axis";
import { buildLines } from "@/lineplot/buildLines";
import { Controls } from "@/lineplot/Controls";
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
  setActiveToolbarTab,
  setAxis,
  setControlState,
  setLegend,
  setLine,
  setMeasureMode,
  setRanges,
  setRemoteCreated,
  setRule,
  setSelectedRule,
  setSelection,
  setXChannel,
  setYChannels,
  shouldDisplayAxis,
  type State,
  storeViewport,
  ZERO_STATE,
} from "@/lineplot/slice";
import { useDownloadAsCSV } from "@/lineplot/useDownloadAsCSV";
import { Range } from "@/range";
import { Workspace } from "@/workspace";

const useSyncComponent = Workspace.createSyncComponent(
  "Line Plot",
  async ({ key, workspace, store, fluxStore, client }) => {
    const s = store.getState();
    if (
      !Access.updateGranted({ id: lineplot.ontologyID(key), store: fluxStore, client })
    )
      return;
    const data = select(s, key);
    if (data == null) return;
    const la = Layout.selectRequired(s, key);
    if (!data.remoteCreated) store.dispatch(setRemoteCreated({ key }));
    await client.lineplots.create(workspace, { key, name: la.name, data });
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
    downloadAsCSV({ timeRanges: [range.timeRange], lines, name: range.name });
  const addRangeToNewPlot = Range.useAddToNewPlot();
  const handleOpenInNewPlot = () => addRangeToNewPlot([range.key]);
  const placeLayout = Layout.usePlacer();
  const handleViewDetails = () => {
    placeLayout({ ...Range.OVERVIEW_LAYOUT, name: range.name, key: range.key });
  };
  return (
    <PMenu.Menu level="small">
      <PMenu.Item itemKey="download" onClick={handleDownloadAsCSV}>
        <Icon.CSV />
        Download as CSV
      </PMenu.Item>
      <PMenu.Item itemKey="line-plot" onClick={handleOpenInNewPlot}>
        <Icon.LinePlot />
        Open in new plot
      </PMenu.Item>
      <PMenu.Item itemKey="metadata" onClick={handleViewDetails}>
        <Icon.Annotate />
        View details
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
  const hasEditPermission = Access.useUpdateGranted(lineplot.ontologyID(layoutKey));

  useEffect(() => {
    if (prevName !== name) syncDispatch(Layout.rename({ key: layoutKey, name }));
  }, [syncDispatch, name, prevName]);

  useAsyncEffect(
    async (signal) => {
      if (client == null) return;
      const toFetch = lines.filter((line) => line.label == null);
      if (toFetch.length === 0) return;
      const fetched = await client.channels.retrieve(
        unique.unique(toFetch.map((line) => line.channels.y)) as channel.KeysOrNames,
      );
      if (signal.aborted) return;
      const update = toFetch.map((l) => ({
        key: l.key,
        label: fetched.find((f) => f.key === l.channels.y)?.name,
      }));
      syncDispatch(setLine({ key: layoutKey, line: update }));
    },
    [layoutKey, client, lines],
  );

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

  useAsyncEffect(async () => {
    const axis = vis.axes.axes.x1;
    const axisKey = axis.key as XAxisKey;
    const key = vis.channels[axisKey];
    const prevKey = prevVis?.channels[axisKey];
    if (client == null || key === prevKey) return;
    let newType: axis.TickType = "time";
    if (primitive.isNonZero(key)) {
      const ch = await client.channels.retrieve(key);
      if (!ch.dataType.equals(DataType.TIMESTAMP)) newType = "linear";
    }
    if (axis.type === newType) return;
    syncDispatch(
      setAxis({
        key: layoutKey,
        axisKey,
        axis: { ...(axis as AxisState), type: newType },
        triggerRender: true,
      }),
    );
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
    (position: sticky.XY) =>
      syncDispatch(setLegend({ key: layoutKey, legend: { position } })),
    100,
    [syncDispatch, layoutKey],
  );

  const handleLegendPositionChange = useCallback(
    (position: sticky.XY) => {
      setLegendPosition(position);
      storeLegendPosition(position);
    },
    [storeLegendPosition],
  );

  const { enableTooltip, clickMode, hold } = useSelectControlState(layoutKey);
  const mode = useSelectViewportMode(layoutKey);
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
    dispatch(setActiveToolbarTab({ key: layoutKey, tab: "data" }));
  }, [windowKey, dispatch, layoutKey]);

  const handleSelectRule = useCallback(
    (ruleKey: string) => {
      dispatch(setSelectedRule({ key: layoutKey, ruleKey }));
    },
    [dispatch, layoutKey],
  );

  const handleMeasureModeChange = useCallback(
    (mode: measure.Mode) => {
      dispatch(setMeasureMode({ key: layoutKey, mode }));
    },
    [dispatch, layoutKey],
  );

  const handleHold = useCallback(
    (hold: boolean) => {
      dispatch(setControlState({ key: layoutKey, state: { hold } }));
    },
    [dispatch, layoutKey],
  );

  const props = PMenu.useContextMenu();
  const linePlotRef = useRef<Base.LinePlotRef | null>(null);

  interface ContextMenuContentProps extends PMenu.ContextMenuMenuProps {
    layoutKey: string;
  }

  const ContextMenuContent = ({ layoutKey }: ContextMenuContentProps): ReactElement => {
    const { box: selection } = useSelectSelection(layoutKey);
    const placeLayout = Layout.usePlacer();
    const handleError = Status.useErrorHandler();

    const getTimeRange = useCallback(async () => {
      const bounds = await linePlotRef.current?.getBounds();
      if (bounds == null) return null;
      const s = scale.Scale.scale<number>(1).scale(bounds.x1);
      return new TimeRange(s.pos(box.left(selection)), s.pos(box.right(selection)));
    }, [selection]);

    const downloadAsCSV = useDownloadAsCSV();

    const handleSelect = (key: string): void => {
      handleError(async () => {
        const tr = await getTimeRange();
        if (tr == null) return;
        switch (key) {
          case "iso":
            await navigator.clipboard.writeText(
              `${tr.start.toString("ISO")} - ${tr.end.toString("ISO")}`,
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
            downloadAsCSV({ timeRanges: [tr], lines, name });
            break;
        }
      }, `Failed to perform ${CONTEXT_MENU_ERROR_MESSAGES[key]}`);
    };

    return (
      <PMenu.Menu onChange={handleSelect} gap="small" level="small">
        {!box.areaIsZero(selection) && (
          <>
            <PMenu.Item itemKey="iso">
              <Icon.Range /> Copy ISO time range
            </PMenu.Item>
            <PMenu.Item itemKey="python">
              <Icon.Python /> Copy Python time range
            </PMenu.Item>
            <PMenu.Item itemKey="typescript">
              <Icon.TypeScript /> Copy TypeScript time range
            </PMenu.Item>
            <PMenu.Divider />
            <PMenu.Item itemKey="range">
              <Ranger.CreateIcon /> Create range from selection
            </PMenu.Item>
            <PMenu.Divider />
            <PMenu.Item itemKey="download">
              <Icon.CSV /> Download region as CSV
            </PMenu.Item>
            <PMenu.Divider />
          </>
        )}
        <Menu.ReloadConsoleItem />
      </PMenu.Menu>
    );
  };

  const rangeProviderProps: Channel.LinePlotProps["rangeProviderProps"] = {
    menu: (props) => <RangeAnnotationContextMenu lines={propsLines} range={props} />,
  };

  return (
    <div
      style={{ height: "100%", width: "100%", padding: "2rem" }}
      className={props.className}
    >
      <PMenu.ContextMenu
        {...props}
        menu={(props) => <ContextMenuContent {...props} layoutKey={layoutKey} />}
      >
        <Channel.LinePlot
          ref={linePlotRef}
          aetherKey={layoutKey}
          hold={hold}
          onContextMenu={props.open}
          title={name}
          axes={axes}
          lines={propsLines}
          rules={vis.rules}
          clearOverScan={{ x: 5, y: 5 }}
          onTitleChange={hasEditPermission ? handleTitleChange : undefined}
          visible={visible}
          titleLevel={vis.title.level}
          showTitle={vis.title.visible}
          showLegend={vis.legend.visible}
          onLineChange={hasEditPermission ? handleLineChange : undefined}
          onRuleChange={hasEditPermission ? handleRuleChange : undefined}
          onAxisChannelDrop={hasEditPermission ? handleChannelAxisDrop : undefined}
          onAxisChange={hasEditPermission ? handleAxisChange : undefined}
          onViewportChange={handleViewportChange}
          initialViewport={initialViewport}
          onLegendPositionChange={
            hasEditPermission ? handleLegendPositionChange : undefined
          }
          legendPosition={legendPosition}
          viewportTriggers={triggers}
          enableTooltip={enableTooltip}
          legendVariant={focused ? "fixed" : "floating"}
          enableMeasure={clickMode === "measure"}
          onDoubleClick={handleDoubleClick}
          onSelectRule={hasEditPermission ? handleSelectRule : undefined}
          onHold={handleHold}
          rangeProviderProps={rangeProviderProps}
          measureMode={vis.measure.mode}
          onMeasureModeChange={hasEditPermission ? handleMeasureModeChange : undefined}
        >
          {!focused && <Controls layoutKey={layoutKey} />}
        </Channel.LinePlot>
      </PMenu.ContextMenu>
      {focused && <Controls layoutKey={layoutKey} />}
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

const useLoadRemote = createLoadRemote<lineplot.LinePlot>({
  useRetrieve: Base.useRetrieveObservable,
  targetVersion: ZERO_STATE.version,
  useSelectVersion,
  actionCreator: (v) => internalCreate({ ...(v.data as State), key: v.key }),
});

export const LinePlot: Layout.Renderer = ({ layoutKey, ...rest }) => {
  const linePlot = useLoadRemote(layoutKey);
  if (linePlot == null) return null;
  return <Loaded layoutKey={layoutKey} {...rest} />;
};
