// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { xy } from "@synnaxlabs/x";
import type * as rf from "reactflow";

import { type Color } from "@/color";
import { type connector } from "@/vis/diagram/edge/connector";

/** The current viewport state of the diagram */
export interface Viewport {
  /*
   * The top-left pixel offset of the diagram pan position.
   * Note that this offset is unscaled by zoom.
   * */
  position: xy.XY;
  /**
   * A decimal of the current diagram zoom. Larger values represent magnification.
   */
  zoom: number;
}

/** Pluto specific info passed to the 'data' attribute on rf.Edge */
export interface RFEdgeData {
  /** The color of the edge. */
  color: Color.Crude;
  /** A list of segments representing the structure of the edge connector. */
  segments: connector.Segment[];
}

/** The properties for an edge within a diagram. */
export interface Edge extends RFEdgeData {
  /** A unique key for identifying the edge within the diagram. */
  key: string;
  /** The key of the source node for the edge. */
  source: string;
  /** The key of the target node for the edge. */
  target: string;
  /** Whether the edge is currently selected. */
  selected: boolean;
  /**
   * The id of handle on the source node that the edge is connected to. Note that this
   * id is unique only within the source node.
   */
  sourceHandle?: string | null;
  /**
   * The id of the handle on the target node that the edge is connected to. Note that this id
   * is unique only within the target node.
   */
  targetHandle?: string | null;
}

/** The properties for a node within a diagram. */
export interface Node {
  /** A unique key for identifying the node within the diagram. */
  key: string;
  /** The XY coordinate of the top left corner of the node. Unscaled by the viewport. */
  position: xy.XY;
  /** Whether the node is currently selected. */
  selected?: boolean;
  /** An optional z-index for the node. */
  zIndex?: number;
}

/**
 * Translates nodes from their pluto representation to their react-flow representation.
 */
export const translateNodesForward = (nodes: Node[]): rf.Node[] =>
  nodes.map((node) => ({
    ...node,
    id: node.key,
    type: "custom",
    data: {},
  }));

/** Translates edges from their pluto representation to their react-flow representation. */
export const translateEdgesForward = (edges: Edge[]): Array<rf.Edge<RFEdgeData>> =>
  edges.map(({ segments, color, ...edge }) => ({
    ...edge,
    id: edge.key,
    data: { segments, color },
  }));

/** Translates nodes from their react-flow representation to their pluto representation. */
export const translateNodesBackward = (nodes: rf.Node[]): Node[] =>
  nodes.map((node) => ({
    key: node.id,
    selected: node.selected,
    ...node,
  }));

/** Translates edges from their react-flow representation to their pluto representation */
export const translateEdgesBackward = (
  edges: Array<rf.Edge<RFEdgeData>>,
  defaultColor: Color.Crude,
): Edge[] =>
  edges.map((edge) => {
    if (edge.data == null) edge.data = { segments: [], color: defaultColor };
    return {
      key: edge.id,
      segments: edge.data?.segments ?? [],
      selected: edge.selected ?? false,
      color: edge.data?.color ?? defaultColor,
      ...edge,
    };
  });

/** Translates the diagram viewport from its pluto representation to its react-flow representation */
export const translateViewportForward = (viewport: Viewport): rf.Viewport => ({
  ...viewport.position,
  zoom: viewport.zoom,
});

/** Translates the diagram viewport from its react-flow representation to its pluto representation */
export const translateViewportBackward = (viewport: rf.Viewport): Viewport => ({
  position: xy.construct(viewport),
  zoom: viewport.zoom,
});

/**
 * Executes the provided callback against the react-flow representation of the given
 * nodes, then converts
 */
export const nodeConverter = (
  nodes: Node[],
  f: (nodes: rf.Node[]) => rf.Node[],
): Node[] => translateNodesBackward(f(translateNodesForward(nodes)));

export const edgeConverter = (
  edges: Edge[],
  f: (edges: rf.Edge[]) => rf.Edge[],
  color: Color.Crude,
): Edge[] => translateEdgesBackward(f(translateEdgesForward(edges)), color);
