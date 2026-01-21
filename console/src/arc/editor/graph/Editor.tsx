// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch, type UnknownAction } from "@reduxjs/toolkit";
import { arc } from "@synnaxlabs/client";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import {
  Access,
  Arc as Base,
  Diagram,
  Haul,
  Menu as PMenu,
  Theming,
  useSyncedRef,
  Viewport,
} from "@synnaxlabs/pluto";
import { box, id, xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo, useRef } from "react";
import { useDispatch } from "react-redux";

import { Controls } from "@/arc/editor/Controls";
import {
  select,
  useSelect,
  useSelectNodeProps,
  useSelectViewportMode,
} from "@/arc/selectors";
import {
  addElement,
  clearSelection,
  copySelection,
  internalCreate,
  pasteSelection,
  selectAll,
  setEdges,
  setEditable,
  setElementProps,
  setFitViewOnResize,
  setNodes,
  setViewport,
  setViewportMode,
  type State,
} from "@/arc/slice";
import { Controls as BaseControls } from "@/components";
import { useUndoableDispatch } from "@/hooks/useUndoableDispatch";
import { Layout } from "@/layout";
import { type RootState } from "@/store";

export const HAUL_TYPE = "arc-element";

interface SymbolRendererProps extends Diagram.SymbolProps {
  layoutKey: string;
  dispatch: Dispatch<UnknownAction>;
}

const StageRenderer = ({
  symbolKey,
  position,
  selected,
  draggable,
  dispatch,
  layoutKey,
}: SymbolRendererProps): ReactElement | null => {
  const props = useSelectNodeProps(layoutKey, symbolKey);
  const { key = "", ...rest } = props ?? {};
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
  const C = Base.Stage.REGISTRY[key];
  if (C == null) throw new Error(`Symbol ${key} not found`);
  return (
    <C.Symbol
      key={key}
      id={symbolKey}
      symbolKey={symbolKey}
      position={position}
      selected={selected}
      draggable={draggable}
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
    30, // roughly the right time needed to prevent actions that get dispatch automatically by Diagram.tsx, like setNodes immediately following addElement
  );

  const theme = Theming.use();
  const viewportRef = useSyncedRef(state.graph.viewport);
  const hasEditPermission = Access.useUpdateGranted(arc.ontologyID(layoutKey));
  const canEdit = hasEditPermission && state.graph.editable;

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
        dispatch(setNodes({ key: layoutKey, nodes }));
      else undoableDispatch(setNodes({ key: layoutKey, nodes }));
    },
    [layoutKey, dispatch, undoableDispatch],
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

  const elRenderer = useCallback(
    (props: Diagram.SymbolProps) => (
      <StageRenderer layoutKey={layoutKey} dispatch={undoableDispatch} {...props} />
    ),
    [layoutKey, undoableDispatch],
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
      <Base.Arc
        viewportMode={viewportMode}
        onViewportModeChange={handleViewportModeChange}
        onViewportChange={handleViewportChange}
        edges={state.graph.edges}
        nodes={state.graph.nodes}
        // Turns out that setting the zoom value to 1 here doesn't have any negative
        // effects on the arc sizing and ensures that we position all the lines
        // in the correct place.
        viewport={{ ...state.graph.viewport, zoom: 1 }}
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
        <Diagram.NodeRenderer>{elRenderer}</Diagram.NodeRenderer>
        <Diagram.Background />
        <BaseControls x>
          <Diagram.FitViewControl />
          {hasEditPermission && <Diagram.ToggleEditControl />}
        </BaseControls>
      </Base.Arc>
      <Controls state={state} />
    </div>
  );
};
