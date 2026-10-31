// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/Schematic.css";

import { type Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Haul } from "@synnaxlabs/lyra/haul";
import { useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { Menu } from "@synnaxlabs/lyra/menu";
import { type Triggers } from "@synnaxlabs/lyra/triggers";
import { box, TimeSpan, xy } from "@synnaxlabs/x";
import { type ReactElement, useRef, useState } from "react";

import { CSS } from "@/css";
import { useClipboard } from "@/schematic/clipboard";
import {
  Diagram,
  edgeChangesToActions,
  nodeChangesToActions,
} from "@/schematic/Diagram";
import { useAddNode } from "@/schematic/editor";
import { Group } from "@/schematic/group";
import { canDropHaulItem, filterHaulItems } from "@/schematic/haul";
import { Node } from "@/schematic/node";
import {
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
import { Tooltip } from "@/schematic/Tooltip";
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
  /** Hides the tooltip that shows a symbol's configuration on hover. */
  tooltipsDisabled?: boolean;
}
interface Hovered {
  nodeKey: string;
  anchor: HTMLElement;
}

const AUTO_RENDER_INTERVAL = TimeSpan.seconds(1).milliseconds;
const DRAG_HANDLE_SELECTOR = `.${Node.DRAG_HANDLE_CLASS}`;
const CONTROL_STATE_SELECTOR = `.${Node.Control.STATE_CLASS}`;

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
  tooltipsDisabled = false,
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
  const lockedNodes = Group.lockMembers(nodes, parentOf, configs);
  const dispatch = useSingleDispatch();
  const handleNodesChange = (changes: BaseDiagram.NodeChange[]) =>
    dispatch(nodeChangesToActions(changes));

  const handleEdgesChange = (changes: BaseDiagram.EdgeChange[]) =>
    dispatch(edgeChangesToActions(changes));

  const handleAddNode = useAddNode();
  const ref = useRef<HTMLDivElement>(null);
  const viewportRef = useSyncedRef(viewport);
  const calculateCursorPosition = (cursor: xy.Crude) => {
    if (ref.current == null) return xy.ZERO;
    return BaseDiagram.calculateCursorPosition(
      box.construct(ref.current),
      cursor,
      viewportRef.current,
    );
  };

  const handleDrop = ({ items, event }: Haul.OnDropProps): Haul.Item[] => {
    const valid = filterHaulItems(items);
    if (event == null) return valid;
    const position = xy.truncate(calculateCursorPosition(event), 0);
    valid.forEach(({ data }) => handleAddNode({ ...data, position }));
    return valid;
  };

  const dropProps = Haul.useDrop({
    type: "Schematic",
    key,
    canDrop: canDropHaulItem,
    onDrop: handleDrop,
  });

  const handleClearSelection = () => onSelectionChange?.([]);
  const handleSelectAll = () => {
    onSelectionChange?.([
      ...nodesRef.current.map((n) => n.key),
      ...edgesRef.current.map((e) => e.key),
    ]);
  };
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
  const handleGroup = () => {
    const selection = group(selectedRef.current ?? []);
    if (selection != null) onSelectionChange?.(selection);
  };
  const handleUngroup = () => {
    const freed = ungroup(selectedRef.current ?? []);
    if (freed != null) onSelectionChange?.(freed);
  };

  const canGroup = Group.canGroup(selected ?? [], nodes, parentOf);
  const canUngroup = Group.canUngroup(selected ?? [], configs);

  // Clicking a member selects its whole group; delete, copy, and cut then cover it.
  const handleSelectionChange = (keys: string[]) =>
    onSelectionChange?.(Group.closure(keys, parentOfRef.current, configsRef.current));

  // Double-clicking a grouped member drills in, selecting only that member.
  const handleNodeDoubleClick: NonNullable<
    BaseDiagram.DiagramProps["onNodeDoubleClick"]
  > = (e, node) => {
    if (editable) {
      const drilled = Group.drillIn(node.id, parentOfRef.current, configsRef.current);
      if (drilled != null) onSelectionChange?.(drilled);
    }
    onNodeDoubleClick?.(e, node);
  };

  const [hovered, setHovered] = useState<Hovered | null>(null);
  // The control chip is inside the node, so moves over it close the tooltip instead.
  const handleNodeMouseMove: NonNullable<
    BaseDiagram.DiagramProps["onNodeMouseMove"]
  > = (e, node) => {
    const { target, currentTarget: anchor } = e;
    const overControl =
      target instanceof Element && target.closest(CONTROL_STATE_SELECTOR) != null;
    setHovered((prev) => {
      if (overControl || !(anchor instanceof HTMLElement)) return null;
      if (prev?.nodeKey === node.id) return prev;
      return { nodeKey: node.id, anchor };
    });
  };
  const clearHovered = () => setHovered(null);
  const hoveredConfig = hovered == null ? undefined : configs[hovered.nodeKey];

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
  const renderMenu: Component.RenderProp<Menu.ContextMenuMenuProps> = (menuProps) => (
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
      onNodeMouseMove={handleNodeMouseMove}
      onNodeMouseLeave={clearHovered}
      onNodeDragStart={clearHovered}
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
      {!editable &&
        !tooltipsDisabled &&
        hovered != null &&
        !contextMenu.visible &&
        hoveredConfig != null &&
        Node.isConfig(hoveredConfig) && (
          <Tooltip
            key={hovered.nodeKey}
            anchor={hovered.anchor}
            config={hoveredConfig}
          />
        )}
      {nodes.length === 0 && emptyContent != null && (
        <Flex.Box center className={CSS.BE("schematic", "empty")}>
          {emptyContent}
        </Flex.Box>
      )}
      <Menu.ContextMenu {...contextMenu} menu={renderMenu} />
    </Diagram>
  );
};
