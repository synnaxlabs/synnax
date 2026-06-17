// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { box, id, type record, xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo, useRef } from "react";

import { create } from "@/arc/Arc";
import { Stage } from "@/arc/functions";
import {
  useDispatch,
  useRedo,
  useSelectEdges,
  useSelectNodeProps,
  useSelectNodes,
  useUndo,
} from "@/arc/queries";
import { parseEdgeKey } from "@/arc/translate";
import { Component } from "@/component";
import { Haul } from "@/haul";
import { useSyncedRef } from "@/hooks";
import { Icon } from "@/icon";
import { Key } from "@/key";
import { Menu } from "@/menu";
import { Theming } from "@/theming";
import { type Triggers } from "@/triggers";
import { Diagram as BaseDiagram } from "@/vis/diagram";

export const HAUL_TYPE = "arc_element";

export type HaulItem = Haul.Item<typeof HAUL_TYPE, string, undefined>;

export const createHaulItem = (key: string): HaulItem => ({ type: HAUL_TYPE, key });

export const isHaulItem = (item: Haul.Item): item is HaulItem =>
  item.type === HAUL_TYPE;

export const filterHaulItems = (items: Haul.Item[]): HaulItem[] =>
  items.filter(isHaulItem);

export const canDropHaulItem = Haul.canDropOfType<HaulItem>(HAUL_TYPE);

// nodeChangesToActions converts diagram node gestures into Arc actions. Dimension
// changes are dropped: Arc graph nodes carry no measured field, so the diagram
// re-measures them on mount.
export const nodeChangesToActions = (
  changes: BaseDiagram.NodeChange[],
): arc.Action[] => {
  const actions: arc.Action[] = [];
  changes.forEach((ch) => {
    switch (ch.type) {
      case "position":
        actions.push(arc.setNodePosition({ key: ch.key, position: ch.position }));
        return;
      case "remove":
        actions.push(arc.removeNode({ key: ch.key }));
    }
  });
  return actions;
};

// edgeChangesToActions converts diagram edge gestures into Arc actions. Removal
// recovers the endpoints from the diagram edge key, since Arc edges carry no key
// on the wire.
export const edgeChangesToActions = (changes: BaseDiagram.EdgeChange[]): arc.Action[] =>
  changes.flatMap((ch) => {
    switch (ch.type) {
      case "add":
        return [
          arc.addEdge({
            edge: {
              source: ch.edge.source,
              target: ch.edge.target,
              kind: arc.ir.EdgeKind.continuous,
            },
          }),
        ];
      case "remove":
        return [arc.removeEdge(parseEdgeKey(ch.key))];
      default:
        return [];
    }
  });

export interface AddNodeProps {
  key: string;
  type: string;
  position?: xy.Crude;
}

// useAddNode returns a callback that appends a node of the given function type at
// the given position, seeding its config from the type's default props.
export const useAddNode = (key: arc.Key) => {
  const theme = Theming.use();
  const { dispatch } = useDispatch();
  return useCallback(
    ({ key: nodeKey, type, position }: AddNodeProps) => {
      const spec = Stage.REGISTRY[type];
      if (spec == null) return;
      dispatch({
        key,
        actions: [
          arc.setNode({
            node: {
              key: nodeKey,
              type,
              config: spec.defaultProps(theme),
              position: xy.construct(position ?? xy.ZERO),
            },
          }),
        ],
      });
    },
    [key, dispatch, theme],
  );
};

const NodeRenderer = ({
  nodeKey,
  position,
  selected,
  draggable,
}: BaseDiagram.NodeProps): ReactElement | null => {
  const key = Key.use<arc.Key>("Arc.Graph.NodeRenderer");
  const props = useSelectNodeProps({ key, nodeKey });
  const { dispatch } = useDispatch();
  const { key: type = "", ...rest } = props ?? {};
  const handleChange = useCallback(
    (config: record.Unknown) =>
      dispatch({
        key,
        actions: [arc.setNodeConfig({ key: nodeKey, config })],
      }),
    [key, nodeKey, dispatch],
  );
  if (props == null) return null;
  const C = Stage.REGISTRY[type as string];
  if (C == null) throw new Error(`Arc function ${type} not found`);
  return (
    <C.Symbol
      nodeKey={nodeKey}
      position={position}
      selected={selected}
      draggable={draggable}
      onChange={handleChange}
      {...rest}
    />
  );
};

const ArcDiagram = create({ node: Component.renderProp(NodeRenderer) });

const UNDO_TRIGGER: Triggers.Trigger = ["Control", "Z"];
const REDO_TRIGGER: Triggers.Trigger = ["Control", "Shift", "Z"];

export interface GraphProps extends Omit<
  BaseDiagram.DiagramProps,
  "nodes" | "edges" | "onNodesChange" | "onEdgesChange" | "onChange"
> {
  resourceKey: arc.Key;
  enableTriggers?: boolean | (() => boolean);
  extraMenuItems?: Component.RenderProp<Menu.ContextMenuMenuProps>;
}

// Graph is the Arc graph editor. It owns the graph document state via the flux
// store: nodes, edges, and node configs are read from the store and every
// gesture is dispatched as an Arc action. UI state (viewport, selection,
// editability) is owned by the caller and threaded through props.
export const Graph = ({
  resourceKey: key,
  viewport,
  selected,
  editable,
  onSelectionChange,
  enableTriggers,
  extraMenuItems,
  children,
  ...props
}: GraphProps): ReactElement => {
  const nodes = useSelectNodes({ key });
  const nodesRef = useSyncedRef(nodes);
  const edges = useSelectEdges({ key });
  const edgesRef = useSyncedRef(edges);
  const { dispatch } = useDispatch();

  const handleNodesChange = useCallback(
    (changes: BaseDiagram.NodeChange[]) =>
      dispatch({ key, actions: nodeChangesToActions(changes) }),
    [key, dispatch],
  );
  const handleEdgesChange = useCallback(
    (changes: BaseDiagram.EdgeChange[]) =>
      dispatch({ key, actions: edgeChangesToActions(changes) }),
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
  const { undo, canUndo } = useUndo({ key });
  const { redo, canRedo } = useRedo({ key });

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
              triggerIndicator={UNDO_TRIGGER}
            >
              <Icon.Undo />
              Undo
            </Menu.Item>
            <Menu.Item
              itemKey="redo"
              onClick={redo}
              disabled={!canRedo}
              triggerIndicator={REDO_TRIGGER}
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
    <Key.Provider value={key}>
      <ArcDiagram
        ref={ref}
        onNodesChange={handleNodesChange}
        onEdgesChange={handleEdgesChange}
        viewport={viewport}
        onSelectionChange={onSelectionChange}
        editable={editable}
        onContextMenu={contextMenu.open}
        nodes={nodes}
        edges={edges}
        selected={selected}
        {...dropProps}
        {...props}
      >
        {children}
        <Menu.ContextMenu {...contextMenu} menu={renderMenu} />
      </ArcDiagram>
    </Key.Provider>
  );
};
