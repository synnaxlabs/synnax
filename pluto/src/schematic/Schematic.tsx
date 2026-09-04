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
import { type ReactElement, useCallback, useMemo, useRef } from "react";

import { type Component } from "@/component";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Haul } from "@/haul";
import { useSyncedRef } from "@/hooks";
import { Menu } from "@/menu";
import { useClipboard } from "@/schematic/clipboard";
import {
  Diagram,
  edgeChangesToActions,
  nodeChangesToActions,
} from "@/schematic/Diagram";
import { Group } from "@/schematic/group";
import { canDropHaulItem, filterHaulItems } from "@/schematic/haul";
import { Node } from "@/schematic/node";
import {
  useAddNode,
  useAllConfigs,
  useAllEdges,
  useAllNodes,
  useGroup,
  useParentOf,
  useRedo,
  useSingleDispatch,
  useUndo,
  useUngroup,
} from "@/schematic/queries";
import { useKey } from "@/schematic/Suspended";
import { type Triggers } from "@/triggers";
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
  enableTriggers?: Triggers.Condition;
  extraMenuItems?: Component.RenderProp<Menu.ContextMenuMenuProps>;
  /** Rendered as a centered overlay when the schematic has no nodes. */
  emptyContent?: ReactElement;
}
const AUTO_RENDER_INTERVAL = TimeSpan.seconds(1).milliseconds;
const DRAG_HANDLE_SELECTOR = `.${Node.DRAG_HANDLE_CLASS}`;

export const Schematic = ({
  className,
  viewport,
  onDoubleClick,
  onNodeDoubleClick,
  onSelectionChange,
  selected,
  enableTriggers,
  extraMenuItems,
  editable,
  emptyContent,
  children,
  ...props
}: SchematicProps): ReactElement => {
  const key = useKey();
  const nodes = useAllNodes();
  const nodesRef = useSyncedRef(nodes);
  const edges = useAllEdges();
  const edgesRef = useSyncedRef(edges);
  const configs = useAllConfigs();
  const configsRef = useSyncedRef(configs);
  const selectedRef = useSyncedRef(selected);
  const parentOf = useParentOf();
  const parentOfRef = useSyncedRef(parentOf);
  const lockedNodes = useMemo(
    () => Group.lockMembers(nodes, parentOf),
    [nodes, parentOf],
  );
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

  const { onCopy, onCut, onPaste, copy, cut, paste } = useClipboard({
    selected,
    onCut: onSelectionChange,
    onPaste: onSelectionChange,
    container: ref,
  });

  const group = useGroup();
  const ungroup = useUngroup();
  const handleGroup = useCallback(() => {
    const selection = group(selectedRef.current ?? []);
    if (selection != null) onSelectionChange?.(selection);
  }, [group, onSelectionChange]);
  const handleUngroup = useCallback(() => {
    const freed = ungroup(selectedRef.current ?? []);
    if (freed != null) onSelectionChange?.(freed);
  }, [ungroup, onSelectionChange]);

  const canGroup = useMemo(
    () => Group.canGroup(selected ?? [], nodes, parentOf),
    [selected, nodes, parentOf],
  );
  const canUngroup = useMemo(
    () => Group.canUngroup(selected ?? [], configs),
    [selected, configs],
  );

  // Clicking a member selects its whole group; delete, copy, and cut then cover it.
  const handleSelectionChange = useCallback(
    (keys: string[]) =>
      onSelectionChange?.(
        Group.closure(keys, parentOfRef.current, configsRef.current),
      ),
    [onSelectionChange],
  );

  // Double-clicking a grouped member drills in, selecting only that member.
  const handleNodeDoubleClick = useCallback<
    NonNullable<BaseDiagram.DiagramProps["onNodeDoubleClick"]>
  >(
    (e, node) => {
      if (editable) {
        const drilled = Group.drillIn(
          node.id,
          parentOfRef.current,
          configsRef.current,
        );
        if (drilled != null) onSelectionChange?.(drilled);
      }
      onNodeDoubleClick?.(e, node);
    },
    [editable, onSelectionChange, onNodeDoubleClick],
  );

  BaseDiagram.useTriggers({
    onSelectAll: handleSelectAll,
    onClearSelection: handleClearSelection,
    onUndo: undo,
    onRedo: redo,
    onGroup: handleGroup,
    onUngroup: handleUngroup,
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
            {(canGroup || canUngroup) && (
              <>
                <Menu.Divider />
                <BaseDiagram.Menu.GroupItems
                  group={handleGroup}
                  ungroup={handleUngroup}
                  canGroup={canGroup}
                  canUngroup={canUngroup}
                />
              </>
            )}
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
      canGroup,
      canUngroup,
      handleGroup,
      handleUngroup,
    ],
  );

  return (
    <Diagram
      ref={ref}
      className={CSS.cls(CSS.B("schematic"), className)}
      dragHandleSelector={DRAG_HANDLE_SELECTOR}
      autoRenderInterval={AUTO_RENDER_INTERVAL}
      onNodesChange={handleNodesChange}
      onEdgesChange={handleEdgesChange}
      viewport={viewport}
      onSelectionChange={handleSelectionChange}
      edgesReconnectable={false}
      editable={editable}
      onDoubleClick={onDoubleClick}
      onNodeDoubleClick={handleNodeDoubleClick}
      onContextMenu={contextMenu.open}
      onCopy={onCopy}
      onCut={onCut}
      onPaste={onPaste}
      nodes={lockedNodes}
      edges={edges}
      selected={selected}
      {...dropProps}
      {...props}
    >
      {children}
      {nodes.length === 0 && emptyContent != null && (
        <Flex.Box center className={CSS.BE("schematic", "empty")}>
          {emptyContent}
        </Flex.Box>
      )}
      <Menu.ContextMenu {...contextMenu} menu={renderMenu} />
    </Diagram>
  );
};
