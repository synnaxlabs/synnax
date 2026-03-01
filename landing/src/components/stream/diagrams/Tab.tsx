import type { ReactElement } from "react";

import type { NodeColors } from "@/components/stream/diagrams/theme";
import { NODE_RX } from "@/components/stream/diagrams/theme";

export const TAB_EXTEND = 10;
export const TAB_OVERLAP = 14;

const TRANSITION = "fill 0.3s ease, stroke 0.3s ease";

interface TabProps {
  nodeEdgeX: number;
  centerY: number;
  side: "left" | "right";
  colors: NodeColors;
}

export const Tab = ({
  nodeEdgeX,
  centerY,
  side,
  colors,
}: TabProps): ReactElement => {
  const x =
    side === "right" ? nodeEdgeX - TAB_OVERLAP : nodeEdgeX - TAB_EXTEND;
  const innerX = x + 3;
  return (
    <g>
      <rect
        x={x}
        y={centerY - 9}
        width={TAB_EXTEND + TAB_OVERLAP}
        height={18}
        rx={NODE_RX}
        fill={colors.tabFill}
        stroke={colors.tabBorder}
        strokeWidth={0.5}
        style={{ transition: TRANSITION }}
      />
      <rect
        x={innerX}
        y={centerY - 5}
        width={TAB_EXTEND + TAB_OVERLAP - 6}
        height={10}
        rx={2}
        fill={colors.tabCenter}
        style={{ transition: TRANSITION }}
      />
    </g>
  );
};

export const MINI_TAB_EXTEND = 6;
const MINI_TAB_OVERLAP = 8;

export const MiniTab = ({
  nodeEdgeX,
  centerY,
  side,
  colors,
}: TabProps): ReactElement => {
  const x =
    side === "right"
      ? nodeEdgeX - MINI_TAB_OVERLAP
      : nodeEdgeX - MINI_TAB_EXTEND;
  const innerX = x + 2;
  return (
    <g>
      <rect
        x={x}
        y={centerY - 6}
        width={MINI_TAB_EXTEND + MINI_TAB_OVERLAP}
        height={12}
        rx={3}
        fill={colors.tabFill}
        stroke={colors.tabBorder}
        strokeWidth={0.5}
        style={{ transition: TRANSITION }}
      />
      <rect
        x={innerX}
        y={centerY - 3}
        width={MINI_TAB_EXTEND + MINI_TAB_OVERLAP - 4}
        height={6}
        rx={1.5}
        fill={colors.tabCenter}
        style={{ transition: TRANSITION }}
      />
    </g>
  );
};
