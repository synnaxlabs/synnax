// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Dispatch,
  type PayloadAction,
  type UnknownAction,
} from "@reduxjs/toolkit";
import { schematic } from "@synnaxlabs/client";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import {
  Button,
  Control,
  Diagram,
  Flex,
  Haul,
  Icon,
  type Legend,
  Menu as PMenu,
  Schematic as Core,
  Text,
  Theming,
  Triggers,
  usePrevious,
  useSyncedRef,
  Viewport,
} from "@synnaxlabs/pluto";
import { box, deep, location, uuid, xy } from "@synnaxlabs/x";
import {
  type ReactElement,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useDispatch } from "react-redux";

import { useLoadRemote } from "@/hooks/useLoadRemote";
import { useUndoableDispatch } from "@/hooks/useUndoableDispatch";
import { Layout } from "@/layout";
import {
  selectHasPermission,
  selectOptional,
  selectRequired,
  useSelectEditable,
  useSelectHasPermission,
  useSelectNodeProps,
  useSelectRequired,
  useSelectRequiredViewportMode,
  useSelectVersion,
} from "@/schematic/selectors";
import {
  calculatePos,
  clearSelection,
  copySelection,
  internalCreate,
  pasteSelection,
  selectAll,
  setControlStatus,
  setEdges,
  setEditable,
  setElementProps,
  setFitViewOnResize,
  setLegend,
  setNodes,
  setRemoteCreated,
  setViewport,
  setViewportMode,
  type State,
  toggleControl,
  ZERO_STATE,
} from "@/schematic/slice";
import { useAddSymbol } from "@/schematic/symbols/useAddSymbol";
import { type Selector } from "@/selector";
import { type RootState } from "@/store";
import { Workspace } from "@/workspace";

interface SyncPayload {
  key?: string;
}

export const HAUL_TYPE = "schematic-element";

const useSyncComponent = (
  layoutKey: string,
  dispatch?: Dispatch<PayloadAction<SyncPayload>>,
): Dispatch<PayloadAction<SyncPayload>> =>
  Workspace.useSyncComponent<SyncPayload>(
    "Schematic",
    layoutKey,
    async (ws, store, client) => {
      const storeState = store.getState();
      if (!selectHasPermission(storeState)) return;
      const data = selectOptional(storeState, layoutKey);
      if (data == null) return;
      const layout = Layout.selectRequired(storeState, layoutKey);
      if (data.snapshot) {
        await client.workspaces.schematic.rename(layoutKey, layout.name);
        return;
      }
      const setData = { ...data, key: undefined };
      if (!data.remoteCreated) store.dispatch(setRemoteCreated({ key: layoutKey }));
      await client.workspaces.schematic.create(ws, {
        key: layoutKey,
        name: layout.name,
        data: setData,
      });
    },
    dispatch,
  );

interface SymbolRendererProps extends Diagram.SymbolProps {
  layoutKey: string;
  dispatch: Dispatch<UnknownAction>;
}

const SymbolRenderer = ({
  symbolKey,
  position,
  selected,
  layoutKey,
  dispatch,
}: SymbolRendererProps): ReactElement | null => {
  const props = useSelectNodeProps(layoutKey, symbolKey);
  const key = props?.key;
  const handleChange = useCallback(
    (props: object) => {
      if (key == null) return;
      dispatch(
        setElementProps({ layoutKey, key: symbolKey, props: { key, ...props } }),
      );
    },
    [symbolKey, layoutKey, key, dispatch],
  );

  if (props == null) return null;

  const C = Core.Symbol.REGISTRY[key as Core.Symbol.Variant];

  if (C == null) throw new Error(`Symbol ${key} not found`);

  // Just here to make sure we don't spread the key into the symbol.
  const { key: _, ...rest } = props;

  return (
    <C.Symbol
      key={key}
      id={symbolKey}
      symbolKey={symbolKey}
      position={position}
      selected={selected}
      onChange={handleChange}
      {...rest}
    />
  );
};

export const ContextMenu: Layout.ContextMenuRenderer = ({ layoutKey }) => (
  <PMenu.Menu level="small" gap="small">
    <Layout.MenuItems layoutKey={layoutKey} />
  </PMenu.Menu>
);

export const Loaded: Layout.Renderer = ({ layoutKey, visible }) => {
  const windowKey = useSelectWindowKey() as string;
  const { name } = Layout.useSelectRequired(layoutKey);
  const schematic = useSelectRequired(layoutKey);
  const dispatch = useDispatch();
  const syncDispatch = useSyncComponent(layoutKey);
  const selector = useCallback(
    (state: RootState) => selectRequired(state, layoutKey),
    [layoutKey],
  );
  const [undoableDispatch_, undo, redo] = useUndoableDispatch<RootState, State>(
    selector,
    internalCreate,
    30, // roughly the right time needed to prevent actions that get dispatch automatically by Diagram.tsx, like setNodes immediately following addElement
  );
  const undoableDispatch = useSyncComponent(layoutKey, undoableDispatch_);

  const theme = Theming.use();
  const viewportRef = useSyncedRef(schematic.viewport);

  const prevName = usePrevious(name);
  useEffect(() => {
    if (prevName !== name) syncDispatch(Layout.rename({ key: layoutKey, name }));
  }, [name, prevName, layoutKey, syncDispatch]);

  const isEditable = useSelectEditable(layoutKey);
  const canBeEditable = useSelectHasPermission();
  useEffect(() => {
    if (!canBeEditable && isEditable)
      syncDispatch(setEditable({ key: layoutKey, editable: false }));
  }, [canBeEditable, isEditable, layoutKey, syncDispatch]);

  const handleEdgesChange: Diagram.DiagramProps["onEdgesChange"] = useCallback(
    (edges) => undoableDispatch(setEdges({ key: layoutKey, edges })),
    [layoutKey, undoableDispatch],
  );

  const handleNodesChange: Diagram.DiagramProps["onNodesChange"] = useCallback(
    (nodes, changes) => {
      // @ts-expect-error - Sometimes, the nodes do have dragging
      if (nodes.some((n) => n.dragging) || changes.some((c) => c.type === "select"))
        // don't remember dragging a node or selecting an element
        syncDispatch(setNodes({ key: layoutKey, nodes }));
      else undoableDispatch(setNodes({ key: layoutKey, nodes }));
    },
    [layoutKey, syncDispatch, undoableDispatch],
  );

  const handleViewportChange: Diagram.DiagramProps["onViewportChange"] = useCallback(
    (vp) => syncDispatch(setViewport({ key: layoutKey, viewport: vp })),
    [layoutKey, syncDispatch],
  );

  const handleEditableChange: Diagram.DiagramProps["onEditableChange"] = useCallback(
    (cbk) => syncDispatch(setEditable({ key: layoutKey, editable: cbk })),
    [layoutKey, syncDispatch],
  );

  const handleSetFitViewOnResize = useCallback(
    (v: boolean) =>
      syncDispatch(setFitViewOnResize({ key: layoutKey, fitViewOnResize: v })),
    [layoutKey, syncDispatch],
  );

  const handleControlStatusChange = useCallback(
    (control: Control.Status) =>
      syncDispatch(setControlStatus({ key: layoutKey, control })),
    [layoutKey, syncDispatch],
  );

  const acquireControl = useCallback(
    (v: boolean) =>
      syncDispatch(
        toggleControl({ key: layoutKey, status: v ? "acquired" : "released" }),
      ),
    [layoutKey, syncDispatch],
  );

  const elRenderer = useCallback(
    (props: Diagram.SymbolProps) => (
      <SymbolRenderer layoutKey={layoutKey} dispatch={undoableDispatch} {...props} />
    ),
    [layoutKey, undoableDispatch],
  );

  const ref = useRef<HTMLDivElement>(null);

  const handleAddElement = useAddSymbol(undoableDispatch, layoutKey);

  const handleDrop = useCallback(
    ({ items, event }: Haul.OnDropProps): Haul.Item[] => {
      const valid = Haul.filterByType(HAUL_TYPE, items);
      if (ref.current == null || event == null) return valid;
      const region = box.construct(ref.current);
      valid.forEach(({ key, data }) => {
        const spec = Core.Symbol.REGISTRY[key as Core.Symbol.Variant];
        if (spec == null) return;
        const pos = xy.truncate(
          calculatePos(
            region,
            { x: event.clientX, y: event.clientY },
            viewportRef.current,
          ),
          0,
        );
        handleAddElement(key.toString(), pos, data);
      });
      return valid;
    },
    [theme, undoableDispatch, layoutKey],
  );

  const dropProps = Haul.useDrop({
    type: "Schematic",
    key: layoutKey,
    canDrop: Haul.canDropOfType(HAUL_TYPE),
    onDrop: handleDrop,
  });

  const mode = useSelectRequiredViewportMode(layoutKey);
  const triggers = useMemo(() => Viewport.DEFAULT_TRIGGERS[mode], [mode]);

  Triggers.use({
    triggers: [
      ["Control", "V"],
      ["Control", "C"],
      ["Escape"],
      ["Control", "Z"],
      ["Control", "Shift", "Z"],
      ["Control", "A"],
    ],
    loose: true,
    region: ref,
    callback: useCallback(
      ({ triggers, cursor, stage }: Triggers.UseEvent) => {
        if (ref.current == null || stage !== "start") return;
        const region = box.construct(ref.current);
        const copy = triggers.some((t) => t.includes("C"));
        const isClear = triggers.some((t) => t.includes("Escape"));
        const isAll = triggers.some((t) => t.includes("A"));
        const isUndo =
          triggers.some((t) => t.includes("Z")) &&
          triggers.some((t) => t.includes("Control")) &&
          !triggers.some((t) => t.includes("Shift"));
        const isRedo =
          triggers.some((t) => t.includes("Z")) &&
          triggers.some((t) => t.includes("Control")) &&
          triggers.some((t) => t.includes("Shift"));
        const pos = calculatePos(region, cursor, viewportRef.current);
        if (copy) syncDispatch(copySelection({ pos }));
        else if (isClear) syncDispatch(clearSelection({ key: layoutKey }));
        else if (isUndo) undo();
        else if (isRedo) redo();
        else if (isAll) syncDispatch(selectAll({ key: layoutKey }));
        else undoableDispatch(pasteSelection({ pos, key: layoutKey }));
      },
      [layoutKey, undoableDispatch, undo, redo, syncDispatch],
    ),
  });

  const handleDoubleClick = useCallback(() => {
    if (!schematic.editable) return;
    syncDispatch(
      Layout.setNavDrawerVisible({ windowKey, key: "visualization", value: true }),
    );
  }, [windowKey, schematic.editable, syncDispatch]);

  const [legendPosition, setLegendPosition] = useState<Legend.StickyXY>(
    schematic.legend.position,
  );

  const storeLegendPosition = useCallback(
    (position: Legend.StickyXY) =>
      syncDispatch(setLegend({ key: layoutKey, legend: { position } })),
    [layoutKey, syncDispatch],
  );

  const handleLegendPositionChange = useCallback(
    (position: Legend.StickyXY) => {
      setLegendPosition(position);
      storeLegendPosition(position);
    },
    [storeLegendPosition, setLegendPosition],
  );

  const canEditSchematic = useSelectHasPermission() && !schematic.snapshot;

  const handleViewportModeChange = useCallback(
    (mode: Viewport.Mode) => dispatch(setViewportMode({ key: layoutKey, mode })),
    [dispatch, layoutKey],
  );

  return (
    <div
      ref={ref}
      onDoubleClick={handleDoubleClick}
      style={{ width: "inherit", height: "inherit", position: "relative" }}
    >
      <Control.Controller
        name={name}
        authority={schematic.authority}
        acquireTrigger={schematic.controlAcquireTrigger}
        onStatusChange={handleControlStatusChange}
      >
        <Diagram.Diagram
          onViewportChange={handleViewportChange}
          viewportMode={mode}
          onViewportModeChange={handleViewportModeChange}
          edges={schematic.edges}
          nodes={schematic.nodes}
          // Turns out that setting the zoom value to 1 here doesn't have any negative
          // effects on the schematic sizing and ensures that we position all the lines
          // in the correct place.
          viewport={{ ...schematic.viewport, zoom: 1 }}
          onEdgesChange={handleEdgesChange}
          onNodesChange={handleNodesChange}
          onEditableChange={handleEditableChange}
          editable={schematic.editable}
          triggers={triggers}
          onDoubleClick={handleDoubleClick}
          fitViewOnResize={schematic.fitViewOnResize}
          setFitViewOnResize={handleSetFitViewOnResize}
          visible={visible}
          dragHandleSelector={`.${Core.DRAG_HANDLE_CLASS}`}
          {...dropProps}
        >
          <Diagram.NodeRenderer>{elRenderer}</Diagram.NodeRenderer>
          <Diagram.Background />
          <Diagram.Controls>
            <Diagram.SelectViewportModeControl />
            <Diagram.FitViewControl />
            <Flex.Box x pack>
              {canEditSchematic && (
                <Diagram.ToggleEditControl
                  disabled={schematic.control === "acquired"}
                />
              )}
              {!schematic.snapshot && (
                <Button.Toggle
                  value={schematic.control === "acquired"}
                  onChange={acquireControl}
                  tooltipLocation={location.BOTTOM_LEFT}
                  uncheckedVariant="outlined"
                  checkedVariant="filled"
                  size="small"
                  tooltip={
                    <Text.Text level="small">
                      {schematic.control === "acquired"
                        ? "Release control"
                        : "Acquire control"}
                    </Text.Text>
                  }
                >
                  <Icon.Circle />
                </Button.Toggle>
              )}
            </Flex.Box>
          </Diagram.Controls>
        </Diagram.Diagram>
        <Control.Legend
          position={legendPosition}
          onPositionChange={handleLegendPositionChange}
          allowVisibleChange={false}
        />
      </Control.Controller>
    </div>
  );
};

export const Schematic: Layout.Renderer = ({ layoutKey, ...rest }) => {
  const loaded = useLoadRemote({
    name: "Schematic",
    targetVersion: ZERO_STATE.version,
    layoutKey,
    useSelectVersion,
    fetcher: async (client, layoutKey) => {
      const { key, data } = await client.workspaces.schematic.retrieve(layoutKey);
      return { key, ...data } as State;
    },
    actionCreator: internalCreate,
  });
  if (!loaded) return null;
  return <Loaded layoutKey={layoutKey} {...rest} />;
};

export const LAYOUT_TYPE = "schematic";
export type LayoutType = typeof LAYOUT_TYPE;

export const SELECTABLE: Selector.Selectable = {
  key: LAYOUT_TYPE,
  title: "Schematic",
  icon: <Icon.Schematic />,
  create: async ({ layoutKey }) => create({ key: layoutKey }),
};

export type CreateArg = Partial<State> & Partial<Layout.BaseState>;

export const create =
  (initial: CreateArg = {}): Layout.Creator =>
  ({ dispatch, store }) => {
    const canEditSchematic = selectHasPermission(store.getState());
    const { name = "Schematic", location = "mosaic", window, tab, ...rest } = initial;
    if (!canEditSchematic && tab?.editable) tab.editable = false;
    const key = schematic.keyZ.safeParse(initial.key).data ?? uuid.create();
    dispatch(internalCreate({ ...deep.copy(ZERO_STATE), ...rest, key }));
    return {
      key,
      location,
      name,
      icon: "Schematic",
      type: LAYOUT_TYPE,
      window: { navTop: true, showTitle: true },
      tab,
    };
  };
