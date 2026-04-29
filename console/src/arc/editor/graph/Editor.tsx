// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import {
  Access,
  Arc as Base,
  Component,
  Diagram,
  Haul,
  Theming,
  useSyncedRef,
  Viewport,
} from "@synnaxlabs/pluto";
import { box, id, xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo, useRef } from "react";
import { useDispatch } from "react-redux";

import { Controls } from "@/arc/editor/Controls";
import { Provider, useArcEditorContext } from "@/arc/editor/graph/Context";
import {
  select,
  useSelect,
  useSelectNodeProps,
  useSelectSelected,
  useSelectViewportMode,
} from "@/arc/selectors";
import {
  addElement,
  applyEdgeChanges,
  applyNodeChanges,
  copySelection,
  internalCreate,
  pasteSelection,
  selectAll,
  setEditable,
  setElementProps,
  setFitViewOnResize,
  setSelected,
  setViewport,
  setViewportMode,
  type State,
} from "@/arc/slice";
import { ContextMenu as CMenu, Controls as BaseControls } from "@/components";
import { useUndoableDispatch } from "@/hooks/useUndoableDispatch";
import { Layout } from "@/layout";
import { type RootState } from "@/store";

export const HAUL_TYPE = "arc-element";

const StageRenderer = ({
  nodeKey,
  position,
  selected,
  draggable,
}: Diagram.NodeProps): ReactElement | null => {
  const { layoutKey, dispatch } = useArcEditorContext("ArcEditor.StageRenderer");
  const props = useSelectNodeProps(layoutKey, nodeKey);
  const { key = "", ...rest } = props ?? {};
  const handleChange = useCallback(
    (props: object) => {
      if (key == null) return;
      dispatch(
        setElementProps({
          layoutKey,
          key: nodeKey,
          props: { key, ...props },
        }),
      );
    },
    [nodeKey, layoutKey, key, dispatch],
  );
  if (props == null) return null;
  const C = Base.Stage.REGISTRY[key];
  if (C == null) throw new Error(`Symbol ${key} not found`);
  return (
    <C.Symbol
      key={key}
      nodeKey={nodeKey}
      position={position}
      selected={selected}
      draggable={draggable}
      onChange={handleChange}
      {...rest}
    />
  );
};

const ArcDiagram = Base.create({
  node: Component.renderProp(StageRenderer),
});

export const ContextMenu: Layout.ContextMenuRenderer = ({ layoutKey }) => (
  <CMenu.Menu>
    <Layout.MenuItems layoutKey={layoutKey} />
  </CMenu.Menu>
);

export const Editor: Layout.Renderer = ({ layoutKey, visible }) => {
  const windowKey = useSelectWindowKey() as string;
  const state = useSelect(layoutKey);

  const dispatch = useDispatch();
  const selector = useCallback(
    (state: RootState) => select(state, layoutKey),
    [layoutKey],
  );
  const [undoableDispatch, undo, redo] = useUndoableDispatch<RootState, State>(
    selector,
    internalCreate,
    30,
  );

  const theme = Theming.use();
  const viewportRef = useSyncedRef(state.graph.viewport);
  const hasUpdatePermission = Access.useUpdateGranted(arc.ontologyID(layoutKey));
  const canEdit = hasUpdatePermission && state.graph.editable;

  const selected = useSelectSelected(layoutKey);

  const handleSelectionChange = useCallback(
    (selected: string[]) => dispatch(setSelected({ key: layoutKey, selected })),
    [layoutKey, dispatch],
  );

  const handleNodesChange = useCallback(
    (changes: Diagram.NodeChange[]) =>
      undoableDispatch(applyNodeChanges({ key: layoutKey, changes })),
    [layoutKey, undoableDispatch],
  );

  const handleEdgesChange = useCallback(
    (changes: Diagram.EdgeChange[]) =>
      undoableDispatch(applyEdgeChanges({ key: layoutKey, changes })),
    [layoutKey, undoableDispatch],
  );

  const handleViewportChange: Diagram.DiagramProps["onViewportChange"] = useCallback(
    (vp) => dispatch(setViewport({ key: layoutKey, viewport: vp })),
    [layoutKey, dispatch],
  );

  const handleEditableChange: Diagram.DiagramProps["onEditableChange"] = useCallback(
    (cbk) => dispatch(setEditable({ key: layoutKey, editable: cbk })),
    [layoutKey, dispatch],
  );

  const handleSetFitViewOnResize = useCallback(
    (v: boolean) =>
      dispatch(setFitViewOnResize({ key: layoutKey, fitViewOnResize: v })),
    [layoutKey, dispatch],
  );

  const ref = useRef<HTMLDivElement>(null);

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
      if (ref.current == null || event == null) return valid;
      valid.forEach(({ key, data }) => {
        const spec = Base.Stage.REGISTRY[key];
        if (spec == null) return;
        const pos = xy.truncate(calculateCursorPosition(event), 0);
        undoableDispatch(
          addElement({
            key: layoutKey,
            elKey: id.create(),
            node: { position: pos, zIndex: spec.zIndex },
            props: { key, ...spec.defaultProps(theme), ...(data ?? {}) },
          }),
        );
      });
      return valid;
    },
    [theme, undoableDispatch, layoutKey],
  );

  const dropProps = Haul.useDrop({
    type: "arc",
    key: layoutKey,
    canDrop: Haul.canDropOfType(HAUL_TYPE),
    onDrop: handleDrop,
  });

  const viewportMode = useSelectViewportMode();
  const triggers = useMemo(
    () => Viewport.DEFAULT_TRIGGERS[viewportMode],
    [viewportMode],
  );

  const handleDoubleClick = useCallback(() => {
    if (!state.graph.editable) return;
    dispatch(
      Layout.setNavDrawerVisible({
        windowKey,
        key: "visualization",
        value: true,
      }),
    );
  }, [windowKey, state.graph.editable, dispatch]);

  const handleViewportModeChange = useCallback(
    (mode: Viewport.Mode) => dispatch(setViewportMode({ mode })),
    [dispatch],
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
    () => dispatch(setSelected({ key: layoutKey, selected: [] })),
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

  const ctxValue = useMemo(
    () => ({ layoutKey, dispatch: undoableDispatch }),
    [layoutKey, undoableDispatch],
  );

  return (
    <Provider value={ctxValue}>
      <ArcDiagram
        viewportMode={viewportMode}
        onViewportModeChange={handleViewportModeChange}
        onViewportChange={handleViewportChange}
        edges={state.graph.edges}
        nodes={state.graph.nodes}
        // Turns out that setting the zoom value to 1 here doesn't have any negative
        // effects on the arc sizing and ensures that we position all the lines
        // in the correct place.
        viewport={{ ...state.graph.viewport, zoom: 1 }}
        selected={selected}
        onSelectionChange={handleSelectionChange}
        onEdgesChange={handleEdgesChange}
        onNodesChange={handleNodesChange}
        onEditableChange={handleEditableChange}
        editable={canEdit}
        triggers={triggers}
        onDoubleClick={handleDoubleClick}
        fitViewOnResize={state.graph.fitViewOnResize}
        setFitViewOnResize={handleSetFitViewOnResize}
        visible={visible}
        {...dropProps}
      >
        <Diagram.Background />
        <BaseControls x>
          <Diagram.Controls.FitView />
          {hasUpdatePermission && <Diagram.Controls.ToggleEdit />}
        </BaseControls>
      </ArcDiagram>
      <Controls state={state} />
    </Provider>
  );
};
