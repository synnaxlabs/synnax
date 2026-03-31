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
import { TimeSpan } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo } from "react";

import { Component } from "@/component";
import { CSS } from "@/css";
import { ConnectionLine, Edge, type EdgeData } from "@/schematic/edge";
import { useDispatch, useRetrieve } from "@/schematic/queries";
import { DRAG_HANDLE_CLASS } from "@/schematic/symbol/Grid";
import { Diagram } from "@/vis/diagram";

export interface SchematicProps extends Omit<
  Diagram.DiagramProps,
  | "dragHandleSelector"
  | "nodes"
  | "edges"
  | "onNodesChange"
  | "onEdgesChange"
  | "nodeRenderer"
  | "edgeRenderer"
  | "connectionLineComponent"
> {
  schematicKey: string;
  selectedNodes?: string[];
  selectedEdges?: string[];
  onSelectionChange?: (nodes: string[], edges: string[]) => void;
  nodeRenderer: Diagram.DiagramProps["nodeRenderer"];
}

const edgeRendererProp = Component.renderProp(Edge) as Diagram.DiagramProps["edgeRenderer"];

const AUTO_RENDER_INTERVAL = TimeSpan.seconds(1).milliseconds;

const nodeChangeToAction = (change: Diagram.NodeChange): schematic.Action | null => {
  switch (change.type) {
    case "position":
      return schematic.setNodePosition({ key: change.key, position: change.position });
    case "remove":
      return schematic.removeNode({ key: change.key });
    case "dimensions":
      return schematic.setNodeDimensions({
        key: change.key,
        dimensions: change.dimensions,
      });
    case "select":
      return null;
  }
};

const edgeChangeToAction = (change: Diagram.EdgeChange): schematic.Action | null => {
  switch (change.type) {
    case "add":
      return schematic.setEdge({ edge: change.edge as schematic.Edge });
    case "remove":
      return schematic.removeEdge({ key: change.key });
    case "data":
      return schematic.setEdgeData({
        key: change.key,
        data: change.data as schematic.EdgeData,
      });
    case "select":
      return null;
  }
};

export const Schematic = ({
  className,
  children,
  schematicKey,
  selectedNodes = [],
  selectedEdges = [],
  onSelectionChange,
  nodeRenderer,
  ...props
}: SchematicProps): ReactElement | null => {
  const { data: doc } = useRetrieve({ key: schematicKey });
  const { update: dispatch } = useDispatch();

  const nodes = useMemo(() => {
    const srcNodes = doc?.nodes ?? [];
    const selectedSet = new Set(selectedNodes);
    return srcNodes.map((n) => ({ ...n, selected: selectedSet.has(n.key) }));
  }, [doc, selectedNodes]);

  const edges = useMemo(() => {
    const srcEdges = doc?.edges ?? [];
    const selectedSet = new Set(selectedEdges);
    return srcEdges.map((e) => ({ ...e, selected: selectedSet.has(e.key) }));
  }, [doc, selectedEdges]);

  const handleNodesChange = useCallback(
    (changes: Diagram.NodeChange[]) => {
      const selChanges = changes.filter(
        (c): c is Diagram.NodeChange & { type: "select" } => c.type === "select",
      );
      if (selChanges.length > 0 && onSelectionChange != null) {
        const current = new Set(selectedNodes);
        for (const c of selChanges)
          if (c.selected) current.add(c.key);
          else current.delete(c.key);

        onSelectionChange([...current], selectedEdges);
      }

      const actions = changes
        .map(nodeChangeToAction)
        .filter((a): a is schematic.Action => a != null);
      if (actions.length > 0) dispatch({ key: schematicKey, actions });
    },
    [schematicKey, dispatch, selectedNodes, selectedEdges, onSelectionChange],
  );

  const handleEdgesChange = useCallback(
    (changes: Diagram.EdgeChange[]) => {
      const selChanges = changes.filter(
        (c): c is Diagram.EdgeChange & { type: "select" } => c.type === "select",
      );
      if (selChanges.length > 0 && onSelectionChange != null) {
        const current = new Set(selectedEdges);
        for (const c of selChanges)
          if (c.selected) current.add(c.key);
          else current.delete(c.key);

        onSelectionChange(selectedNodes, [...current]);
      }

      const actions = changes
        .map(edgeChangeToAction)
        .filter((a): a is schematic.Action => a != null);
      if (actions.length > 0) dispatch({ key: schematicKey, actions });
    },
    [schematicKey, dispatch, selectedNodes, selectedEdges, onSelectionChange],
  );

  return (
    <Diagram.Diagram
      className={CSS(CSS.B("schematic"), className)}
      dragHandleSelector={`.${DRAG_HANDLE_CLASS}`}
      autoRenderInterval={AUTO_RENDER_INTERVAL}
      nodes={nodes}
      edges={edges}
      onNodesChange={handleNodesChange}
      onEdgesChange={handleEdgesChange}
      nodeRenderer={nodeRenderer}
      edgeRenderer={edgeRendererProp}
      connectionLineComponent={ConnectionLine}
      {...props}
    >
      {children}
    </Diagram.Diagram>
  );
};
