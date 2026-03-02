import type { ReactElement } from "react";

import { Text } from "@synnaxlabs/pluto";

import type { NodeColors, NodeState } from "@/components/stream/diagrams/theme";
import { NODE_BG, NODE_BORDER, NODE_RX } from "@/components/stream/diagrams/theme";
import type { NodeIcon } from "@/components/stream/diagrams/types";

const TRANSITION = "all 0.3s ease";

interface NodeProps {
  x: number;
  y: number;
  w: number;
  h: number;
  label: string;
  icon: NodeIcon;
  value?: string;
  colors: NodeColors;
  state: NodeState;
  filterId?: string;
}

export const Node = ({
  x,
  y,
  w,
  h,
  label,
  icon: NodeIcon,
  value,
  colors,
  state: _state,
}: NodeProps): ReactElement => {
  const hw = w / 2;
  const hh = h / 2;
  return (
    <g>
      <rect
        x={x - hw}
        y={y - hh}
        width={w}
        height={h}
        rx={NODE_RX}
        fill={NODE_BG}
        stroke={NODE_BORDER}
        strokeWidth={0.5}
      />
      <foreignObject x={x - hw} y={y - hh} width={w} height={h}>
        <div
          style={{
            width: "100%",
            height: "100%",
            display: "flex",
            flexDirection: "column",
            justifyContent: "space-between",
            padding: "7px 8px",
            boxSizing: "border-box",
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: "6px" }}>
            <div
              style={{
                width: "18px",
                height: "18px",
                borderRadius: "4px",
                background: colors.iconBg,
                border: `0.5px solid ${colors.iconBorder}`,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                flexShrink: 0,
                transition: TRANSITION,
              }}
            >
              <NodeIcon
                style={{
                  width: "12px",
                  height: "12px",
                  color: colors.iconColor,
                  transition: TRANSITION,
                }}
              />
            </div>
            <Text.Text
              level="small"
              weight={400}
              color={colors.textColor}
              wrap={false}
              style={{ transition: TRANSITION }}
            >
              {label}
            </Text.Text>
          </div>
          <Text.Text
            level="p"
            weight={500}
            variant="code"
            color={colors.valueColor}
            style={{
              transition: TRANSITION,
              opacity: value != null ? 1 : 0,
            }}
          >
            {value ?? "\u00A0"}
          </Text.Text>
        </div>
      </foreignObject>
    </g>
  );
};

const PILL_TEXT = "var(--pluto-gray-l9)";
const PILL_ICON = "var(--pluto-gray-l7)";

const PILL_DESC = "var(--pluto-gray-l6)";

export const PillNode = ({
  x,
  y,
  w,
  h,
  label,
  icon: IconComp,
  value,
  colors,
  state,
  filterId,
}: NodeProps): ReactElement => {
  const hw = w / 2;
  const hh = h / 2;
  const isActive = state === "active";
  const iconColor = isActive ? colors.iconColor : PILL_ICON;
  const textColor = isActive ? colors.textColor : PILL_TEXT;
  const iconSize = isActive ? `${Math.round(h * 0.45)}px` : "14px";
  const hasDesc = value != null && value !== "";
  if (isActive) {
    return (
      <g>
        <rect
          x={x - hw}
          y={y - hh}
          width={w}
          height={h}
          rx={12}
          fill="white"
          stroke="var(--pluto-primary-z-30)"
          strokeWidth={1}
          filter={filterId != null ? `url(#${filterId})` : undefined}
        />
        <foreignObject x={x - hw} y={y - hh} width={w} height={h}>
          <div
            style={{
              width: "100%",
              height: "100%",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              boxSizing: "border-box",
            }}
          >
            <IconComp
              style={{
                width: iconSize,
                height: iconSize,
                color: "black",
                transition: TRANSITION,
              }}
            />
          </div>
        </foreignObject>
      </g>
    );
  }

  return (
    <g>
      <rect
        x={x - hw}
        y={y - hh}
        width={w}
        height={h}
        rx={8}
        fill={NODE_BG}
        stroke={NODE_BORDER}
        strokeWidth={0.5}
      />
      <foreignObject x={x - hw} y={y - hh} width={w} height={h}>
        <div
          style={{
            width: "100%",
            height: "100%",
            display: "flex",
            flexDirection: "column",
            alignItems: "flex-start",
            justifyContent: "space-between",
            gap: "2px",
            padding: "7px 12px",
            boxSizing: "border-box",
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: "6px" }}>
            <IconComp
              style={{
                width: iconSize,
                height: iconSize,
                color: iconColor,
                flexShrink: 0,
                transition: TRANSITION,
              }}
            />
            {label !== "" && (
              <Text.Text
                level="small"
                weight={500}
                variant="code"
                color={textColor}
                wrap={false}
                style={{
                  transition: TRANSITION,
                  fontSize: "11px",
                  textTransform: "uppercase",
                }}
              >
                {label}
              </Text.Text>
            )}
          </div>
          {hasDesc && (
            <Text.Text
              level="small"
              weight={400}
              variant="code"
              color={PILL_DESC}
              wrap={false}
              style={{ transition: TRANSITION, fontSize: "10px" }}
            >
              {value}
            </Text.Text>
          )}
        </div>
      </foreignObject>
    </g>
  );
};
