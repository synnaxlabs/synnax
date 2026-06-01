// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, lineplot, type ranger } from "@synnaxlabs/client";
import {
  Access,
  type axis,
  Channel,
  Icon,
  LinePlot as PLinePlot,
  Menu,
  Ranger,
  Status,
  Synnax,
  useAsyncEffect,
  useDebouncedCallback,
  Viewport,
} from "@synnaxlabs/pluto";
import { type measure } from "@synnaxlabs/pluto/ether";
import {
  box,
  color,
  DataType,
  location,
  primitive,
  scale,
  type sticky,
  TimeRange,
  TimeSpan,
  unique,
} from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo, useRef, useState } from "react";
import { useDispatch } from "react-redux";

import { ContextMenu } from "@/components";
import { Layout } from "@/layout";
import { Controls } from "@/lineplot/Controls";
import {
  useSelect,
  useSelectControlState,
  useSelectPendingUpload,
  useSelectSelection,
  useSelectViewportMode,
} from "@/lineplot/selectors";
import {
  DEFAULT_RULE_COLOR,
  setActiveToolbarTab,
  setControlState,
  setMeasureMode,
  setSelectedRule,
  setSelection,
  storeViewport,
} from "@/lineplot/slice";
import { useAssignedDispatch } from "@/lineplot/useAssignedDispatch";
import { useDownloadAsCSV } from "@/lineplot/useDownloadAsCSV";
import { useUndoRedoTriggers } from "@/lineplot/useUndoRedoTriggers";
import { useAutoUpload } from "@/lineplot/useUpload";
import { Range } from "@/range";

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
    <ContextMenu.Menu>
      <Menu.Item itemKey="download" onClick={handleDownloadAsCSV}>
        <Icon.CSV />
        Download as CSV
      </Menu.Item>
      <Menu.Item itemKey="line-plot" onClick={handleOpenInNewPlot}>
        <Icon.LinePlot />
        Open in new plot
      </Menu.Item>
      <Menu.Item itemKey="metadata" onClick={handleViewDetails}>
        <Icon.Annotate />
        View details
      </Menu.Item>
    </ContextMenu.Menu>
  );
};

const shouldDisplayAxis = (
  key: PLinePlot.AxisKey,
  channels: lineplot.Channels,
): boolean => {
  if (key === "x1" || key === "y1") return true;
  if (key === "x2") return channels.x2 !== 0;
  return channels[key].length > 0;
};

const Loaded: Layout.Renderer = ({ layoutKey, focused, visible }) => {
  const { name } = Layout.useSelectRequired(layoutKey);
  const vis = useSelect(layoutKey);
  const client = Synnax.use();
  const dispatch = useDispatch();
  const { dispatch: pdispatch } = PLinePlot.useDispatch();
  const assignedDispatch = useAssignedDispatch(layoutKey);
  useUndoRedoTriggers(layoutKey);
  const title = PLinePlot.useSelectTitle({ key: layoutKey });
  const legend = PLinePlot.useSelectLegend({ key: layoutKey });
  const channels = PLinePlot.useSelectChannels({ key: layoutKey });
  const rules = PLinePlot.useSelectRules({ key: layoutKey });
  const axes = PLinePlot.useSelectAxes({ key: layoutKey });
  const storedLines = PLinePlot.useSelectLines({ key: layoutKey });
  const derived = PLinePlot.useDerivedLines(layoutKey);
  const allRangeKeys = useMemo(
    () => Array.from(new Set(derived.map((d) => d.range).filter((k) => k !== ""))),
    [derived],
  );
  const resolvedRanges = Range.useSelectMultiple(allRangeKeys);
  const rangeByKey = useMemo(() => {
    const m = new Map<string, Range.Range>();
    for (const r of resolvedRanges) m.set(r.key, r);
    return m;
  }, [resolvedRanges]);
  const hasUpdatePermission = Access.useUpdateGranted(lineplot.ontologyID(layoutKey));
  const theme = Layout.useSelectTheme();

  const propsLines = useMemo<Channel.LineProps[]>(() => {
    const palette = theme?.colors.visualization.palettes.default ?? [];
    const out: Channel.LineProps[] = [];
    derived.forEach((d, i) => {
      const range = rangeByKey.get(d.range);
      if (range == null) return;
      const colorVal = PLinePlot.resolveLineColor(d.color, i, palette);
      const base = {
        key: d.key,
        axes: { x: d.xAxis, y: d.yAxis },
        channels: { x: d.xChannel, y: d.yChannel },
        color: colorVal,
        strokeWidth: d.strokeWidth,
        downsample: d.downsample,
        downsampleMode: d.downsampleMode,
        label: d.label,
      };
      if (range.variant === "dynamic")
        out.push({
          ...base,
          variant: "dynamic",
          timeSpan: new TimeSpan(range.span),
        });
      else
        out.push({
          ...base,
          variant: "static",
          timeRange: new TimeRange(range.timeRange),
        });
    });
    return out;
  }, [derived, rangeByKey, theme]);

  useAsyncEffect(
    async (signal) => {
      if (client == null) return;
      const overrides = new Set(storedLines.map((l) => l.key));
      const toMaterialize = derived.filter(
        (l) => l.label == null && !overrides.has(l.key),
      );
      if (toMaterialize.length === 0) return;
      const fetched = await client.channels.retrieve(
        unique.unique(toMaterialize.map((l) => l.yChannel)),
      );
      if (signal.aborted) return;
      const palette = theme?.colors.visualization.palettes.default ?? [];
      pdispatch({
        key: layoutKey,
        actions: toMaterialize.map((l) =>
          lineplot.setLine({
            line: {
              ...l,
              label: fetched.find((f) => f.key === l.yChannel)?.name ?? undefined,
              color: PLinePlot.resolveLineColor(l.color, derived.indexOf(l), palette),
            },
          }),
        ),
      });
    },
    [layoutKey, client, derived, storedLines, theme],
  );

  const handleLineChange = useCallback<
    Exclude<Channel.LinePlotProps["onLineChange"], undefined>
  >(
    (d) => {
      const existing =
        storedLines.find((l) => l.key === d.key) ??
        derived.find((l) => l.key === d.key);
      if (existing == null) return;
      const next: lineplot.Line = {
        ...existing,
        ...d,
        color: d.color != null ? color.construct(d.color) : existing.color,
      };
      pdispatch({ key: layoutKey, actions: [lineplot.setLine({ line: next })] });
    },
    [pdispatch, layoutKey, storedLines, derived],
  );

  const handleRuleChange = useCallback<
    Exclude<Channel.LinePlotProps["onRuleChange"], undefined>
  >(
    (rule) => {
      const existing = rules.find((r) => r.key === rule.key);
      if (existing == null) return;
      const nextAxis =
        rule.axis != null && PLinePlot.isAxisKey(rule.axis) ? rule.axis : existing.axis;
      const next: lineplot.Rule = {
        ...existing,
        ...rule,
        color: rule.color != null ? color.construct(rule.color) : existing.color,
        axis: nextAxis,
      };
      pdispatch({ key: layoutKey, actions: [lineplot.setRule({ rule: next })] });
    },
    [pdispatch, layoutKey, rules],
  );

  const handleAxisChange = useCallback(
    (a: Partial<Channel.AxisProps> & { key: string }) => {
      if (!PLinePlot.isAxisKey(a.key)) return;
      const existing = axes[a.key];
      if (existing == null) return;
      const { bounds: aBounds, label: aLabel } = a;
      const next: lineplot.Axis = {
        ...existing,
        key: a.key,
        ...(aBounds != null
          ? {
              bounds: {
                lower: Number(aBounds.lower),
                upper: Number(aBounds.upper),
              },
            }
          : {}),
        ...(aLabel != null ? { label: aLabel } : {}),
      };
      pdispatch({ key: layoutKey, actions: [lineplot.setAxis({ axis: next })] });
    },
    [pdispatch, layoutKey, axes],
  );

  useAsyncEffect(async () => {
    const ax = axes.x1;
    const key = channels.x1;
    if (client == null) return;
    let newType: axis.TickType = "time";
    if (primitive.isNonZero(key)) {
      const ch = await client.channels.retrieve(key);
      if (!ch.dataType.equals(DataType.TIMESTAMP)) newType = "linear";
    }
    if (ax.type === newType) return;
    pdispatch({
      key: layoutKey,
      actions: [lineplot.setAxis({ axis: { ...ax, type: newType } })],
    });
  }, [channels.x1, axes.x1.type, client]);

  const handleChannelAxisDrop = useCallback(
    (axisKey: string, dropped: channel.Key[]): void => {
      const actions: lineplot.Action[] = [];
      if (PLinePlot.isXAxisKey(axisKey))
        actions.push(lineplot.setXChannel({ axisKey, channel: dropped[0] }));
      else if (PLinePlot.isYAxisKey(axisKey)) {
        const existing = new Set(channels[axisKey]);
        for (const c of dropped)
          if (!existing.has(c))
            actions.push(lineplot.addChannel({ axisKey, channel: c }));
      } else return;
      assignedDispatch(actions);
    },
    [assignedDispatch, channels],
  );

  const handleViewportChange: Viewport.UseHandler = useDebouncedCallback(
    ({ box: b, stage, mode }) => {
      if (stage !== "end") return;
      if (mode === "select") dispatch(setSelection({ key: layoutKey, box: b }));
      else
        dispatch(
          storeViewport({ key: layoutKey, pan: box.bottomLeft(b), zoom: box.dims(b) }),
        );
    },
    TimeSpan.milliseconds(100),
    [dispatch, layoutKey],
  );

  const [legendPosition, setLegendPosition] = useState(legend.position);

  const storeLegendPosition = useDebouncedCallback(
    (position: sticky.XY) =>
      pdispatch({
        key: layoutKey,
        actions: [lineplot.setLegend({ legend: { ...legend, position } })],
      }),
    TimeSpan.milliseconds(100),
    [pdispatch, layoutKey, legend],
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
    dispatch(Layout.setNavDrawerVisible({ key: "visualization", value: true }));
    dispatch(setActiveToolbarTab({ key: layoutKey, tab: "data" }));
  }, [dispatch, layoutKey]);

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

  const handleTitleChange = useCallback(
    (value: string) => {
      dispatch(Layout.rename({ key: layoutKey, name: value }));
    },
    [dispatch, layoutKey],
  );

  const props = Menu.useContextMenu();
  const linePlotRef = useRef<PLinePlot.CoreRef | null>(null);

  const builtAxes = useMemo<Channel.AxisProps[]>(
    () =>
      PLinePlot.AXIS_KEYS.filter((k) => shouldDisplayAxis(k, channels)).map(
        (k): Channel.AxisProps => ({ location: PLinePlot.axisLocation(k), ...axes[k] }),
      ),
    [axes, channels],
  );

  interface ContextMenuContentProps extends Menu.ContextMenuMenuProps {
    layoutKey: string;
  }

  const ContextMenuContent = ({ layoutKey }: ContextMenuContentProps): ReactElement => {
    const { box: selection } = useSelectSelection(layoutKey);
    const placeLayout = Layout.usePlacer();
    const handleError = Status.useErrorHandler();

    const getTimeRange = useCallback(async (): Promise<TimeRange> => {
      const bounds = await linePlotRef.current?.getBounds();
      if (bounds == null) throw new Error("No bounds available");
      const s = scale.Scale.scale<number>(1).scale(bounds.x1);
      return new TimeRange(s.pos(box.left(selection)), s.pos(box.right(selection)));
    }, [selection]);

    const downloadAsCSV = useDownloadAsCSV();

    const getISOText = useCallback(async () => {
      const tr = await getTimeRange();
      return `${tr.start.toString("ISO")} - ${tr.end.toString("ISO")}`;
    }, [getTimeRange]);

    const getPythonText = useCallback(async () => {
      const tr = await getTimeRange();
      return `sy.TimeRange(${tr.start.valueOf()}, ${tr.end.valueOf()})`;
    }, [getTimeRange]);

    const getTypeScriptText = useCallback(async () => {
      const tr = await getTimeRange();
      return `new TimeRange(${tr.start.valueOf()}, ${tr.end.valueOf()})`;
    }, [getTimeRange]);

    const handleCreateRange = () =>
      handleError(async () => {
        const tr = await getTimeRange();
        placeLayout(Range.createCreateLayout({ timeRange: tr.numeric }));
      }, "Failed to create range from selection");

    const handleDownloadCSV = () =>
      handleError(async () => {
        const tr = await getTimeRange();
        downloadAsCSV({ timeRanges: [tr], lines: propsLines, name });
      }, "Failed to download region as CSV");

    return (
      <ContextMenu.Menu>
        {!box.areaIsZero(selection) && (
          <>
            <Menu.CopyItem
              itemKey="iso"
              text={getISOText}
              successMessage="Copied ISO time range to clipboard"
            >
              <Icon.Range /> Copy ISO time range
            </Menu.CopyItem>
            <Menu.CopyItem
              itemKey="python"
              text={getPythonText}
              successMessage="Copied Python time range to clipboard"
            >
              <Icon.Python /> Copy Python time range
            </Menu.CopyItem>
            <Menu.CopyItem
              itemKey="typescript"
              text={getTypeScriptText}
              successMessage="Copied TypeScript time range to clipboard"
            >
              <Icon.TypeScript /> Copy TypeScript time range
            </Menu.CopyItem>
            <Menu.Divider />
            <Menu.Item itemKey="range" onClick={handleCreateRange}>
              <Ranger.CreateIcon /> Create range from selection
            </Menu.Item>
            <Menu.Divider />
            <Menu.Item itemKey="download" onClick={handleDownloadCSV}>
              <Icon.CSV /> Download region as CSV
            </Menu.Item>
            <Menu.Divider />
          </>
        )}
        <ContextMenu.ReloadConsoleItem />
      </ContextMenu.Menu>
    );
  };

  const [hasAnnotations, setHasAnnotations] = useState(false);
  const rangeProviderProps: Channel.LinePlotProps["rangeProviderProps"] = {
    visible: vis.annotations.visible,
    onHasAnnotationsChange: setHasAnnotations,
    menu: (props) => <RangeAnnotationContextMenu lines={propsLines} range={props} />,
  };

  return (
    <div
      style={{ height: "100%", width: "100%", padding: "2rem" }}
      className={props.className}
    >
      <Menu.ContextMenu
        {...props}
        menu={(props) => <ContextMenuContent {...props} layoutKey={layoutKey} />}
      >
        <Channel.LinePlot
          ref={linePlotRef}
          aetherKey={layoutKey}
          hold={hold}
          onContextMenu={props.open}
          title={name}
          axes={builtAxes}
          lines={propsLines}
          rules={rules.map((r) => ({ ...r, color: r.color ?? DEFAULT_RULE_COLOR }))}
          clearOverScan={{ x: 5, y: 5 }}
          onTitleChange={hasUpdatePermission ? handleTitleChange : undefined}
          visible={visible}
          titleLevel={title.level}
          showTitle={title.visible}
          showLegend={legend.visible}
          onLineChange={hasUpdatePermission ? handleLineChange : undefined}
          onRuleChange={hasUpdatePermission ? handleRuleChange : undefined}
          onAxisChannelDrop={hasUpdatePermission ? handleChannelAxisDrop : undefined}
          onAxisChange={hasUpdatePermission ? handleAxisChange : undefined}
          onViewportChange={handleViewportChange}
          initialViewport={initialViewport}
          onLegendPositionChange={
            hasUpdatePermission ? handleLegendPositionChange : undefined
          }
          legendPosition={legendPosition}
          viewportTriggers={triggers}
          enableTooltip={enableTooltip}
          legendVariant={focused ? "fixed" : "floating"}
          enableMeasure={clickMode === "measure"}
          onDoubleClick={handleDoubleClick}
          onSelectRule={hasUpdatePermission ? handleSelectRule : undefined}
          onHold={handleHold}
          rangeProviderProps={rangeProviderProps}
          measureMode={vis.measure.mode}
          onMeasureModeChange={
            hasUpdatePermission ? handleMeasureModeChange : undefined
          }
        >
          {!focused && (
            <Controls layoutKey={layoutKey} hasAnnotations={hasAnnotations} />
          )}
        </Channel.LinePlot>
      </Menu.ContextMenu>
      {focused && <Controls layoutKey={layoutKey} hasAnnotations={hasAnnotations} />}
    </div>
  );
};

const Internal: Layout.Renderer = (props) => {
  PLinePlot.useEnsureRetrieved({ key: props.layoutKey });
  return <Loaded {...props} />;
};

export const LinePlot: Layout.Renderer = (props) => {
  // Fire the upload effect unconditionally so a freshly placed plot still
  // gets pushed to the server. The Redux pendingUpload flag is the synchronous
  // gate: while it is non-null the canonical record may not yet be in Pluto's
  // flux cache, so suspending on useEnsureRetrieved would tear the layout
  // down through the surrounding error boundary.
  useAutoUpload(props.layoutKey);
  const pendingUpload = useSelectPendingUpload(props.layoutKey);
  if (pendingUpload != null) return null;
  return <Internal {...props} />;
};

LinePlot.useName = Layout.createUseFluxName(
  PLinePlot.useRename,
  PLinePlot.useRetrieveObservableName,
);
