// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/arc/graph/Editor.css";

import { box, id, xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useRef } from "react";

import { useClipboard } from "@/arc/graph/clipboard";
import {
  Diagram,
  edgeChangesToActions,
  nodeChangesToActions,
} from "@/arc/graph/Diagram";
import { canDropHaulItem, filterHaulItems } from "@/arc/haul";
import {
  useAddNode,
  useAllEdges,
  useAllNodes,
  useRedo,
  useSingleDispatch,
  useUndo,
} from "@/arc/queries";
import { Scope } from "@/arc/scope";
import { useKey } from "@/arc/Suspended";
import { type Component } from "@/component";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { useSyncedRef } from "@/hooks";
import { Menu } from "@/menu";
import { type Triggers } from "@/triggers";
import { Diagram as BaseDiagram } from "@/vis/diagram";

const FIT_VIEW_OPTIONS: BaseDiagram.FitViewOptions = {
  minZoom: 0.5,
  maxZoom: 0.9,
  padding: 0.1,
};

const SNAP_GRID: xy.Couple = [2, 2];

export interface EditorProps extends Omit<
  BaseDiagram.DiagramProps,
  "nodes" | "edges" | "onNodesChange" | "onEdgesChange" | "onChange"
> {
  enableTriggers?: Triggers.Condition;
  extraMenuItems?: Component.RenderProp<Menu.ContextMenuMenuProps>;
}

export const Editor = ({
  viewport,
  className,
  selected,
  editable,
  onSelectionChange,
  enableTriggers,
  extraMenuItems,
  children,
  ...props
}: EditorProps): ReactElement => {
  const key = useKey();
  const nodes = useAllNodes();
  const nodesRef = useSyncedRef(nodes);
  const edges = useAllEdges();
  const edgesRef = useSyncedRef(edges);
  const dispatch = useSingleDispatch();

  const handleNodesChange = useCallback(
    (changes: BaseDiagram.NodeChange[]) => dispatch(nodeChangesToActions(changes)),
    [dispatch],
  );
  const handleEdgesChange = useCallback(
    (changes: BaseDiagram.EdgeChange[]) => dispatch(edgeChangesToActions(changes)),
    [dispatch],
  );

  const handleAddNode = useAddNode();
  const ref = useRef<HTMLDivElement>(null);
  const viewportRef = useSyncedRef(viewport);
  const calculateCursorPosition = useCallback((cursor: xy.Crude) => {
    if (ref.current == null) return xy.ZERO;
    return BaseDiagram.calculateCursorPosition(
      box.construct(ref.current),
      cursor,
      viewportRef.current,
    );
  }, []);

  const handleDrop = useCallback(
    ({ items, event }: Haul.OnDropProps): Haul.Item[] => {
      const valid = filterHaulItems(items);
      if (event == null) return valid;
      const position = xy.truncate(calculateCursorPosition(event), 0);
      valid.forEach(({ key: type }) =>
        handleAddNode({ key: id.create(), type, position }),
      );
      return valid;
    },
    [handleAddNode, calculateCursorPosition],
  );

  const dropProps = Haul.useDrop({
    type: "arc",
    key,
    canDrop: canDropHaulItem,
    onDrop: handleDrop,
  });

  const handleClearSelection = useCallback(
    () => onSelectionChange?.([]),
    [onSelectionChange],
  );
  const handleSelectAll = useCallback(
    () =>
      onSelectionChange?.([
        ...nodesRef.current.map((n) => n.key),
        ...edgesRef.current.map((e) => e.key),
      ]),
    [onSelectionChange],
  );
  const { undo, canUndo } = useUndo();
  const { redo, canRedo } = useRedo();

  const { onCopy, onCut, onPaste, copy, cut, paste } = useClipboard({
    key,
    selected,
    onCut: onSelectionChange,
    onPaste: onSelectionChange,
    container: ref,
  });

  BaseDiagram.useTriggers({
    onSelectAll: handleSelectAll,
    onClearSelection: handleClearSelection,
    onUndo: undo,
    onRedo: redo,
    enabled: enableTriggers,
    editable,
  });

  const contextMenu = Menu.useContextMenu();
  const renderMenu = useCallback<Component.RenderProp<Menu.ContextMenuMenuProps>>(
    (menuProps) => (
      <Menu.Menu level="small" gap="small">
        {editable && (
          <>
            <BaseDiagram.Menu.ClipboardItems
              cut={cut}
              copy={copy}
              paste={paste}
              hasSelection={(selected?.length ?? 0) > 0}
            />
            <Menu.Divider />
            <Menu.UndoRedoItems
              undo={undo}
              redo={redo}
              canUndo={canUndo}
              canRedo={canRedo}
            />
            {extraMenuItems != null && <Menu.Divider />}
          </>
        )}
        {extraMenuItems?.(menuProps)}
      </Menu.Menu>
    ),
    [
      undo,
      redo,
      canUndo,
      canRedo,
      editable,
      extraMenuItems,
      selected,
      cut,
      copy,
      paste,
    ],
  );

  return (
    <Scope.Provider value={key}>
      <Diagram
        ref={ref}
        className={CSS.cls(className, CSS.B("arc"))}
        fitViewOptions={FIT_VIEW_OPTIONS}
        snapGrid={SNAP_GRID}
        snapToGrid
        onNodesChange={handleNodesChange}
        onEdgesChange={handleEdgesChange}
        viewport={viewport}
        onSelectionChange={onSelectionChange}
        editable={editable}
        onContextMenu={contextMenu.open}
        onCopy={onCopy}
        onCut={onCut}
        onPaste={onPaste}
        nodes={nodes}
        edges={edges}
        selected={selected}
        {...dropProps}
        {...props}
      >
        {children}
        <Menu.ContextMenu {...contextMenu} menu={renderMenu} />
      </Diagram>
    </Scope.Provider>
  );
};
