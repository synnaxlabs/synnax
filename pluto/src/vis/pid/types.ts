// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type * as rf from "reactflow";

import { type Color } from "@/color";
import { xy } from "@synnaxlabs/x";

export interface Viewport {
  position: xy.XY;
  zoom: number;
}

export interface Edge {
  key: string;
  source: string;
  target: string;
  selected: boolean;
  color: Color.Crude;
  sourceHandle?: string | null;
  targetHandle?: string | null;
  points: xy.XY[];
}

export interface Node {
  key: string;
  position: xy.XY;
  selected?: boolean;
}

export const translateNodesForward = (nodes: Node[]): rf.Node[] =>
  nodes.map((node) => ({
    ...node,
    id: node.key,
    type: "custom",
    data: {},
  }));

export const translateEdgesForward = (edges: Edge[]): rf.Edge[] =>
  edges.map(({ points, color, ...edge }) => ({
    ...edge,
    id: edge.key,
    data: { points, color },
  }));

export const translateNodesBackward = (nodes: rf.Node[]): Node[] =>
  nodes.map((node) => ({
    key: node.id,
    selected: node.selected,
    ...node,
  }));

export const translateEdgesBackward = (edges: rf.Edge[]): Edge[] =>
  edges.map((edge) => ({
    key: edge.id,
    points: edge.data?.points ?? [],
    selected: edge.selected ?? false,
    color: edge.data?.color,
    ...edge,
  }));

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
): Node[] => translateNodesBackward(f(translateNodesForward(nodes)));

export const edgeConverter = (
  edges: Edge[],
  f: (edges: rf.Edge[]) => rf.Edge[],
): Edge[] => translateEdgesBackward(f(translateEdgesForward(edges)));
