// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/lineplot/LinePlot.css";

import { lineplot, ranger } from "@synnaxlabs/client";
import { useDebouncedCallback } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import {
  Access,
  Channel,
  LinePlot as Base,
  Panel as PlutoPanel,
  Ranger,
  Viewport,
} from "@synnaxlabs/pluto";
import type { lineplot as pLineplot } from "@synnaxlabs/pluto/ether";
import {
  box,
  type color,
  location,
  TimeRange,
  TimeSpan,
  TimeStamp,
  type xy,
} from "@synnaxlabs/x";
import { type ReactElement, type RefObject, useRef, useState } from "react";

import { Controls } from "@/feature/lineplot/Controls";
import { CSV } from "@/feature/lineplot/csv";
import { Hold } from "@/feature/lineplot/hold";
import { T0 } from "@/feature/lineplot/t0";
import { Visibility } from "@/feature/lineplot/visibility";
import { Zoom } from "@/feature/lineplot/zoom";
import { Sift } from "@/feature/sift";
import { FLAGS } from "@/flags";
import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import type { Panel } from "@/platform/panel";
import { Range } from "@/platform/range";
import { Vis } from "@/platform/vis";
import { Session } from "@/session";

const CLEAR_OVERSCAN: xy.XY = { x: 5, y: 5 };
const VIEWPORT_DEBOUNCE = TimeSpan.milliseconds(100);

interface RangeAnnotationContextMenuProps {
  lines: CSV.DownloadLine[];
  range: Base.Range.MenuProps;
}

const RangeAnnotationContextMenu = ({
  lines,
  range,
}: RangeAnnotationContextMenuProps): ReactElement => {
  const downloadAsCSV = CSV.useDownload();
  return (
    <Range.ContextMenu
      keys={[range.key]}
      primaryMenuItems={
        <Menu.Item itemKey="zoom" onClick={range.onZoom}>
          {range.zoomed ? <Icon.Expand /> : <Icon.Zoom />}
          {range.zoomed ? "Zoom out" : "Zoom to range"}
        </Menu.Item>
      }
      onDownloadCSV={(ranges) =>
        downloadAsCSV({
          timeRanges: ranges.map((r) => r.timeRange),
          lines,
          name: range.name,
        })
      }
      extraMenuItems={
        <Sift.Range.UploadMenuItem rangeKey={range.key} rangeName={range.name} />
      }
    />
  );
};

interface BreaksMenuItemsProps {
  linePlotRef: RefObject<Base.FrameRef | null>;
  breaks: pLineplot.BreakCounts;
}

const BreaksMenuItems = ({
  linePlotRef,
  breaks,
}: BreaksMenuItemsProps): ReactElement => (
  <>
    {breaks.closed > 0 && (
      <Menu.Item
        itemKey="openBreaks"
        onClick={() => linePlotRef.current?.setAllBreaks(true)}
      >
        <Icon.Expand /> Open all breaks
      </Menu.Item>
    )}
    {breaks.open > 0 && (
      <Menu.Item
        itemKey="closeBreaks"
        onClick={() => linePlotRef.current?.setAllBreaks(false)}
      >
        <Icon.Collapse /> Close all breaks
      </Menu.Item>
    )}
    {breaks.open + breaks.closed > 0 && <Menu.Divider />}
  </>
);

// Compare holds what the menus of a plot on a relative x axis act on.
interface Compare {
  refs: ranger.Ref[];
  ranges: Base.RangeSpec[];
  hovered: Base.Hovered | null;
  onItemVisibleChange: (item: Visibility.Item, visible: boolean) => void;
  onIsolate: (item: Visibility.Item) => void;
  hidden: Session.LinePlot.Hidden;
  scope: Visibility.Scope;
  /** The key of the range the plot is zoomed to, if any. */
  zoomed: string | null;
  onShowAll: () => void;
}

interface CompareMenuItemsProps {
  menuKey: string;
  compare: Compare;
  linePlotRef: RefObject<Base.FrameRef | null>;
  editable: boolean;
}

// CompareMenuItems lists the actions on what the user clicked: a range name or a
// channel in the legend, or the line under the pointer. Off any range, it lists only
// Show all.
const CompareMenuItems = ({
  menuKey,
  compare: {
    refs,
    ranges,
    hovered,
    onItemVisibleChange,
    onIsolate,
    hidden,
    scope,
    zoomed,
    onShowAll,
  },
  linePlotRef,
  editable,
}: CompareMenuItemsProps): ReactElement | null => {
  const { dispatch } = Base.useDispatch();
  const key = Base.useKey();
  const setT0Modal = T0.useSetModal();
  const handleError = Status.useErrorHandler();
  const heading = ranges.find((r) => r.key === menuKey);
  const trace = heading == null ? Base.parseTraceKey(menuKey, ranges) : null;
  const pointed = heading == null && trace == null ? hovered : null;
  const target = trace ?? pointed;
  const range = heading ?? ranges.find((r) => r.key === target?.range);
  const showAll =
    Visibility.countHidden(hidden) > 0 ? (
      <Menu.Item itemKey="showAll" onClick={onShowAll}>
        <Icon.Visible /> Show all
      </Menu.Item>
    ) : null;
  if (range?.zero == null)
    return showAll == null ? null : (
      <>
        {showAll}
        <Menu.Divider />
      </>
    );
  const { zero } = range;
  const applyT0 = (t0: TimeStamp | undefined) =>
    dispatch({ key, actions: [lineplot.setRangeT0({ range: range.key, t0 })] });
  const isKey =
    refs.find((r) => Session.Range.keyOf(r) === range.key)?.variant === "key";
  const timeRange =
    range.window.variant === "static"
      ? TimeRange.z.parse(range.window.timeRange)
      : TimeRange.ZERO;
  // A running range ends at TimeStamp.MAX, so the zoom stops at now.
  const shown = timeRange.boundBy(new TimeRange(TimeStamp.ZERO, TimeStamp.now()));
  const zoomable = !shown.span.isZero;
  const t0Editable = editable && isKey;
  const lineHidden =
    target != null && Visibility.isLineHidden(hidden, target.line, scope);
  const handleSetT0 = () =>
    handleError(async () => {
      const t0 = await setT0Modal({
        name: range.label ?? "",
        timeRange,
        value: Number(zero.valueOf()),
      });
      if (t0 != null) applyT0(new TimeStamp(t0));
    }, "Failed to set T0");
  const handleZoom = () =>
    handleError(async () => {
      const positions = await linePlotRef.current?.getXPositions({
        lower: Number(shown.start.valueOf() - zero.valueOf()),
        upper: Number(shown.end.valueOf() - zero.valueOf()),
      });
      if (positions == null) return;
      linePlotRef.current?.toggleZoom(range.key, positions.x1);
    }, "Failed to zoom to range");
  return (
    <>
      {target == null ? (
        <>
          <Menu.Item
            itemKey="isolateRange"
            onClick={() => onIsolate({ variant: "range", range: range.key })}
          >
            <Ranger.VisibleIcon /> Isolate
          </Menu.Item>
          {showAll}
          {zoomable && (
            <Menu.Item itemKey="zoomToRange" onClick={handleZoom}>
              {zoomed === range.key ? <Icon.Expand /> : <Icon.Zoom />}
              {zoomed === range.key ? "Zoom out" : "Zoom to range"}
            </Menu.Item>
          )}
          {t0Editable && (
            <Menu.Item itemKey="setT0" onClick={handleSetT0}>
              <Icon.Time /> Set T0...
            </Menu.Item>
          )}
          {t0Editable && range.customZero === true && (
            <Menu.Item itemKey="resetT0" onClick={() => applyT0(undefined)}>
              <Icon.Reset /> Reset T0
            </Menu.Item>
          )}
        </>
      ) : (
        <>
          {pointed != null && t0Editable && (
            <Menu.Item itemKey="setT0Here" onClick={() => applyT0(pointed.time)}>
              <Icon.Time /> Set T0 here
            </Menu.Item>
          )}
          <Menu.Item
            itemKey="isolateTrace"
            onClick={() =>
              onIsolate({ variant: "trace", line: target.line, range: target.range })
            }
          >
            <Channel.VisibleIcon /> Isolate in this range
          </Menu.Item>
          <Menu.Item
            itemKey="isolateLine"
            onClick={() => onIsolate({ variant: "line", line: target.line })}
          >
            <Channel.VisibleIcon /> Isolate in all ranges
          </Menu.Item>
          {lineHidden ? (
            <Menu.Item
              itemKey="showLine"
              onClick={() =>
                onItemVisibleChange({ variant: "line", line: target.line }, true)
              }
            >
              <Channel.VisibleIcon /> Show in all ranges
            </Menu.Item>
          ) : (
            <Menu.Item
              itemKey="hideLine"
              onClick={() =>
                onItemVisibleChange({ variant: "line", line: target.line }, false)
              }
            >
              <Channel.HiddenIcon /> Hide in all ranges
            </Menu.Item>
          )}
          {showAll}
        </>
      )}
      <Menu.Divider />
    </>
  );
};

interface ContextMenuContentProps {
  csvLines: CSV.DownloadLine[];
  linePlotRef: RefObject<Base.FrameRef | null>;
  editable: boolean;
  breaks: pLineplot.BreakCounts;
  breaksHidden: boolean;
  menuKeys: string[];
  compare: Compare | null;
}

const ContextMenuContent = ({
  csvLines,
  linePlotRef,
  editable,
  breaks,
  breaksHidden,
  menuKeys,
  compare,
}: ContextMenuContentProps): ReactElement => {
  const name = Base.useName({});
  const { undo, canUndo } = Base.useUndo({});
  const { redo, canRedo } = Base.useRedo({});
  const { box: selection } = Session.LinePlot.useSelectSelection();
  const openCreateRange = Range.useCreateModal();
  const hasRangeCreatePermission = Access.useCreateGranted(ranger.TYPE_RESOURCE_ID);
  const handleError = Status.useErrorHandler();
  const downloadAsCSV = CSV.useDownload();
  const getTimeRange = async (): Promise<TimeRange> => {
    const bounds = await linePlotRef.current?.getXBounds(box.xBounds(selection));
    if (bounds == null) throw new Error("No bounds available");
    return new TimeRange(bounds.x1.lower, bounds.x1.upper);
  };
  const copyText = (fmt: (tr: TimeRange) => string) => async () =>
    fmt(await getTimeRange());
  const handleCreateRange = () =>
    handleError(async () => {
      const tr = await getTimeRange();
      await openCreateRange({ timeRange: tr.numeric });
    }, "Failed to create range from selection");
  const handleDownloadCSV = () =>
    handleError(async () => {
      const tr = await getTimeRange();
      downloadAsCSV({ timeRanges: [tr], lines: csvLines, name });
    }, "Failed to download region as CSV");
  return (
    <ContextMenu.Menu>
      {editable && (
        <>
          <Menu.UndoRedoItems
            undo={undo}
            redo={redo}
            canUndo={canUndo}
            canRedo={canRedo}
          />
          <Menu.Divider />
        </>
      )}
      {FLAGS.breaks && !breaksHidden && (
        <BreaksMenuItems breaks={breaks} linePlotRef={linePlotRef} />
      )}
      {compare != null && (
        <CompareMenuItems
          menuKey={menuKeys[0] ?? ""}
          compare={compare}
          linePlotRef={linePlotRef}
          editable={editable}
        />
      )}
      {compare == null && !box.areaIsZero(selection) && (
        <>
          <Menu.CopyItem
            itemKey="iso"
            text={copyText(
              (tr) => `${tr.start.toString("ISO")} - ${tr.end.toString("ISO")}`,
            )}
            successMessage="Copied ISO time range to clipboard"
          >
            <Icon.Range /> Copy ISO time range
          </Menu.CopyItem>
          <Menu.CopyItem
            itemKey="python"
            text={copyText(
              (tr) => `sy.TimeRange(${tr.start.valueOf()}, ${tr.end.valueOf()})`,
            )}
            successMessage="Copied Python time range to clipboard"
          >
            <Icon.Python /> Copy Python time range
          </Menu.CopyItem>
          <Menu.CopyItem
            itemKey="typescript"
            text={copyText(
              (tr) => `new TimeRange(${tr.start.valueOf()}, ${tr.end.valueOf()})`,
            )}
            successMessage="Copied TypeScript time range to clipboard"
          >
            <Icon.TypeScript /> Copy TypeScript time range
          </Menu.CopyItem>
          <Menu.Divider />
          {hasRangeCreatePermission && (
            <Menu.Item itemKey="range" onClick={handleCreateRange}>
              <Ranger.CreateIcon /> Create range from selection
            </Menu.Item>
          )}
          <Menu.Divider />
          <Menu.Item itemKey="download" onClick={handleDownloadCSV}>
            <Icon.CSV /> Download region as CSV
          </Menu.Item>
          <Sift.Range.UploadRegionMenuItem name={name} getTimeRange={getTimeRange} />
          <Menu.Divider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const Internal = (): ReactElement => {
  const key = Base.useKey();
  const visible = Session.Panel.useSelectIsTabVisible();
  const vis = Session.LinePlot.useSelect();
  const dispatch = Session.useDispatch();
  const hasUpdatePermission = Access.useUpdateGranted(lineplot.resourceID(key));
  const refs = Base.useRanges();
  const ranges = Session.Range.useResolveRanges(refs, Base.useRangeColors({}));
  const x1 = Base.useXAxis({ axisKey: "x1" });
  const relative = FLAGS.compare && Base.isRelative(x1);

  const derived = Base.useLines();
  const scope: Visibility.Scope = {
    ranges: ranges.filter((r) => r.zero != null).map((r) => r.key),
    lines: derived.map((d) => d.key),
  };
  const hidden = Visibility.inScope(Session.LinePlot.useSelectHidden(), scope);
  const isolation = Session.LinePlot.useSelectIsolation();
  const { dispatch: dispatchPlot } = Base.useDispatch();
  const hoveredRef = useRef<Base.Hovered | null>(null);
  const setVisibility = (next: Visibility.Visibility) =>
    dispatch(Session.LinePlot.setHidden({ key, ...next }));
  const handleItemVisibleChange = (item: Visibility.Item, itemVisible: boolean) =>
    setVisibility(
      Visibility.setItemVisible({ hidden, isolation }, item, itemVisible, scope),
    );
  const handleIsolate = (item: Visibility.Item) =>
    setVisibility(Visibility.isolate({ hidden, isolation }, item, scope));
  const handleLegendChange = (
    item: Visibility.Item,
    itemVisible: boolean,
    isolated: boolean,
  ) =>
    isolated
      ? setVisibility(Visibility.toggleIsolation({ hidden, isolation }, item, scope))
      : handleItemVisibleChange(item, itemVisible);
  const handleShowAll = () =>
    setVisibility({ hidden: Visibility.NONE_HIDDEN, isolation: null });
  const handleLineVisibleChange = (lineKey: string, v: boolean, isolated: boolean) => {
    const trace = relative ? Base.parseTraceKey(lineKey, ranges) : null;
    if (!relative) handleLegendChange({ variant: "line", line: lineKey }, v, isolated);
    else if (trace != null)
      handleLegendChange({ variant: "trace", ...trace }, v, isolated);
  };
  const handleRangeVisibleChange = (range: string, v: boolean, isolated: boolean) =>
    handleLegendChange({ variant: "range", range }, v, isolated);
  const handleRangeColorChange = (rangeKey: string, c: color.Color) => {
    const ref = refs.find((r) => Session.Range.keyOf(r) === rangeKey);
    if (ref == null) return;
    dispatchPlot({
      key,
      actions: [
        lineplot.setRangeColor({ range: ref, color: { type: "fixed", value: c } }),
      ],
    });
  };
  const handleHoveredChange = (h: Base.Hovered | null) => {
    hoveredRef.current = h;
  };
  const csvLines: CSV.DownloadLine[] = derived.map((d) => ({
    channels: { x: d.xChannel, y: d.yChannel },
  }));

  const { enableTooltip, clickMode, hold } = Session.LinePlot.useSelectControlState();
  const storedBreaksHidden = Session.LinePlot.useSelectBreaksHidden();
  const breaksHidden = !FLAGS.breaks || storedBreaksHidden;
  const mode = Session.LinePlot.useSelectViewportMode();
  const triggers = Viewport.DEFAULT_TRIGGERS[mode];
  // Only setViewport bumps the trigger. A stored viewport came from the plot itself.
  const [applied, setApplied] = useState(vis.viewport);
  if (applied.renderTrigger !== vis.viewport.renderTrigger) setApplied(vis.viewport);
  const initialViewport = box.reRoot(
    box.construct(applied.pan, applied.zoom),
    location.BOTTOM_LEFT,
  );

  Hold.useTrigger({ key, enabled: hasUpdatePermission });
  Zoom.useResetOnRangeChange({ key });

  const handleViewportChange = useDebouncedCallback(
    ({ box: b, stage, mode }: Viewport.UseEvent) => {
      if (stage !== "end") return;
      if (mode === "select") dispatch(Session.LinePlot.setSelection({ key, box: b }));
      else
        dispatch(
          Session.LinePlot.storeViewport({
            key,
            pan: box.bottomLeft(b),
            zoom: box.dims(b),
          }),
        );
    },
    VIEWPORT_DEBOUNCE,
    [dispatch, key],
  );

  const handleOpenBreaksChange = ({
    openBreaks: open,
    viewport,
    selection,
  }: Base.OpenBreaksChange) => {
    // A view stored after the change would hold the times of the layout before it.
    handleViewportChange.flush();
    dispatch(
      Session.LinePlot.storeBreaks({
        key,
        open,
        pan: box.bottomLeft(viewport),
        zoom: box.dims(viewport),
        selection,
      }),
    );
  };

  const handleSelectRule = (ruleKey: string) =>
    dispatch(Session.LinePlot.setSelectedRule({ key, ruleKey }));
  const handleMeasureModeChange = (mode: pLineplot.measure.Mode) =>
    dispatch(Session.LinePlot.setMeasureMode({ key, mode }));
  const handleHold = (hold: boolean) =>
    dispatch(Session.LinePlot.setControlHold({ key, hold }));
  const handleDoubleClick = () => {
    dispatch(Session.Nav.showBottom({}));
    dispatch(Session.LinePlot.setActiveToolbarTab({ key, tab: "data" }));
  };

  const menuProps = Menu.useContextMenu();
  // The pointer leaves the plot for the menu, so the menu keeps the line it opened on.
  const [menuHovered, setMenuHovered] = useState<Base.Hovered | null>(null);
  const handleContextMenu: Menu.ContextMenuOpen = (e) => {
    setMenuHovered(hoveredRef.current);
    menuProps.open(e);
  };
  const linePlotRef = useRef<Base.FrameRef | null>(null);
  const [hasAnnotations, setHasAnnotations] = useState(false);
  const [zoomed, setZoomed] = useState<string | null>(null);

  const rangeProviderProps = {
    visible: vis.annotations.visible,
    onHasAnnotationsChange: setHasAnnotations,
    menu: (p: Base.Range.MenuProps) => (
      <RangeAnnotationContextMenu lines={csvLines} range={p} />
    ),
  };

  const [breaks, setBreaks] = useState<pLineplot.BreakCounts>({ open: 0, closed: 0 });
  const menuRenderProp = ({ keys }: Menu.ContextMenuMenuProps) => (
    <ContextMenuContent
      csvLines={csvLines}
      linePlotRef={linePlotRef}
      editable={hasUpdatePermission}
      breaks={breaks}
      breaksHidden={breaksHidden}
      menuKeys={keys}
      compare={
        relative
          ? {
              refs,
              ranges,
              hovered: keys.length === 0 ? menuHovered : null,
              onItemVisibleChange: handleItemVisibleChange,
              onIsolate: handleIsolate,
              hidden,
              scope,
              zoomed,
              onShowAll: handleShowAll,
            }
          : null
      }
    />
  );

  return (
    <div className={CSS.cls(CSS.BE("line-plot", "container"), menuProps.className)}>
      <Menu.ContextMenu {...menuProps} menu={menuRenderProp}>
        <Base.LinePlot
          ref={linePlotRef}
          aetherKey={key}
          editable={hasUpdatePermission}
          enableTriggers={hasUpdatePermission}
          ranges={ranges}
          enableTooltip={enableTooltip}
          enableMeasure={clickMode === "measure"}
          measureMode={vis.measure.mode}
          onMeasureModeChange={handleMeasureModeChange}
          initialViewport={initialViewport}
          onViewportChange={handleViewportChange}
          viewportTriggers={triggers}
          rangeProviderProps={rangeProviderProps}
          onSelectRule={handleSelectRule}
          hiddenLines={new Set(hidden.lines)}
          hiddenRanges={new Set(hidden.ranges)}
          hiddenTraces={new Set(hidden.traces)}
          onLineVisibleChange={handleLineVisibleChange}
          onRangeVisibleChange={handleRangeVisibleChange}
          onRangeColorChange={hasUpdatePermission ? handleRangeColorChange : undefined}
          onHoveredChange={handleHoveredChange}
          relativeDisabled={!FLAGS.compare}
          onZoomedChange={setZoomed}
          hold={hold}
          onHold={handleHold}
          onContextMenu={handleContextMenu}
          onDoubleClick={handleDoubleClick}
          clearOverScan={CLEAR_OVERSCAN}
          visible={visible}
          breaksHidden={breaksHidden}
          onBreakCountsChange={setBreaks}
          initialOpenBreaks={vis.breaks.open}
          onOpenBreaksChange={handleOpenBreaksChange}
        >
          <Controls
            hasAnnotations={hasAnnotations}
            hasBreaks={breaks.open + breaks.closed > 0}
          />
        </Base.LinePlot>
      </Menu.ContextMenu>
    </div>
  );
};

export const LinePlot: Panel.Content = () => {
  const { key } = PlutoPanel.useTabResource();
  return (
    <Vis.GPUBoundary name="Line plots">
      <Base.Suspended linePlotKey={key}>
        <Internal />
      </Base.Suspended>
    </Vis.GPUBoundary>
  );
};
