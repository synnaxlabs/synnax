// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import {
  Box,
  CONCRETE,
  Cylinder,
  DARK,
  Face,
  LIGHT,
  METAL,
  type Point,
  Wire,
} from "@/components/IndustrialGeometry";

import { Label, Node, Pad, Rack, Trace } from "./DeploymentGeometry";

// A faceted surface of revolution keeps the vehicle and engine bells in the same
// orthographic material language as the rest of the physical equipment.
const Taper = ({
  x,
  y,
  z,
  height,
  bottomRadius,
  topRadius,
  dark = false,
}: {
  x: number;
  y: number;
  z: number;
  height: number;
  bottomRadius: number;
  topRadius: number;
  dark?: boolean;
}): ReactElement => {
  const ring = (radius: number, elevation: number): Point[] =>
    Array.from({ length: 32 }, (_, i) => [
      x + Math.cos((i * Math.PI) / 16) * radius,
      y + Math.sin((i * Math.PI) / 16) * radius,
      elevation,
    ]);
  const bottom = ring(bottomRadius, z);
  const top = ring(topRadius, z + height);
  return (
    <g className="scene-cylinder">
      {Array.from({ length: 16 }, (_, i) => {
        const n = (i + 28) % 32;
        const next = (n + 1) % 32;
        const shade = Math.round((dark ? 35 : 73) + 24 * Math.cos((i / 16) * Math.PI));
        return (
          <Face
            key={i}
            vertices={[bottom[n], bottom[next], top[next], top[n]]}
            fill={`rgb(${shade} ${shade + 15} ${shade + 33})`}
          />
        );
      })}
      {topRadius > 0 && <Face vertices={top} fill={dark ? DARK.top : LIGHT.top} />}
    </g>
  );
};

const Pipe = ({ vertices }: { vertices: readonly Point[] }): ReactElement => (
  <g>
    <Wire vertices={vertices} className="scene-pipe-shadow" />
    <Wire vertices={vertices} className="scene-pipe" />
  </g>
);

const ServiceTower = (): ReactElement => (
  <g>
    <Box x={63} y={58} width={60} depth={57} height={12} />
    {[74, 108].map((x) =>
      [68, 102].map((y) => (
        <Box
          key={`${x}-${y}`}
          x={x}
          y={y}
          z={12}
          width={5}
          depth={5}
          height={208}
          material={METAL}
        />
      )),
    )}
    {[17, 55, 93, 131, 169, 207].map((z) => (
      <g key={z}>
        <Box x={71} y={65} z={z} width={45} depth={45} height={3} material={METAL} />
        {z < 207 && (
          <>
            <Wire
              vertices={[
                [77, 108, z + 3],
                [110, 108, z + 38],
                [77, 108, z + 38],
                [110, 108, z + 3],
              ]}
              className="scene-railing"
            />
            <Wire
              vertices={[
                [114, 70, z + 3],
                [114, 103, z + 38],
                [114, 70, z + 38],
                [114, 103, z + 3],
              ]}
              className="scene-railing"
            />
          </>
        )}
      </g>
    ))}
    <Box x={82} y={77} z={12} width={10} depth={14} height={192} material={DARK} />
    {[94, 170].map((z) => (
      <g key={z}>
        <Box x={111} y={94} z={z} width={40} depth={10} height={4} material={LIGHT} />
        <Wire
          vertices={[
            [111, 105, z + 13],
            [151, 105, z + 13],
            [151, 105, z + 4],
          ]}
          className="scene-railing"
        />
        <Wire
          vertices={[
            [112, 101, z - 15],
            [147, 101, z],
          ]}
          className="scene-railing"
        />
        <Pipe
          vertices={[
            [113, 96, z + 6],
            [147, 96, z + 6],
            [147, 109, z + 6],
          ]}
        />
      </g>
    ))}
    <Wire
      vertices={[
        [78, 69, 220],
        [78, 69, 240],
      ]}
      className="scene-mast"
    />
  </g>
);

const LaunchComplex = (): ReactElement => (
  <g className="scene-physical">
    <Box x={97} y={70} width={119} depth={157} height={3} material={DARK} />
    <Face
      vertices={[
        [140, 101, 4],
        [178, 101, 4],
        [178, 244, 4],
        [140, 244, 4],
      ]}
      fill="#080f19"
    />
    <Box x={105} y={82} width={35} depth={145} height={15} />
    <Box x={178} y={82} width={30} depth={145} height={15} />
    <Box x={105} y={72} width={103} depth={20} height={22} />
    <Face
      vertices={[
        [140, 126, 25],
        [178, 126, 25],
        [178, 184, 4],
        [140, 184, 4],
      ]}
      fill={METAL.right}
      className="scene-panel"
    />
    <ServiceTower />
    {[130, 178].map((x) => (
      <Box
        key={x}
        x={x}
        y={99}
        z={15}
        width={9}
        depth={26}
        height={19}
        material={DARK}
      />
    ))}
    <Cylinder x={159} y={109} z={31} radius={28} height={7} dark />
    <Taper x={159} y={109} z={38} height={14} bottomRadius={9} topRadius={5} dark />
    <Taper x={159} y={109} z={49} height={11} bottomRadius={11} topRadius={14} />
    <Cylinder x={159} y={109} z={60} radius={14} height={104} />
    <Cylinder x={159} y={109} z={164} radius={14} height={7} dark />
    <Cylinder x={159} y={109} z={171} radius={14} height={36} />
    <Taper x={159} y={109} z={207} height={7} bottomRadius={14} topRadius={16} />
    <Cylinder x={159} y={109} z={214} radius={16} height={9} />
    <Taper x={159} y={109} z={223} height={23} bottomRadius={16} topRadius={0} />
    <Wire
      vertices={[
        [169, 119, 63],
        [169, 119, 159],
      ]}
      className="scene-detail"
    />
    {[143, 171].map((x) => (
      <Wire
        key={x}
        vertices={[
          [x, 125, 36],
          [x, 125, 49],
          [159 + (x - 159) * 0.7, 117, 54],
        ]}
        className="scene-railing"
      />
    ))}
    <Box x={241} y={103} width={26} depth={43} height={33} material={METAL} />
    {[13, 20, 27].map((z) => (
      <Wire
        key={z}
        vertices={[
          [245, 147, z],
          [263, 147, z],
        ]}
        className="scene-rack-slot"
      />
    ))}
    <Pipe
      vertices={[
        [247, 105, 12],
        [225, 105, 12],
        [225, 69, 12],
        [119, 69, 12],
        [119, 86, 12],
      ]}
    />
    <Wire
      vertices={[
        [48, 233, 0],
        [114, 233, 0],
      ]}
      className="scene-road-line"
    />
  </g>
);

const EngineStand = (): ReactElement => (
  <g>
    <Box x={469} y={74} width={97} depth={105} height={5} material={CONCRETE} />
    <Face
      vertices={[
        [498, 100, 6],
        [538, 100, 6],
        [538, 219, 6],
        [498, 219, 6],
      ]}
      fill="#080f19"
    />
    {[477, 549].map((x) => (
      <Box
        key={x}
        x={x}
        y={82}
        z={5}
        width={8}
        depth={8}
        height={104}
        material={METAL}
      />
    ))}
    <Box x={477} y={82} z={96} width={80} depth={8} height={11} material={LIGHT} />
    <Box x={477} y={82} z={104} width={80} depth={72} height={5} material={METAL} />
    <Cylinder x={517} y={116} z={79} radius={10} height={24} />
    <Taper x={517} y={116} z={47} height={32} bottomRadius={19} topRadius={8} dark />
    <Cylinder x={517} y={116} z={44} radius={19} height={3} dark />
    <Wire
      vertices={[
        [507, 116, 97],
        [500, 116, 87],
        [500, 116, 76],
      ]}
      className="scene-pipe"
    />
    {[477, 549].map((x) => (
      <g key={x}>
        <Box x={x} y={146} z={5} width={8} depth={8} height={104} material={METAL} />
        <Wire
          vertices={[
            [x + 5, 88, 14],
            [x + 5, 147, 95],
            [x + 5, 88, 95],
            [x + 5, 147, 14],
          ]}
          className="scene-railing"
        />
      </g>
    ))}
    <Box x={477} y={146} z={96} width={80} depth={8} height={11} material={LIGHT} />
    <Wire
      vertices={[
        [477, 155, 119],
        [558, 155, 119],
        [558, 82, 119],
      ]}
      className="scene-railing"
    />
    {[479, 505, 531, 557].map((x) => (
      <Wire
        key={x}
        vertices={[
          [x, 155, 109],
          [x, 155, 119],
        ]}
        className="scene-railing"
      />
    ))}
    <Box x={500} y={93} z={109} width={30} depth={26} height={16} material={DARK} />
    <Box x={565} y={154} z={5} width={21} depth={28} height={27} material={METAL} />
    {[15, 23].map((z) => (
      <Wire
        key={z}
        vertices={[
          [569, 183, z],
          [582, 183, z],
        ]}
        className="scene-rack-slot"
      />
    ))}
    <Face
      vertices={[
        [498, 136, 31],
        [538, 136, 31],
        [538, 189, 6],
        [498, 189, 6],
      ]}
      fill={CONCRETE.right}
      className="scene-panel"
    />
  </g>
);

const PropellantSystem = (): ReactElement => (
  <g>
    {[
      [668, 83, 76],
      [711, 141, 59],
    ].map(([x, y, height]) => (
      <g key={x}>
        <Cylinder x={x} y={y} radius={24} height={5} dark />
        <Cylinder x={x} y={y} z={5} radius={20} height={height} />
        <Taper x={x} y={y} z={height + 5} height={6} bottomRadius={20} topRadius={13} />
        <Cylinder x={x} y={y} z={height + 11} radius={6} height={4} dark />
        <Wire
          vertices={[
            [x + 22, y, 5],
            [x + 22, y, height + 11],
            [x + 12, y, height + 11],
          ]}
          className="scene-railing"
        />
        {Array.from({ length: 7 }, (_, i) => (
          <Wire
            key={i}
            vertices={[
              [x + 20, y, 10 + i * 8],
              [x + 25, y, 10 + i * 8],
            ]}
            className="scene-detail"
          />
        ))}
      </g>
    ))}
    {[
      [668, 83, 0],
      [711, 141, 8],
    ].map(([x, y, offset]) => (
      <g key={x}>
        <Pipe
          vertices={[
            [x - 19, y, 19],
            [615 + offset, y, 19],
            [615 + offset, 191 + offset, 19],
            [557 + offset, 191 + offset, 19],
            [557 + offset, 132 + offset, 19],
            [557 + offset, 132 + offset, 85],
            [524, 132 + offset, 85],
          ]}
        />
        <Box x={609 + offset} y={170} width={8} depth={8} height={18} material={DARK} />
        <Box
          x={612 + offset}
          y={152}
          z={16}
          width={7}
          depth={8}
          height={6}
          material={LIGHT}
        />
      </g>
    ))}
    <Box x={690} y={184} width={21} depth={15} height={22} material={METAL} />
  </g>
);

const AvionicsLab = (): ReactElement => (
  <g className="scene-physical">
    <Box x={145} y={329} width={270} depth={150} height={5} material={LIGHT} />
    <Box x={145} y={329} z={5} width={270} depth={6} height={73} />
    <Box x={145} y={335} z={5} width={6} depth={137} height={73} />
    <Wire
      vertices={[
        [154, 336, 49],
        [397, 336, 49],
        [397, 358, 49],
      ]}
      className="scene-pipe"
    />
    {[169, 215, 261, 307].map((x) => (
      <Rack key={x} x={x} y={347} z={5} width={29} depth={30} height={64} />
    ))}
    <Rack x={366} y={356} z={5} width={28} depth={34} height={69} />
    <Wire
      vertices={[
        [166, 393, 6],
        [402, 393, 6],
        [402, 453, 6],
      ]}
      className="scene-detail"
    />
    {[178, 340].map((x) =>
      [421, 447].map((y) => (
        <Box
          key={`${x}-${y}`}
          x={x}
          y={y}
          z={5}
          width={4}
          depth={4}
          height={27}
          material={METAL}
        />
      )),
    )}
    <Box x={174} y={416} z={32} width={175} depth={39} height={4} material={LIGHT} />
    {[188, 240, 292].map((x) => (
      <g key={x}>
        <Box x={x} y={428} z={36} width={38} depth={21} height={11} material={DARK} />
        <Face
          vertices={[
            [x + 3, 449.5, 38],
            [x + 35, 449.5, 38],
            [x + 35, 449.5, 44],
            [x + 3, 449.5, 44],
          ]}
          fill={METAL.left}
        />
        {[8, 14, 20, 26, 32].map((offset) => (
          <Wire
            key={offset}
            vertices={[
              [x + offset, 450, 39],
              [x + offset, 450, 43],
            ]}
            className="scene-rack-slot"
          />
        ))}
        <Box x={x + 16} y={418} z={36} width={6} depth={6} height={9} material={DARK} />
        <Box
          x={x + 4}
          y={417}
          z={44}
          width={30}
          depth={3}
          height={19}
          material={METAL}
        />
        <Face
          vertices={[
            [x + 7, 420.5, 47],
            [x + 31, 420.5, 47],
            [x + 31, 420.5, 60],
            [x + 7, 420.5, 60],
          ]}
          fill={DARK.right}
        />
        <Wire
          vertices={[
            [x + 10, 421, 52],
            [x + 14, 421, 52],
            [x + 17, 421, 56],
            [x + 21, 421, 50],
            [x + 25, 421, 54],
            [x + 28, 421, 54],
          ]}
          className="scene-rack-slot"
        />
      </g>
    ))}
    <Wire
      vertices={[
        [151, 473, 78],
        [151, 335, 78],
        [415, 335, 78],
        [415, 473, 78],
        [151, 473, 78],
      ]}
      className="scene-roof-outline"
    />
    <Box x={411} y={466} z={5} width={4} depth={7} height={73} material={CONCRETE} />
  </g>
);

const NODES = [
  { x: 302, y: 225, label: "F.01" },
  { x: 590, y: 246, label: "F.02" },
  { x: 713, y: 215, label: "F.03" },
  { x: 440, y: 463, label: "F.04" },
] as const;

// The intersite trunk remains outside each equipment envelope. Each independent
// local route terminates at the acquisition cabinet for that physical system.
const TRUNK: readonly Point[][] = [
  [
    [302, 225, 6],
    [302, 280, 6],
    [440, 280, 6],
  ],
  [
    [440, 463, 6],
    [440, 280, 6],
    [590, 280, 6],
    [590, 246, 6],
  ],
  [
    [590, 280, 6],
    [713, 280, 6],
    [713, 215, 6],
  ],
];

const LOCAL_ROUTES: readonly Point[][] = [
  [
    [253, 148, 6],
    [283, 148, 6],
    [283, 225, 6],
    [302, 225, 6],
  ],
  [
    [576, 183, 6],
    [590, 183, 6],
    [590, 246, 6],
  ],
  [
    [701, 200, 6],
    [701, 215, 6],
    [713, 215, 6],
  ],
  [
    [393, 386, 6],
    [402, 386, 6],
    [402, 463, 6],
    [440, 463, 6],
  ],
];

export const AerospaceScene = (): ReactElement => (
  <g className="deployment-aerospace">
    <Pad x={30} y={28} width={285} depth={223} />
    <Pad x={438} y={33} width={314} depth={223} />
    <Pad x={131} y={320} width={331} depth={175} />
    {TRUNK.map((vertices, i) => (
      <Trace key={`trunk-${i}`} vertices={vertices} reverse={i === 1} delay={i * 0.7} />
    ))}
    <LaunchComplex />
    <g className="scene-physical">
      <EngineStand />
      <PropellantSystem />
    </g>
    <AvionicsLab />
    {LOCAL_ROUTES.map((vertices, i) => (
      <Trace key={`local-${i}`} vertices={vertices} muted delay={i * 0.45} />
    ))}
    {NODES.map((node) => (
      <Node key={node.label} {...node} />
    ))}
    <g className="scene-annotations">
      <Label
        anchor={[79, 75, 185]}
        x={278}
        y={43}
        title="LAUNCH OPERATIONS"
        detail="Vehicle · ground systems"
        width={238}
      />
      <Label
        anchor={[530, 147, 111]}
        x={1113}
        y={232}
        title="PROPULSION TEST"
        detail="Hotfire · feed systems"
        width={230}
        side="right"
      />
      <Label
        anchor={[272, 446, 43]}
        x={173}
        y={621}
        title="HITL AVIONICS LAB"
        detail="Flight hardware · simulation"
        width={265}
      />
    </g>
  </g>
);
