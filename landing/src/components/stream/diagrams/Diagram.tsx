import { type ReactElement, useId } from "react";

import type { CalcDiagramState } from "@/components/stream/calcTimeline";
import { Node, PillNode } from "@/components/stream/diagrams/Node";
import { MiniTab, MINI_TAB_EXTEND, Tab, TAB_EXTEND } from "@/components/stream/diagrams/Tab";
import { Trace } from "@/components/stream/diagrams/Trace";
import { BLUE_TAB_COLORS, deriveNodeState, resolveNodeColors } from "@/components/stream/diagrams/theme";
import type { DiagramDef } from "@/components/stream/diagrams/types";

const DOT_SIZE = 0.5;
const DOT_SPACING = 16;
const DOT_COLOR = "var(--pluto-gray-l5)";
const SURFACE_BG = "var(--pluto-gray-l2-30)";

const WRAPPER_STYLE: React.CSSProperties = {
  position: "relative",
  width: "100%",
  height: "100%",
  overflow: "hidden",
  backgroundColor: SURFACE_BG,
  backgroundImage: `radial-gradient(circle, ${DOT_COLOR} ${DOT_SIZE}px, transparent ${DOT_SIZE}px)`,
  backgroundSize: `${DOT_SPACING}px ${DOT_SPACING}px`,
};

const SVG_STYLE: React.CSSProperties = {
  display: "block",
  width: "100%",
  height: "100%",
};

interface DiagramProps {
  def: DiagramDef;
  state: CalcDiagramState;
}

export const Diagram = ({ def, state }: DiagramProps): ReactElement => {
  const uid = useId();
  const nodeMap = new Map(def.nodes.map((n) => [n.id, n]));
  const isPill = def.variant === "pill";

  const hasRightTab = new Set(def.edges.map((e) => e.from));
  const hasLeftTab = new Set(def.edges.map((e) => e.to));

  const NodeComp = isPill ? PillNode : Node;

  return (
    <div style={WRAPPER_STYLE}>
      <svg viewBox={def.viewBox} style={SVG_STYLE}>
        {/* Traces */}
        {def.edges.map((edge, i) => {
          const from = nodeMap.get(edge.from)!;
          const to = nodeMap.get(edge.to)!;
          const fromActive = state.activeNodes.includes(edge.from);
          const toActive = state.activeNodes.includes(edge.to);
          const flowing = fromActive || toActive;
          const excluded = state.excludedNodes.includes(edge.from);
          const pad = isPill ? MINI_TAB_EXTEND : TAB_EXTEND;
          const x1 = from.x + from.w / 2 + pad;
          const x2 = to.x - to.w / 2 - pad;
          return (
            <Trace
              key={`${edge.from}-${edge.to}`}
              pathId={`${uid}-t-${i}`}
              x1={x1}
              y1={from.y}
              x2={x2}
              y2={to.y}
              flowing={flowing}
              excluded={excluded}
            />
          );
        })}

        {/* Tabs (behind nodes) */}
        {def.nodes.map((node) => {
          const nodeState = deriveNodeState(
            node.id,
            state.activeNodes,
            state.excludedNodes,
            state.alarmNodes,
          );
          const colors = resolveNodeColors(nodeState);
          const hw = node.w / 2;
          const TabComp = isPill ? MiniTab : Tab;
          const tabColors = isPill ? { ...colors, ...BLUE_TAB_COLORS } : colors;
          return (
            <g key={`tabs-${node.id}`}>
              {hasRightTab.has(node.id) && (
                <TabComp
                  nodeEdgeX={node.x + hw}
                  centerY={node.y}
                  side="right"
                  colors={tabColors}
                />
              )}
              {hasLeftTab.has(node.id) && (
                <TabComp
                  nodeEdgeX={node.x - hw}
                  centerY={node.y}
                  side="left"
                  colors={tabColors}
                />
              )}
            </g>
          );
        })}

        {/* Node bodies (on top) */}
        {def.nodes.map((node) => {
          const nodeState = deriveNodeState(
            node.id,
            state.activeNodes,
            state.excludedNodes,
            state.alarmNodes,
          );
          const colors = resolveNodeColors(nodeState);
          return (
            <NodeComp
              key={node.id}
              x={node.x}
              y={node.y}
              w={node.w}
              h={node.h}
              label={node.label}
              icon={node.icon}
              value={state.nodeValues[node.id]}
              colors={colors}
              state={nodeState}
            />
          );
        })}
      </svg>
    </div>
  );
};
