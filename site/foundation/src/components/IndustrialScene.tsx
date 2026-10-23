// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useId } from "react";

import { IndustrialControlFlow } from "@/components/IndustrialControlFlow";
import {
  Box,
  CONCRETE,
  Cylinder,
  DARK,
  Face,
  LIGHT,
  METAL,
  type Point,
  project,
  SIGNAL,
  Wire,
} from "@/components/IndustrialGeometry";
import {
  FoundationNode as Node,
  InfrastructurePad as Site,
} from "@/components/IndustrialNodes";
import { IndustrialSoftware } from "@/components/IndustrialSoftware";

export type SystemLayer = "operation" | "connectivity" | "applications" | "control";

const Hall = (): ReactElement => (
  <g>
    <Box x={38} y={55} width={202} depth={130} height={45} />
    {Array.from({ length: 5 }, (_, i) => {
      const y = 55 + i * 26;
      return (
        <g key={y} className="scene-solid">
          <Face
            vertices={[
              [38, y, 45],
              [240, y, 45],
              [240, y + 20, 68],
              [38, y + 20, 68],
            ]}
            fill={LIGHT.top}
          />
          <Face
            vertices={[
              [38, y + 20, 68],
              [240, y + 20, 68],
              [240, y + 26, 45],
              [38, y + 26, 45],
            ]}
            fill={DARK.top}
          />
          <Face
            vertices={[
              [240, y, 45],
              [240, y + 20, 68],
              [240, y + 26, 45],
            ]}
            fill={CONCRETE.right}
          />
          {Array.from({ length: 5 }, (_, j) => (
            <Wire
              key={j}
              className="scene-detail"
              vertices={[
                [56 + j * 38, y, 45],
                [56 + j * 38, y + 20, 68],
              ]}
            />
          ))}
        </g>
      );
    })}
    {Array.from({ length: 7 }, (_, i) => (
      <Wire
        key={i}
        className="scene-detail"
        vertices={[
          [50 + i * 28, 185, 1],
          [50 + i * 28, 185, 44],
        ]}
      />
    ))}
    {[70, 119, 168].map((x) => (
      <g key={x}>
        <Face
          vertices={[
            [x, 185.5, 2],
            [x + 32, 185.5, 2],
            [x + 32, 185.5, 27],
            [x, 185.5, 27],
          ]}
          fill={DARK.right}
        />
        {[9, 16, 23].map((z) => (
          <Wire
            key={z}
            vertices={[
              [x, 186, z],
              [x + 32, 186, z],
            ]}
            className="scene-door"
          />
        ))}
      </g>
    ))}
    <Box x={15} y={80} width={15} depth={63} height={22} material={METAL} />
    <Box x={22} y={90} z={22} width={6} depth={12} height={49} material={DARK} />
    <Box x={22} y={118} z={22} width={6} depth={12} height={37} material={DARK} />
  </g>
);

const TankFarm = (): ReactElement => (
  <g>
    <Box x={309} y={24} width={155} depth={148} height={4} material={LIGHT} />
    {[
      [337, 57],
      [394, 57],
      [337, 120],
      [394, 120],
    ].map(([x, y]) => (
      <g key={`${x}-${y}`}>
        <Cylinder x={x} y={y} z={4} radius={22} height={64} />
        <Cylinder x={x} y={y} z={68} radius={7} height={5} dark />
        <Wire
          vertices={[
            [x + 23, y, 4],
            [x + 23, y, 68],
            [x + 9, y, 68],
          ]}
          className="scene-railing"
        />
        {Array.from({ length: 6 }, (_, i) => (
          <Wire
            key={i}
            vertices={[
              [x + 22, y, 10 + i * 10],
              [x + 27, y, 10 + i * 10],
            ]}
            className="scene-detail"
          />
        ))}
      </g>
    ))}
    {[36, 68, 100, 132].map((y) => (
      <Box key={y} x={440} y={y} width={7} depth={7} height={36} material={DARK} />
    ))}
    {[0, 1, 2].map((i) => (
      <g key={i}>
        <Wire
          vertices={[
            [324, 48 + i * 9, 26],
            [425 + i * 7, 48 + i * 9, 26],
            [425 + i * 7, 157, 26],
            [480, 157, 26],
            [480, 205, 26],
          ]}
          className="scene-pipe-shadow"
        />
        <Wire
          vertices={[
            [324, 48 + i * 9, 26],
            [425 + i * 7, 48 + i * 9, 26],
            [425 + i * 7, 157, 26],
            [480, 157, 26],
            [480, 205, 26],
          ]}
          className="scene-pipe"
        />
      </g>
    ))}
  </g>
);

const RemoteSite = (): ReactElement => (
  <g>
    <Box x={533} y={51} width={112} depth={112} height={4} material={LIGHT} />
    <Box x={550} y={65} z={4} width={76} depth={67} height={42} />
    <Box x={546} y={61} z={46} width={84} depth={75} height={4} material={LIGHT} />
    {Array.from({ length: 3 }, (_, row) =>
      Array.from({ length: 5 }, (_, col) => (
        <Face
          key={`${row}-${col}`}
          vertices={[
            [552 + col * 14, 70 + row * 19, 54],
            [564 + col * 14, 70 + row * 19, 54],
            [564 + col * 14, 85 + row * 19, 51],
            [552 + col * 14, 85 + row * 19, 51],
          ]}
          fill={DARK.top}
          className="scene-panel"
        />
      )),
    )}
    <Box x={557} y={133} z={4} width={15} depth={9} height={23} material={DARK} />
    <Box x={582} y={135} z={4} width={22} depth={14} height={13} material={METAL} />
    <Wire
      vertices={[
        [639, 79, 4],
        [639, 79, 95],
      ]}
      className="scene-mast"
    />
    <Wire
      vertices={[
        [630, 79, 81],
        [648, 79, 81],
        [644, 79, 86],
        [634, 79, 86],
      ]}
      className="scene-railing"
    />
    <Wire
      vertices={[
        [639, 79, 92],
        [651, 91, 68],
        [627, 67, 68],
        [639, 79, 92],
      ]}
      className="scene-railing"
    />
  </g>
);

const TestStand = (): ReactElement => (
  <g>
    <Box x={42} y={274} width={184} depth={132} height={5} material={LIGHT} />
    {[72, 139].map((x) => (
      <g key={x}>
        <Box x={x} y={292} z={5} width={42} depth={76} height={13} />
        <Box
          x={x + 7}
          y={300}
          z={18}
          width={28}
          depth={58}
          height={18}
          material={DARK}
        />
        <Cylinder x={x + 21} y={321} z={36} radius={12} height={13} />
        <Box
          x={x + 4}
          y={366}
          z={5}
          width={34}
          depth={19}
          height={38}
          material={METAL}
        />
        <Face
          vertices={[
            [x + 7, 385, 24],
            [x + 33, 385, 24],
            [x + 33, 385, 37],
            [x + 7, 385, 37],
          ]}
          fill={DARK.right}
        />
      </g>
    ))}
    {[55, 215].map((x) => (
      <g key={x}>
        <Box x={x} y={280} z={5} width={5} depth={5} height={78} material={DARK} />
        <Box x={x} y={390} z={5} width={5} depth={5} height={78} material={DARK} />
      </g>
    ))}
    <Box x={55} y={280} z={79} width={165} depth={6} height={5} material={METAL} />
    <Box x={55} y={390} z={79} width={165} depth={6} height={5} material={METAL} />
    <Box x={114} y={280} z={84} width={9} depth={116} height={5} material={METAL} />
    <Wire
      vertices={[
        [119, 328, 84],
        [119, 328, 49],
      ]}
      className="scene-railing"
    />
    <Box x={184} y={309} z={5} width={14} depth={32} height={32} material={LIGHT} />
    {[313, 322, 331].map((y) => (
      <Box
        key={y}
        x={198}
        y={y}
        z={21}
        width={1}
        depth={5}
        height={2}
        material={SIGNAL}
      />
    ))}
  </g>
);

const Warehouse = (): ReactElement => (
  <g>
    <Box x={318} y={283} width={143} depth={130} height={46} />
    <Box x={315} y={280} z={46} width={149} depth={136} height={4} material={LIGHT} />
    {[0, 1, 2, 3].map((i) => (
      <g key={i}>
        <Box
          x={337 + i * 29}
          y={299}
          z={50}
          width={17}
          depth={53}
          height={5}
          material={METAL}
        />
        <Face
          vertices={[
            [334 + i * 30, 413.5, 1],
            [353 + i * 30, 413.5, 1],
            [353 + i * 30, 413.5, 30],
            [334 + i * 30, 413.5, 30],
          ]}
          fill={DARK.top}
        />
        {[9, 18, 27].map((z) => (
          <Wire
            key={z}
            vertices={[
              [334 + i * 30, 414, z],
              [353 + i * 30, 414, z],
            ]}
            className="scene-door"
          />
        ))}
      </g>
    ))}
    <Box x={330} y={420} width={22} depth={40} height={21} material={LIGHT} />
    <Box x={331} y={456} width={20} depth={13} height={17} material={METAL} />
    <Box x={390} y={420} width={22} depth={33} height={21} material={LIGHT} />
    <Box x={391} y={450} width={20} depth={13} height={17} material={METAL} />
    <Box x={467} y={304} width={18} depth={19} height={11} material={METAL} />
    <Box x={467} y={329} width={18} depth={19} height={11} material={METAL} />
  </g>
);

const DataRoom = (): ReactElement => (
  <g>
    <Box x={535} y={298} width={104} depth={106} height={5} material={LIGHT} />
    <Box x={539} y={302} z={5} width={91} depth={7} height={59} material={CONCRETE} />
    <Box x={627} y={309} z={5} width={7} depth={89} height={59} material={CONCRETE} />
    {[0, 1, 2].map((row) =>
      [0, 1, 2, 3].map((col) => (
        <g key={`${row}-${col}`}>
          <Box
            x={548 + col * 18}
            y={319 + row * 25}
            z={5}
            width={13}
            depth={17}
            height={40}
            material={DARK}
          />
          {[15, 26, 37].map((z) => (
            <Wire
              key={z}
              vertices={[
                [550 + col * 18, 336 + row * 25, z],
                [559 + col * 18, 336 + row * 25, z],
              ]}
              className="scene-rack-slot"
            />
          ))}
        </g>
      )),
    )}
    <Wire
      vertices={[
        [539, 398, 64],
        [539, 302, 64],
        [634, 302, 64],
        [634, 398, 64],
        [539, 398, 64],
        [539, 398, 5],
      ]}
      className="scene-roof-outline"
    />
  </g>
);

// Every site has its own node. Connections share the same world coordinates as
// the equipment, so the raised network remains registered to the physical scene.
const NODES = [
  { x: 250, y: 205, label: "F.01" },
  { x: 477, y: 164, label: "F.02" },
  { x: 239, y: 393, label: "F.03" },
  { x: 481, y: 422, label: "F.04" },
  { x: 687, y: 153, label: "F.05" },
  { x: 704, y: 442, label: "F.06" },
] as const;
const LOCAL_PATHS: readonly Point[][] = [
  [
    [118, 187, 5],
    [118, 216, 5],
    [261, 216, 5],
  ],
  [
    [421, 168, 5],
    [421, 175, 5],
    [488, 175, 5],
  ],
  [
    [205, 328, 7],
    [250, 328, 7],
    [250, 404, 7],
  ],
  [
    [463, 370, 5],
    [492, 370, 5],
    [492, 433, 5],
  ],
  [
    [686, 97, 5],
    [698, 97, 5],
    [698, 164, 5],
  ],
  [
    [683, 414, 5],
    [715, 414, 5],
    [715, 453, 5],
  ],
];
const CLUSTER_PATHS: readonly Point[][] = [
  [
    [261, 216, 6],
    [488, 216, 6],
    [488, 175, 6],
  ],
  [
    [261, 216, 6],
    [261, 304, 6],
    [250, 304, 6],
    [250, 404, 6],
  ],
  [
    [250, 404, 6],
    [250, 450, 6],
    [492, 450, 6],
    [492, 433, 6],
  ],
  [
    [488, 175, 6],
    [548, 175, 6],
    [548, 164, 6],
    [698, 164, 6],
  ],
  [
    [488, 175, 6],
    [512, 175, 6],
    [512, 433, 6],
    [492, 433, 6],
  ],
  [
    [492, 433, 6],
    [560, 433, 6],
    [560, 453, 6],
    [715, 453, 6],
  ],
  [
    [698, 164, 6],
    [757, 164, 6],
    [757, 453, 6],
    [715, 453, 6],
  ],
];

const Trace = ({
  vertices,
  className = "",
}: {
  vertices: readonly Point[];
  className?: string;
}): ReactElement => (
  <g className={`scene-trace ${className}`}>
    <Wire vertices={vertices} className="scene-signal-underlay" />
    <Wire vertices={vertices} className="scene-signal-path" />
    <Wire vertices={vertices} className="scene-signal-packets" />
  </g>
);

const shift = (x: number, y: number): string =>
  `translate(${(x - y) * 1.1} ${(x + y) * 0.47})`;

// Leader lines end at the edge of a label's opaque backing. Nothing runs
// underneath the type, including when the network moves to its raised plane.
const Annotation = ({
  anchor,
  x,
  y,
  title,
  detail,
  width = 242,
  side = "left",
}: {
  anchor: Point;
  x: number;
  y: number;
  title: string;
  detail: string;
  width?: number;
  side?: "left" | "right";
}): ReactElement => {
  const [ax, ay] = project(anchor);
  const edgeX = side === "left" ? x + width : x;
  const bendX = side === "left" ? edgeX + 18 : edgeX - 18;
  return (
    <g className="scene-annotation">
      <path d={`M${ax} ${ay} L${bendX} ${y + 22} H${edgeX}`} />
      <circle cx={ax} cy={ay} r={2.5} />
      <rect x={x - 8} y={y - 5} width={width + 16} height={49} rx={4} />
      <text x={x} y={y + 12}>
        {title}
      </text>
      <text x={x} y={y + 32} className="scene-annotation-detail">
        {detail}
      </text>
    </g>
  );
};

const DESCRIPTIONS: Record<SystemLayer, string> = {
  operation:
    "Production equipment, test stands, process instrumentation, existing SCADA software, and remote facilities connect to six local Foundation nodes.",
  connectivity:
    "Six Foundation nodes rise above the dimmed equipment. The connected mesh joins a main facility, a remote site, and existing server infrastructure into one cluster.",
  applications:
    "The software surfaces and telemetry routes are emphasized. Applications, data platforms, models, and agents receive a shared live picture from the cluster.",
  control:
    "A supervisory command travels from an application through Foundation to production equipment. Rules, interlocks, and access control guard the route; an acknowledgment returns to the application.",
};

export const IndustrialScene = ({ layer }: { layer: SystemLayer }): ReactElement => {
  const titleId = useId();
  const descriptionId = useId();
  const lift = layer === "operation" ? 0 : 62;
  const appBus: readonly Point[] = [
    [337, -25, 172],
    [637, -25, 172],
    [637, 175, 172],
    [488, 175, 172],
    [488, 175, 14 + lift],
  ];
  return (
    <svg
      className="industrial-scene"
      data-layer={layer}
      viewBox="60 -20 1500 890"
      role="img"
      aria-labelledby={`${titleId} ${descriptionId}`}
    >
      <title id={titleId}>The operation, connected through Foundation</title>
      <desc id={descriptionId}>{DESCRIPTIONS[layer]}</desc>
      <g className="scene-datum">
        {Array.from({ length: 21 }, (_, i) => (
          <Wire
            key={`x-${i}`}
            vertices={[
              [-25 + i * 40, -30, -12],
              [-25 + i * 40, 535, -12],
            ]}
          />
        ))}
        {Array.from({ length: 15 }, (_, i) => (
          <Wire
            key={`y-${i}`}
            vertices={[
              [-25, -30 + i * 40, -12],
              [790, -30 + i * 40, -12],
            ]}
          />
        ))}
      </g>
      <Site x={0} y={0} width={530} depth={495} />
      <Site x={600} y={-10} width={178} depth={221} />
      <Site x={575} y={305} width={203} depth={212} />
      <g className="scene-campus-detail">
        {[
          [0, 198, 530, 60],
          [245, 0, 53, 495],
        ].map(([x, y, width, depth]) => (
          <Face
            key={`${x}-${y}`}
            vertices={[
              [x, y, 0],
              [x + width, y, 0],
              [x + width, y + depth, 0],
              [x, y + depth, 0],
            ]}
            fill="#1a2432"
          />
        ))}
        <Wire
          vertices={[
            [5, 249, 0],
            [526, 249, 0],
          ]}
          className="scene-road-line"
        />
        <Wire
          vertices={[
            [293, 12, 0],
            [293, 476, 0],
          ]}
          className="scene-road-line"
        />
        {[21, 27, 33, 39, 45, 51].map((x) => (
          <Wire
            key={x}
            vertices={[
              [x, 211, 0],
              [x, 233, 0],
            ]}
            className="scene-crosswalk"
          />
        ))}
        <Wire
          vertices={[
            [17, 469, 0],
            [221, 469, 0],
            [221, 437, 0],
            [17, 437, 0],
          ]}
          className="scene-parking"
        />
        {Array.from({ length: 11 }, (_, i) => (
          <Wire
            key={i}
            vertices={[
              [17 + i * 19, 437, 0],
              [17 + i * 19, 469, 0],
            ]}
            className="scene-parking"
          />
        ))}
        {[50, 91, 167, 186].map((x) => (
          <Box
            key={x}
            x={x}
            y={443}
            width={11}
            depth={20}
            height={5}
            material={x === 91 ? DARK : CONCRETE}
          />
        ))}
      </g>
      <g className="scene-physical scene-production">
        <Hall />
      </g>
      <g className="scene-physical">
        <TankFarm />
      </g>
      <g className="scene-physical">
        <TestStand />
      </g>
      <g className="scene-physical">
        <Warehouse />
      </g>
      <g className="scene-physical" transform={shift(100, -42)}>
        <RemoteSite />
      </g>
      <g className="scene-physical" transform={shift(70, 50)}>
        <DataRoom />
      </g>
      <g className="scene-local-links">
        {LOCAL_PATHS.map((vertices, i) => (
          <Trace key={i} vertices={vertices} />
        ))}
      </g>
      <g className="scene-node-risers">
        {NODES.map(({ x, y, label }) => (
          <Wire
            key={label}
            vertices={[
              [x + 11, y + 11, 0],
              [x + 11, y + 11, 62],
            ]}
            className="scene-layer-riser"
          />
        ))}
      </g>
      <g className="scene-network" style={{ transform: `translateY(-${lift}px)` }}>
        <g className="scene-cluster-links">
          {CLUSTER_PATHS.map((vertices, i) => (
            <Trace key={i} vertices={vertices} />
          ))}
        </g>
        {NODES.map((node) => (
          <Node key={node.label} {...node} />
        ))}
      </g>
      <g className="scene-software-links">
        <Trace vertices={appBus} />
        {[337, 482, 627].map((x) => (
          <Wire
            key={x}
            vertices={[
              [x, -42, 172],
              [x, -25, 172],
            ]}
            className="scene-signal-path"
          />
        ))}
      </g>
      <IndustrialSoftware />
      <IndustrialControlFlow />
      <g className="scene-annotations" key={layer}>
        {layer === "operation" && (
          <>
            <Annotation
              anchor={[100, 60, 68]}
              x={376}
              y={69}
              width={242}
              title="PRODUCTION LINE"
              detail="PLCs · equipment control"
            />
            <Annotation
              anchor={[75, 299, 82]}
              x={95}
              y={205}
              width={224}
              title="TEST OPERATIONS"
              detail="DAQs · instruments"
            />
            <Annotation
              anchor={[624, 365, 64]}
              x={1145}
              y={662}
              width={280}
              side="right"
              title="EXISTING SYSTEMS"
              detail="LabVIEW · SCADA · local services"
            />
          </>
        )}
        {layer === "connectivity" && (
          <>
            <Annotation
              anchor={[261, 216, 83]}
              x={302}
              y={126}
              width={218}
              title="LOCAL FOUNDATION NODES"
              detail="Alongside your equipment"
            />
            <Annotation
              anchor={[698, 164, 83]}
              x={1242}
              y={292}
              width={224}
              side="right"
              title="REMOTE SITE"
              detail="The same cluster"
            />
            <Annotation
              anchor={[715, 453, 83]}
              x={1115}
              y={680}
              width={266}
              side="right"
              title="YOUR INFRASTRUCTURE"
              detail="On premises · cloud · both"
            />
          </>
        )}
        {layer === "applications" && (
          <>
            <Annotation
              anchor={[637, 140, 172]}
              x={1250}
              y={378}
              width={232}
              side="right"
              title="UNIFIED TELEMETRY"
              detail="One live context"
            />
            <Annotation
              anchor={[715, 453, 83]}
              x={1108}
              y={682}
              width={266}
              side="right"
              title="ACROSS THE OPERATION"
              detail="Every connected site"
            />
          </>
        )}
        {layer === "control" && (
          <>
            <Annotation
              anchor={[118, 187, 5]}
              x={295}
              y={128}
              width={220}
              title="EQUIPMENT SETPOINT"
              detail="Command → acknowledgment"
            />
            <Annotation
              anchor={[488, 175, 83]}
              x={1139}
              y={384}
              width={300}
              side="right"
              title="SUPERVISORY CONTROL"
              detail="Rules · interlocks · access control"
            />
          </>
        )}
      </g>
    </svg>
  );
};
