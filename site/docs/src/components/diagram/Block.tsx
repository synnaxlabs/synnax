// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/components/diagram/Block.css";

import { type ReactElement } from "react";

import {
  type Direction,
  type Edge,
  layout,
  type Node,
  type PlacedNode,
  TAB_EXTEND,
} from "@/components/diagram/layout";

export interface BlockProps {
  nodes: Node[];
  edges: Edge[];
  /** Describes the diagram for screen readers. */
  title: string;
}

const TAB_OVERLAP = 8;
const TAB_WIDTH = TAB_EXTEND + TAB_OVERLAP;
const TAB_HEIGHT = 12;

interface TabProps {
  /** The center of the edge of the block that the tab sits on. */
  x: number;
  y: number;
  /** Which edge of the block the tab sits on. */
  side: "left" | "right" | "top" | "bottom";
}

// The tab overlaps the block and extends past its edge.
const Tab = ({ x, y, side }: TabProps): ReactElement => {
  const across = side === "left" || side === "right";
  const width = across ? TAB_WIDTH : TAB_HEIGHT;
  const height = across ? TAB_HEIGHT : TAB_WIDTH;
  const start = side === "left" || side === "top" ? -TAB_EXTEND : -TAB_OVERLAP;
  const left = across ? x + start : x - width / 2;
  const top = across ? y - height / 2 : y + start;
  return (
    <g className="block-diagram__tab">
      <rect
        x={left}
        y={top}
        width={width}
        height={height}
        rx={3}
        className="block-diagram__tab-body"
      />
      <rect
        x={left + (across ? 2 : width / 4)}
        y={top + (across ? height / 4 : 2)}
        width={across ? width - 4 : width / 2}
        height={across ? height / 2 : height - 4}
        rx={1.5}
        className="block-diagram__tab-center"
      />
    </g>
  );
};

interface NodeProps {
  node: Omit<PlacedNode, "key" | "column">;
}

/** The icon chip and text of a block, drawn over the given box. */
export const Content = ({
  node: { label, caption, icon: Icon, x, y, width, height },
}: NodeProps): ReactElement => (
  <foreignObject x={x} y={y} width={width} height={height}>
    <div className="block-diagram__content">
      {Icon != null && (
        <span className="block-diagram__icon">
          <Icon />
        </span>
      )}
      <span className="block-diagram__text">
        <span className="block-diagram__label">{label}</span>
        {caption != null && <span className="block-diagram__caption">{caption}</span>}
      </span>
    </div>
  </foreignObject>
);

export const Item = ({ node }: NodeProps): ReactElement => (
  <g>
    <rect
      x={node.x}
      y={node.y}
      width={node.width}
      height={node.height}
      rx={6}
      className="block-diagram__node"
    />
    <Content node={node} />
  </g>
);

interface SvgProps extends BlockProps {
  direction: Direction;
}

const Svg = ({ nodes, edges, title, direction }: SvgProps): ReactElement => {
  const {
    width,
    height,
    nodes: placed,
    edges: lines,
    sources,
    targets,
  } = layout(nodes, edges, direction);
  const right = direction === "right";
  return (
    <svg
      viewBox={`0 0 ${width} ${height}`}
      width={width}
      height={height}
      role="img"
      aria-label={title}
      className={`block-diagram__svg block-diagram__svg--${right ? "wide" : "narrow"}`}
    >
      {lines.map(({ from, to, path }) => (
        <path key={`${from}-${to}`} d={path} className="block-diagram__edge" />
      ))}
      {placed.map((node) => (
        <g key={`tabs-${node.key}`}>
          {sources.has(node.key) &&
            (right ? (
              <Tab x={node.x + node.width} y={node.y + node.height / 2} side="right" />
            ) : (
              <Tab x={node.x + node.width / 2} y={node.y + node.height} side="bottom" />
            ))}
          {targets.has(node.key) &&
            (right ? (
              <Tab x={node.x} y={node.y + node.height / 2} side="left" />
            ) : (
              <Tab x={node.x + node.width / 2} y={node.y} side="top" />
            ))}
        </g>
      ))}
      {placed.map((node) => (
        <Item key={node.key} node={node} />
      ))}
    </svg>
  );
};

/**
 * Draws a block diagram that flows from left to right, or from top to bottom in a
 * narrow column. Give each node a column, and the diagram places, sizes, and connects
 * the blocks.
 */
export const Block = (props: BlockProps): ReactElement => (
  <figure className="block-diagram">
    <Svg {...props} direction="right" />
    <Svg {...props} direction="down" />
  </figure>
);
