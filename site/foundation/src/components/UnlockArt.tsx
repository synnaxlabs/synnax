// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ReactElement, ReactNode } from "react";

export type UnlockKind =
  | "predictive"
  | "anomaly"
  | "root-cause"
  | "fleet"
  | "quality"
  | "tests"
  | "optimization"
  | "intelligence"
  | "automation";

type Point = readonly [number, number, number];

const SLATE = "#1d2b3c";
const EDGE = "#456079";
const LIGHT = "#86a9ce";
const INK = "#b8cee2";

// One shallow orthographic camera gives the studies a common material language.
const project = ([x, y, z]: Point): readonly [number, number] => [
  66 + 0.99 * x + 0.66 * y,
  180 + 0.27 * x - 0.48 * y - z,
];
const coordinate = (p: Point): string =>
  project(p)
    .map((n) => n.toFixed(2))
    .join(",");
const path = (points: readonly Point[]): string =>
  points.map((p, i) => `${i === 0 ? "M" : "L"}${coordinate(p)}`).join(" ");

const Line = ({
  points,
  color = EDGE,
  width = 1.2,
  className,
  dashed = false,
}: {
  points: readonly Point[];
  color?: string;
  width?: number;
  className?: string;
  dashed?: boolean;
}): ReactElement => (
  <path
    d={path(points)}
    fill="none"
    stroke={color}
    strokeWidth={width}
    strokeLinejoin="round"
    strokeDasharray={dashed ? "3 5" : undefined}
    pathLength={className?.includes("unlock-art-trace") ? 1 : undefined}
    className={className}
  />
);

const Face = ({
  points,
  fill = SLATE,
  stroke = EDGE,
  className,
}: {
  points: readonly Point[];
  fill?: string;
  stroke?: string;
  className?: string;
}): ReactElement => (
  <polygon
    points={points.map(coordinate).join(" ")}
    fill={fill}
    stroke={stroke}
    strokeWidth="0.85"
    strokeLinejoin="round"
    className={className}
  />
);

const Plane = ({
  x = 0,
  y = 0,
  w = 390,
  d = 150,
  z = 0,
  fill = SLATE,
  children,
  className,
}: {
  x?: number;
  y?: number;
  w?: number;
  d?: number;
  z?: number;
  fill?: string;
  children?: ReactNode;
  className?: string;
}): ReactElement => (
  <g className={className}>
    <Face
      points={[
        [x, y, z - 4],
        [x + w, y, z - 4],
        [x + w, y, z],
        [x, y, z],
      ]}
      fill="#111b28"
      stroke="#2b3d51"
    />
    <Face
      points={[
        [x, y, z],
        [x + w, y, z],
        [x + w, y + d, z],
        [x, y + d, z],
      ]}
      fill={fill}
    />
    {children}
  </g>
);

const Mark = ({ point, size = 3 }: { point: Point; size?: number }): ReactElement => {
  const [x, y, z] = point;
  return (
    <Face
      points={[
        [x - size, y - size, z],
        [x + size, y - size, z],
        [x + size, y + size, z],
        [x - size, y + size, z],
      ]}
      fill={INK}
      stroke={INK}
    />
  );
};

const samples = (
  x: number,
  w: number,
  y: number,
  height: (t: number) => number,
  count = 50,
): Point[] =>
  Array.from({ length: count + 1 }, (_, i) => [
    x + (i / count) * w,
    y,
    height(i / count),
  ]);

const Ribbon = ({
  points,
  base = 0,
  color = LIGHT,
  className,
}: {
  points: Point[];
  base?: number;
  color?: string;
  className?: string;
}): ReactElement => (
  <g className={className}>
    <Face
      points={[
        [points[0][0], points[0][1], base],
        ...points,
        [points[points.length - 1][0], points[points.length - 1][1], base],
      ]}
      fill="#223449"
      stroke="#344d67"
    />
    <Line points={points} color={color} width={1.5} />
  </g>
);

const Predictive = (): ReactElement => (
  <>
    <Plane y={4} w={390} d={150} fill="#152131" />
    {[130, 104, 78, 52, 26].map((y, i) => {
      const z = (t: number): number =>
        10 + 18 * t + 34 * t ** 3 + Math.sin(t * 22 + i * 0.3) * 3;
      return (
        <g key={y}>
          <Ribbon points={samples(12, 270, y, (t) => z(t * 0.76))} />
          <Line
            points={samples(282, 84, y, (t) => z(0.76 + t * 0.24))}
            color={LIGHT}
            dashed
          />
          <Line
            points={samples(282, 84, y, (t) => z(0.76 + t * 0.24))}
            color={INK}
            width={2}
            className="unlock-art-trace"
          />
        </g>
      );
    })}
    <g className="unlock-art-reveal">
      <Face
        points={[
          [332, 12, 0],
          [356, 12, 0],
          [356, 144, 0],
          [332, 144, 0],
        ]}
        fill="#2b3d51"
        stroke={LIGHT}
      />
      <Line
        points={[
          [344, 16, 60],
          [344, 145, 60],
        ]}
        color={INK}
        dashed
      />
      <Line
        points={[
          [344, 16, 0],
          [344, 16, 66],
        ]}
        color={LIGHT}
      />
    </g>
  </>
);

const Anomaly = (): ReactElement => (
  <>
    <Plane w={380} d={160} fill="#152131" />
    {[140, 112, 84, 56, 28].map((y, i) => (
      <Ribbon
        key={y}
        points={samples(
          10,
          355,
          y,
          (t) =>
            17 +
            Math.sin(t * 19) * 7 +
            (i === 2 ? 54 * Math.exp(-(((t - 0.6) / 0.065) ** 2)) : 0),
        )}
        color={i === 2 ? INK : EDGE}
        className={i === 2 ? undefined : "unlock-art-muted"}
      />
    ))}
    <g className="unlock-art-reveal">
      <Line
        points={[
          [223, 55, 0],
          [223, 55, 90],
          [223, 110, 90],
          [223, 110, 0],
        ]}
        color={LIGHT}
      />
      <Mark point={[223, 84, 64]} size={4} />
    </g>
  </>
);

const RootCause = (): ReactElement => {
  const branches: Point[][] = [
    [
      [20, 24, 18],
      [128, 24, 18],
      [182, 76, 35],
    ],
    [
      [20, 76, 18],
      [128, 76, 18],
      [182, 76, 35],
    ],
    [
      [20, 128, 18],
      [128, 128, 18],
      [182, 76, 35],
    ],
    [
      [182, 76, 35],
      [244, 76, 35],
      [292, 32, 54],
      [365, 32, 54],
    ],
    [
      [182, 76, 35],
      [244, 76, 35],
      [292, 119, 54],
      [365, 119, 54],
    ],
  ];
  return (
    <>
      <Plane x={8} y={6} w={112} d={143} z={18} fill="#152131" />
      <Plane x={148} y={31} w={102} d={92} z={35} />
      <Plane x={288} y={6} w={96} d={143} z={54} fill="#223449" />
      {branches.map((points, i) => (
        <Line key={i} points={points} color={EDGE} />
      ))}
      {[24, 76, 128].map((y) => (
        <Mark key={y} point={[37, y, 19]} />
      ))}
      <Mark point={[182, 76, 36]} size={6} />
      <Line
        points={[
          [365, 119, 55],
          [292, 119, 55],
          [244, 76, 36],
          [182, 76, 36],
          [128, 76, 19],
          [37, 76, 19],
        ]}
        color={INK}
        width={2}
        className="unlock-art-trace"
      />
      <g className="unlock-art-reveal">
        <Line
          points={[
            [164, 60, 37],
            [200, 60, 37],
            [200, 94, 37],
            [164, 94, 37],
            [164, 60, 37],
          ]}
          color={LIGHT}
        />
      </g>
    </>
  );
};

const Fleet = (): ReactElement => (
  <>
    {[2, 1, 0].flatMap((row) =>
      [0, 1, 2].map((col) => {
        const x = col * 128;
        const y = row * 64;
        return (
          <Plane key={`${row}-${col}`} x={x} y={y} w={111} d={47} z={row * 5}>
            <Line
              points={samples(
                x + 9,
                92,
                y + 22,
                (t) =>
                  row * 5 + 6 + 18 * Math.exp(-(((t - 0.43 - col * 0.035) / 0.2) ** 2)),
              )}
              color={LIGHT}
              width={1.6}
            />
            <Line
              points={[
                [x + 9, y + 10, row * 5 + 1],
                [x + 101, y + 10, row * 5 + 1],
              ]}
              color="#2b3d51"
            />
            <Line
              points={[
                [x + 58, y + 2, row * 5 + 1],
                [x + 58, y + 43, row * 5 + 1],
              ]}
              color={INK}
              className="unlock-art-reveal"
            />
          </Plane>
        );
      }),
    )}
    <Line
      points={[
        [58, 0, 34],
        [58, 177, 49],
      ]}
      color={INK}
      dashed
      className="unlock-art-reveal"
    />
    <Line
      points={[
        [186, 0, 34],
        [186, 177, 49],
      ]}
      color={INK}
      dashed
      className="unlock-art-reveal"
    />
    <Line
      points={[
        [314, 0, 34],
        [314, 177, 49],
      ]}
      color={INK}
      dashed
      className="unlock-art-reveal"
    />
  </>
);

const Quality = (): ReactElement => (
  <>
    <Plane x={9} y={20} w={370} d={120} fill="#111b28" />
    {[330, 270, 210, 150, 90, 30].map((x, i) => (
      <g key={x}>
        <Face
          points={[
            [x, 23, 1],
            [x, 136, 1],
            [x, 136, 68],
            [x, 23, 68],
          ]}
          fill={i % 2 === 0 ? "#1d2b3c" : "#172332"}
        />
        <Line
          points={[
            [x, 32, 9],
            [x, 32, 59],
            [x, 127, 59],
          ]}
          color="#344d67"
        />
        <Line
          points={[
            [x, 49, 25],
            [x, 103, 25],
          ]}
          color={EDGE}
        />
        <Mark point={[x, 76, 34]} size={3.5} />
      </g>
    ))}
    <Line
      points={[
        [7, 76, 34],
        [375, 76, 34],
      ]}
      color={LIGHT}
      width={1.6}
    />
    <Line
      points={[
        [7, 76, 34],
        [375, 76, 34],
      ]}
      color={INK}
      width={2.3}
      className="unlock-art-trace"
    />
    <g className="unlock-art-reveal">
      {[30, 90, 150, 210, 270, 330].map((x) => (
        <Line
          key={x}
          points={[
            [x, 76, 34],
            [x, 76, 90],
          ]}
          color={LIGHT}
          dashed
        />
      ))}
    </g>
  </>
);

const Tests = (): ReactElement => (
  <>
    {[2, 1, 0].map((i) => {
      const y = 15 + i * 59;
      const z = i * 13;
      return (
        <Plane
          key={i}
          x={12}
          y={y}
          w={362}
          d={43}
          z={z}
          fill={i === 1 ? "#223449" : SLATE}
        >
          <Line
            points={[
              [23, y + 9, z + 1],
              [362, y + 9, z + 1],
            ]}
            color="#344d67"
          />
          <Ribbon
            points={samples(
              24,
              337,
              y + 24,
              (t) =>
                z +
                5 +
                (t > 0.32
                  ? Math.exp(-(t - 0.32) * 4) *
                    (22 + i * 6) *
                    (1 + 0.22 * Math.sin((t - 0.32) * 38))
                  : 0),
            )}
            base={z}
            color={i === 1 ? INK : LIGHT}
          />
        </Plane>
      );
    })}
    <Line
      points={[
        [132, 0, 36],
        [132, 183, 76],
      ]}
      color={INK}
      width={1.5}
    />
    <Line
      points={[
        [132, 0, 0],
        [132, 0, 55],
      ]}
      color={LIGHT}
    />
    <g className="unlock-art-reveal">
      <Line
        points={[
          [260, 0, 36],
          [260, 183, 76],
        ]}
        color={LIGHT}
        dashed
      />
      <Line
        points={[
          [132, 0, 55],
          [260, 0, 55],
        ]}
        color={INK}
      />
      <Line
        points={[
          [260, 0, 49],
          [260, 0, 61],
        ]}
        color={INK}
      />
    </g>
  </>
);

const Optimization = (): ReactElement => (
  <>
    <Plane x={8} w={376} d={165} fill="#152131" />
    {[0, 1, 2, 3, 4, 5, 6].map((i) => {
      const points: Point[] = Array.from({ length: 81 }, (_, j) => {
        const angle = (j / 80) * Math.PI * 2;
        const r = 21 + i * 16;
        return [
          194 + Math.cos(angle) * r * 1.4,
          80 + Math.sin(angle) * r * 0.64,
          4 + (6 - i) * 4 + Math.cos(angle * 2) * 3,
        ];
      });
      return (
        <Line
          key={i}
          points={points}
          color={i === 0 ? LIGHT : EDGE}
          width={i === 0 ? 1.6 : 1}
        />
      );
    })}
    <Line
      points={[
        [40, 23, 5],
        [112, 32, 14],
        [143, 68, 22],
        [194, 80, 31],
      ]}
      color={LIGHT}
      width={1.5}
    />
    <Line
      points={[
        [40, 23, 5],
        [112, 32, 14],
        [143, 68, 22],
        [194, 80, 31],
      ]}
      color={INK}
      width={2.2}
      className="unlock-art-trace"
    />
    <Mark point={[194, 80, 31]} size={4} />
    <g className="unlock-art-reveal">
      <Line
        points={[
          [194, 80, 31],
          [194, 80, 91],
        ]}
        color={LIGHT}
        dashed
      />
      <Mark point={[194, 80, 91]} size={3} />
    </g>
  </>
);

const Intelligence = (): ReactElement => (
  <>
    <Plane x={4} y={4} w={137} d={149} fill="#152131" />
    {[18, 42, 66, 90, 114, 138].map((y, i) => (
      <Line
        key={y}
        points={samples(
          13,
          113,
          y,
          (t) => 8 + 10 * Math.sin(t * (11 + i) + i) * Math.sin(t * Math.PI),
        )}
        color={i % 2 === 0 ? LIGHT : EDGE}
      />
    ))}
    <Plane x={177} y={24} w={44} d={107} z={25} fill="#2b3d51" />
    {[18, 42, 66, 90, 114, 138].map((y) => (
      <Line
        key={y}
        points={[
          [127, y, 8],
          [176, 76, 25],
          [222, 76, 25],
        ]}
        color={EDGE}
      />
    ))}
    {[2, 1, 0].map((i) => (
      <Plane
        key={i}
        x={260}
        y={12 + i * 57}
        w={121}
        d={39}
        z={35}
        fill={i === 1 ? "#2b3d51" : SLATE}
      >
        {[0, 1, 2, 3, 4].map((j) => (
          <Face
            key={j}
            points={[
              [272 + j * 20, 21 + i * 57, 36],
              [283 + j * 20, 21 + i * 57, 36],
              [283 + j * 20, 34 + i * 57, 36],
              [272 + j * 20, 34 + i * 57, 36],
            ]}
            fill={j <= i + 1 ? "#456079" : "#223449"}
            stroke="none"
          />
        ))}
      </Plane>
    ))}
    {[31, 88, 145].map((y) => (
      <Line
        key={y}
        points={[
          [222, 76, 25],
          [246, y, 35],
          [267, y, 35],
        ]}
        color={LIGHT}
      />
    ))}
    <Line
      points={[
        [13, 66, 8],
        [127, 66, 8],
        [176, 76, 25],
        [222, 76, 25],
        [246, 88, 35],
        [367, 88, 35],
      ]}
      color={INK}
      width={2}
      className="unlock-art-trace"
    />
  </>
);

const Automation = (): ReactElement => (
  <>
    <Plane x={12} y={4} w={365} d={152} fill="#152131" />
    {[306, 211, 116].map((x, i) => (
      <g key={x} className={i === 1 ? "unlock-art-shift" : undefined}>
        <Face
          points={[
            [x, 22, 0],
            [x, 138, 0],
            [x, 138, 78],
            [x, 22, 78],
            [x, 22, 0],
            [x, 43, 0],
            [x, 43, 57],
            [x, 117, 57],
            [x, 117, 0],
            [x, 138, 0],
          ]}
          fill={i === 1 ? "#2b3d51" : SLATE}
          stroke={i === 1 ? LIGHT : EDGE}
        />
        <Line
          points={[
            [x, 22, 82],
            [x, 138, 82],
          ]}
          color={EDGE}
        />
      </g>
    ))}
    <Line
      points={[
        [25, 96, 25],
        [360, 96, 25],
      ]}
      color={LIGHT}
      width={1.6}
    />
    <Line
      points={[
        [360, 96, 25],
        [360, 40, 12],
        [25, 40, 12],
      ]}
      color={EDGE}
      width={1.4}
    />
    <Line
      points={[
        [25, 96, 25],
        [360, 96, 25],
        [360, 40, 12],
        [25, 40, 12],
      ]}
      color={INK}
      width={2}
      className="unlock-art-trace"
    />
    <Mark point={[25, 96, 25]} size={4} />
    <Mark point={[360, 96, 25]} size={4} />
    <g className="unlock-art-reveal">
      <Mark point={[25, 40, 12]} size={4} />
      <Line
        points={[
          [29, 40, 12],
          [43, 40, 12],
        ]}
        color={INK}
        width={2}
      />
    </g>
  </>
);

const ART: Record<UnlockKind, () => ReactElement> = {
  predictive: Predictive,
  anomaly: Anomaly,
  "root-cause": RootCause,
  fleet: Fleet,
  quality: Quality,
  tests: Tests,
  optimization: Optimization,
  intelligence: Intelligence,
  automation: Automation,
};

export const UnlockArt = ({ kind }: { kind: UnlockKind }): ReactElement => {
  const Geometry = ART[kind];
  return (
    <svg
      className={`unlock-art unlock-art--${kind}`}
      viewBox="0 0 600 340"
      preserveAspectRatio="xMidYMid meet"
      aria-hidden="true"
      focusable="false"
    >
      <Geometry />
    </svg>
  );
};
