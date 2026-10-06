// Copyright 2026 Synnax Labs, Inc. Licensed under licenses/BSL.txt.

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

const Column = ({
  x,
  y,
  height,
}: {
  x: number;
  y: number;
  height: number;
}): ReactElement => (
  <g>
    <Box x={x - 28} y={y - 28} width={56} depth={56} height={8} material={CONCRETE} />
    <Cylinder x={x} y={y} z={8} radius={19} height={height} />
    <Cylinder x={x} y={y} z={height + 8} radius={21} height={4} dark />
    <Cylinder x={x} y={y} z={height + 12} radius={7} height={6} />
    {[0.3, 0.64].map((level) => (
      <g key={level}>
        {[-27, 22].map((offset) => (
          <g key={offset}>
            <Box
              x={x + offset}
              y={y - 27}
              z={height * level}
              width={5}
              depth={54}
              height={3}
              material={DARK}
            />
            <Box
              x={x - 22}
              y={y + offset}
              z={height * level}
              width={44}
              depth={5}
              height={3}
              material={DARK}
            />
          </g>
        ))}
        <Wire
          vertices={[
            [x - 27, y + 27, height * level + 13],
            [x + 27, y + 27, height * level + 13],
            [x + 27, y - 27, height * level + 13],
          ]}
          className="scene-railing"
        />
        {[-24, 0, 24].map((offset) => (
          <Wire
            key={offset}
            vertices={[
              [x + offset, y + 27, height * level + 3],
              [x + offset, y + 27, height * level + 13],
            ]}
            className="scene-detail"
          />
        ))}
      </g>
    ))}
    <Wire
      vertices={[
        [x + 28, y, 8],
        [x + 28, y, height + 8],
      ]}
      className="scene-railing"
    />
    <Wire
      vertices={[
        [x + 33, y, 8],
        [x + 33, y, height + 8],
      ]}
      className="scene-railing"
    />
    {Array.from({ length: Math.floor(height / 9) }, (_, i) => (
      <Wire
        key={i}
        vertices={[
          [x + 28, y, 14 + i * 9],
          [x + 33, y, 14 + i * 9],
        ]}
        className="scene-detail"
      />
    ))}
    <Wire
      vertices={[
        [x, y + 20, 38],
        [x, y + 38, 38],
        [x, y + 38, 49],
        [x, 253, 49],
      ]}
      className="scene-pipe-shadow"
    />
    <Wire
      vertices={[
        [x, y + 20, 38],
        [x, y + 38, 38],
        [x, y + 38, 49],
        [x, 253, 49],
      ]}
      className="scene-pipe"
    />
  </g>
);

const Exchanger = ({ x, y }: { x: number; y: number }): ReactElement => {
  const ring = (along: number): Point[] =>
    Array.from({ length: 20 }, (_, i) => [
      along,
      y + Math.cos((i * Math.PI) / 10) * 18,
      32 + Math.sin((i * Math.PI) / 10) * 18,
    ]);
  const a = ring(x),
    b = ring(x + 90);
  return (
    <g className="scene-solid">
      <Box x={x + 8} y={y - 22} width={10} depth={44} height={18} material={DARK} />
      <Box x={x + 72} y={y - 22} width={10} depth={44} height={18} material={DARK} />
      {Array.from({ length: 10 }, (_, i) => {
        const n = (i + 15) % 20,
          next = (n + 1) % 20;
        return (
          <Face
            key={i}
            vertices={[a[n], b[n], b[next], a[next]]}
            fill={i < 5 ? LIGHT.top : METAL.left}
          />
        );
      })}
      <Face vertices={b} fill={DARK.top} />
      <Wire vertices={[...b, b[0]]} className="scene-railing" />
      {[x + 8, x + 80].map((along) => (
        <Wire
          key={along}
          vertices={[...ring(along), ring(along)[0]]}
          className="scene-detail"
        />
      ))}
      <Box
        x={x + 88}
        y={y - 8}
        z={24}
        width={11}
        depth={16}
        height={16}
        material={METAL}
      />
    </g>
  );
};

const Production = (): ReactElement => (
  <g>
    <Box x={45} y={318} width={284} depth={137} height={56} />
    <Box x={41} y={314} z={56} width={292} depth={145} height={4} material={LIGHT} />
    {Array.from({ length: 12 }, (_, i) => (
      <Wire
        key={i}
        vertices={[
          [49 + i * 24, 320, 60.5],
          [49 + i * 24, 453, 60.5],
        ]}
        className="scene-detail"
      />
    ))}
    {[73, 161, 249].map((x) => (
      <g key={x}>
        <Box x={x} y={351} z={60} width={45} depth={57} height={7} material={DARK} />
        {[0, 1, 2, 3].map((i) => (
          <Wire
            key={i}
            vertices={[
              [x + 6, 358 + i * 12, 67],
              [x + 39, 358 + i * 12, 67],
            ]}
            className="scene-railing"
          />
        ))}
        <Face
          vertices={[
            [x, 455.5, 1],
            [x + 42, 455.5, 1],
            [x + 42, 455.5, 36],
            [x, 455.5, 36],
          ]}
          fill={DARK.right}
        />
        {[9, 18, 27].map((z) => (
          <Wire
            key={z}
            vertices={[
              [x, 456, z],
              [x + 42, 456, z],
            ]}
            className="scene-door"
          />
        ))}
        <Box x={x - 2} y={456} width={46} depth={18} height={4} material={DARK} />
      </g>
    ))}
  </g>
);

export const PlantScene = (): ReactElement => (
  <g>
    <Pad x={5} y={15} width={766} depth={475} />
    <g className="deployment-physical">
      <Box x={33} y={39} width={160} depth={135} height={38} />
      <Box x={29} y={35} z={38} width={168} depth={143} height={4} material={LIGHT} />
      {[70, 130].map((x) => (
        <g key={x}>
          <Cylinder x={x} y={91} z={42} radius={23} height={26} dark />
          <Cylinder x={x} y={91} z={68} radius={21} height={3} />
          <Cylinder x={x} y={91} z={71} radius={15} height={2} dark />
          {Array.from({ length: 6 }, (_, i) => (
            <Wire
              key={i}
              vertices={[
                [x - 10, 79 + i * 4, 73],
                [x + 10, 79 + i * 4, 73],
              ]}
              className="scene-detail"
            />
          ))}
        </g>
      ))}
      <Column x={302} y={84} height={221} />
      <Column x={406} y={89} height={192} />
      <Column x={367} y={178} height={146} />
      {[590, 678].map((x) =>
        [65, 154].map((y) => (
          <g key={`${x}-${y}`}>
            <Cylinder x={x} y={y} z={0} radius={33} height={51} />
            <Cylinder x={x} y={y} z={51} radius={8} height={5} dark />
            <Wire
              vertices={[
                [x + 34, y, 0],
                [x + 34, y, 52],
                [x + 10, y, 52],
              ]}
              className="scene-railing"
            />
            <Wire
              vertices={[
                [x, y + 33, 15],
                [x, y + 43, 15],
                [x, 247, 15],
              ]}
              className="scene-pipe"
            />
          </g>
        )),
      )}
      {[191, 285, 475, 568, 720].map((x) => (
        <g key={x}>
          <Box x={x} y={238} width={5} depth={5} height={47} material={DARK} />
          <Box x={x} y={266} width={5} depth={5} height={47} material={DARK} />
          <Box x={x} y={235} z={45} width={5} depth={39} height={4} material={METAL} />
        </g>
      ))}
      {[245, 253, 261].map((y) => (
        <g key={y}>
          <Wire
            vertices={[
              [176, y, 50],
              [732, y, 50],
            ]}
            className="scene-pipe-shadow"
          />
          <Wire
            vertices={[
              [176, y, 50],
              [732, y, 50],
            ]}
            className="scene-pipe"
          />
        </g>
      ))}
      <Production />
      <Box x={478} y={320} width={150} depth={115} height={5} material={CONCRETE} />
      <Exchanger x={502} y={348} />
      <Exchanger x={502} y={407} />
      <Pad x={661} y={321} width={92} depth={119} />
      <Rack x={676} y={336} width={28} depth={27} height={68} />
      <Rack x={715} y={336} width={25} depth={27} height={68} />
    </g>
    <g>
      <Trace
        muted
        vertices={[
          [156, 174, 8],
          [156, 205, 8],
          [209, 205, 8],
        ]}
      />
      <Trace
        muted
        vertices={[
          [406, 126, 9],
          [475, 126, 9],
          [475, 205, 9],
        ]}
      />
      <Trace
        muted
        vertices={[
          [700, 179, 8],
          [736, 179, 8],
          [736, 205, 8],
        ]}
      />
      <Trace
        muted
        vertices={[
          [329, 391, 8],
          [359, 391, 8],
          [359, 465, 8],
        ]}
      />
      <Trace
        muted
        vertices={[
          [627, 406, 8],
          [645, 406, 8],
          [645, 465, 8],
        ]}
      />
      <Trace
        vertices={[
          [209, 205, 8],
          [227, 205, 8],
          [227, 289, 8],
          [449, 289, 8],
          [449, 465, 8],
          [645, 465, 8],
        ]}
        delay={-1.1}
      />
      <Trace
        vertices={[
          [475, 205, 8],
          [475, 289, 8],
          [449, 289, 8],
        ]}
        delay={-2.5}
      />
      <Trace
        vertices={[
          [736, 205, 8],
          [755, 205, 8],
          [755, 289, 8],
          [475, 289, 8],
        ]}
        delay={-0.5}
      />
      <Trace
        vertices={[
          [359, 465, 8],
          [449, 465, 8],
        ]}
        delay={-3}
      />
      <Trace
        vertices={[
          [645, 465, 8],
          [715, 465, 8],
          [715, 382, 8],
        ]}
      />
      {[
        [209, 205],
        [475, 205],
        [736, 205],
        [359, 465],
        [645, 465],
      ].map(([x, y], i) => (
        <Node key={i} x={x} y={y} label={`F.0${i + 1}`} />
      ))}
    </g>
    <Label
      anchor={[302, 84, 237]}
      x={335}
      y={10}
      width={210}
      title="PROCESS UNITS"
      detail="Pressure · temperature · flow"
    />
    <Label
      anchor={[678, 65, 55]}
      x={1155}
      y={130}
      width={258}
      title="UTILITIES & STORAGE"
      detail="Local PLCs · process telemetry"
      side="right"
    />
    <Label
      anchor={[165, 360, 66]}
      x={50}
      y={607}
      width={245}
      title="PRODUCTION HALL"
      detail="Equipment state · supervisory control"
    />
  </g>
);
