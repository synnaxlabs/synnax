// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type graph, xy } from "@synnaxlabs/x";
import type * as rf from "@xyflow/react";
import { MarkerType } from "@xyflow/react";

export interface Viewport {
  position: xy.XY;
  zoom: number;
}

export interface Edge extends graph.Edge {}

export interface Node extends graph.Node {
  position: xy.XY;
}

export const translateNodesForward = (
  nodes: Node[],
  dragHandleSelector?: string,
): rf.Node[] =>
  nodes.map((node) => ({
    id: node.key,
    type: "custom",
    position: { ...node.position },
    data: {},
    dragHandle: dragHandleSelector,
  }));

export const translateEdgesForward = (edges: Edge[]): Array<rf.Edge<{}>> =>
  edges.map((edge) => ({
    source: edge.source.node,
    target: edge.target.node,
    sourceHandle: edge.source.param,
    targetHandle: edge.target.param,
    id: edge.key,
    markerEnd: {
      type: MarkerType.ArrowClosed,
      strokeWidth: 2,
      color: "var(--pluto-gray-l8)",
    },
  }));

export const translateViewportForward = (viewport: Viewport): rf.Viewport => ({
  ...viewport.position,
  zoom: viewport.zoom,
});

export const translateViewportBackward = (viewport: rf.Viewport): Viewport => ({
  position: xy.construct(viewport),
  zoom: viewport.zoom,
});

export type NodeChange =
  | {
      type: "position";
      key: string;
      position: xy.XY;
    }
  | {
      type: "remove";
      key: string;
    };

export const translateNodeChangeForward = (change: rf.NodeChange): NodeChange => {
  switch (change.type) {
    case "position":
      return {
        type: "position",
        key: change.id,
        position: xy.construct(change.position ?? { x: 0, y: 0 }),
      };
    case "remove":
      return { type: "remove", key: change.id };
  }
  throw new Error(`Unknown node change type: ${change.type}`);
};

export type EdgeChange =
  | {
      type: "add";
      edge: Edge;
    }
  | {
      type: "remove";
      key: string;
    };

export const translateEdgeChangeForward = (change: rf.EdgeChange): EdgeChange => {
  switch (change.type) {
    case "add":
      return {
        type: "add",
        edge: {
          key: change.item.id,
          source: {
            node: change.item.source,
            param: change.item.sourceHandle ?? "",
          },
          target: {
            node: change.item.target,
            param: change.item.targetHandle ?? "",
          },
        },
      };
    case "remove":
      return { type: "remove", key: change.id };
  }
  throw new Error(`Unknown edge change type: ${change.type}`);
};
