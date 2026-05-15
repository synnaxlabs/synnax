// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/Schematic.css";

import { schematic } from "@synnaxlabs/client";
import { box, TimeSpan, xy } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useCallback, useRef } from "react";

import { CSS } from "@/css";
import { Flux } from "@/flux";
import { Haul } from "@/haul";
import { useSyncedRef } from "@/hooks";
import { Icon } from "@/icon";
import { Key } from "@/key";
import { Menu } from "@/menu";
import { useClipboard } from "@/schematic/clipboard";
import {
  Diagram,
  edgeChangesToActions,
  nodeChangesToActions,
} from "@/schematic/Diagram";
import * as Groups from "@/schematic/groups";
import { canDropHaulItem, filterHaulItems } from "@/schematic/haul";
import { Node } from "@/schematic/node";
import {
  type FluxSubStore,
  useAddNode,
  useDispatch,
  useGroup,
  useRedo,
  useSelectAllEdges,
  useSelectAllNodes,
  useSelectCanGroup,
  useSelectCanUngroup,
  useUndo,
  useUngroup,
} from "@/schematic/queries";
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
  resourceKey: schematic.Key;
  extraContextMenuItems?: ReactNode;
}
const AUTO_RENDER_INTERVAL = TimeSpan.seconds(1).milliseconds;
const DRAG_HANDLE_SELECTOR = `.${Node.DRAG_HANDLE_CLASS}`;

export const Schematic = ({
  className,
  resourceKey: key,
  viewport,
  onDoubleClick,
  onSelectionChange,
  selected,
  enableTriggers,
  extraContextMenuItems,
  ...props
}: SchematicProps): ReactElement => {
  const nodes = useSelectAllNodes({ key });
  const nodesRef = useSyncedRef(nodes);
  const edges = useSelectAllEdges({ key });
  const edgesRef = useSyncedRef(edges);
  const { dispatch } = useDispatch();
  const store = Flux.useStore<FluxSubStore>();
  const handleNodesChange = useCallback(
    (changes: BaseDiagram.NodeChange[]) => {
      const schem = store.schematics.get(key);
      // Cascade: removing a group container also removes its members.
      let expandedChanges: BaseDiagram.NodeChange[] = changes;
      if (schem != null) {
        const removeKeys = changes
          .filter((c) => c.type === "remove")
          .map((c) => c.key);
        if (removeKeys.length > 0) {
          const cascaded = Groups.cascadeRemovedKeys(
            removeKeys,
            schem.nodes,
            schem.configs,
          );
          const original = new Set(removeKeys);
          const extraRemoves: BaseDiagram.NodeChange[] = cascaded
            .filter((k) => !original.has(k))
            .map((k) => ({ type: "remove" as const, key: k }));
          expandedChanges = [...changes, ...extraRemoves];
        }
      }
      const positionChanges = expandedChanges.filter((c) => c.type === "position");
      const otherChanges = expandedChanges.filter((c) => c.type !== "position");
      let allPositionChanges: BaseDiagram.NodeChange[] = positionChanges;
      if (positionChanges.length > 0 && schem != null) {
        const draggingByKey = new Map(
          positionChanges.map((c) => [c.key, c.dragging]),
        );
        const propagated = Groups.propagateGroupDrag(
          positionChanges.map((c) => ({ key: c.key, position: c.position })),
          schem.nodes,
          schem.configs,
        );
        allPositionChanges = propagated.map((p) => ({
          type: "position" as const,
          key: p.key,
          position: p.position,
          dragging: draggingByKey.get(p.key) ?? false,
        }));
      }
      const actions = nodeChangesToActions([...otherChanges, ...allPositionChanges]);
      if (actions.length === 0) return;
      dispatch({ key, actions });
      // Audit: after the dispatch, drop orphan/single-member group containers
      // and recompute bounding boxes for groups whose members moved.
      const next = store.schematics.get(key);
      if (next == null) return;
      const audit = Groups.auditGroups(next.nodes, next.configs);
      const auditActions: schematic.Action[] = [];
      for (const node of audit.clearGroupIdNodes)
        auditActions.push(schematic.setNode({ node, config: undefined }));
      for (const k of audit.removeGroupKeys)
        auditActions.push(schematic.removeNode({ key: k }));
      for (const r of audit.resizeGroups) {
        const node = next.nodes.find((n) => n.key === r.key);
        if (node == null) continue;
        const existing = next.configs[r.key] as Record<string, unknown> | undefined;
        auditActions.push(
          schematic.setNode({
            node: { ...node, position: r.position },
            config: { ...(existing ?? {}), dimensions: r.dimensions },
          }),
        );
      }
      if (auditActions.length > 0) dispatch({ key, actions: auditActions });
    },
    [key, dispatch, store],
  );

  const handleEdgesChange = useCallback(
    (changes: BaseDiagram.EdgeChange[]) => {
      const actions = edgeChangesToActions(changes);
      if (actions.length > 0) dispatch({ key, actions });
    },
    [key, dispatch],
  );

  const handleAddNode = useAddNode(key);
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
  const { undo } = useUndo({ key });
  const { redo } = useRedo({ key });

  const { onCopy, onCut, onPaste, copy, cut, paste } = useClipboard({
    key,
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

  const handleGroup = useGroup(key);
  const handleUngroup = useUngroup(key);
  const canGroup = useSelectCanGroup({ key, selected: selected ?? [] });
  const canUngroup = useSelectCanUngroup({ key, selected: selected ?? [] });
  const selectedRef = useSyncedRef(selected ?? []);
  Triggers.use({
    triggers: [
      ["Control", "G"],
      ["Control", "U"],
    ],
    region: ref,
    loose: true,
    callback: useCallback(
      ({ triggers, stage }: Triggers.UseEvent) => {
        if (stage !== "start") return;
        if (enableTriggers === false) return;
        if (typeof enableTriggers === "function" && !enableTriggers()) return;
        if (triggers.flat().includes("U")) {
          if (canUngroup) handleUngroup(selectedRef.current);
        } else if (canGroup) handleGroup(selectedRef.current);
      },
      [enableTriggers, canGroup, canUngroup, handleGroup, handleUngroup],
    ),
  });

  const menuProps = Menu.useContextMenu();
  // React Flow doesn't select nodes on right-click, so context menu actions like
  // Copy would operate on an empty selection without this.
  const handleContextMenu = useCallback(
    (e: React.MouseEvent) => {
      const target = e.target as HTMLElement;
      const nodeEl = target.closest<HTMLElement>("[data-id]");
      if (nodeEl != null) {
        const nodeKey = nodeEl.dataset.id;
        if (nodeKey != null && !selectedRef.current.includes(nodeKey))
          onSelectionChange?.([nodeKey]);
      }
      menuProps.open(e);
    },
    [menuProps, onSelectionChange],
  );
  const canvasMenu = useCallback(
    () => (
      <Menu.Menu level="small" gap="small">
        <Menu.Item
          itemKey="cut"
          trigger={["Control", "X"]}
          triggerIndicator
          onClick={cut}
        >
          <Icon.Cut />
          Cut
        </Menu.Item>
        <Menu.Item
          itemKey="copy"
          trigger={["Control", "C"]}
          triggerIndicator
          onClick={copy}
        >
          <Icon.Copy />
          Copy
        </Menu.Item>
        <Menu.Item
          itemKey="paste"
          trigger={["Control", "V"]}
          triggerIndicator
          onClick={() => paste(menuProps.cursor)}
        >
          <Icon.Paste />
          Paste
        </Menu.Item>
        {(canGroup || canUngroup) && <Menu.Divider />}
        {canGroup && (
          <Menu.Item
            itemKey="group"
            trigger={["Control", "G"]}
            triggerIndicator
            onClick={() => handleGroup(selectedRef.current)}
          >
            Group
          </Menu.Item>
        )}
        {canUngroup && (
          <Menu.Item
            itemKey="ungroup"
            trigger={["Control", "U"]}
            triggerIndicator
            onClick={() => handleUngroup(selectedRef.current)}
          >
            Ungroup
          </Menu.Item>
        )}
        {extraContextMenuItems != null && (
          <>
            <Menu.Divider />
            {extraContextMenuItems}
          </>
        )}
      </Menu.Menu>
    ),
    [
      cut,
      copy,
      paste,
      menuProps.cursor,
      canGroup,
      canUngroup,
      handleGroup,
      handleUngroup,
      extraContextMenuItems,
    ],
  );

  return (
    <Key.Provider value={key}>
      <Menu.ContextMenu {...menuProps} menu={canvasMenu}>
        <Diagram
          ref={ref}
          className={CSS(CSS.B("schematic"), className, menuProps.className)}
          dragHandleSelector={DRAG_HANDLE_SELECTOR}
          autoRenderInterval={AUTO_RENDER_INTERVAL}
          onNodesChange={handleNodesChange}
          onEdgesChange={handleEdgesChange}
          viewport={viewport}
          onSelectionChange={onSelectionChange}
          onDoubleClick={onDoubleClick}
          onContextMenu={handleContextMenu}
          onCopy={onCopy}
          onCut={onCut}
          onPaste={onPaste}
          nodes={nodes}
          edges={edges}
          selected={selected}
          {...dropProps}
          {...props}
        />
      </Menu.ContextMenu>
    </Key.Provider>
  );
};
