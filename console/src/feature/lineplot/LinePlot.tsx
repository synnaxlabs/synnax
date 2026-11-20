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
import { Button } from "@synnaxlabs/lyra/button";
import { useDebouncedCallback } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import {
  Access,
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
  scale,
  TimeRange,
  TimeSpan,
  TimeStamp,
  type xy,
} from "@synnaxlabs/x";
import { type ReactElement, type RefObject, useRef, useState } from "react";

import { Controls } from "@/feature/lineplot/Controls";
import {
  type DownloadLine,
  useDownloadAsCSV,
} from "@/feature/lineplot/useDownloadAsCSV";
import { useResetZoomOnRangeChange } from "@/feature/lineplot/useResetZoomOnRangeChange";
import { useSetT0Modal } from "@/feature/lineplot/useSetT0Modal";
import { useTriggerHold } from "@/feature/lineplot/useTriggerHold";
import {
  countHidden,
  inScope,
  type Item,
  NONE_HIDDEN,
  type Scope,
  setItemVisible,
  showOnly,
  toggleSolo,
  type Visibility,
} from "@/feature/lineplot/visibility";
import { Sift } from "@/feature/sift";
import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import type { Panel } from "@/platform/panel";
import { Range } from "@/platform/range";
import { Vis } from "@/platform/vis";
import { Session } from "@/session";

const CLEAR_OVERSCAN: xy.XY = { x: 5, y: 5 };
const VIEWPORT_DEBOUNCE = TimeSpan.milliseconds(100);

interface RangeAnnotationContextMenuProps {
  lines: DownloadLine[];
  range: Base.Range.MenuProps;
}

const RangeAnnotationContextMenu = ({
  lines,
  range,
}: RangeAnnotationContextMenuProps): ReactElement => {
  const downloadAsCSV = useDownloadAsCSV();
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

// Compare holds what the menus of a plot on a relative x axis act on.
interface Compare {
  refs: ranger.Ref[];
  ranges: Base.RangeSpec[];
  hovered: Base.Hovered | null;
  onItemVisibleChange: (item: Item, visible: boolean) => void;
  onShowOnly: (item: Item) => void;
  hiddenCount: number;
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

// CompareMenuItems lists the actions on a range heading or a line of one range in the
// legend, or on the line under the pointer.
const CompareMenuItems = ({
  menuKey,
  compare: {
    refs,
    ranges,
    hovered,
    onItemVisibleChange,
    onShowOnly,
    hiddenCount,
    zoomed,
    onShowAll,
  },
  linePlotRef,
  editable,
}: CompareMenuItemsProps): ReactElement | null => {
  const { dispatch } = Base.useDispatch();
  const key = Base.useKey();
  const setT0Modal = useSetT0Modal();
  const handleError = Status.useErrorHandler();
  const heading = ranges.find((r) => r.key === menuKey);
  const trace = heading == null ? Base.parseTraceKey(menuKey, ranges) : null;
  const target = hovered ?? trace;
  const range = heading ?? ranges.find((r) => r.key === target?.range);
  if (range?.zero == null) return null;
  const { zero } = range;
  const applyT0 = (t0: TimeStamp | undefined) =>
    dispatch({ key, actions: [lineplot.setRangeT0({ range: range.key, t0 })] });
  const isKey =
    refs.find((r) => Session.Range.keyOf(r) === range.key)?.variant === "key";
  const timeRange =
    range.window.variant === "static"
      ? TimeRange.z.parse(range.window.timeRange)
      : TimeRange.ZERO;
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
      const bounds = await linePlotRef.current?.getBounds();
      if (bounds == null) return;
      const s = scale.Scale.scale<number>(bounds.x1).scale(1);
      linePlotRef.current?.toggleZoom(range.key, {
        lower: s.pos(Number(timeRange.start.valueOf() - zero.valueOf())),
        upper: s.pos(Number(timeRange.end.valueOf() - zero.valueOf())),
      });
    }, "Failed to zoom to range");
  return (
    <>
      {hovered != null && editable && isKey && (
        <Menu.Item itemKey="setT0Here" onClick={() => applyT0(hovered.time)}>
          <Icon.Time /> Set T0 here
        </Menu.Item>
      )}
      <Menu.Item
        itemKey="showOnlyRange"
        onClick={() => onShowOnly({ variant: "range", range: range.key })}
      >
        <Icon.Visible /> Show only this range
      </Menu.Item>
      {target != null && (
        <>
          <Menu.Item
            itemKey="showOnlyLine"
            onClick={() => onShowOnly({ variant: "line", line: target.line })}
          >
            <Icon.Visible /> Show only this channel
          </Menu.Item>
          <Menu.Item
            itemKey="hideLine"
            onClick={() =>
              onItemVisibleChange({ variant: "line", line: target.line }, false)
            }
          >
            <Icon.Hidden /> Hide this channel in all ranges
          </Menu.Item>
        </>
      )}
      {hiddenCount > 0 && (
        <Menu.Item itemKey="showAll" onClick={onShowAll}>
          <Icon.Visible /> Show all
        </Menu.Item>
      )}
      {heading != null && (
        <>
          <Menu.Divider />
          <Menu.Item itemKey="zoomToRange" onClick={handleZoom}>
            {zoomed === range.key ? <Icon.Expand /> : <Icon.Zoom />}
            {zoomed === range.key ? "Zoom out" : "Zoom to range"}
          </Menu.Item>
          {editable && isKey && (
            <>
              <Menu.Item itemKey="setT0" onClick={handleSetT0}>
                <Icon.Time /> Set T0...
              </Menu.Item>
              {range.customZero === true && (
                <Menu.Item itemKey="resetT0" onClick={() => applyT0(undefined)}>
                  <Icon.Reset /> Reset T0
                </Menu.Item>
              )}
            </>
          )}
        </>
      )}
      <Menu.Divider />
    </>
  );
};

interface ContextMenuContentProps {
  csvLines: DownloadLine[];
  linePlotRef: RefObject<Base.FrameRef | null>;
  editable: boolean;
  menuKeys: string[];
  compare: Compare | null;
}

const ContextMenuContent = ({
  csvLines,
  linePlotRef,
  editable,
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
  const downloadAsCSV = useDownloadAsCSV();
  const getTimeRange = async (): Promise<TimeRange> => {
    const bounds = await linePlotRef.current?.getBounds();
    if (bounds == null) throw new Error("No bounds available");
    const s = scale.Scale.scale<number>(1).scale(bounds.x1);
    return new TimeRange(s.pos(box.left(selection)), s.pos(box.right(selection)));
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
  const relative = Base.isRelative(x1);

  const derived = Base.useLines();
  const scope: Scope = {
    ranges: ranges.filter((r) => r.zero != null).map((r) => r.key),
    lines: derived.map((d) => d.key),
  };
  const hidden = inScope(Session.LinePlot.useSelectHidden(), scope);
  const solo = Session.LinePlot.useSelectSolo();
  const { dispatch: dispatchPlot } = Base.useDispatch();
  const hoveredRef = useRef<Base.Hovered | null>(null);
  const setVisibility = (next: Visibility) =>
    dispatch(Session.LinePlot.setHidden({ key, ...next }));
  const handleItemVisibleChange = (item: Item, itemVisible: boolean) =>
    setVisibility(setItemVisible({ hidden, solo }, item, itemVisible, scope));
  const handleShowOnly = (item: Item) =>
    setVisibility(showOnly({ hidden, solo }, item, scope));
  const handleLegendChange = (item: Item, itemVisible: boolean, isSolo: boolean) =>
    isSolo
      ? setVisibility(toggleSolo({ hidden, solo }, item, scope))
      : handleItemVisibleChange(item, itemVisible);
  const handleShowAll = () => setVisibility({ hidden: NONE_HIDDEN, solo: null });
  const handleLineVisibleChange = (lineKey: string, v: boolean, isSolo: boolean) => {
    const trace = relative ? Base.parseTraceKey(lineKey, ranges) : null;
    if (!relative) handleLegendChange({ variant: "line", line: lineKey }, v, isSolo);
    else if (trace != null)
      handleLegendChange({ variant: "trace", ...trace }, v, isSolo);
  };
  const handleRangeVisibleChange = (range: string, v: boolean, isSolo: boolean) =>
    handleLegendChange({ variant: "range", range }, v, isSolo);
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
  const hiddenCount = countHidden(hidden);
  const legendFooter =
    relative && hiddenCount > 0 ? (
      <Button.Button variant="text" size="small" onClick={handleShowAll}>
        {`Show all (${hiddenCount} hidden)`}
      </Button.Button>
    ) : undefined;
  const handleHoveredChange = (h: Base.Hovered | null) => {
    hoveredRef.current = h;
  };
  const csvLines: DownloadLine[] = derived.map((d) => ({
    channels: { x: d.xChannel, y: d.yChannel },
  }));

  const { enableTooltip, clickMode, hold } = Session.LinePlot.useSelectControlState();
  const mode = Session.LinePlot.useSelectViewportMode();
  const triggers = Viewport.DEFAULT_TRIGGERS[mode];
  // Only setViewport bumps the trigger. A stored viewport came from the plot itself.
  const [applied, setApplied] = useState(vis.viewport);
  if (applied.renderTrigger !== vis.viewport.renderTrigger) setApplied(vis.viewport);
  const initialViewport = box.reRoot(
    box.construct(applied.pan, applied.zoom),
    location.BOTTOM_LEFT,
  );

  useTriggerHold({ key, enabled: hasUpdatePermission });
  useResetZoomOnRangeChange({ key });

  const handleViewportChange: Viewport.UseHandler = useDebouncedCallback(
    ({ box: b, stage, mode }) => {
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

  const menuRenderProp = ({ keys }: Menu.ContextMenuMenuProps) => (
    <ContextMenuContent
      csvLines={csvLines}
      linePlotRef={linePlotRef}
      editable={hasUpdatePermission}
      menuKeys={keys}
      compare={
        relative
          ? {
              refs,
              ranges,
              hovered: keys.length === 0 ? menuHovered : null,
              onItemVisibleChange: handleItemVisibleChange,
              onShowOnly: handleShowOnly,
              hiddenCount,
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
          legendFooter={legendFooter}
          onHoveredChange={handleHoveredChange}
          onZoomedChange={setZoomed}
          hold={hold}
          onHold={handleHold}
          onContextMenu={handleContextMenu}
          onDoubleClick={handleDoubleClick}
          clearOverScan={CLEAR_OVERSCAN}
          visible={visible}
        >
          <Controls hasAnnotations={hasAnnotations} />
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
