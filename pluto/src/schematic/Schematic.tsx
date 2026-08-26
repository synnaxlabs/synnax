// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/Schematic.css";

import { box, type dimensions, TimeSpan, xy } from "@synnaxlabs/x";
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
import { Status } from "@/status";
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
const CUT_LOCKED_MESSAGE = "Cannot cut grouped symbols. Ungroup first.";

export const Schematic = ({
  className,
  viewport,
  onDoubleClick,
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
  const addStatus = Status.useAdder();
  const lockedSelected = useCallback(
    () =>
      Group.lockedKeys(selectedRef.current ?? [], parentOfRef.current).length > 0,
    [],
  );
  const blockLocked = useCallback(
    (message: string): boolean => {
      if (!lockedSelected()) return false;
      addStatus({ variant: "error", message });
      return true;
    },
    [lockedSelected, addStatus],
  );
  const handleNodesChange = useCallback(
    (changes: BaseDiagram.NodeChange[]) => {
      let allowed = changes;
      if (
        changes.some((c) => c.type === "remove") &&
        blockLocked("Cannot delete grouped symbols. Ungroup first.")
      )
        allowed = changes.filter((c) => c.type !== "remove");
      dispatch(nodeChangesToActions(allowed));
    },
    [dispatch, blockLocked],
  );

  const handleEdgesChange = useCallback(
    (changes: BaseDiagram.EdgeChange[]) => {
      let allowed = changes;
      if (changes.some((c) => c.type === "remove") && lockedSelected())
        allowed = changes.filter((c) => c.type !== "remove");
      dispatch(edgeChangesToActions(allowed));
    },
    [dispatch, lockedSelected],
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

  const {
    onCopy,
    onCut: baseOnCut,
    onPaste,
    copy,
    cut: baseCut,
    paste,
  } = useClipboard({
    selected,
    onCut: onSelectionChange,
    onPaste: onSelectionChange,
    container: ref,
  });
  const onCut = useCallback<BaseDiagram.ClipboardHandler>(
    (e, cursor) => {
      if (!blockLocked(CUT_LOCKED_MESSAGE)) baseOnCut(e, cursor);
    },
    [blockLocked, baseOnCut],
  );
  const cut = useCallback(() => {
    if (!blockLocked(CUT_LOCKED_MESSAGE)) baseCut();
  }, [blockLocked, baseCut]);

  // Sizes come from the DOM: measured dimensions are not persisted, and this
  // component sits outside the React Flow provider.
  const measure = useCallback((key: string): dimensions.Dimensions | null => {
    try {
      const { width, height } = BaseDiagram.selectNode(key).getBoundingClientRect();
      const zoom = viewportRef.current.zoom;
      return { width: width / zoom, height: height / zoom };
    } catch {
      return null;
    }
  }, []);

  const group = useGroup();
  const ungroup = useUngroup();
  const handleGroup = useCallback(() => {
    const selection = group(selectedRef.current ?? [], measure);
    if (selection != null) onSelectionChange?.(selection);
  }, [group, measure, onSelectionChange]);
  const handleUngroup = useCallback(() => {
    const freed = ungroup(selectedRef.current ?? []);
    if (freed != null) onSelectionChange?.(freed);
  }, [ungroup, onSelectionChange]);

  // Selecting a group selects its members; delete, copy, and cut then cover them.
  const handleSelectionChange = useCallback(
    (keys: string[]) =>
      onSelectionChange?.(Group.withMembers(keys, configsRef.current)),
    [onSelectionChange],
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
