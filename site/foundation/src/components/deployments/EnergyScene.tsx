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

import { Label, Node, Pad, Trace } from "./DeploymentGeometry";

const Battery = ({
  x,
  y,
  width = 74,
  depth = 30,
  height = 50,
}: {
  x: number;
  y: number;
  width?: number;
  depth?: number;
  height?: number;
}): ReactElement => (
  <g>
    <Box
      x={x - 2}
      y={y - 2}
      width={width + 4}
      depth={depth + 4}
      height={4}
      material={DARK}
    />
    <Box
      x={x}
      y={y}
      z={4}
      width={width}
      depth={depth}
      height={height}
      material={LIGHT}
    />
    <Box
      x={x - 1}
      y={y - 1}
      z={height + 4}
      width={width + 2}
      depth={depth + 2}
      height={2}
      material={METAL}
    />
    {[0, 1, 2].map((i) => {
      const left = x + 4 + (i * (width - 8)) / 3;
      const right = left + (width - 14) / 3;
      return (
        <g key={i}>
          <Face
            vertices={[
              [left, y + depth + 0.2, 9],
              [right, y + depth + 0.2, 9],
              [right, y + depth + 0.2, height],
              [left, y + depth + 0.2, height],
            ]}
            fill={CONCRETE.left}
            className="scene-panel"
          />
          {[0, 1, 2, 3, 4].map((rib) => (
            <Wire
              key={rib}
              vertices={[
                [left + 3, y + depth + 0.5, 15 + rib * 4],
                [right - 3, y + depth + 0.5, 15 + rib * 4],
              ]}
              className="scene-door"
            />
          ))}
          <Wire
            vertices={[
              [right - 3, y + depth + 0.7, height - 9],
              [right - 3, y + depth + 0.7, height - 15],
            ]}
            className="scene-railing"
          />
        </g>
      );
    })}
    {[0, 1].map((i) => (
      <g key={i}>
        <Box
          x={x + 12 + i * (width - 33)}
          y={y + 7}
          z={height + 6}
          width={13}
          depth={depth - 14}
          height={3}
          material={DARK}
        />
        <Cylinder
          x={x + 18.5 + i * (width - 33)}
          y={y + depth / 2}
          z={height + 9}
          radius={4.5}
          height={1}
          dark
        />
      </g>
    ))}
    {Array.from({ length: 5 }, (_, i) => (
      <Wire
        key={i}
        vertices={[
          [x + width + 0.3, y + 4, 13 + i * 6],
          [x + width + 0.3, y + depth - 4, 13 + i * 6],
        ]}
        className="scene-detail"
      />
    ))}
  </g>
);

const Inverter = ({ x, y }: { x: number; y: number }): ReactElement => (
  <g>
    <Box x={x} y={y} width={25} depth={21} height={33} material={METAL} />
    <Face
      vertices={[
        [x + 4, y + 21.3, 20],
        [x + 20, y + 21.3, 20],
        [x + 20, y + 21.3, 28],
        [x + 4, y + 21.3, 28],
      ]}
      fill={DARK.right}
    />
    {[8, 11, 14].map((z) => (
      <Wire
        key={z}
        vertices={[
          [x + 4, y + 21.5, z],
          [x + 20, y + 21.5, z],
        ]}
        className="scene-door"
      />
    ))}
  </g>
);

const Transformer = ({ x, y }: { x: number; y: number }): ReactElement => (
  <g>
    <Box x={x - 3} y={y - 3} width={40} depth={33} height={4} material={DARK} />
    <Box
      x={x + 3}
      y={y + 3}
      z={4}
      width={24}
      depth={20}
      height={24}
      material={CONCRETE}
    />
    {[0, 1, 2, 3, 4, 5].map((i) => (
      <Box
        key={i}
        x={x + i * 5}
        y={y + 22}
        z={6}
        width={2}
        depth={6}
        height={19}
        material={METAL}
      />
    ))}
    {[7, 17, 27].map((dx) => (
      <g key={dx}>
        <Cylinder x={x + dx} y={y + 11} z={28} radius={2.6} height={10} dark />
        <Cylinder x={x + dx} y={y + 11} z={31} radius={4} height={2} />
        <Cylinder x={x + dx} y={y + 11} z={35} radius={4} height={2} />
      </g>
    ))}
  </g>
);

const Dispatch = (): ReactElement => (
  <g>
    <Box x={604} y={323} width={85} depth={61} height={43} />
    <Box x={600} y={319} z={43} width={93} depth={69} height={4} material={LIGHT} />
    <Box x={615} y={333} z={47} width={29} depth={22} height={8} material={METAL} />
    {[613, 637, 661].map((x) => (
      <Face
        key={x}
        vertices={[
          [x, 384.3, 18],
          [x + 17, 384.3, 18],
          [x + 17, 384.3, 33],
          [x, 384.3, 33],
        ]}
        fill={DARK.right}
        className="scene-panel"
      />
    ))}
    <Transformer x={719} y={341} />
    {[610, 654, 698, 742].map((x) => (
      <g key={x}>
        <Box x={x} y={426} width={7} depth={7} height={37} material={DARK} />
        <Cylinder x={x + 3.5} y={429.5} z={37} radius={4.5} height={10} />
        {[40, 44].map((z) => (
          <Cylinder key={z} x={x + 3.5} y={429.5} z={z} radius={6} height={1.5} dark />
        ))}
      </g>
    ))}
    {[0, 7, 14].map((offset) => (
      <Wire
        key={offset}
        vertices={[
          [607, 427 + offset, 50],
          [750, 427 + offset, 50],
          [750, 364, 50],
          [740, 364, 39],
        ]}
        className="scene-pipe"
      />
    ))}
    <Wire
      vertices={[
        [688, 332, 47],
        [688, 332, 99],
        [682, 332, 88],
        [694, 332, 88],
      ]}
      className="scene-railing"
    />
  </g>
);

const ROUTES: readonly (readonly Point[])[] = [
  [
    [270, 215, 7],
    [270, 261, 7],
    [475, 261, 7],
    [475, 190, 7],
  ],
  [
    [270, 215, 7],
    [323, 215, 7],
    [323, 286, 7],
    [365, 286, 7],
    [365, 420, 7],
  ],
  [
    [475, 190, 7],
    [555, 190, 7],
    [555, 375, 7],
  ],
  [
    [365, 420, 7],
    [427, 420, 7],
    [427, 489, 7],
    [555, 489, 7],
    [555, 375, 7],
  ],
];

export const EnergyScene = (): ReactElement => (
  <g className="deployment-energy">
    <g className="scene-datum">
      {Array.from({ length: 10 }, (_, i) => (
        <Wire
          key={`x-${i}`}
          vertices={[
            [i * 85, -10, -12],
            [i * 85, 500, -12],
          ]}
        />
      ))}
      {Array.from({ length: 7 }, (_, i) => (
        <Wire
          key={`y-${i}`}
          vertices={[
            [0, i * 80, -12],
            [775, i * 80, -12],
          ]}
        />
      ))}
    </g>
    <g className="scene-campus-detail" opacity={0.45}>
      {[0, 23, 46].map((offset) => (
        <Wire
          key={offset}
          vertices={[
            [15, 230 + offset, -6],
            [110, 240 + offset, -6],
            [168, 268 + offset, -6],
            [276, 268 + offset, -6],
            [417, 223 + offset, -6],
            [480, 236 + offset, -6],
            [628, 205 + offset, -6],
            [772, 220 + offset, -6],
          ]}
          className="scene-site-edge"
        />
      ))}
      {[0, 24].map((offset) => (
        <Wire
          key={offset}
          vertices={[
            [20, 488 + offset, -6],
            [160, 496 + offset, -6],
            [329, 462 + offset, -6],
            [408, 482 + offset, -6],
            [485, 454 + offset, -6],
            [612, 467 + offset, -6],
            [769, 480 + offset, -6],
          ]}
          className="scene-site-edge"
        />
      ))}
    </g>
    <Pad x={40} y={55} width={240} depth={144} />
    <Pad x={420} y={10} width={275} depth={151} />
    <Pad x={65} y={315} width={278} depth={162} />
    <Pad x={590} y={306} width={180} depth={163} />
    <g className="scene-physical">
      {[71, 126].map((y) =>
        [55, 147].map((x) => <Battery key={`${x}-${y}`} x={x} y={y} />),
      )}
      <Inverter x={242} y={80} />
      <Transformer x={239} y={137} />
      {[437, 522, 607].map((x) => (
        <Battery key={x} x={x} y={27} width={67} depth={64} height={59} />
      ))}
      {[442, 530, 618].map((x) => (
        <Inverter key={x} x={x} y={118} />
      ))}
      <Wire
        vertices={[
          [449, 111, 7],
          [650, 111, 7],
          [650, 139, 7],
        ]}
        className="scene-pipe"
      />
      {[334, 393].map((y) =>
        [83, 136, 189].map((x) => (
          <Battery key={`${x}-${y}`} x={x} y={y} width={40} depth={31} height={54} />
        )),
      )}
      <Inverter x={276} y={345} />
      <Transformer x={276} y={410} />
      <Dispatch />
    </g>
    <g className="scene-local-links">
      <Trace
        muted
        vertices={[
          [254, 102, 7],
          [290, 102, 7],
          [290, 215, 7],
          [270, 215, 7],
        ]}
      />
      <Trace
        muted
        vertices={[
          [455, 142, 7],
          [455, 173, 7],
          [475, 173, 7],
          [475, 190, 7],
        ]}
      />
      <Trace
        muted
        vertices={[
          [300, 387, 7],
          [365, 387, 7],
          [365, 420, 7],
        ]}
      />
      <Trace
        muted
        vertices={[
          [604, 365, 7],
          [577, 365, 7],
          [577, 375, 7],
          [555, 375, 7],
        ]}
      />
    </g>
    <g className="scene-network">
      {ROUTES.map((vertices, i) => (
        <Trace key={i} vertices={vertices} delay={i * -0.8} />
      ))}
      <Node x={270} y={215} label="E.01" />
      <Node x={475} y={190} label="E.02" />
      <Node x={365} y={420} label="E.03" />
      <Node x={555} y={375} label="E.04" />
    </g>
    <g className="scene-annotations">
      <Label
        anchor={[92, 86, 60]}
        x={200}
        y={106}
        title="DISTRIBUTED STORAGE"
        detail="Battery telemetry · local control"
        width={270}
      />
      <Label
        anchor={[632, 56, 69]}
        x={1190}
        y={208}
        title="UTILITY SCALE"
        detail="Inverters · thermal systems"
        width={248}
        side="right"
      />
      <Label
        anchor={[645, 360, 46]}
        x={1018}
        y={663}
        title="REGIONAL DISPATCH"
        detail="One view across every site"
        width={253}
        side="right"
      />
    </g>
  </g>
);
