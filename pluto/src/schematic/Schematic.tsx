// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/Schematic.css";

import { box, TimeSpan, xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useRef } from "react";

import { type Component } from "@/component";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { useSyncedRef } from "@/hooks";
import { Icon } from "@/icon";
import { Menu } from "@/menu";
import { useClipboard } from "@/schematic/clipboard";
import {
  Diagram,
  edgeChangesToActions,
  nodeChangesToActions,
} from "@/schematic/Diagram";
import { canDropHaulItem, filterHaulItems } from "@/schematic/haul";
import { Node } from "@/schematic/node";
import {
  useAddNode,
  useRedo,
  useSelectAllEdges,
  useSelectAllNodes,
  useSingleDispatch,
  useUndo,
} from "@/schematic/queries";
import { useKey } from "@/schematic/Suspended";
import { Triggers } from "@/triggers";
import { Diagram as BaseDiagram } from "@/vis/diagram";

export interface SchematicProps extends Omit<
  BaseDiagram.DiagramProps,
  | "dragHandleSelector"
  | "nodes"
  | "edges"
  | "onNodesChange"
  | "onEdgesChange"
  | "onChange"
> {
  enableTriggers?: boolean | (() => boolean);
  extraMenuItems?: Component.RenderProp<Menu.ContextMenuMenuProps>;
}
const AUTO_RENDER_INTERVAL = TimeSpan.seconds(1).milliseconds;
const DRAG_HANDLE_SELECTOR = `.${Node.DRAG_HANDLE_CLASS}`;

export const Schematic = ({
  className,
  viewport,
  onDoubleClick,
  onSelectionChange,
  selected,
  enableTriggers,
  extraMenuItems,
  editable,
  children,
  ...props
}: SchematicProps): ReactElement => {
  const key = useKey();
  const nodes = useSelectAllNodes();
  const nodesRef = useSyncedRef(nodes);
  const edges = useSelectAllEdges();
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
      valid.forEach(({ data }) => handleAddNode({ ...data, position }));
      return valid;
    },
    [handleAddNode, calculateCursorPosition],
  );

  const dropProps = Haul.useDrop({
    type: "Schematic",
    key,
    canDrop: canDropHaulItem,
    onDrop: handleDrop,
  });

  const handleClearSelection = useCallback(
    () => onSelectionChange?.([]),
    [onSelectionChange],
  );
  const handleSelectAll = useCallback(() => {
    onSelectionChange?.([
      ...nodesRef.current.map((n) => n.key),
      ...edgesRef.current.map((e) => e.key),
    ]);
  }, [onSelectionChange]);
  const { undo, canUndo } = useUndo();
  const { redo, canRedo } = useRedo();

  const { onCopy, onPaste } = useClipboard({
    selected,
    onPaste: onSelectionChange,
  });

  BaseDiagram.useTriggers({
    onSelectAll: handleSelectAll,
    onClearSelection: handleClearSelection,
    onUndo: undo,
    onRedo: redo,
    enabled: enableTriggers,
  });

  const contextMenu = Menu.useContextMenu();
  const renderMenu = useCallback<Component.RenderProp<Menu.ContextMenuMenuProps>>(
    (menuProps) => (
      <Menu.Menu level="small" gap="small">
        {editable && (
          <>
            <Menu.Item
              itemKey="undo"
              onClick={undo}
              disabled={!canUndo}
              triggerIndicator={Triggers.UNDO}
            >
              <Icon.Undo />
              Undo
            </Menu.Item>
            <Menu.Item
              itemKey="redo"
              onClick={redo}
              disabled={!canRedo}
              triggerIndicator={Triggers.REDO}
            >
              <Icon.Redo />
              Redo
            </Menu.Item>
            {extraMenuItems != null && <Menu.Divider />}
          </>
        )}
        {extraMenuItems?.(menuProps)}
      </Menu.Menu>
    ),
    [undo, redo, canUndo, canRedo, editable, extraMenuItems],
  );

  return (
    <Diagram
      ref={ref}
      className={CSS(CSS.B("schematic"), className)}
      dragHandleSelector={DRAG_HANDLE_SELECTOR}
      autoRenderInterval={AUTO_RENDER_INTERVAL}
      onNodesChange={handleNodesChange}
      onEdgesChange={handleEdgesChange}
      viewport={viewport}
      onSelectionChange={onSelectionChange}
      editable={editable}
      onDoubleClick={onDoubleClick}
      onContextMenu={contextMenu.open}
      onCopy={onCopy}
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
  );
};
