// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Icon } from "@synnaxlabs/media";
import {
  Channel,
  Color,
  Legend,
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
  deep,
  getEntries,
  location,
  scale,
  TimeRange,
  unique,
  type UnknownRecord,
} from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement, useCallback, useEffect, useMemo, useState } from "react";
import { useDispatch } from "react-redux";
import { v4 as uuidv4 } from "uuid";

import { Menu } from "@/components/menu";
import { UseSyncerArgs, useSyncerDispatch } from "@/hooks/dispatchers";
import { Layout } from "@/layout";
import {
  AxisKey,
  axisLocation,
  MultiXAxisRecord,
  X_AXIS_KEYS,
  XAxisKey,
  YAxisKey,
} from "@/lineplot/axis";
import { download } from "@/lineplot/download";
import {
  select,
  selectRanges,
  useSelect,
  useSelectAxisBounds,
  useSelectControlState,
  useSelectSelection,
  useSelectViewportMode,
} from "@/lineplot/selectors";
import {
  type AxisState,
  internalCreate,
  type LineState,
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
  type StoreState,
  storeViewport,
  typedLineKeyToString,
  ZERO_STATE,
} from "@/lineplot/slice";
import { Range } from "@/range";
import { Workspace } from "@/workspace";

interface SyncPayload {
  key?: string;
}

const Loaded = ({ layoutKey }: { layoutKey: string }): ReactElement => {
  const windowKey = useSelectWindowKey() as string;
  const { name } = Layout.useSelectRequired(layoutKey);
  const placer = Layout.usePlacer();
  const vis = useSelect(layoutKey);
  const ranges = selectRanges(layoutKey);
  const client = Synnax.use();
  const addStatus = Status.useAggregator();

  const syncLayout = useMutation<
    void,
    Error,
    UseSyncerArgs<Layout.StoreState & StoreState & Workspace.StoreState, SyncPayload>
  >({
    retry: 3,
    mutationFn: async ({ client, action: { key }, store }) => {
      if (key == null) return;
      const s = store.getState();
      const ws = Workspace.selectActiveKey(s);
      if (ws == null) return;
      const data = select(s, key);
      const la = Layout.selectRequired(s, key);
      if (!data.remoteCreated) store.dispatch(setRemoteCreated({ key }));
      await client.workspaces.linePlot.create(ws, {
        key,
        name: la.name,
        data: data as unknown as UnknownRecord,
      });
    },
    onError: (e, { store, action: { key } }) => {
      let message = "Failed to save line plot";
      if (key != null) {
        const data = Layout.select(store.getState(), key);
        if (data?.name != null) message += ` ${data.name}`;
      }
      addStatus({
        key: layoutKey,
        variant: "error",
        message,
        description: e.message,
      });
    },
  });

  const syncDispatch = useSyncerDispatch<
    Layout.StoreState & Workspace.StoreState & StoreState,
    SyncPayload
  >(syncLayout.mutate, 500);

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
      unique(toFetch.map((line) => line.channels.y)) as channel.KeysOrNames,
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
            axis: rule.axis as XAxisKey,
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
          axisKey: axis.key as AxisKey,
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

  const initialViewport = useMemo(() => {
    return box.reRoot(
      box.construct(vis.viewport.pan, vis.viewport.zoom),
      location.BOTTOM_LEFT,
    );
  }, [vis.viewport.renderTrigger]);

  const handleDoubleClick = useCallback(
    () =>
      dispatch(
        Layout.setNavDrawerVisible({ windowKey, key: "visualization", value: true }),
      ),
    [windowKey, dispatch],
  );

  const props = PMenu.useContextMenu();

  interface ContextMenuContentProps {
    layoutKey: string;
  }

  const ContextMenuContent = ({ layoutKey }: ContextMenuContentProps): ReactElement => {
    const { box: selection } = useSelectSelection(layoutKey);
    const bounds = useSelectAxisBounds(layoutKey, "x1");

    const s = scale.Scale.scale(1).scale(bounds);
    const timeRange = new TimeRange(
      s.pos(box.left(selection)),
      s.pos(box.right(selection)),
    );

    const newLayout = Layout.usePlacer();
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
          dispatch(
            Range.setBuffer({
              timeRange: {
                start: Number(timeRange.start.valueOf()),
                end: Number(timeRange.end.valueOf()),
              },
            }),
          );
          newLayout(Range.createEditLayout());
          break;
        case "download":
          if (client == null) return;
          download({ timeRange, lines, client });
          break;
      }
    };

    return (
      <PMenu.Menu onChange={handleSelect} iconSpacing="small" level="small">
        {!box.areaIsZero(selection) && (
          <>
            <PMenu.Item itemKey="iso" startIcon={<Icon.Range />}>
              Copy time range as ISO
            </PMenu.Item>
            <PMenu.Item itemKey="python" startIcon={<Icon.Python />}>
              Copy time range as Python
            </PMenu.Item>
            <PMenu.Item itemKey="typescript" startIcon={<Icon.TypeScript />}>
              Copy time range as TypeScript
            </PMenu.Item>
            <PMenu.Item itemKey="range" startIcon={<Icon.Add />}>
              Create new range from selection
            </PMenu.Item>
            <PMenu.Item itemKey="download" startIcon={<Icon.Download />}>
              Download data as CSV
            </PMenu.Item>
          </>
        )}
        <Menu.HardReloadItem />
      </PMenu.Menu>
    );
  };

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
          enableMeasure={clickMode === "measure"}
          onDoubleClick={handleDoubleClick}
          onHold={(hold) => dispatch(setControlState({ state: { hold } }))}
          annotationProvider={{
            menu: ({ key, timeRange, name }) => {
              const handleSelect = (itemKey: string) => {
                switch (itemKey) {
                  case "download":
                    if (client == null) return;
                    download({ client, lines, timeRange, name });
                    break;
                  case "meta-data":
                    placer({
                      ...Range.metaDataWindowLayout,
                      name: `${name} Meta Data`,
                      key: key,
                    });
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
                  <PMenu.Item itemKey="line-plot" startIcon={<Icon.Visualize />}>
                    Open in New Plot
                  </PMenu.Item>
                  <PMenu.Item itemKey="meta-data" startIcon={<Icon.Annotate />}>
                    View Meta Data
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
  getEntries(vis.axes.axes)
    .filter(([key]) => shouldDisplayAxis(key, vis))
    .map(([key, axis]): Channel.AxisProps => {
      return {
        location: axisLocation(key as AxisKey),
        type: X_AXIS_KEYS.includes(key as XAxisKey) ? "time" : "linear",
        ...axis,
      };
    });

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
              xAxis: xAxis as XAxisKey,
              yAxis: yAxis as YAxisKey,
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

export type LayoutType = "lineplot";
export const LAYOUT_TYPE = "lineplot";

export const create =
  (initial: Partial<State> & Omit<Partial<Layout.State>, "type">): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "Line Plot", location = "mosaic", window, tab, ...rest } = initial;
    const key = initial.key ?? uuidv4();
    dispatch(internalCreate({ ...deep.copy(ZERO_STATE), ...rest, key }));
    return {
      key,
      name,
      location,
      type: LAYOUT_TYPE,
      icon: "Visualize",
      window,
      tab,
    };
  };

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

export const SELECTABLE: Layout.Selectable = {
  key: LAYOUT_TYPE,
  title: "Line Plot",
  icon: <Icon.Visualize />,
  create: (layoutKey: string) => create({ key: layoutKey }),
};
