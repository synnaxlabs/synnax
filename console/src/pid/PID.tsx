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
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Icon } from "@synnaxlabs/media";
import {
  Control,
  Button,
  Haul,
  Theming,
  Text,
  Viewport,
  useSyncedRef,
  Synnax,
  useAsyncEffect,
  Diagram,
  PID as Core,
} from "@synnaxlabs/pluto";
import { Triggers } from "@synnaxlabs/pluto/triggers";
import { type UnknownRecord, box } from "@synnaxlabs/x";
import { nanoid } from "nanoid/non-secure";
import { useDispatch } from "react-redux";

import { useSyncerDispatch, type Syncer } from "@/hooks/dispatchers";
import { Layout } from "@/layout";
import {
  select,
  useSelect,
  useSelectNodeProps,
  useSelectViewport,
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
}: Diagram.SymbolProps & { layoutKey: string }): ReactElement | null => {
  const { key, ...props } = useSelectNodeProps(layoutKey, symbolKey);
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
          props: { key, ...props },
        }),
      );
    },
    [dispatch, symbolKey, layoutKey, key],
  );

  const C = Core.SYMBOLS[key as Core.Variant];
  if (C == null) {
    throw new Error(`Symbol ${key} not found`);
  }

  const zoom = useSelectViewport(layoutKey);

  return (
    <C.Symbol
      aetherKey={symbolKey}
      position={position}
      selected={selected}
      onChange={handleChange}
      zoom={zoom.zoom}
      {...props}
    />
  );
};

export const Loaded: Layout.Renderer = ({ layoutKey }) => {
  const windowKey = useSelectWindowKey() as string;
  const { name } = Layout.useSelectRequired(layoutKey);
  const pid = useSelect(layoutKey);

  const dispatch = useSyncerDispatch<Layout.StoreState & StoreState, SyncPayload>(
    // @ts-expect-error - typescript can't identify property keys set as constants.
    syncer,
    1000,
  );
  const theme = Theming.use();
  const viewportRef = useSyncedRef(pid.viewport);

  const handleEdgesChange: Diagram.DiagramProps["onEdgesChange"] = useCallback(
    (edges) => {
      dispatch(setEdges({ layoutKey, edges }));
    },
    [dispatch, layoutKey],
  );

  const handleNodesChange: Diagram.DiagramProps["onNodesChange"] = useCallback(
    (nodes) => {
      dispatch(setNodes({ layoutKey, nodes }));
    },
    [dispatch, layoutKey],
  );

  const handleViewportChange: Diagram.DiagramProps["onViewportChange"] = useCallback(
    (vp) => {
      dispatch(setViewport({ layoutKey, viewport: vp }));
    },
    [layoutKey],
  );

  const handleEditableChange: Diagram.DiagramProps["onEditableChange"] = useCallback(
    (cbk) => {
      dispatch(setEditable({ layoutKey, editable: cbk }));
    },
    [layoutKey],
  );

  const handleControlStatusChange = useCallback(
    (control: Control.Status) => {
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
    (props: Diagram.SymbolProps) => {
      return <SymbolRenderer layoutKey={layoutKey} {...props} />;
    },
    [layoutKey],
  );

  const ref = useRef<HTMLDivElement>(null);

  const handleDrop = useCallback(
    ({ items, event }: Haul.OnDropProps): Haul.Item[] => {
      console.log(items);
      const valid = Haul.filterByType(HAUL_TYPE, items);
      if (ref.current == null) return valid;
      const region = box.construct(ref.current);
      valid.forEach(({ key, data }) => {
        const spec = Core.SYMBOLS[key as Core.Variant];
        if (spec == null) return;
        const pos = calculatePos(
          region,
          { x: event.clientX, y: event.clientY },
          viewportRef.current,
        );
        dispatch(
          addElement({
            layoutKey,
            key: nanoid(),
            node: {
              position: pos,
              zIndex: spec.zIndex,
            },
            props: {
              key,
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
        windowKey,
        key: "visualization",
        value: true,
      }) as PayloadAction<SyncPayload>,
    );
  }, [windowKey, dispatch, pid.editable]);

  return (
    <div
      ref={ref}
      onDoubleClick={handleDoubleClick}
      style={{ width: "inherit", height: "inherit", position: "relative" }}
    >
      <Control.Controller
        name={name}
        authority={1}
        acquireTrigger={pid.controlAcquireTrigger}
        onStatusChange={handleControlStatusChange}
      >
        <Diagram.Diagram
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
          <Diagram.NodeRenderer>{elRenderer}</Diagram.NodeRenderer>
          <Diagram.Background />
          <Diagram.Controls>
            {!pid.snapshot && (
              <Diagram.ToggleEditControl disabled={pid.control === "acquired"} />
            )}
            <Diagram.FitViewControl />
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
          </Diagram.Controls>
        </Diagram.Diagram>
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
