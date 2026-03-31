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
import { type EdgeProps as RFEdgeProps } from "@xyflow/react";
import { type ReactElement, useCallback } from "react";

import { CSS } from "@/css";
import { ConnectionLine, Edge } from "@/schematic/edge";
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
  | "selected"
  | "onSelectionChange"
> {
  schematicKey: string;
  selected?: string[];
  onSelectionChange?: (selected: string[]) => void;
  nodeRenderer: Diagram.DiagramProps["nodeRenderer"];
}

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
    default:
      return null;
  }
};

const edgeChangeToActions = (change: Diagram.EdgeChange): schematic.Action[] => {
  switch (change.type) {
    case "add":
      return [
        schematic.setEdge({ edge: change.edge as schematic.Edge }),
        schematic.setProps({
          key: change.edge.key,
          props: { waypoints: [], variant: "pipe" },
        }),
      ];
    case "remove":
      return [schematic.removeEdge({ key: change.key })];
    default:
      return [];
  }
};

export const Schematic = ({
  className,
  children,
  schematicKey,
  nodeRenderer,
  ...props
}: SchematicProps): ReactElement | null => {
  const { data: doc } = useRetrieve({ key: schematicKey });
  const { update: dispatch } = useDispatch();

  const handleNodesChange = useCallback(
    (changes: Diagram.NodeChange[]) => {
      const actions = changes
        .map(nodeChangeToAction)
        .filter((a): a is schematic.Action => a != null);
      if (actions.length > 0) dispatch({ key: schematicKey, actions });
    },
    [schematicKey, dispatch],
  );

  const handleEdgesChange = useCallback(
    (changes: Diagram.EdgeChange[]) => {
      const actions = changes.flatMap(edgeChangeToActions);
      if (actions.length > 0) dispatch({ key: schematicKey, actions });
    },
    [schematicKey, dispatch],
  );

  const edgeRenderer = useCallback(
    (rfProps: RFEdgeProps) => <Edge schematicKey={schematicKey} {...rfProps} />,
    [schematicKey],
  );

  return (
    <Diagram.Diagram
      className={CSS(CSS.B("schematic"), className)}
      dragHandleSelector={`.${DRAG_HANDLE_CLASS}`}
      autoRenderInterval={AUTO_RENDER_INTERVAL}
      nodes={doc?.nodes ?? []}
      edges={doc?.edges ?? []}
      onNodesChange={handleNodesChange}
      onEdgesChange={handleEdgesChange}
      nodeRenderer={nodeRenderer}
      edgeRenderer={edgeRenderer}
      connectionLineComponent={ConnectionLine}
      {...props}
    >
      {children}
    </Diagram.Diagram>
  );
};
