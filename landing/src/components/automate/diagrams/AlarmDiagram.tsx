import type { ReactElement } from "react";

import {
  ACCENT,
  COMPUTE_LABEL_STYLE,
  LABEL_STYLE,
  LINE_IDLE,
  RULE_OFF,
  RULE_ON,
  TEXT_OFF,
  TEXT_ON,
  VALUE_OFF,
  VALUE_ON,
  VALUE_STYLE,
  WARNING_ACCENT,
} from "@/components/diagrams/diagramConstants";
import { DiagramDefs, useDiagramDefs } from "@/components/diagrams/DiagramDefs";
import type { DiagramState } from "@/components/automate/timeline";

interface AlarmDiagramProps {
  state: DiagramState;
}

interface NodeDef {
  x: number;
  y: number;
  label: string;
  id: string;
}

const NODES: NodeDef[] = [
  { x: 50, y: 130, label: "press_pt", id: "sensor" },
  { x: 150, y: 130, label: "check", id: "check" },
  { x: 250, y: 130, label: "stable_for", id: "stable" },
];

const SELECT_X = 320;
const SELECT_Y = 130;
const WARN_X = 290;
const WARN_Y = 55;
const NOM_X = 290;
const NOM_Y = 210;

const isNodeActive = (activeNode: string, nodeId: string): boolean => {
  if (activeNode === nodeId) return true;
  if (activeNode === "select-true" || activeNode === "select-false") {
    return nodeId === "sensor" || nodeId === "check" || nodeId === "stable";
  }
  return false;
};

export const AlarmDiagram = ({ state }: AlarmDiagramProps): ReactElement => {
  const ids = useDiagramDefs();
  const selectActive =
    state.activeNode === "select-true" || state.activeNode === "select-false";
  const trueActive = state.activeNode === "select-true";
  const falseActive = state.activeNode === "select-false";

  return (
    <svg viewBox="0 0 380 260" className="automate-diagram-svg">
      <DiagramDefs ids={ids} />

      {/* Pipeline nodes */}
      {NODES.map((node, i) => {
        const active = isNodeActive(state.activeNode, node.id);

        return (
          <g key={node.id}>
            <text
              x={node.x}
              y={node.y - 10}
              textAnchor="middle"
              fill={active ? TEXT_ON : TEXT_OFF}
              style={LABEL_STYLE}
            >
              {node.label}
            </text>
            <line
              x1={node.x - 28}
              y1={node.y}
              x2={node.x + 28}
              y2={node.y}
              stroke={active ? RULE_ON : RULE_OFF}
              strokeWidth="1"
              style={{ transition: "stroke 0.5s ease" }}
            />

            {active && (
              <circle
                cx={node.x + 28}
                cy={node.y}
                r="12"
                fill={`url(#${ids.accentEndpoint})`}
              />
            )}

            {/* Connector bezier to next node */}
            {i < NODES.length - 1 && (
              <path
                d={`M${node.x + 32},${node.y} C${node.x + 55},${node.y} ${NODES[i + 1].x - 50},${NODES[i + 1].y} ${NODES[i + 1].x - 32},${NODES[i + 1].y}`}
                fill="none"
                stroke={
                  active ? `url(#${ids.accentFlow})` : LINE_IDLE
                }
                strokeWidth={active ? "0.75" : "0.5"}
                style={{ transition: "all 0.6s ease" }}
              />
            )}
          </g>
        );
      })}

      {/* Last node → select bezier */}
      <path
        d={`M${NODES[2].x + 32},${NODES[2].y} C${NODES[2].x + 55},${NODES[2].y} ${SELECT_X - 40},${SELECT_Y} ${SELECT_X - 22},${SELECT_Y}`}
        fill="none"
        stroke={selectActive ? `url(#${ids.accentFlow})` : LINE_IDLE}
        strokeWidth={selectActive ? "0.75" : "0.5"}
        style={{ transition: "all 0.6s ease" }}
      />

      {/* Select circle */}
      <circle
        cx={SELECT_X}
        cy={SELECT_Y}
        r="20"
        fill="none"
        stroke={selectActive ? ACCENT : "rgba(255,255,255,0.04)"}
        strokeWidth="0.75"
        style={{ transition: "stroke 0.5s ease" }}
      />
      {selectActive && (
        <circle
          cx={SELECT_X}
          cy={SELECT_Y}
          r="28"
          fill={`url(#${ids.accentEndpoint})`}
        />
      )}
      <text
        x={SELECT_X}
        y={SELECT_Y + 4}
        textAnchor="middle"
        fill={selectActive ? TEXT_ON : TEXT_OFF}
        style={COMPUTE_LABEL_STYLE}
      >
        select
      </text>

      {/* Select → warning branch (up-left bezier) */}
      <path
        d={`M${SELECT_X - 14},${SELECT_Y - 14} C${SELECT_X - 30},${SELECT_Y - 40} ${WARN_X + 40},${WARN_Y + 10} ${WARN_X + 28},${WARN_Y}`}
        fill="none"
        stroke={
          trueActive ? `url(#${ids.warningFlow})` : LINE_IDLE
        }
        strokeWidth={trueActive ? "0.75" : "0.5"}
        style={{ transition: "all 0.6s ease" }}
      />

      {trueActive && (
        <circle
          cx={WARN_X + 28}
          cy={WARN_Y}
          r="12"
          fill={`url(#${ids.warningEndpoint})`}
        />
      )}
      <text
        x={WARN_X}
        y={WARN_Y - 10}
        textAnchor="middle"
        fill={trueActive ? WARNING_ACCENT : TEXT_OFF}
        style={LABEL_STYLE}
      >
        warning
      </text>
      <line
        x1={WARN_X - 28}
        y1={WARN_Y}
        x2={WARN_X + 28}
        y2={WARN_Y}
        stroke={trueActive ? WARNING_ACCENT : RULE_OFF}
        strokeWidth="1"
        style={{ transition: "stroke 0.5s ease" }}
      />

      {/* Select → nominal branch (down-left bezier) */}
      <path
        d={`M${SELECT_X - 14},${SELECT_Y + 14} C${SELECT_X - 30},${SELECT_Y + 40} ${NOM_X + 40},${NOM_Y - 10} ${NOM_X + 28},${NOM_Y}`}
        fill="none"
        stroke={
          falseActive ? `url(#${ids.accentFlow})` : LINE_IDLE
        }
        strokeWidth={falseActive ? "0.75" : "0.5"}
        style={{ transition: "all 0.6s ease" }}
      />

      {falseActive && (
        <circle
          cx={NOM_X + 28}
          cy={NOM_Y}
          r="12"
          fill={`url(#${ids.accentEndpoint})`}
        />
      )}
      <text
        x={NOM_X}
        y={NOM_Y - 10}
        textAnchor="middle"
        fill={falseActive ? TEXT_ON : TEXT_OFF}
        style={LABEL_STYLE}
      >
        nominal
      </text>
      <line
        x1={NOM_X - 28}
        y1={NOM_Y}
        x2={NOM_X + 28}
        y2={NOM_Y}
        stroke={falseActive ? RULE_ON : RULE_OFF}
        strokeWidth="1"
        style={{ transition: "stroke 0.5s ease" }}
      />

      {/* Pressure value below press_pt */}
      <text
        x={NODES[0].x}
        y={NODES[0].y + 16}
        textAnchor="middle"
        fill={VALUE_OFF}
        style={VALUE_STYLE}
      >
        {state.pressure} PSI
      </text>
    </svg>
  );
};
