// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color, record, xy } from "@synnaxlabs/x";
import type * as rf from "@xyflow/react";
import { MarkerType } from "@xyflow/react";
import { z } from "zod/v4";

/**
 * The current viewport state of the diagram.
 */
export const viewportZ = z.object({
  /*
   * The top-left pixel offset of the diagram pan position. Note that this
   * offset is unscaled by zoom.
   */
  position: xy.xy,

  /**
   * A decimal of the current diagram zoom. Larger values represent
   * magnification.
   */
  zoom: z.number(),
});

/**
 * The current viewport state of the diagram.
 */
export type Viewport = z.infer<typeof viewportZ>;

/*
 * The properties for an edge within a diagram.
 */
export const edgeZ = z.object({
  /**
   * A unique key for identifying the edge within the diagram.
   */
  key: z.string(),

  /**
   * The key of the source node for the edge.
   */
  source: z.string(),

  /**
   * The key of the target node for the edge.
   */
  target: z.string(),

  id: z.string(),
  data: record.unknownZ.optional(),

  /**
   * Whether the edge is currently selected.
   */
  selected: z.boolean(),

  /**
   * The id of handle on the source node that the edge is connected to. Note
   * that this id is unique only within the source node.
   */
  sourceHandle: z.string().nullable().optional(),

  /**
   * The id of the handle on the target node that the edge is connected to. Note
   * that this id is unique only within the target node.
   */
  targetHandle: z.string().nullable().optional(),
});

/**
 * The properties for an edge within a diagram.
 */
export type Edge = z.infer<typeof edgeZ>;

/**
 * The properties for a node within a diagram.
 */
export const nodeZ = z.object({
  /** A unique key for identifying the node within the diagram. */
  key: z.string(),
  /** The XY coordinate of the top left corner of the node. Unscaled by the viewport. */
  position: xy.xy,
  /** Whether the node is currently selected. */
  selected: z.boolean().optional(),
  /** An optional z-index for the node. */
  zIndex: z.number().optional(),
  /** The type of the node. */
  type: z.string().optional(),
  /** The data associated with the node. */
  data: record.unknownZ.optional(),
  /** The measured dimensions of the node. */
  measured: z
    .object({ width: z.number().optional(), height: z.number().optional() })
    .optional(),
});

/**
 * The properties for a node within a diagram.
 */
export type Node = z.infer<typeof nodeZ>;

/**
 * Translates nodes from their pluto representation to their react-flow representation.
 */
export const translateNodesForward = (
  nodes: Node[],
  dragHandleSelector?: string,
): rf.Node[] =>
  nodes.map((node) => ({
    id: node.key,
    type: "custom",
    zIndex: node.zIndex,
    measured: { ...node.measured },
    position: { ...node.position },
    selected: node.selected,
    data: {},
    dragHandle: dragHandleSelector,
  }));

/** Translates edges from their pluto representation to their react-flow representation. */
export const translateEdgesForward = (edges: Edge[]): Array<rf.Edge<record.Unknown>> =>
  edges.map(({ data, ...edge }) => ({
    ...edge,
    id: edge.key,
    data,
    markerEnd: {
      type: MarkerType.ArrowClosed,
      strokeWidth: 2,
      color: "var(--pluto-gray-l8)",
    },
  }));

/** Translates nodes from their react-flow representation to their pluto representation. */
export const translateNodesBackward = (nodes: rf.Node[]): Node[] =>
  nodes.map((node) => ({ key: node.id, ...node }));

/** Translates edges from their react-flow representation to their pluto representation */
export const translateEdgesBackward = (
  edges: Array<rf.Edge<record.Unknown>>,
  defaultColor: color.Crude,
): Edge[] =>
  edges.map((edge) => {
    edge.data ??= { segments: [], color: defaultColor, variant: "pipe" };
    return {
      key: edge.id,
      selected: edge.selected ?? false,
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
  f: (edges: rf.Edge<record.Unknown>[]) => rf.Edge<record.Unknown>[],
  color: color.Crude,
): Edge[] => translateEdgesBackward(f(translateEdgesForward(edges)), color);
