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

/** A line from one block to the next one in the flow. */
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

/** The way the flow runs: columns from left to right, or ranks from top to bottom. */
export type Direction = "right" | "down";

export const TAB_EXTEND = 6;
const PADDING = TAB_EXTEND + 2;
const COLUMN_GAP = 72;
export const ROW_GAP = 20;
const RANK_GAP = 40;
const SIBLING_GAP = 12;
// The widest rank that fits a phone column at full size.
const NARROW_WIDTH = 300;
const NODE_HEIGHT = 40;
const CAPTION_HEIGHT = 16;
const LABEL_LINE_HEIGHT = 16;
const CAPTION_LINE_HEIGHT = 14;
const LABEL_CHAR_WIDTH = 7.4;
const CAPTION_CHAR_WIDTH = 6.8;
const ICON_SPACE = 26;
const X_PADDING = 24;
const CORNER_RADIUS = 8;

/** The total length of the given lengths with a gap between each pair. */
export const span = (lengths: number[], gap: number): number =>
  lengths.reduce((a, b) => a + b, 0) + gap * Math.max(0, lengths.length - 1);

// A label or caption can hold line breaks.
const lines = (text: string): number => text.split("\n").length;

const longest = (text?: string): number =>
  text == null ? 0 : Math.max(...text.split("\n").map((line) => line.length));

export const nodeWidth = ({
  label,
  caption,
  icon,
}: Pick<Node, "label" | "caption" | "icon">): number => {
  const text = Math.max(
    longest(label) * LABEL_CHAR_WIDTH,
    longest(caption) * CAPTION_CHAR_WIDTH,
  );
  return Math.ceil(text + (icon == null ? 0 : ICON_SPACE) + X_PADDING);
};

export const nodeHeight = ({
  label,
  caption,
}: Pick<Node, "label" | "caption">): number =>
  NODE_HEIGHT +
  (lines(label) - 1) * LABEL_LINE_HEIGHT +
  (caption == null ? 0 : CAPTION_HEIGHT + (lines(caption) - 1) * CAPTION_LINE_HEIGHT);

// Breaks text at the space that makes its longest line shortest. A space after a comma
// wins, so a list never splits inside an item, as in "PLCs, OPC UA, HTTP".
const balance = (text: string): string => {
  const afterComma = [...text.matchAll(/, /g)].map(({ index }) => index + 1);
  const spaces =
    afterComma.length > 0
      ? afterComma
      : [...text.matchAll(/ /g)].map(({ index }) => index);
  if (spaces.length === 0) return text;
  const at = spaces.reduce((best, i) =>
    Math.max(i, text.length - i - 1) < Math.max(best, text.length - best - 1)
      ? i
      : best,
  );
  return `${text.slice(0, at)}\n${text.slice(at + 1)}`;
};

const wrap = (node: Node): Node => ({
  ...node,
  label: balance(node.label),
  caption: node.caption == null ? undefined : balance(node.caption),
});

const pathRight = (
  x1: number,
  y1: number,
  x2: number,
  y2: number,
  midX: number,
): string => {
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

const pathDown = (
  x1: number,
  y1: number,
  x2: number,
  y2: number,
  midY: number,
): string => {
  if (Math.abs(x1 - x2) < 1) return `M${x1},${y1} L${x2},${y2}`;
  const dx = x2 > x1 ? 1 : -1;
  const r = Math.min(CORNER_RADIUS, Math.abs(x2 - x1) / 2, Math.abs(midY - y1) / 2);
  return [
    `M${x1},${y1}`,
    `L${x1},${midY - r}`,
    `Q${x1},${midY} ${x1 + dx * r},${midY}`,
    `L${x2 - dx * r},${midY}`,
    `Q${x2},${midY} ${x2},${midY + r}`,
    `L${x2},${y2}`,
  ].join(" ");
};

interface Placement {
  width: number;
  height: number;
  nodes: Map<string, PlacedNode>;
  path: (from: PlacedNode, to: PlacedNode) => string;
}

// Each column is as wide as its widest block and is centered vertically. Lines bend at
// right angles in the gap before their target column.
const placeRight = (columns: Node[][]): Placement => {
  const columnWidths = columns.map((col) => Math.max(0, ...col.map(nodeWidth)));
  const columnHeights = columns.map((col) => span(col.map(nodeHeight), ROW_GAP));
  const contentHeight = Math.max(...columnHeights);
  const nodes = new Map<string, PlacedNode>();
  let x = PADDING;
  columns.forEach((col, i) => {
    let y = PADDING + (contentHeight - columnHeights[i]) / 2;
    for (const node of col) {
      const height = nodeHeight(node);
      const width = columnWidths[i];
      nodes.set(node.key, { ...node, x, y, width, height });
      y += height + ROW_GAP;
    }
    x += columnWidths[i] + COLUMN_GAP;
  });
  return {
    width: x - COLUMN_GAP + PADDING,
    height: contentHeight + 2 * PADDING,
    nodes,
    path: (from, to) =>
      pathRight(
        from.x + from.width + TAB_EXTEND,
        from.y + from.height / 2,
        to.x - TAB_EXTEND,
        to.y + to.height / 2,
        to.x - COLUMN_GAP / 2,
      ),
  };
};

const rankWidth = (rank: Node[]): number => span(rank.map(nodeWidth), SIBLING_GAP);

// Each column becomes a rank, with its blocks side by side at their own widths and
// centered. A rank too wide for a phone breaks its text onto two lines. Lines bend at
// right angles in the gap before their target rank.
const placeDown = (columns: Node[][]): Placement => {
  const ranks = columns.map((col) =>
    rankWidth(col) > NARROW_WIDTH ? col.map(wrap) : col,
  );
  const rankWidths = ranks.map(rankWidth);
  const rankHeights = ranks.map((rank) => Math.max(0, ...rank.map(nodeHeight)));
  const contentWidth = Math.max(...rankWidths);
  const nodes = new Map<string, PlacedNode>();
  const rankTops: number[] = [];
  let y = PADDING;
  ranks.forEach((rank, i) => {
    rankTops.push(y);
    let x = PADDING + (contentWidth - rankWidths[i]) / 2;
    for (const node of rank) {
      const width = nodeWidth(node);
      const height = nodeHeight(node);
      nodes.set(node.key, {
        ...node,
        x,
        y: y + (rankHeights[i] - height) / 2,
        width,
        height,
      });
      x += width + SIBLING_GAP;
    }
    y += rankHeights[i] + RANK_GAP;
  });
  return {
    width: contentWidth + 2 * PADDING,
    height: y - RANK_GAP + PADDING,
    nodes,
    path: (from, to) =>
      pathDown(
        from.x + from.width / 2,
        from.y + from.height + TAB_EXTEND,
        to.x + to.width / 2,
        to.y - TAB_EXTEND,
        rankTops[to.column] - RANK_GAP / 2,
      ),
  };
};

/**
 * Places and connects the blocks of a diagram. Blocks keep their columns, which run
 * from left to right or, as ranks, from top to bottom.
 *
 * @throws Error if an edge refers to a key that no node has.
 */
export const layout = (
  nodes: Node[],
  edges: Edge[],
  direction: Direction = "right",
): Layout => {
  const columnCount = Math.max(...nodes.map(({ column }) => column)) + 1;
  const columns = Array.from({ length: columnCount }, (_, i) =>
    nodes.filter(({ column }) => column === i),
  );
  const {
    width,
    height,
    nodes: placed,
    path,
  } = (direction === "right" ? placeRight : placeDown)(columns);
  const get = (key: string): PlacedNode => {
    const node = placed.get(key);
    if (node == null) throw new Error(`diagram edge refers to unknown node "${key}"`);
    return node;
  };
  return {
    width,
    height,
    nodes: [...placed.values()],
    edges: edges.map((edge) => ({ ...edge, path: path(get(edge.from), get(edge.to)) })),
    sources: new Set(edges.map(({ from }) => from)),
    targets: new Set(edges.map(({ to }) => to)),
  };
};
