// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/Schematic.css";

import { schematic } from "@synnaxlabs/client";
import { type record, TimeSpan } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { Component } from "@/component";
import { CSS } from "@/css";
import { Key } from "@/key";
import { ConnectionLine, Edge, type EdgeData } from "@/schematic/edge";
import { useRetrieve, useSelectNodePropsAndType, useUpdate } from "@/schematic/queries";
import { Symbol } from "@/schematic/symbol";
import { DRAG_HANDLE_CLASS } from "@/schematic/symbol/Grid";
import { Diagram } from "@/vis/diagram";

export interface SchematicProps
  extends Omit<
    Diagram.DiagramProps,
    "dragHandleSelector" | "edges" | "nodes" | "onNodesChange" | "onEdgesChange"
  > {
  schematicKey: string;
}

const edgeRenderer = Component.renderProp(Edge);

const AUTO_RENDER_INTERVAL = TimeSpan.seconds(1).milliseconds;

const nodeChangeToAction = (change: Diagram.NodeChange): schematic.Action => {
  switch (change.type) {
    case "position":
      return schematic.setNodePosition({ key: change.key, position: change.position });
    case "remove":
      return schematic.removeNode({ key: change.key });
  }
};

const edgeChangeToAction = (change: Diagram.EdgeChange): schematic.Action => {
  switch (change.type) {
    case "add":
      return schematic.setEdge(change.edge);
    case "remove":
      return schematic.removeEdge({ key: change.key });
  }
};

export const NodeRenderer = ({
  symbolKey,
  position,
  selected,
  draggable,
}: Diagram.SymbolProps): ReactElement => {
  const { type, props } = useSelectNodePropsAndType(symbolKey);
  const key = Key.use();
  const { update: dispatch } = useUpdate();
  const handleChange = useCallback(
    (props: record.Unknown) =>
      dispatch({ key, ...schematic.setNodeProps({ key, props }) }),
    [symbolKey, dispatch],
  );
  const C = Symbol.REGISTRY[type as Symbol.Variant];
  return (
    <C.Symbol
      key={symbolKey}
      id={symbolKey}
      symbolKey={symbolKey}
      position={position}
      selected={selected}
      onChange={handleChange}
      draggable={draggable}
      {...props}
    />
  );
};

const nodeRenderer = Component.renderProp(NodeRenderer);

export const Schematic = ({
  className,
  children,
  schematicKey,
  ...props
}: SchematicProps): ReactElement | null => {
  const result = useRetrieve({ key: schematicKey });
  const { update: dispatch } = useUpdate();

  const handleNodesChange = useCallback(
    (nodes: Diagram.NodeChange[]) =>
      nodes.forEach((node) =>
        dispatch({ key: schematicKey, ...nodeChangeToAction(node) }),
      ),
    [schematicKey, dispatch],
  );

  const handleEdgesChange = useCallback(
    (edges: Diagram.EdgeChange[]) =>
      edges.forEach((edge) =>
        dispatch({ key: schematicKey, ...edgeChangeToAction(edge) }),
      ),
    [schematicKey, dispatch],
  );

  if (result.variant !== "success") return null;

  return (
    <Key.Provider itemKey={schematicKey}>
      <Diagram.Diagram
        className={CSS(CSS.B("schematic"), className)}
        dragHandleSelector={`.${DRAG_HANDLE_CLASS}`}
        autoRenderInterval={AUTO_RENDER_INTERVAL}
        onNodesChange={handleNodesChange}
        onEdgesChange={handleEdgesChange}
        edges={result.data.edges}
        nodes={result.data.nodes}
        {...props}
      >
        <Diagram.EdgeRenderer<EdgeData> connectionLineComponent={ConnectionLine}>
          {edgeRenderer}
        </Diagram.EdgeRenderer>
        <Diagram.NodeRenderer>{nodeRenderer}</Diagram.NodeRenderer>
      </Diagram.Diagram>
    </Key.Provider>
  );
};
