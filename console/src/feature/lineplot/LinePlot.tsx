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
  LinePlot as Base,
  Panel as PlutoPanel,
  Ranger,
  Viewport,
} from "@synnaxlabs/pluto";
import type { lineplot as pLineplot } from "@synnaxlabs/pluto/ether";
import { box, location, TimeRange, TimeSpan, type xy } from "@synnaxlabs/x";
import { type ReactElement, type RefObject, useRef, useState } from "react";

import { Controls } from "@/feature/lineplot/Controls";
import {
  type DownloadLine,
  useDownloadAsCSV,
} from "@/feature/lineplot/useDownloadAsCSV";
import { useResetZoomOnRangeChange } from "@/feature/lineplot/useResetZoomOnRangeChange";
import { useTriggerHold } from "@/feature/lineplot/useTriggerHold";
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

interface ContextMenuContentProps {
  csvLines: DownloadLine[];
  linePlotRef: RefObject<Base.FrameRef | null>;
  editable: boolean;
}

const ContextMenuContent = ({
  csvLines,
  linePlotRef,
  editable,
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
      {!box.areaIsZero(selection) && (
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
  const ranges = Session.Range.useResolveWindows(Base.useRanges()).map(
    (window, i): Base.RangeSpec => ({ key: i.toString(), window }),
  );

  const hiddenLineKeys = Session.LinePlot.useSelectHiddenLines();
  const hiddenLines = new Set(hiddenLineKeys);
  const handleLineVisibleChange = (lineKey: string, lineVisible: boolean) =>
    dispatch(Session.LinePlot.setLineVisible({ key, lineKey, visible: lineVisible }));

  const derived = Base.useLines();
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
  const linePlotRef = useRef<Base.FrameRef | null>(null);
  const [hasAnnotations, setHasAnnotations] = useState(false);

  const rangeProviderProps = {
    visible: vis.annotations.visible,
    onHasAnnotationsChange: setHasAnnotations,
    menu: (p: Base.Range.MenuProps) => (
      <RangeAnnotationContextMenu lines={csvLines} range={p} />
    ),
  };

  const menuRenderProp = () => (
    <ContextMenuContent
      csvLines={csvLines}
      linePlotRef={linePlotRef}
      editable={hasUpdatePermission}
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
          relativeDisabled
          enableTooltip={enableTooltip}
          enableMeasure={clickMode === "measure"}
          measureMode={vis.measure.mode}
          onMeasureModeChange={handleMeasureModeChange}
          initialViewport={initialViewport}
          onViewportChange={handleViewportChange}
          viewportTriggers={triggers}
          rangeProviderProps={rangeProviderProps}
          onSelectRule={handleSelectRule}
          hiddenLines={hiddenLines}
          onLineVisibleChange={handleLineVisibleChange}
          hold={hold}
          onHold={handleHold}
          onContextMenu={menuProps.open}
          onDoubleClick={handleDoubleClick}
          clearOverScan={CLEAR_OVERSCAN}
          visible={visible}
          gapsCollapsed={FLAGS.breaks}
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
