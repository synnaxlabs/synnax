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
import { type ReactElement, useCallback } from "react";

import { CSS } from "@/css";
import { Provider } from "@/schematic/Context";
import { Edge } from "@/schematic/edge";
import { Node } from "@/schematic/node";
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
  | "selected"
  | "onSelectionChange"
> {
  resourceKey: string;
  selected?: string[];
  onSelectionChange?: (selected: string[]) => void;
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
          props: { waypoints: [], variant: "pipe", color: "#000000" },
        }),
      ];
    case "remove":
      return [schematic.removeEdge({ key: change.key })];
    default:
      return [];
  }
};

const SchematicDiagram = Diagram.create({
  node: (props) => <Node.Node {...props} />,
  edge: (props) => <Edge.Edge {...props} />,
  connectionLine: (props) => <Edge.ConnectionLine {...props} />,
});

export const Schematic = ({
  className,
  children,
  resourceKey: key,
  ...props
}: SchematicProps): ReactElement | null => {
  const { data: doc } = useRetrieve({ key });
  const { update: dispatch } = useDispatch();

  const handleNodesChange = useCallback(
    (changes: Diagram.NodeChange[]) => {
      const actions = changes
        .map(nodeChangeToAction)
        .filter((a): a is schematic.Action => a != null);
      if (actions.length > 0) dispatch({ key, actions });
    },
    [key, dispatch],
  );

  const handleEdgesChange = useCallback(
    (changes: Diagram.EdgeChange[]) => {
      const actions = changes.flatMap(edgeChangeToActions);
      if (actions.length > 0) dispatch({ key, actions });
    },
    [key, dispatch],
  );

  return (
    <Provider value={key}>
      <SchematicDiagram
        className={CSS(CSS.B("schematic"), className)}
        dragHandleSelector={`.${DRAG_HANDLE_CLASS}`}
        autoRenderInterval={AUTO_RENDER_INTERVAL}
        nodes={doc?.nodes ?? []}
        edges={doc?.edges ?? []}
        onNodesChange={handleNodesChange}
        onEdgesChange={handleEdgesChange}
        {...props}
      >
        {children}
      </SchematicDiagram>
    </Provider>
  );
};
