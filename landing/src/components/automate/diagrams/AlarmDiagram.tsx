import type { DiagramState } from "@/components/automate/timeline";
import type { ReactElement } from "react";

interface AlarmDiagramProps {
  state: DiagramState;
}

const ACTIVE = "var(--pluto-primary-p1)";
const INACTIVE = "var(--pluto-gray-l9)";
const WARNING = "var(--pluto-warning-z)";

interface NodeDef {
  x: number;
  y: number;
  label: string;
  id: string;
}

const NODES: NodeDef[] = [
  { x: 40, y: 100, label: "press_pt", id: "sensor" },
  { x: 130, y: 100, label: "check", id: "check" },
  { x: 220, y: 100, label: "stable_for", id: "stable" },
];

const isNodeActive = (activeNode: string, nodeId: string): boolean => {
  if (activeNode === nodeId) return true;
  if (activeNode === "select-true" || activeNode === "select-false") {
    return nodeId === "sensor" || nodeId === "check" || nodeId === "stable";
  }
  return false;
};

export const AlarmDiagram = ({ state }: AlarmDiagramProps): ReactElement => {
  const selectActive =
    state.activeNode === "select-true" || state.activeNode === "select-false";
  const trueActive = state.activeNode === "select-true";
  const falseActive = state.activeNode === "select-false";

  return (
    <svg viewBox="0 0 320 260" className="automate-diagram-svg">
      {/* Pipeline nodes */}
      {NODES.map((node, i) => {
        const active = isNodeActive(state.activeNode, node.id);
        const nodeColor = active ? ACTIVE : INACTIVE;
        return (
          <g key={node.id}>
            <rect
              x={node.x - 32}
              y={node.y - 16}
              width="64"
              height="32"
              rx="6"
              fill="none"
              stroke={nodeColor}
              strokeWidth="2"
              className="diagram-node"
            />
            <text
              x={node.x}
              y={node.y + 4}
              textAnchor="middle"
              className="diagram-label"
              fill={active ? "var(--pluto-text-color)" : "var(--pluto-gray-l6)"}
            >
              {node.label}
            </text>
            {/* Connector to next node */}
            {i < NODES.length - 1 && (
              <line
                x1={node.x + 32}
                y1={node.y}
                x2={NODES[i + 1].x - 32}
                y2={NODES[i + 1].y}
                stroke={active ? ACTIVE : INACTIVE}
                strokeWidth="2"
                strokeDasharray={active ? "4 3" : "none"}
                className={active ? "flow-line" : ""}
              />
            )}
          </g>
        );
      })}

      {/* select node */}
      <line
        x1="252"
        y1="100"
        x2="280"
        y2="100"
        stroke={selectActive ? ACTIVE : INACTIVE}
        strokeWidth="2"
        className={selectActive ? "flow-line" : ""}
      />
      <polygon
        points="280,80 320,100 280,120"
        fill="none"
        stroke={selectActive ? ACTIVE : INACTIVE}
        strokeWidth="2"
        className="diagram-node"
      />
      <text
        x="296"
        y="104"
        textAnchor="middle"
        className="diagram-label-sm"
        fill={selectActive ? "var(--pluto-text-color)" : "var(--pluto-gray-l6)"}
      >
        select
      </text>

      {/* True branch (warning) */}
      <line
        x1="300"
        y1="80"
        x2="260"
        y2="40"
        stroke={trueActive ? WARNING : INACTIVE}
        strokeWidth="2"
      />
      <rect
        x="190"
        y="24"
        width="72"
        height="32"
        rx="6"
        fill={trueActive ? WARNING : "none"}
        fillOpacity={trueActive ? 0.15 : 0}
        stroke={trueActive ? WARNING : INACTIVE}
        strokeWidth="2"
        className="diagram-node"
      />
      <text
        x="226"
        y="44"
        textAnchor="middle"
        className="diagram-label"
        fill={trueActive ? WARNING : "var(--pluto-gray-l6)"}
      >
        warning
      </text>

      {/* False branch (ok) */}
      <line
        x1="300"
        y1="120"
        x2="260"
        y2="170"
        stroke={falseActive ? ACTIVE : INACTIVE}
        strokeWidth="2"
      />
      <rect
        x="198"
        y="154"
        width="64"
        height="32"
        rx="6"
        fill={falseActive ? ACTIVE : "none"}
        fillOpacity={falseActive ? 0.15 : 0}
        stroke={falseActive ? ACTIVE : INACTIVE}
        strokeWidth="2"
        className="diagram-node"
      />
      <text
        x="230"
        y="174"
        textAnchor="middle"
        className="diagram-label"
        fill={falseActive ? "var(--pluto-text-color)" : "var(--pluto-gray-l6)"}
      >
        nominal
      </text>

      {/* Pressure reading */}
      <text
        x="40"
        y="145"
        textAnchor="middle"
        className="diagram-value-sm"
        fill="var(--pluto-gray-l9)"
      >
        {state.pressure} PSI
      </text>
    </svg>
  );
};
