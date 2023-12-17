// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback, useMemo, useRef } from "react";

import { type PayloadAction } from "@reduxjs/toolkit";
import { Icon } from "@synnaxlabs/media";
import {
  PIDSymbols,
  PID as Core,
  Control,
  Button,
  Haul,
  Theming,
  Text,
  Viewport,
  Triggers,
  useSyncedRef,
  Synnax,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { type UnknownRecord, box, scale, xy } from "@synnaxlabs/x";
import { nanoid } from "nanoid";
import { useDispatch } from "react-redux";

import { useSyncerDispatch, type Syncer } from "@/hooks/dispatchers";
import { Layout } from "@/layout";
import {
  select,
  useSelect,
  useSelectElementProps,
  useSelectViewportMode,
} from "@/pid/selectors";
import {
  toggleControl,
  setControlStatus,
  setEdges,
  setEditable,
  setElementProps,
  setNodes,
  setViewport,
  addElement,
  copySelection,
  calculatePos,
  pasteSelection,
  type StoreState,
  type State,
  internalCreate,
  setRemoteCreated,
} from "@/pid/slice";
import { Workspace } from "@/workspace";

interface SyncPayload {
  layoutKey?: string;
}

const syncer: Syncer<
  Layout.StoreState & StoreState & Workspace.StoreState,
  SyncPayload
> = async (client, { layoutKey }, store) => {
  if (layoutKey == null) return;
  const s = store.getState();
  const ws = Workspace.selectActiveKey(s);
  if (ws == null) return;
  const data = select(s, layoutKey);
  if (data.snapshot) return;
  const la = Layout.selectRequired(s, layoutKey);
  const setData = {
    ...data,
    key: undefined,
    snapshot: undefined,
  } as unknown as UnknownRecord;
  if (!data.remoteCreated) {
    store.dispatch(setRemoteCreated({ layoutKey }));
    await client.workspaces.pid.create(ws, {
      key: layoutKey,
      name: la.name,
      data: setData,
    });
  } else await client.workspaces.pid.setData(layoutKey, setData);
};

export const HAUL_TYPE = "pid-element";

const SymbolRenderer = ({
  symbolKey,
  position,
  selected,
  layoutKey,
  editable,
  zoom,
}: Core.SymbolProps & { layoutKey: string }): ReactElement | null => {
  const el = useSelectElementProps(layoutKey, symbolKey);
  if (el == null) return null;
  const {
    props: { variant, ...props },
  } = el;
  const dispatch = useSyncerDispatch<
    Layout.StoreState & Workspace.StoreState & StoreState,
    SyncPayload
  >(syncer, 1000);

  const handleChange = useCallback(
    (props: object) => {
      dispatch(
        setElementProps({
          layoutKey,
          key: symbolKey,
          props: { variant, ...props },
        }),
      );
    },
    [dispatch, symbolKey, layoutKey, variant],
  );

  const C = PIDSymbols.registry[variant];

  const refZoom = useRef(zoom);

  return (
    <C.Symbol
      position={position}
      selected={selected}
      onChange={handleChange}
      editable={editable}
      zoom={refZoom.current}
      {...props}
    />
  );
};

export const Loaded: Layout.Renderer = ({ layoutKey }) => {
  const { name } = Layout.useSelectRequired(layoutKey);
  const pid = useSelect(layoutKey);

  const dispatch = useSyncerDispatch<Layout.StoreState & StoreState, SyncPayload>(
    syncer,
    1000,
  );
  const theme = Theming.use();
  const viewportRef = useSyncedRef(pid.viewport);

  const handleEdgesChange: Core.PIDProps["onEdgesChange"] = useCallback(
    (edges) => {
      dispatch(setEdges({ layoutKey, edges }));
    },
    [dispatch, layoutKey],
  );

  const handleNodesChange: Core.PIDProps["onNodesChange"] = useCallback(
    (nodes) => {
      dispatch(setNodes({ layoutKey, nodes }));
    },
    [dispatch, layoutKey],
  );

  const handleViewportChange: Core.PIDProps["onViewportChange"] = useCallback(
    (vp) => {
      dispatch(setViewport({ layoutKey, viewport: vp }));
    },
    [layoutKey],
  );

  const handleEditableChange: Core.PIDProps["onEditableChange"] = useCallback(
    (cbk) => {
      dispatch(setEditable({ layoutKey, editable: cbk }));
    },
    [layoutKey],
  );

  const handleControlStatusChange: Control.ControllerProps["onStatusChange"] =
    useCallback(
      (control) => {
        dispatch(setControlStatus({ layoutKey, control }));
      },
      [layoutKey],
    );

  const acquireControl = useCallback(
    (v: boolean) => {
      dispatch(
        toggleControl({
          layoutKey,
          status: v ? "acquired" : "released",
        }),
      );
    },
    [layoutKey],
  );

  const elRenderer = useCallback(
    (props: Core.ElementProps) => {
      return <SymbolRenderer layoutKey={layoutKey} {...props} />;
    },
    [layoutKey],
  );

  const ref = useRef<HTMLDivElement>(null);

  const handleDrop = useCallback(
    ({ items, event }: Haul.OnDropProps): Haul.Item[] => {
      const valid = Haul.filterByType(HAUL_TYPE, items);
      if (ref.current == null) return valid;
      const region = box.construct(ref.current);
      const OFFSET = 20;
      valid.forEach(({ key: variant, data }, i) => {
        const spec = PIDSymbols.registry[variant as PIDSymbols.Variant];
        if (spec == null) return;
        const zoomXY = xy.construct(pid.viewport.zoom);
        const s = scale.XY.translate(xy.scale(box.topLeft(region), -1))
          .magnify({
            x: 1 / zoomXY.x,
            y: 1 / zoomXY.y,
          })
          .translate(xy.scale(pid.viewport.position, -1));
        dispatch(
          addElement({
            layoutKey,
            key: nanoid(),
            node: {
              position: s.pos({
                x: event.clientX + OFFSET * i,
                y: event.clientY + OFFSET * i,
              }),
            },
            props: {
              variant,
              ...spec.defaultProps(theme),
              ...(data ?? {}),
            },
          }),
        );
      });
      return valid;
    },
    [pid.viewport, theme],
  );

  const dropProps = Haul.useDrop({
    type: "PID",
    key: layoutKey,
    canDrop: Haul.canDropOfType(HAUL_TYPE),
    onDrop: handleDrop,
  });

  const mode = useSelectViewportMode();
  const triggers = useMemo(() => Viewport.DEFAULT_TRIGGERS[mode], [mode]);

  Triggers.use({
    triggers: [
      ["Control", "V"],
      ["Control", "C"],
    ],
    region: ref,
    callback: useCallback(
      ({ triggers, cursor, stage }: Triggers.UseEvent) => {
        if (ref.current == null || stage !== "start") return;
        const region = box.construct(ref.current);
        const copy = triggers.some((t) => t.includes("C"));
        const pos = calculatePos(region, cursor, viewportRef.current);
        if (copy) dispatch(copySelection({ pos }));
        else dispatch(pasteSelection({ pos, layoutKey }));
      },
      [dispatch, layoutKey, viewportRef],
    ),
  });

  const handleDoubleClick = useCallback(() => {
    if (!pid.editable) return;
    dispatch(
      Layout.setNavdrawerVisible({
        key: "visualization",
        value: true,
      }) as PayloadAction<SyncPayload>,
    );
  }, [dispatch, pid.editable]);

  Triggers.use({
    triggers: [["MouseLeft", "MouseLeft"]],
    region: ref,
    callback: handleDoubleClick,
  });

  return (
    <div
      ref={ref}
      style={{ width: "inherit", height: "inherit", position: "relative" }}
    >
      <Control.Controller
        name={name}
        authority={1}
        acquireTrigger={pid.controlAcquireTrigger}
        onStatusChange={handleControlStatusChange}
      >
        <Core.PID
          onViewportChange={handleViewportChange}
          edges={pid.edges}
          nodes={pid.nodes}
          viewport={pid.viewport}
          onEdgesChange={handleEdgesChange}
          onNodesChange={handleNodesChange}
          onEditableChange={handleEditableChange}
          editable={pid.editable}
          triggers={triggers}
          onDoubleClick={handleDoubleClick}
          {...dropProps}
        >
          <Core.NodeRenderer>{elRenderer}</Core.NodeRenderer>
          <Core.Background />
          <Core.Controls>
            {!pid.snapshot && (
              <Core.ToggleEditControl disabled={pid.control === "acquired"} />
            )}
            <Core.FitViewControl />
            {!pid.snapshot && (
              <Button.ToggleIcon
                value={pid.control === "acquired"}
                onChange={acquireControl}
                tooltipLocation={{ x: "right", y: "center" }}
                variant="outlined"
                tooltip={
                  <Text.Text level="small">
                    {pid.control === "acquired" ? "Release control" : "Acquire control"}
                  </Text.Text>
                }
              >
                <Icon.Circle />
              </Button.ToggleIcon>
            )}
          </Core.Controls>
        </Core.PID>
        <Control.Legend />
      </Control.Controller>
    </div>
  );
};

export const PID: Layout.Renderer = ({ layoutKey, ...props }): ReactElement | null => {
  const pid = useSelect(layoutKey);
  const dispatch = useDispatch();
  const client = Synnax.use();
  useAsyncEffect(async () => {
    if (client == null || pid != null) return;
    const { data } = await client.workspaces.pid.retrieve(layoutKey);
    dispatch(internalCreate({ key: layoutKey, ...(data as unknown as State) }));
  }, [client, pid]);
  if (pid == null) return null;
  return <Loaded layoutKey={layoutKey} {...props} />;
};
