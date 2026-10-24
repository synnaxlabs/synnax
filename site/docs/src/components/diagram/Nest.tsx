// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/components/diagram/Block.css";

import { type Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement, useId } from "react";

import { Content, Item } from "@/components/diagram/Block";
import { nodeHeight, nodeWidth, ROW_GAP, span } from "@/components/diagram/layout";

/** A box in a nested diagram. A box with children draws as a group around them. */
export interface NestNode {
  key: string;
  label: string;
  caption?: string;
  icon?: Icon.FC;
  /** Children lay out in this many columns. Defaults to 1. */
  columns?: number;
  /** Columns on narrow screens. Defaults to columns. */
  narrowColumns?: number;
  children?: NestNode[];
}

export interface NestProps {
  root: NestNode;
  /** Describes the diagram for screen readers. */
  title: string;
}

const PADDING = 12;
const GAP = ROW_GAP;
const HATCH_SPACING = 6;
// Keeps the hatch clear of the border, which is 0.5 wide and centered on the box edge.
const HATCH_INSET = 1;

type Columns = (node: NestNode) => number;

const wideColumns: Columns = (node) => node.columns ?? 1;
const narrowColumns: Columns = (node) => node.narrowColumns ?? wideColumns(node);

interface Placed {
  node: NestNode;
  x: number;
  y: number;
  width: number;
  height: number;
  children: Placed[];
}

const groupDepth = (node: NestNode): number =>
  node.children == null ? 0 : 1 + Math.max(0, ...node.children.map(groupDepth));

// Every leaf gets the width of the widest leaf, so the boxes line up across groups.
const leafWidth = (node: NestNode): number =>
  node.children == null ? nodeWidth(node) : Math.max(...node.children.map(leafWidth));

const place = (
  node: NestNode,
  x: number,
  y: number,
  leaf: number,
  columns: Columns,
): Placed => {
  if (node.children == null)
    return { node, x, y, width: leaf, height: nodeHeight(node), children: [] };
  const count = columns(node);
  const header = nodeHeight(node);
  // Place each child at the origin first to learn its size.
  const sizes = node.children.map((c) => place(c, 0, 0, leaf, columns));
  const colWidths = Array.from({ length: count }, (_, c) =>
    Math.max(0, ...sizes.filter((_, i) => i % count === c).map((s) => s.width)),
  );
  const rowHeights = Array.from({ length: Math.ceil(sizes.length / count) }, (_, r) =>
    Math.max(...sizes.slice(r * count, (r + 1) * count).map((s) => s.height)),
  );
  const children = node.children.map((child, i) => {
    const c = i % count;
    const r = Math.floor(i / count);
    const cx = x + PADDING + span(colWidths.slice(0, c), GAP) + (c > 0 ? GAP : 0);
    const cy = y + header + span(rowHeights.slice(0, r), GAP) + (r > 0 ? GAP : 0);
    return place(child, cx, cy, leaf, columns);
  });
  return {
    node,
    x,
    y,
    width: span(colWidths, GAP) + 2 * PADDING,
    height: header + span(rowHeights, GAP) + PADDING,
    children,
  };
};

interface GroupProps {
  placed: Placed;
  depth: number;
  hatch: string;
}

const Group = ({ placed, depth, hatch }: GroupProps): ReactElement => {
  const { node, x, y, width, height, children } = placed;
  const box = { x, y, width, height, rx: 6 };
  return (
    <g data-depth={depth}>
      <rect {...box} className="block-diagram__group" />
      <rect
        x={x + HATCH_INSET}
        y={y + HATCH_INSET}
        width={width - 2 * HATCH_INSET}
        height={height - 2 * HATCH_INSET}
        rx={box.rx - HATCH_INSET}
        fill={`url(#${hatch}-${depth})`}
      />
      <rect {...box} className="block-diagram__border" />
      <g className="block-diagram__header">
        <Content node={{ ...node, x, y, width, height: nodeHeight(node) }} />
      </g>
      {children.map((c) =>
        c.node.children != null ? (
          <Group key={c.node.key} placed={c} depth={depth + 1} hatch={hatch} />
        ) : (
          <Item
            key={c.node.key}
            node={{ ...c.node, x: c.x, y: c.y, width: c.width, height: c.height }}
          />
        ),
      )}
    </g>
  );
};

const Svg = ({
  root,
  title,
  columns,
  variant,
}: NestProps & { columns: Columns; variant: "wide" | "narrow" }): ReactElement => {
  const hatch = `nest-hatch-${useId()}`;
  const placed = place(root, 0, 0, leafWidth(root), columns);
  return (
    <svg
      viewBox={`0 0 ${placed.width} ${placed.height}`}
      width={placed.width}
      height={placed.height}
      role="img"
      aria-label={title}
      className={`block-diagram__svg block-diagram__svg--${variant}`}
    >
      {/* A pattern's lines take their color from the pattern, not from the shape that
      uses it, so each depth gets its own. */}
      <defs>
        {Array.from({ length: groupDepth(root) }, (_, depth) => (
          <pattern
            key={depth}
            id={`${hatch}-${depth}`}
            data-depth={depth}
            width={HATCH_SPACING}
            height={HATCH_SPACING}
            patternUnits="userSpaceOnUse"
            patternTransform="rotate(45)"
          >
            {/* Centered, so the tile edge does not clip half of the stroke. */}
            <line
              x1={HATCH_SPACING / 2}
              y1={0}
              x2={HATCH_SPACING / 2}
              y2={HATCH_SPACING}
              className="block-diagram__hatch-line"
            />
          </pattern>
        ))}
      </defs>
      <Group placed={placed} depth={0} hatch={hatch} />
    </svg>
  );
};

/**
 * Draws boxes nested in groups, in the style of a block diagram. Leaves are blocks, and
 * each group shows its label above its children. Narrow screens get a second layout
 * that uses narrowColumns.
 */
export const Nest = (props: NestProps): ReactElement => (
  <figure className="block-diagram block-diagram--nest">
    <Svg {...props} columns={wideColumns} variant="wide" />
    <Svg {...props} columns={narrowColumns} variant="narrow" />
  </figure>
);
