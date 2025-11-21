// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch, type UnknownAction } from "@reduxjs/toolkit";
import { schematic } from "@synnaxlabs/client";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import {
  Button,
  Control,
  Diagram,
  Flex,
  Haul,
  Icon,
  Schematic as Core,
  Text,
  Theming,
  usePrevious,
  useSyncedRef,
  Viewport,
} from "@synnaxlabs/pluto";
import { box, deep, location, type sticky, uuid, xy } from "@synnaxlabs/x";
import {
  type ReactElement,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useDispatch } from "react-redux";

import { createLoadRemote } from "@/hooks/useLoadRemote";
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

export const HAUL_TYPE = "schematic-element";

const useSyncComponent = Workspace.createSyncComponent(
  "Schematic",
  async ({ key, workspace, store, client }) => {
    const storeState = store.getState();
    if (!selectHasPermission(storeState)) return;
    const data = selectOptional(storeState, key);
    if (data == null) return;
    const layout = Layout.selectRequired(storeState, key);
    if (data.snapshot) {
      await client.workspaces.schematics.rename(key, layout.name);
      return;
    }
    const setData = { ...data, key: undefined };
    if (!data.remoteCreated) store.dispatch(setRemoteCreated({ key }));
    await client.workspaces.schematics.create(workspace, {
      key,
      name: layout.name,
      data: setData,
    });
  },
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
        setElementProps({
          layoutKey,
          key: symbolKey,
          props: { key, ...props },
        }),
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
      if (
        // @ts-expect-error - Sometimes, the nodes do have dragging
        nodes.some((n) => n.dragging) ||
        changes.some((c) => c.type === "select")
      )
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

  const calculateCursorPosition = useCallback(
    (cursor: xy.Crude) =>
      Diagram.calculateCursorPosition(
        box.construct(ref.current ?? box.ZERO),
        cursor,
        viewportRef.current,
      ),
    [],
  );

  const handleDrop = useCallback(
    ({ items, event }: Haul.OnDropProps): Haul.Item[] => {
      const valid = Haul.filterByType(HAUL_TYPE, items);
      if (event == null) return valid;
      valid.forEach(({ key, data }) => {
        const spec = Core.Symbol.REGISTRY[key as Core.Symbol.Variant];
        if (spec == null) return;
        const pos = xy.truncate(calculateCursorPosition(event), 0);
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

  const handleDoubleClick = useCallback(() => {
    if (!schematic.editable) return;
    syncDispatch(
      Layout.setNavDrawerVisible({
        windowKey,
        key: "visualization",
        value: true,
      }),
    );
  }, [windowKey, schematic.editable, syncDispatch]);

  const [legendPosition, setLegendPosition] = useState<sticky.XY>(
    schematic.legend.position,
  );

  const storeLegendPosition = useCallback(
    (position: sticky.XY) =>
      syncDispatch(setLegend({ key: layoutKey, legend: { position } })),
    [layoutKey, syncDispatch],
  );

  const handleLegendPositionChange = useCallback(
    (position: sticky.XY) => {
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

  const handleCopySelection = useCallback(
    (cursor: xy.XY) =>
      dispatch(copySelection({ pos: calculateCursorPosition(cursor) })),
    [dispatch, calculateCursorPosition],
  );

  const handlePasteSelection = useCallback(
    (cursor: xy.XY) =>
      dispatch(
        pasteSelection({
          pos: calculateCursorPosition(cursor),
          key: layoutKey,
        }),
      ),
    [dispatch, calculateCursorPosition, layoutKey],
  );

  const handleSelectAll = useCallback(
    () => dispatch(selectAll({ key: layoutKey })),
    [dispatch, layoutKey],
  );

  const handleClearSelection = useCallback(
    () => dispatch(clearSelection({ key: layoutKey })),
    [dispatch, layoutKey],
  );

  Diagram.useTriggers({
    onCopy: handleCopySelection,
    onPaste: handlePasteSelection,
    onSelectAll: handleSelectAll,
    onClear: handleClearSelection,
    onUndo: undo,
    onRedo: redo,
    region: ref,
  });

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
        <Core.Schematic
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
        </Core.Schematic>
        <Control.Legend
          position={legendPosition}
          onPositionChange={handleLegendPositionChange}
          allowVisibleChange={false}
        />
      </Control.Controller>
    </div>
  );
};

const useLoadRemote = createLoadRemote<schematic.Schematic>({
  useRetrieve: Core.useRetrieveObservable,
  targetVersion: ZERO_STATE.version,
  useSelectVersion,
  actionCreator: (v) => internalCreate({ ...(v.data as State), key: v.key }),
});

export const Schematic: Layout.Renderer = ({ layoutKey, ...rest }) => {
  const schematic = useLoadRemote(layoutKey);
  if (schematic == null) return null;
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
