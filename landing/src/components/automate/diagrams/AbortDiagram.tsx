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
  const aTankX = 75;
  const aTankY = 110;
  const aTankW = 70;
  const aTankH = 100;
  const aRX = 35;
  const aRY = 20;
  const abortTankPath = `M${aTankX},${aTankY + aRY} a${aRX},${aRY} 0 0 1 ${aTankW},0 v${aTankH - 2 * aRY} a${aRX},${aRY} 0 0 1 -${aTankW},0 Z`;
  const fillHeight = Math.min((state.pressure / 900) * 65, 65);
  const fillY = aTankY + aTankH - fillHeight;
  const authBarWidth = (state.authority / 255) * 90;

  return (
    <svg viewBox="0 0 300 290" className="automate-diagram-svg">
      {/* Top pipe */}
      <line
        x1="110"
        y1="0"
        x2="110"
        y2="78"
        stroke={pressColor}
        strokeWidth="2"
        strokeDasharray={pressActive ? "8 4" : "none"}
        className={pressActive ? "flow-line" : ""}
      />

      <Valve x={110} y={90} color={pressColor} active={pressActive} label="Press Valve" />

      {/* Pipe to tank */}
      <line
        x1="110"
        y1="103"
        x2="110"
        y2="110"
        stroke={pressColor}
        strokeWidth="2"
        strokeDasharray={pressActive ? "8 4" : "none"}
        className={pressActive ? "flow-line" : ""}
      />

      {/* Tank */}
      <path
        d={abortTankPath}
        fill="none"
        stroke={tankStroke}
        strokeWidth="2"
        className={isEmergency ? "emergency-border" : ""}
      />
      <defs>
        <clipPath id="tank-clip-abort">
          <path d={abortTankPath} />
        </clipPath>
      </defs>
      <rect
        x={aTankX}
        y={fillY}
        width={aTankW}
        height={fillHeight}
        fill={isEmergency ? COLOR_EMERGENCY : COLOR_ACTIVE}
        opacity="0.2"
        className="vessel-fill"
        clipPath="url(#tank-clip-abort)"
      />
      {/* Pressure value (Pluto Value style) */}
      <text
        x="110"
        y="166"
        textAnchor="middle"
        className="diagram-value-sm"
        fill="var(--pluto-text-color)"
      >
        {state.pressure.toFixed(1)} psi
      </text>

      {/* Pipe to vent */}
      <line
        x1="110"
        y1="210"
        x2="110"
        y2="225"
        stroke={ventColor}
        strokeWidth="2"
        strokeDasharray={ventActive ? "8 4" : "none"}
        className={ventActive ? "flow-line" : ""}
      />

      <Valve x={110} y={238} color={ventColor} active={ventActive} label="Vent Valve" />

      {/* Bottom pipe */}
      <line
        x1="110"
        y1="251"
        x2="110"
        y2="290"
        stroke={ventColor}
        strokeWidth="2"
        strokeDasharray={ventActive ? "8 4" : "none"}
        className={ventActive ? "flow-line" : ""}
      />

      {/* Authority meter */}
      <g transform="translate(185, 50)">
        <text
          x="0"
          y="0"
          className="diagram-label"
          fill="var(--pluto-gray-l9)"
        >
          authority
        </text>
        <rect
          x="0"
          y="10"
          width="90"
          height="14"
          rx="3"
          fill="var(--pluto-gray-l2)"
          stroke="var(--pluto-gray-l4)"
          strokeWidth="1"
        />
        <rect
          x="0"
          y="10"
          width={authBarWidth}
          height="14"
          rx="3"
          fill={isEmergency ? COLOR_EMERGENCY : COLOR_ACTIVE}
          opacity="0.7"
          className="authority-fill"
        />
        <text
          x="45"
          y="21"
          textAnchor="middle"
          className="diagram-badge"
          fill="var(--pluto-text-color)"
        >
          {state.authority}
        </text>
      </g>

      {/* Stage badge */}
      <rect
        x="185"
        y="90"
        width="90"
        height="24"
        rx="4"
        fill={isEmergency ? COLOR_EMERGENCY : "var(--pluto-gray-l2)"}
        fillOpacity={isEmergency ? 0.15 : 1}
        stroke={isEmergency ? COLOR_EMERGENCY : "var(--pluto-gray-l4)"}
        strokeWidth="1"
        className={isEmergency ? "emergency-border" : ""}
      />
      <text
        x="230"
        y="106"
        textAnchor="middle"
        className="diagram-badge"
        fill={isEmergency ? COLOR_EMERGENCY : "var(--pluto-text-color)"}
      >
        {state.stage}
      </text>

      {/* Safed indicator */}
      {state.stage === "safed" && (
        <g transform="translate(185, 128)">
          <rect
            x="0"
            y="0"
            width="90"
            height="24"
            rx="4"
            fill={COLOR_ACTIVE}
            fillOpacity="0.15"
            stroke={COLOR_ACTIVE}
            strokeWidth="1"
          />
          <text
            x="45"
            y="16"
            textAnchor="middle"
            className="diagram-badge"
            fill={COLOR_ACTIVE}
          >
            SYSTEM SAFE
          </text>
        </g>
      )}
    </svg>
  );
};
