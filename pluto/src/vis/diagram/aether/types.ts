// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type location, xy } from "@synnaxlabs/x";
import type * as rf from "@xyflow/react";
import { MarkerType } from "@xyflow/react";
import type React from "react";
import { z } from "zod/v4";

export const viewportZ = z.object({
  position: xy.xyZ,
  zoom: z.number(),
});
export type Viewport = z.infer<typeof viewportZ>;

export const handleZ = z.object({
  node: z.string(),
  param: z.string(),
});
export type Handle = z.infer<typeof handleZ>;

export const edgeZ = z.object({
  key: z.string(),
  source: handleZ,
  target: handleZ,
});
export type Edge = z.infer<typeof edgeZ>;

export const nodeZ = z.object({
  key: z.string(),
  position: xy.xyZ,
  zIndex: z.number().optional(),
  type: z.string().optional(),
  measured: z
    .object({ width: z.number().optional(), height: z.number().optional() })
    .optional(),
});
export type Node = z.infer<typeof nodeZ>;

export const FIT_VIEW_OPTIONS: FitViewOptions = {
  maxZoom: 1,
  minZoom: 0.5,
  padding: 0.05,
};

export const translateNodesForward = (
  nodes: Node[],
  selected: Set<string>,
  dragHandleSelector?: string,
): rf.Node[] =>
  nodes.map((node) => ({
    id: node.key,
    type: "custom",
    zIndex: node.zIndex,
    measured: { ...node.measured },
    position: { ...node.position },
    selected: selected.has(node.key),
    data: {},
    dragHandle: dragHandleSelector,
  }));

export const translateEdgesForward = (
  edges: Edge[],
  selected: Set<string>,
): rf.Edge[] =>
  edges.map((edge) => ({
    id: edge.key,
    source: edge.source.node,
    target: edge.target.node,
    sourceHandle: edge.source.param,
    targetHandle: edge.target.param,
    selected: selected.has(edge.key),
    markerEnd: {
      type: MarkerType.ArrowClosed,
      strokeWidth: 2,
      color: "var(--pluto-gray-l8)",
    },
  }));

export const translateNodesBackward = (nodes: rf.Node[]): Node[] =>
  nodes.map((node) => ({ key: node.id, ...node }));

export const translateViewportForward = (viewport: Viewport): rf.Viewport => ({
  ...viewport.position,
  zoom: viewport.zoom,
});

export const translateViewportBackward = (viewport: rf.Viewport): Viewport => ({
  position: xy.construct(viewport),
  zoom: viewport.zoom,
});

export const nodeConverter = (
  nodes: Node[],
  f: (nodes: rf.Node[]) => rf.Node[],
): Node[] => translateNodesBackward(f(translateNodesForward(nodes, new Set())));

export type NodeChange =
  | { type: "position"; key: string; position: xy.XY; dragging: boolean }
  | { type: "remove"; key: string }
  | { type: "select"; key: string; selected: boolean }
  | {
      type: "dimensions";
      key: string;
      dimensions: { width: number; height: number };
    };

export const translateNodeChangeForward = (
  change: rf.NodeChange,
): NodeChange | null => {
  switch (change.type) {
    case "position":
      if (change.position == null) return null;
      return {
        type: "position",
        key: change.id,
        position: xy.construct(change.position),
        dragging: change.dragging ?? false,
      };
    case "remove":
      return { type: "remove", key: change.id };
    case "select":
      return { type: "select", key: change.id, selected: change.selected };
    case "dimensions":
      if (change.dimensions == null) return null;
      return {
        type: "dimensions",
        key: change.id,
        dimensions: change.dimensions,
      };
    default:
      return null;
  }
};

export type EdgeChange =
  | { type: "add"; edge: Edge }
  | { type: "remove"; key: string }
  | { type: "select"; key: string; selected: boolean };

export const translateEdgeChangeForward = (
  change: rf.EdgeChange,
): EdgeChange | null => {
  switch (change.type) {
    case "add": {
      const item = change.item;
      return {
        type: "add",
        edge: {
          key: item.id,
          source: { node: item.source, param: item.sourceHandle ?? "" },
          target: { node: item.target, param: item.targetHandle ?? "" },
        },
      };
    }
    case "remove":
      return { type: "remove", key: change.id };
    case "select":
      return { type: "select", key: change.id, selected: change.selected };
    default:
      return null;
  }
};

export type FitViewOptions = rf.FitViewOptions;

export interface EdgeEndpoint {
  position: xy.XY;
  orientation: location.Outer;
}

export interface EdgeProps {
  edgeKey: string;
  source: EdgeEndpoint;
  target: EdgeEndpoint;
  sourceNode: string;
  targetNode: string;
  selected: boolean;
}

export interface ConnectionLineProps {
  source: EdgeEndpoint;
  target: EdgeEndpoint;
  status: "valid" | "invalid" | null;
  style: React.CSSProperties;
}

export const createEndpoint = (
  x: number,
  y: number,
  position: string,
): EdgeEndpoint => ({
  position: { x, y },
  orientation: position as location.Outer,
});
