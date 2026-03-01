import type { ReactElement } from "react";

import { manhattanPath, manhattanMidpoint } from "@/components/stream/diagrams/routing";
import {
  EXCLUDED_MARKER_COLOR,
  PARTICLE_COLOR,
  PARTICLE_COUNT,
  resolveTraceColor,
} from "@/components/stream/diagrams/theme";

const TRANSITION = "stroke 0.3s ease";

interface TraceProps {
  pathId: string;
  x1: number;
  y1: number;
  x2: number;
  y2: number;
  flowing: boolean;
  excluded: boolean;
}

export const Trace = ({
  pathId,
  x1,
  y1,
  x2,
  y2,
  flowing,
  excluded,
}: TraceProps): ReactElement => {
  const d = manhattanPath(x1, y1, x2, y2);
  const color = resolveTraceColor(flowing, excluded);
  const mid = manhattanMidpoint(x1, y1, x2, y2);
  return (
    <g>
      <path
        id={pathId}
        d={d}
        fill="none"
        stroke={color}
        strokeWidth={2}
        strokeLinecap="round"
        style={{ transition: TRANSITION }}
      />
      {flowing &&
        !excluded &&
        Array.from({ length: PARTICLE_COUNT }).map((_, i) => {
          const dur = 2.2 + i * 0.15;
          const delay = (i / PARTICLE_COUNT) * dur;
          return (
            <circle key={i} r={2} fill={PARTICLE_COLOR}>
              <animateMotion
                dur={`${dur}s`}
                begin={`${delay}s`}
                repeatCount="indefinite"
              >
                <mpath href={`#${pathId}`} />
              </animateMotion>
              <animate
                attributeName="fill-opacity"
                values="0;0.8;0.8;0"
                keyTimes="0;0.08;0.88;1"
                dur={`${dur}s`}
                begin={`${delay}s`}
                repeatCount="indefinite"
              />
            </circle>
          );
        })}
      {excluded && (
        <g transform={`translate(${mid.x},${mid.y})`}>
          <line
            x1={-4}
            y1={-4}
            x2={4}
            y2={4}
            stroke={EXCLUDED_MARKER_COLOR}
            strokeWidth={2}
            strokeLinecap="round"
          />
          <line
            x1={4}
            y1={-4}
            x2={-4}
            y2={4}
            stroke={EXCLUDED_MARKER_COLOR}
            strokeWidth={2}
            strokeLinecap="round"
          />
        </g>
      )}
    </g>
  );
};
