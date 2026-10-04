// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Icon } from "@synnaxlabs/lyra/icon";

/** A block in a diagram. Blocks in one column stack in the order they are given. */
export interface Node {
  key: string;
  label: string;
  /** A second, smaller line under the label. */
  caption?: string;
  icon?: Icon.FC;
  column: number;
}

/** A line from the right side of one block to the left side of another. */
export interface Edge {
  from: string;
  to: string;
}

export interface PlacedNode extends Node {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface PlacedEdge extends Edge {
  path: string;
}

export interface Layout {
  width: number;
  height: number;
  nodes: PlacedNode[];
  edges: PlacedEdge[];
  /** Keys of the blocks that have a line leaving them. */
  sources: Set<string>;
  /** Keys of the blocks that have a line entering them. */
  targets: Set<string>;
}

export const TAB_EXTEND = 6;
const PADDING = TAB_EXTEND + 2;
const COLUMN_GAP = 72;
const ROW_GAP = 20;
const NODE_HEIGHT = 40;
const CAPTION_HEIGHT = 16;
const LABEL_CHAR_WIDTH = 7.4;
const CAPTION_CHAR_WIDTH = 6.8;
const ICON_SPACE = 26;
const X_PADDING = 24;
const CORNER_RADIUS = 8;

export const nodeWidth = ({
  label,
  caption,
  icon,
}: Pick<Node, "label" | "caption" | "icon">): number => {
  const text = Math.max(
    label.length * LABEL_CHAR_WIDTH,
    (caption?.length ?? 0) * CAPTION_CHAR_WIDTH,
  );
  return Math.ceil(text + (icon == null ? 0 : ICON_SPACE) + X_PADDING);
};

export const nodeHeight = ({ caption }: Pick<Node, "caption">): number =>
  caption == null ? NODE_HEIGHT : NODE_HEIGHT + CAPTION_HEIGHT;

const path = (x1: number, y1: number, x2: number, y2: number, midX: number): string => {
  if (Math.abs(y1 - y2) < 1) return `M${x1},${y1} L${x2},${y2}`;
  const dy = y2 > y1 ? 1 : -1;
  const r = Math.min(CORNER_RADIUS, Math.abs(y2 - y1) / 2, Math.abs(midX - x1) / 2);
  return [
    `M${x1},${y1}`,
    `L${midX - r},${y1}`,
    `Q${midX},${y1} ${midX},${y1 + dy * r}`,
    `L${midX},${y2 - dy * r}`,
    `Q${midX},${y2} ${midX + r},${y2}`,
    `L${x2},${y2}`,
  ].join(" ");
};

/**
 * Places blocks in columns from left to right. Each column is as wide as its widest
 * block and is centered vertically. Lines bend at right angles in the gap before their
 * target column.
 *
 * @throws Error if an edge refers to a key that no node has.
 */
export const layout = (nodes: Node[], edges: Edge[]): Layout => {
  const columnCount = Math.max(...nodes.map(({ column }) => column)) + 1;
  const columns = Array.from({ length: columnCount }, (_, i) =>
    nodes.filter(({ column }) => column === i),
  );
  const columnWidths = columns.map((col) => Math.max(0, ...col.map(nodeWidth)));
  const columnHeights = columns.map(
    (col) =>
      col.reduce((sum, n) => sum + nodeHeight(n), 0) +
      Math.max(0, col.length - 1) * ROW_GAP,
  );
  const contentHeight = Math.max(...columnHeights);
  const placed = new Map<string, PlacedNode>();
  let x = PADDING;
  columns.forEach((col, i) => {
    let y = PADDING + (contentHeight - columnHeights[i]) / 2;
    for (const node of col) {
      const height = nodeHeight(node);
      const width = columnWidths[i];
      placed.set(node.key, { ...node, x, y, width, height });
      y += height + ROW_GAP;
    }
    x += columnWidths[i] + COLUMN_GAP;
  });
  const get = (key: string): PlacedNode => {
    const node = placed.get(key);
    if (node == null) throw new Error(`diagram edge refers to unknown node "${key}"`);
    return node;
  };
  const placedEdges = edges.map((edge) => {
    const from = get(edge.from);
    const to = get(edge.to);
    const x1 = from.x + from.width + TAB_EXTEND;
    const x2 = to.x - TAB_EXTEND;
    const y1 = from.y + from.height / 2;
    const y2 = to.y + to.height / 2;
    return { ...edge, path: path(x1, y1, x2, y2, to.x - COLUMN_GAP / 2) };
  });
  return {
    width: x - COLUMN_GAP + PADDING,
    height: contentHeight + 2 * PADDING,
    nodes: [...placed.values()],
    edges: placedEdges,
    sources: new Set(edges.map(({ from }) => from)),
    targets: new Set(edges.map(({ to }) => to)),
  };
};
