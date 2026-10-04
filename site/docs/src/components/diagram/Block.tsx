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

const Tab = ({ x, y }: { x: number; y: number }): ReactElement => (
  <g className="block-diagram__tab">
    <rect
      x={x}
      y={y - TAB_HEIGHT / 2}
      width={TAB_WIDTH}
      height={TAB_HEIGHT}
      rx={3}
      className="block-diagram__tab-body"
    />
    <rect
      x={x + 2}
      y={y - TAB_HEIGHT / 4}
      width={TAB_WIDTH - 4}
      height={TAB_HEIGHT / 2}
      rx={1.5}
      className="block-diagram__tab-center"
    />
  </g>
);

/** The icon chip and text of a block, drawn over the given box. */
export const Content = ({
  node: { label, caption, icon: Icon, x, y, width, height },
}: {
  node: PlacedNode;
}): ReactElement => (
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

export const Item = ({ node }: { node: PlacedNode }): ReactElement => (
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

/**
 * Draws a left-to-right block diagram. Give each node a column, and the diagram places,
 * sizes, and connects the blocks.
 */
export const Block = ({ nodes, edges, title }: BlockProps): ReactElement => {
  const {
    width,
    height,
    nodes: placed,
    edges: lines,
    sources,
    targets,
  } = layout(nodes, edges);
  return (
    <figure className="block-diagram">
      <svg
        viewBox={`0 0 ${width} ${height}`}
        width={width}
        height={height}
        role="img"
        aria-label={title}
        className="block-diagram__svg"
      >
        {lines.map(({ from, to, path }) => (
          <path key={`${from}-${to}`} d={path} className="block-diagram__edge" />
        ))}
        {placed.map((node) => (
          <g key={`tabs-${node.key}`}>
            {sources.has(node.key) && (
              <Tab x={node.x + node.width - TAB_OVERLAP} y={node.y + node.height / 2} />
            )}
            {targets.has(node.key) && (
              <Tab x={node.x - TAB_EXTEND} y={node.y + node.height / 2} />
            )}
          </g>
        ))}
        {placed.map((node) => (
          <Item key={node.key} node={node} />
        ))}
      </svg>
    </figure>
  );
};
