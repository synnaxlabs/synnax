import type { DiagramState } from "@/components/automate/timeline";
import type { ReactElement } from "react";

interface PressureDiagramProps {
  state: DiagramState;
}

const COLOR_ACTIVE = "var(--pluto-primary-p1)";
const COLOR_INACTIVE = "var(--pluto-gray-l9)";

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

export const PressureDiagram = ({ state }: PressureDiagramProps): ReactElement => {
  const pressActive = state.pressValve;
  const ventActive = state.ventValve;
  const pressColor = pressActive ? COLOR_ACTIVE : COLOR_INACTIVE;
  const ventColor = ventActive ? COLOR_ACTIVE : COLOR_INACTIVE;
  const tankX = 110;
  const tankY = 110;
  const tankW = 80;
  const tankH = 120;
  const rX = 40;
  const rY = 22;
  const tankPath = `M${tankX},${tankY + rY} a${rX},${rY} 0 0 1 ${tankW},0 v${tankH - 2 * rY} a${rX},${rY} 0 0 1 -${tankW},0 Z`;
  const fillHeight = Math.min((state.pressure / 600) * 80, 80);
  const fillY = tankY + tankH - fillHeight;

  return (
    <svg viewBox="0 0 300 300" className="automate-diagram-svg">
      {/* Top pipe */}
      <line
        x1="150"
        y1="0"
        x2="150"
        y2="53"
        stroke={pressColor}
        strokeWidth="2"
        strokeDasharray={pressActive ? "8 4" : "none"}
        className={pressActive ? "flow-line" : ""}
      />

      <Valve x={150} y={68} color={pressColor} active={pressActive} label="Press Valve" />

      {/* Pipe from press valve to tank */}
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

      {/* Tank vessel */}
      <path
        d={tankPath}
        fill="none"
        stroke="var(--pluto-gray-l9)"
        strokeWidth="2"
      />
      {/* Tank fill level — clipped to tank shape */}
      <defs>
        <clipPath id="tank-clip-pressure">
          <path d={tankPath} />
        </clipPath>
      </defs>
      <rect
        x={tankX}
        y={fillY}
        width={tankW}
        height={fillHeight}
        fill={COLOR_ACTIVE}
        opacity="0.2"
        className="vessel-fill"
        clipPath="url(#tank-clip-pressure)"
      />

      {/* Pressure value (Pluto Value style) */}
      <text
        x="150"
        y="175"
        textAnchor="middle"
        className="diagram-value"
        fill="var(--pluto-text-color)"
      >
        {state.pressure.toFixed(1)} psi
      </text>

      {/* Pipe from tank to vent valve */}
      <line
        x1="150"
        y1="230"
        x2="150"
        y2="240"
        stroke={ventColor}
        strokeWidth="2"
        strokeDasharray={ventActive ? "8 4" : "none"}
        className={ventActive ? "flow-line" : ""}
      />

      <Valve x={150} y={255} color={ventColor} active={ventActive} label="Vent Valve" />

      {/* Bottom pipe */}
      <line
        x1="150"
        y1="270"
        x2="150"
        y2="300"
        stroke={ventColor}
        strokeWidth="2"
        strokeDasharray={ventActive ? "8 4" : "none"}
        className={ventActive ? "flow-line" : ""}
      />

      {/* Stage badge */}
      <rect
        x="205"
        y="165"
        width="72"
        height="24"
        rx="4"
        fill="var(--pluto-gray-l2)"
        stroke="var(--pluto-gray-l4)"
        strokeWidth="1"
      />
      <text
        x="241"
        y="181"
        textAnchor="middle"
        className="diagram-badge"
        fill="var(--pluto-text-color)"
      >
        {state.stage}
      </text>
    </svg>
  );
};
