// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useId } from "react";

const TICK_COUNT = 12;
const CYCLE_DURATION = 4;
const ROWS = 3;

interface FlashProps {
  attribute: string;
  /** The values at the start, the peak, the end of the flash, and the cycle end. */
  values: string;
  /** Seconds before the first flash. */
  begin: number;
}

/** Flash animates an attribute through a short flash at the start of each cycle. */
const Flash = ({ attribute, values, begin }: FlashProps): ReactElement => (
  <animate
    attributeName={attribute}
    values={values}
    keyTimes="0;0.04;0.1;1"
    dur={`${CYCLE_DURATION}s`}
    begin={`${begin}s`}
    repeatCount="indefinite"
  />
);

export function ControlSystemGraphic({
  className,
  size = 400,
}: {
  className?: string;
  size?: number;
}): ReactElement {
  const rawId = useId();
  const id = rawId.replace(/[^a-zA-Z0-9]/g, "");
  const w = size;
  const h = size * 0.75;
  const padX = 72;
  const trackWidth = w - padX * 2;
  const rowSpacing = h / (ROWS + 1);

  const rows = Array.from({ length: ROWS }, (_, rowIdx) => {
    const y = rowSpacing * (rowIdx + 1);
    const ticks = Array.from({ length: TICK_COUNT }, (_, i) => {
      const x = padX + (i / (TICK_COUNT - 1)) * trackWidth;
      return { x, y };
    });
    return { y, ticks, delay: rowIdx * 0.6 };
  });

  return (
    <svg
      width="100%"
      viewBox={`0 0 ${w} ${h}`}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
      role="img"
      aria-label="Deterministic timeline representing predictable real-time execution"
    >
      <defs>
        <radialGradient id={`${id}-glow`}>
          <stop offset="0%" stopColor="var(--pluto-gray-l8)" stopOpacity={0.4} />
          <stop offset="50%" stopColor="var(--pluto-gray-l6)" stopOpacity={0.12} />
          <stop offset="100%" stopColor="var(--pluto-gray-l4)" stopOpacity={0} />
        </radialGradient>
        <linearGradient id={`${id}-trail`} x1="0" y1="0" x2="1" y2="0">
          <stop offset="0%" stopColor="var(--pluto-gray-l6)" stopOpacity={0} />
          <stop offset="100%" stopColor="var(--pluto-gray-l7)" stopOpacity={0.5} />
        </linearGradient>
      </defs>

      {rows.map((row, rowIdx) => (
        <g key={rowIdx}>
          {/* Track line */}
          <line
            x1={padX}
            y1={row.y}
            x2={padX + trackWidth}
            y2={row.y}
            stroke="var(--pluto-gray-l4)"
            strokeWidth={1}
          />

          {/* Tick marks and flash dots */}
          {row.ticks.map((tick, i) => {
            const tickDelay =
              row.delay + (i / (TICK_COUNT - 1)) * (CYCLE_DURATION * 0.85);
            return (
              <g key={i}>
                {/* Tick mark */}
                <line
                  x1={tick.x}
                  y1={tick.y - 10}
                  x2={tick.x}
                  y2={tick.y + 10}
                  stroke="var(--pluto-gray-l4)"
                  strokeWidth={1}
                />
                {/* Flash dot at tick */}
                <circle
                  cx={tick.x}
                  cy={tick.y}
                  r={4}
                  fill="var(--pluto-gray-l9)"
                  opacity={0.2}
                >
                  <Flash attribute="opacity" values="0.2;1;0.2;0.2" begin={tickDelay} />
                  <Flash attribute="r" values="4;6.4;4;4" begin={tickDelay} />
                </circle>
                {/* Vertical drop line */}
                <line
                  x1={tick.x}
                  y1={tick.y + 10}
                  x2={tick.x}
                  y2={tick.y + rowSpacing * 0.65}
                  stroke="var(--pluto-gray-l5)"
                  strokeWidth={0.75}
                  opacity={0}
                >
                  <Flash attribute="opacity" values="0;0.3;0;0" begin={tickDelay} />
                </line>
              </g>
            );
          })}

          {/* Sweeping marker group */}
          <g>
            <animateTransform
              attributeName="transform"
              type="translate"
              values={`0;${trackWidth};0;0`}
              keyTimes="0;0.85;0.8501;1"
              dur={`${CYCLE_DURATION}s`}
              begin={`${row.delay}s`}
              repeatCount="indefinite"
            />
            {/* Glow halo */}
            <ellipse cx={padX} cy={row.y} rx={30} ry={24} fill={`url(#${id}-glow)`} />
            {/* Trail */}
            <rect
              x={padX - 60}
              y={row.y - 1.5}
              width={60}
              height={3}
              rx={1.5}
              fill={`url(#${id}-trail)`}
            />
            {/* Outer ring */}
            <circle
              cx={padX}
              cy={row.y}
              r={9}
              stroke="var(--pluto-gray-l6)"
              strokeWidth={1}
              opacity={0.35}
            />
            {/* Main dot */}
            <circle
              cx={padX}
              cy={row.y}
              r={5.5}
              fill="var(--pluto-gray-l9)"
              opacity={0.9}
            />
          </g>
        </g>
      ))}

      {/* Row labels */}
      {rows.map((row, i) => (
        <text
          key={`label-${i}`}
          x={padX - 12}
          y={row.y + 1}
          textAnchor="end"
          fill="var(--pluto-gray-l6)"
          dominantBaseline="middle"
          style={{ fontFamily: "var(--pluto-code-font-family)", fontSize: 13 }}
        >
          {`Auto ${i}`}
        </text>
      ))}
    </svg>
  );
}
