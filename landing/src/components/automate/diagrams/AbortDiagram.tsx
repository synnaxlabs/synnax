import type { DiagramState } from "@/components/automate/timeline";
import type { ReactElement } from "react";

interface AbortDiagramProps {
  state: DiagramState;
}

const COLOR_ACTIVE = "var(--pluto-primary-p1)";
const COLOR_INACTIVE = "var(--pluto-gray-l9)";
const COLOR_EMERGENCY = "var(--pluto-error-z)";

const VALVE_PATH =
  "M0 0L-13-6.6C-13.7-6.9-14.7-6.4-14.7-5.3V5.3C-14.7 6.4-13.7 6.9-13 6.6L0 0ZM0 0L13-6.6C13.7-6.9 14.7-6.4 14.7-5.3V5.3C14.7 6.4 13.7 6.9 13 6.6L0 0Z";

interface ValveProps {
  x: number;
  y: number;
  color: string;
  active: boolean;
  label: string;
}

const Valve = ({ x, y, color, active, label }: ValveProps): ReactElement => (
  <g transform={`translate(${x}, ${y})`}>
    <g transform="rotate(90)">
      <path
        d={VALVE_PATH}
        stroke={color}
        strokeWidth="2"
        fill={active ? color : "none"}
        className="valve-shape"
      />
      <line x1="0" y1="0" x2="0" y2="-12" stroke={color} strokeWidth="2" />
      <rect
        x="-6"
        y="-21"
        width="12"
        height="9"
        rx="1"
        stroke={color}
        strokeWidth="2"
        fill="none"
      />
    </g>
    <text
      x="26"
      y="4"
      className="diagram-label"
      fill="var(--pluto-gray-l9)"
    >
      {label}
    </text>
  </g>
);

export const AbortDiagram = ({ state }: AbortDiagramProps): ReactElement => {
  const isEmergency = state.stage === "emergency";
  const pressActive = state.pressValve;
  const ventActive = state.ventValve;
  const pressColor = pressActive ? COLOR_ACTIVE : COLOR_INACTIVE;
  const ventColor = ventActive
    ? isEmergency
      ? COLOR_EMERGENCY
      : COLOR_ACTIVE
    : COLOR_INACTIVE;
  const tankStroke = isEmergency ? COLOR_EMERGENCY : "var(--pluto-gray-l9)";
  const tankX = 110;
  const tankY = 110;
  const tankW = 80;
  const tankH = 120;
  const rX = 40;
  const rY = 22;
  const tankPath = `M${tankX},${tankY + rY} a${rX},${rY} 0 0 1 ${tankW},0 v${tankH - 2 * rY} a${rX},${rY} 0 0 1 -${tankW},0 Z`;
  const fillHeight = Math.min((state.pressure / 900) * 80, 80);
  const fillY = tankY + tankH - fillHeight;

  return (
    <svg viewBox="-30 -10 360 350" className="automate-diagram-svg">
      {/* Top pipe */}
      <line
        x1="150"
        y1="-10"
        x2="150"
        y2="53"
        stroke={pressColor}
        strokeWidth="2"
        strokeDasharray={pressActive ? "8 4" : "none"}
        className={pressActive ? "flow-line" : ""}
      />

      <Valve x={150} y={68} color={pressColor} active={pressActive} label="Press Valve" />

      {/* Pipe to tank */}
      <line
        x1="150"
        y1="83"
        x2="150"
        y2="110"
        stroke={pressColor}
        strokeWidth="2"
        strokeDasharray={pressActive ? "8 4" : "none"}
        className={pressActive ? "flow-line" : ""}
      />

      {/* Tank */}
      <path
        d={tankPath}
        fill="var(--pluto-gray-l2)"
        stroke={tankStroke}
        strokeWidth="2"
        className={isEmergency ? "emergency-border" : ""}
      />
      <defs>
        <clipPath id="tank-clip-abort">
          <path d={tankPath} />
        </clipPath>
      </defs>
      <rect
        x={tankX}
        y={fillY}
        width={tankW}
        height={fillHeight}
        fill={isEmergency ? COLOR_EMERGENCY : COLOR_ACTIVE}
        opacity="0.2"
        className="vessel-fill"
        clipPath="url(#tank-clip-abort)"
      />
      {/* Pressure value */}
      <text
        x="150"
        y="175"
        textAnchor="middle"
        className="diagram-value-sm"
        fill="var(--pluto-text-color)"
      >
        {state.pressure.toFixed(1)} psi
      </text>

      {/* Pipe to vent */}
      <line
        x1="150"
        y1="230"
        x2="150"
        y2="260"
        stroke={ventColor}
        strokeWidth="2"
        strokeDasharray={ventActive ? "8 4" : "none"}
        className={ventActive ? "flow-line" : ""}
      />

      <Valve x={150} y={275} color={ventColor} active={ventActive} label="Vent Valve" />

      {/* Bottom pipe */}
      <line
        x1="150"
        y1="290"
        x2="150"
        y2="340"
        stroke={ventColor}
        strokeWidth="2"
        strokeDasharray={ventActive ? "8 4" : "none"}
        className={ventActive ? "flow-line" : ""}
      />

      {/* Authority meter */}
      <g transform="translate(188, -2)">
        <rect
          x="0"
          y="0"
          width="64"
          height="20"
          rx="2"
          fill="var(--pluto-gray-l2)"
          stroke="var(--pluto-gray-l4)"
          strokeWidth="1"
        />
        <rect
          x="0"
          y="0"
          width={(state.authority / 255) * 64}
          height="20"
          rx="2"
          fill={isEmergency ? COLOR_EMERGENCY : COLOR_ACTIVE}
          opacity="0.25"
          className="authority-fill"
        />
        <text
          x="32"
          y="10"
          textAnchor="middle"
          dominantBaseline="central"
          className="diagram-badge-sm"
          style={{ fill: "var(--pluto-gray-l9)" }}
        >
          auth {state.authority}
        </text>
      </g>

      {/* Stage badge */}
      <rect
        x="258"
        y="-2"
        width="64"
        height="20"
        rx="2"
        fill={isEmergency ? COLOR_EMERGENCY : "var(--pluto-gray-l2)"}
        fillOpacity={isEmergency ? 0.15 : 1}
        stroke={isEmergency ? COLOR_EMERGENCY : "var(--pluto-gray-l4)"}
        strokeWidth="1"
        className={isEmergency ? "emergency-border" : ""}
      />
      <text
        x="290"
        y="8"
        textAnchor="middle"
        dominantBaseline="central"
        className="diagram-badge-sm"
        style={{ fill: isEmergency ? COLOR_EMERGENCY : "var(--pluto-gray-l9)" }}
      >
        {state.stage}
      </text>

      {/* Safed indicator */}
      {state.stage === "safed" && (
        <g transform="translate(258, 22)">
          <rect
            x="0"
            y="0"
            width="64"
            height="20"
            rx="2"
            fill="var(--pluto-secondary-z)"
            fillOpacity="0.15"
            stroke="var(--pluto-secondary-z)"
            strokeWidth="1"
          />
          <text
            x="32"
            y="10"
            textAnchor="middle"
            dominantBaseline="central"
            className="diagram-badge-sm"
            style={{ fill: "var(--pluto-secondary-z)" }}
          >
            SAFE
          </text>
        </g>
      )}
    </svg>
  );
};
