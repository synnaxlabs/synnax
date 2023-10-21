// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback, useMemo, useRef } from "react";

import { Icon } from "@synnaxlabs/media";
import {
  PID as Core,
  PIDElement,
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
  useSelectViewporMode,
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

const ElementRenderer = ({
  elementKey,
  position,
  selected,
  layoutKey,
  editable,
  zoom,
}: Core.ElementProps & { layoutKey: string }): ReactElement | null => {
  const el = useSelectElementProps(layoutKey, elementKey);
  if (el == null) return null;
  const {
    props: { type, ...props },
  } = el;
  const dispatch = useSyncerDispatch<
    Layout.StoreState & Workspace.StoreState & StoreState,
    SyncPayload
  >(syncer, 1000);

  const handleChange = useCallback(
    (props: object) => {
      dispatch(
        setElementProps({ layoutKey, key: elementKey, props: { type, ...props } }),
      );
    },
    [dispatch, elementKey, layoutKey, type],
  );

  const C = PIDElement.REGISTRY[type];

  const refZoom = useRef(zoom);

  return (
    <C.Element
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
      return <ElementRenderer layoutKey={layoutKey} {...props} />;
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
      valid.forEach(({ key: type, data }, i) => {
        const spec = PIDElement.REGISTRY[type];
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
              type,
              ...spec.initialProps(theme),
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

  const mode = useSelectViewporMode();
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

  return (
    <div ref={ref} style={{ width: "inherit", height: "inherit" }}>
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
          {...dropProps}
        >
          <Core.NodeRenderer>{elRenderer}</Core.NodeRenderer>
          <Core.Background />
          <Core.Controls reverse>
            {!pid.snapshot && (
              <Core.ToggleEditControl disabled={pid.control === "acquired"} />
            )}
            <Core.FitViewControl />
            {!pid.snapshot && (
              <Button.ToggleIcon
                value={pid.control === "acquired"}
                onChange={acquireControl}
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
