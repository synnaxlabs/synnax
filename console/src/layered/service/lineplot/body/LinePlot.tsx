// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, type ranger } from "@synnaxlabs/client";
import {
  Access,
  Icon,
  LinePlot as PLinePlot,
  Menu,
  Ranger,
  Status,
  useDebouncedCallback,
  Viewport,
} from "@synnaxlabs/pluto";
import { type lineplot as etherLineplot } from "@synnaxlabs/pluto/ether";
import { box, location, scale, TimeRange, TimeSpan, unique } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo, useRef, useState } from "react";
import { useDispatch, useStore } from "react-redux";

import { ContextMenu } from "@/components";
import { Controls } from "@/layered/service/lineplot/body/Controls";
import {
  type DownloadLine,
  useDownloadAsCSV,
} from "@/layered/service/lineplot/useDownloadAsCSV";
import { Session } from "@/layered/session";
import { Layout } from "@/layout";
import { Range } from "@/range";
import { type RootState } from "@/store";

interface RangeAnnotationContextMenuProps {
  lines: DownloadLine[];
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

const Internal: Layout.Renderer = ({ focused, visible }) => {
  const key = PLinePlot.useKey();
  const { name } = Layout.useSelectRequired(key);
  const vis = Session.LinePlot.useSelect();
  const dispatch = useDispatch();
  const store = useStore<RootState>();
  const hasUpdatePermission = Access.useUpdateGranted(lineplot.ontologyID(key));

  const ranges = PLinePlot.useSelectRanges();
  const rangeKeys = unique.unique([...ranges.x1, ...ranges.x2]);
  const resolved = Range.useSelectMultiple(rangeKeys);
  const resolvedRanges = useMemo(() => {
    const m = new Map<string, PLinePlot.ResolvedRange>();
    for (const r of resolved)
      m.set(
        r.key,
        r.variant === "dynamic"
          ? { variant: "dynamic", span: new TimeSpan(r.span) }
          : { variant: "static", timeRange: new TimeRange(r.timeRange) },
      );
    return m;
  }, [resolved]);

  const hiddenLineKeys = Session.LinePlot.useSelectHiddenLines();
  const hiddenLines = useMemo(() => new Set(hiddenLineKeys), [hiddenLineKeys]);
  const handleLineVisibleChange = useCallback(
    (lineKey: string, lineVisible: boolean) =>
      dispatch(Session.LinePlot.setLineVisible({ key, lineKey, visible: lineVisible })),
    [dispatch, key],
  );

  const derived = PLinePlot.useSelectLines();
  const csvLines = useMemo<DownloadLine[]>(
    () =>
      derived.map((d) => ({
        channels: { x: d.xChannel, y: d.yChannel },
      })),
    [derived],
  );

  const { enableTooltip, clickMode, hold } = Session.LinePlot.useSelectControlState();
  const mode = Session.LinePlot.useSelectViewportMode();
  const triggers = useMemo(() => Viewport.DEFAULT_TRIGGERS[mode], [mode]);
  const initialViewport = useMemo(
    () =>
      box.reRoot(
        box.construct(vis.viewport.pan, vis.viewport.zoom),
        location.BOTTOM_LEFT,
      ),
    [vis.viewport.renderTrigger],
  );

  const enableTriggers = useCallback(
    () =>
      Layout.selectActiveMosaicTabKeyAndNotBlurred(store.getState()) === key &&
      hasUpdatePermission,
    [store, key, hasUpdatePermission],
  );

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
    TimeSpan.milliseconds(100),
    [dispatch, key],
  );

  const handleSelectRule = useCallback(
    (ruleKey: string) => dispatch(Session.LinePlot.setSelectedRule({ key, ruleKey })),
    [dispatch, key],
  );
  const handleMeasureModeChange = useCallback(
    (m: etherLineplot.measure.Mode) =>
      dispatch(Session.LinePlot.setMeasureMode({ key, mode: m })),
    [dispatch, key],
  );
  const handleHold = useCallback(
    (h: boolean) =>
      dispatch(Session.LinePlot.setControlState({ key, state: { hold: h } })),
    [dispatch, key],
  );
  const handleDoubleClick = useCallback(() => {
    dispatch(Layout.setNavDrawerVisible({ key: "visualization", value: true }));
    dispatch(Session.LinePlot.setActiveToolbarTab({ key, tab: "data" }));
  }, [dispatch, key]);

  const menuProps = Menu.useContextMenu();
  const linePlotRef = useRef<PLinePlot.FrameRef | null>(null);
  const [hasAnnotations, setHasAnnotations] = useState(false);

  const ContextMenuContent = (): ReactElement => {
    const { box: selection } = Session.LinePlot.useSelectSelection();
    const placeLayout = Layout.usePlacer();
    const handleError = Status.useErrorHandler();
    const downloadAsCSV = useDownloadAsCSV();
    const getTimeRange = useCallback(async (): Promise<TimeRange> => {
      const bounds = await linePlotRef.current?.getBounds();
      if (bounds == null) throw new Error("No bounds available");
      const s = scale.Scale.scale<number>(1).scale(bounds.x1);
      return new TimeRange(s.pos(box.left(selection)), s.pos(box.right(selection)));
    }, [selection]);
    const copyText = (fmt: (tr: TimeRange) => string) => async () =>
      fmt(await getTimeRange());
    const handleCreateRange = () =>
      handleError(async () => {
        const tr = await getTimeRange();
        placeLayout(Range.createCreateLayout({ timeRange: tr.numeric }));
      }, "Failed to create range from selection");
    const handleDownloadCSV = () =>
      handleError(async () => {
        const tr = await getTimeRange();
        downloadAsCSV({ timeRanges: [tr], lines: csvLines, name });
      }, "Failed to download region as CSV");
    return (
      <ContextMenu.Menu>
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

  return (
    <div
      style={{ height: "100%", width: "100%", padding: "2rem" }}
      className={menuProps.className}
    >
      <Menu.ContextMenu {...menuProps} menu={() => <ContextMenuContent />}>
        <PLinePlot.LinePlot
          ref={linePlotRef}
          aetherKey={key}
          editable={hasUpdatePermission}
          enableTriggers={enableTriggers}
          resolvedRanges={resolvedRanges}
          legendVariant={focused ? "fixed" : "floating"}
          enableTooltip={enableTooltip}
          enableMeasure={clickMode === "measure"}
          measureMode={vis.measure.mode}
          onMeasureModeChange={handleMeasureModeChange}
          initialViewport={initialViewport}
          onViewportChange={handleViewportChange}
          viewportTriggers={triggers}
          rangeProviderProps={{
            visible: vis.annotations.visible,
            onHasAnnotationsChange: setHasAnnotations,
            menu: (p) => <RangeAnnotationContextMenu lines={csvLines} range={p} />,
          }}
          onSelectRule={handleSelectRule}
          hiddenLines={hiddenLines}
          onLineVisibleChange={handleLineVisibleChange}
          hold={hold}
          onHold={handleHold}
          onContextMenu={menuProps.open}
          onDoubleClick={handleDoubleClick}
          clearOverScan={{ x: 5, y: 5 }}
          visible={visible}
        >
          {!focused && <Controls hasAnnotations={hasAnnotations} />}
        </PLinePlot.LinePlot>
      </Menu.ContextMenu>
      {focused && <Controls hasAnnotations={hasAnnotations} />}
    </div>
  );
};

export const LinePlot: Layout.Renderer = (props) => (
  <PLinePlot.Suspended linePlotKey={props.layoutKey}>
    <Internal {...props} />
  </PLinePlot.Suspended>
);

LinePlot.useName = Layout.createUseFluxName(
  PLinePlot.useRename,
  PLinePlot.useRetrieveObservableName,
);
