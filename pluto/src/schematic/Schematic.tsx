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
import { box, color, TimeSpan, xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useRef } from "react";

import { Component } from "@/component";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { useSyncedRef } from "@/hooks";
import { Provider } from "@/schematic/Context";
import { Edge } from "@/schematic/edge";
import { Node } from "@/schematic/node";
import { useAddNode, useDispatch, useRetrieve } from "@/schematic/queries";
import { DRAG_HANDLE_CLASS } from "@/schematic/symbol/Grid";
import { Theming } from "@/theming";
import { Diagram } from "@/vis/diagram";

export interface SchematicProps extends Omit<
  Diagram.DiagramProps,
  | "dragHandleSelector"
  | "nodes"
  | "edges"
  | "onNodesChange"
  | "onEdgesChange"
  | "selected"
  | "onSelectionChange"
> {
  resourceKey: string;
  selected?: string[];
  onSelectionChange?: (selected: string[]) => void;
}

const AUTO_RENDER_INTERVAL = TimeSpan.seconds(1).milliseconds;
export const HAUL_TYPE = "schematic-element";

const nodeChangeToAction = (change: Diagram.NodeChange): schematic.Action | null => {
  switch (change.type) {
    case "position":
      return schematic.setNodePosition({ key: change.key, position: change.position });
    case "remove":
      return schematic.removeNode({ key: change.key });
    default:
      return null;
  }
};

const edgeChangeToActions = (
  change: Diagram.EdgeChange,
  baseEdgeColor: color.Color,
): schematic.Action[] => {
  switch (change.type) {
    case "add":
      return [
        schematic.setEdge({ edge: change.edge }),
        schematic.setProps({
          key: change.edge.key,
          props: { segments: [], variant: "pipe", color: color.hex(baseEdgeColor) },
        }),
      ];
    case "remove":
      return [schematic.removeEdge({ key: change.key })];
    default:
      return [];
  }
};

const SchematicDiagram = Diagram.create({
  node: Component.renderProp(Node.Node),
  edge: Component.renderProp(Edge.Edge),
  connectionLine: Component.renderProp(Edge.ConnectionLine),
});

export const Schematic = ({
  className,
  resourceKey,
  viewport,
  onDoubleClick,
  onSelectionChange,
  ...props
}: SchematicProps): ReactElement => {
  const { data: doc } = useRetrieve({ key: resourceKey });
  const { update: dispatch } = useDispatch();
  const theme = Theming.use();

  const handleNodesChange = useCallback(
    (changes: Diagram.NodeChange[]) => {
      const actions = changes
        .map(nodeChangeToAction)
        .filter((a): a is schematic.Action => a != null);
      if (actions.length > 0) dispatch({ key: resourceKey, actions });
    },
    [resourceKey, dispatch],
  );

  const handleEdgesChange = useCallback(
    (changes: Diagram.EdgeChange[]) => {
      const actions = changes.flatMap((change) =>
        edgeChangeToActions(change, theme.colors.gray.l10),
      );
      if (actions.length > 0) dispatch({ key: resourceKey, actions });
    },
    [resourceKey, dispatch, theme.colors.gray.l10],
  );

  const handleAddNode = useAddNode(resourceKey);
  const ref = useRef<HTMLDivElement>(null);
  const viewportRef = useSyncedRef(viewport);
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
      if (event == null) return valid;
      valid.forEach(({ key, data }) => {
        const pos = xy.truncate(calculateCursorPosition(event), 0);
        handleAddNode(key.toString(), pos, data);
      });
      return valid;
    },
    [handleAddNode, calculateCursorPosition],
  );

  const dropProps = Haul.useDrop({
    type: "Schematic",
    key: resourceKey,
    canDrop: Haul.canDropOfType(HAUL_TYPE),
    onDrop: handleDrop,
  });

  const handleClearSelection = useCallback(() => onSelectionChange?.([]), []);

  Diagram.useTriggers({
    onCopy: () => {},
    onPaste: () => {},
    onSelectAll: () => {},
    onClear: handleClearSelection,
    onUndo: () => {},
    onRedo: () => {},
    region: ref,
  });

  return (
    <div
      ref={ref}
      onDoubleClick={onDoubleClick}
      className={CSS(CSS.BE("schematic", "container"))}
    >
      <Provider value={resourceKey}>
        <SchematicDiagram
          className={CSS(CSS.B("schematic"), className)}
          dragHandleSelector={`.${DRAG_HANDLE_CLASS}`}
          autoRenderInterval={AUTO_RENDER_INTERVAL}
          nodes={doc?.nodes ?? []}
          edges={doc?.edges ?? []}
          onNodesChange={handleNodesChange}
          onEdgesChange={handleEdgesChange}
          viewport={viewport}
          onSelectionChange={onSelectionChange}
          onDoubleClick={onDoubleClick}
          {...dropProps}
          {...props}
        />
      </Provider>
    </div>
  );
};
