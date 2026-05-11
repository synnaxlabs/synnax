// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import {
  Access,
  Control,
  Diagram,
  Flux,
  Haul,
  Icon,
  type Pluto,
  Schematic as Base,
  Status,
  Synnax,
  Theming,
  useSyncedRef,
  Viewport,
} from "@synnaxlabs/pluto";
import { box, type color, deep, type sticky, uuid, xy } from "@synnaxlabs/x";
import { useCallback, useMemo, useRef, useState } from "react";
import { useDispatch, useStore } from "react-redux";

import { ContextMenu as CContextMenu } from "@/components";
import { createLoadRemote } from "@/hooks/useLoadRemote";
import { useUndoableDispatch } from "@/hooks/useUndoableDispatch";
import { Layout } from "@/layout";
import { Controller } from "@/schematic/Controller";
import { Controls } from "@/schematic/Controls";
import { canDropHaulItem, isHaulItem } from "@/schematic/haul";
import {
  selectConfig,
  selectOptional,
  selectRequired,
  useSelectConfig,
  useSelectIsRemoteCreated,
  useSelectLegendVisible,
  useSelectRequired,
  useSelectRequiredViewportMode,
  useSelectSelected,
  useSelectVersion,
} from "@/schematic/selectors";
import {
  applyEdgeChanges,
  applyNodeChanges,
  clearSelection,
  copySelection,
  fromRemote,
  internalCreate,
  pasteSelection,
  selectAll,
  setEditable,
  setElementConfig,
  setFitViewOnResize,
  setLegend,
  setRemoteCreated,
  setSelected,
  setViewport,
  setViewportMode,
  type State,
  ZERO_STATE,
} from "@/schematic/slice";
import { type AddNodeProps, useAddNode } from "@/schematic/symbols/useAddNode";
import { Selector } from "@/selector";
import { type RootState } from "@/store";
import { Workspace } from "@/workspace";

type SchematicRetriever = (key: string) => Promise<schematic.Schematic>;

const navigateToLinkedSchematic = async (
  retrieve: SchematicRetriever,
  page: string,
  placeLayout: Layout.Placer,
): Promise<void> => {
  const s = await retrieve(page);
  placeLayout(create(fromRemote(s)));
};

type NodeClickHandler = (nodeId: string, dblClick: boolean) => void;

const useHandleNodeClickAction = (layoutKey: string): NodeClickHandler => {
  const store = useStore<RootState>();
  const client = Synnax.use();
  const fluxStore = Flux.useStore<Pluto.FluxStore>();
  const retrieve: SchematicRetriever | null = useMemo(
    () =>
      client != null
        ? (key: string) =>
            Base.retrieveSingle({ store: fluxStore, client, query: { key } })
        : null,
    [fluxStore, client],
  );
  const handleError = Status.useErrorHandler();
  const placeLayout = Layout.usePlacer();

  return useCallback(
    (nodeId: string, dblClick: boolean) => {
      const storeState = store.getState();
      const state = selectOptional(storeState, layoutKey);
      if (state == null || state.editable || retrieve == null) return;
      const props = selectConfig(storeState, layoutKey, nodeId);
      if (
        props?.variant !== "offPageReference" ||
        typeof props.page !== "string" ||
        props.page.length === 0
      )
        return;
      const dblClickNav = props.dblClickNav !== false;
      if (dblClick !== dblClickNav) return;
      const { page } = props;
      const label = props.label?.label;
      const name = label != null && label.length > 0 ? label : "Referenced schematic";
      handleError(
        () => navigateToLinkedSchematic(retrieve, page, placeLayout),
        `Schematic "${name}" not found`,
      );
    },
    [store, layoutKey, retrieve, placeLayout, handleError],
  );
};

const useSyncComponent = Workspace.createSyncComponent(
  "Schematic",
  async ({ key, workspace, store, fluxStore, client }) => {
    const storeState = store.getState();
    if (
      !Access.updateGranted({ id: schematic.ontologyID(key), store: fluxStore, client })
    )
      return;
    const data = selectOptional(storeState, key);
    if (data == null || data.snapshot) return;
    const layout = Layout.selectRequired(storeState, key);
    if (!data.remoteCreated) store.dispatch(setRemoteCreated({ key }));
    await client.schematics.create(workspace, {
      key,
      name: layout.name,
      snapshot: data.snapshot,
      authority: data.authority,
      legend: data.legend,
      nodes: data.nodes,
      edges: data.edges,
      configs: data.configs,
    });
  },
);

const useConfig: Base.UseConfig = (key, elKey) => {
  const dispatch = useDispatch();
  const config = useSelectConfig(key, elKey);
  return [
    config,
    useCallback(
      (config: Partial<Base.ElementConfig>) =>
        dispatch(setElementConfig({ key, elKey, config })),
      [key, dispatch],
    ),
  ];
};

const SchematicComponent = Base.create({ useConfig });

export const ContextMenu: Layout.ContextMenuRenderer = ({ layoutKey }) => (
  <CContextMenu.Menu>
    <Layout.MenuItems layoutKey={layoutKey} />
  </CContextMenu.Menu>
);

export const Loaded: Layout.Renderer = ({ layoutKey, visible }) => {
  const windowKey = useSelectWindowKey() as string;
  const state = useSelectRequired(layoutKey);
  const legendVisible = useSelectLegendVisible(layoutKey);
  const dispatch = useDispatch();
  const syncDispatch = useSyncComponent(layoutKey);
  const selector = useCallback(
    (state: RootState) => selectRequired(state, layoutKey),
    [layoutKey],
  );
  const [undoableDispatch_, undo, redo] = useUndoableDispatch<RootState, State>(
    selector,
    internalCreate,
    30, // roughly the right time needed to prevent actions that get dispatch
    // automatically by Diagram.tsx, like setNodes immediately following addElement
  );
  const undoableDispatch = useSyncComponent(layoutKey, undoableDispatch_);

  const theme = Theming.use();
  const viewportRef = useSyncedRef(state.viewport);

  const hasUpdatePermission =
    Access.useUpdateGranted(schematic.ontologyID(layoutKey)) && !state.snapshot;
  const canEdit = hasUpdatePermission && state.editable;

  const handleEdgesChange = useCallback(
    (changes: Diagram.EdgeChange[]) =>
      undoableDispatch(applyEdgeChanges({ key: layoutKey, changes })),
    [layoutKey, undoableDispatch],
  );

  const handleNodesChange = useCallback(
    (changes: Diagram.NodeChange[]) => {
      const dragging = changes.some(
        (c) => c.type === "position" && c.dragging === true,
      );
      const action = applyNodeChanges({ key: layoutKey, changes });
      if (dragging) syncDispatch(action);
      else undoableDispatch(action);
    },
    [layoutKey, syncDispatch, undoableDispatch],
  );

  const selected = useSelectSelected(layoutKey);
  const handleSelectionChange = useCallback(
    (next: string[]) => syncDispatch(setSelected({ key: layoutKey, selected: next })),
    [layoutKey, syncDispatch],
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

  const ref = useRef<HTMLDivElement>(null);

  const handleAddElement = useAddNode(layoutKey, undoableDispatch);

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
      const valid = items.filter(isHaulItem);
      if (event == null) return valid;
      valid.forEach(({ data }) => {
        const pos = xy.truncate(calculateCursorPosition(event), 0);
        handleAddElement({ ...(data as AddNodeProps), position: pos });
      });
      return valid;
    },
    [theme, undoableDispatch, layoutKey],
  );

  const dropProps = Haul.useDrop({
    type: "schematic",
    key: layoutKey,
    canDrop: canDropHaulItem,
    onDrop: handleDrop,
  });

  const mode = useSelectRequiredViewportMode(layoutKey);
  const triggers = useMemo(() => Viewport.DEFAULT_TRIGGERS[mode], [mode]);

  const handleDoubleClick = useCallback(() => {
    if (!state.editable) return;
    syncDispatch(
      Layout.setNavDrawerVisible({
        windowKey,
        key: "visualization",
        value: true,
      }),
    );
  }, [windowKey, state.editable, syncDispatch]);

  const handleNodeClickAction = useHandleNodeClickAction(layoutKey);

  const handleNodeClick = useCallback(
    (_event: React.MouseEvent, node: { id: string }) =>
      handleNodeClickAction(node.id, false),
    [handleNodeClickAction],
  );

  const handleNodeDoubleClick = useCallback(
    (_event: React.MouseEvent, node: { id: string }) =>
      handleNodeClickAction(node.id, true),
    [handleNodeClickAction],
  );

  const [legendPosition, setLegendPosition] = useState<sticky.XY>(
    state.legend.position,
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

  const handleLegendColorsChange = useCallback(
    (colors: Record<string, color.Color>) =>
      syncDispatch(setLegend({ key: layoutKey, legend: { colors } })),
    [layoutKey, syncDispatch],
  );

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

  const modals = Layout.useSelectWindowModals();
  Diagram.useTriggers({
    onCopy: handleCopySelection,
    onPaste: handlePasteSelection,
    onSelectAll: handleSelectAll,
    onClear: handleClearSelection,
    onUndo: undo,
    onRedo: redo,
    region: ref,
    disabled: modals.length > 0,
  });

  return (
    <Controller resourceKey={layoutKey} authority={state.authority}>
      <SchematicComponent
        itemKey={layoutKey}
        ref={ref}
        onViewportChange={handleViewportChange}
        viewportMode={mode}
        onViewportModeChange={handleViewportModeChange}
        edges={state.edges}
        nodes={state.nodes}
        selected={selected}
        onSelectionChange={handleSelectionChange}
        // Turns out that setting the zoom value to 1 here doesn't have any negative
        // effects on the schematic sizing and ensures that we position all the lines
        // in the correct place.
        viewport={{ ...state.viewport, zoom: 1 }}
        onEdgesChange={handleEdgesChange}
        onNodesChange={handleNodesChange}
        onEditableChange={handleEditableChange}
        editable={canEdit}
        triggers={triggers}
        onDoubleClick={handleDoubleClick}
        onNodeClick={handleNodeClick}
        onNodeDoubleClick={handleNodeDoubleClick}
        fitViewOnResize={state.fitViewOnResize}
        setFitViewOnResize={handleSetFitViewOnResize}
        visible={visible}
        {...dropProps}
      >
        <Diagram.Background />
        <Controls
          hasUpdatePermission={hasUpdatePermission}
          controlStatus={state.control}
          snapshot={state.snapshot}
        />
      </SchematicComponent>
      {legendVisible && (
        <Control.Legend
          position={legendPosition}
          onPositionChange={handleLegendPositionChange}
          colors={state.legend.colors}
          onColorsChange={handleLegendColorsChange}
          allowVisibleChange={false}
        />
      )}
    </Controller>
  );
};

const useLoadRemote = createLoadRemote<schematic.Schematic>({
  useRetrieve: Base.useRetrieveObservable,
  targetVersion: ZERO_STATE.version,
  useSelectVersion,
  actionCreator: (v) => internalCreate(fromRemote(v)),
});

export const Schematic: Layout.Renderer = ({ layoutKey, ...rest }) => {
  const schematic = useLoadRemote(layoutKey);
  if (schematic == null) return null;
  return <Loaded layoutKey={layoutKey} {...rest} />;
};

Schematic.useName = Layout.createUseFluxName(
  Base.useRename,
  Base.useRetrieveObservableName,
  useSelectIsRemoteCreated,
);

export const LAYOUT_TYPE = "schematic";
export type LayoutType = typeof LAYOUT_TYPE;

export const Selectable: Selector.Selectable = ({ layoutKey, onPlace }) => {
  const hasCreatePermission = Access.useCreateGranted(schematic.TYPE_ONTOLOGY_ID);
  const handleClick = useCallback(() => {
    onPlace(create({ key: layoutKey }));
  }, [onPlace, layoutKey]);

  if (!hasCreatePermission) return null;

  return (
    <Selector.Item
      key={LAYOUT_TYPE}
      title="Schematic"
      icon={<Icon.Schematic />}
      onClick={handleClick}
    />
  );
};
Selectable.type = LAYOUT_TYPE;
Selectable.useVisible = () => Access.useCreateGranted(schematic.TYPE_ONTOLOGY_ID);

export type CreateArg = Partial<State> & Partial<Layout.BaseState>;

export const create =
  (initial: CreateArg = {}): Layout.Creator =>
  ({ dispatch }) => {
    const canEditSchematic = true;
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
