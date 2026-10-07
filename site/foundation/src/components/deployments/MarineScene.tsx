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
  project,
  Wire,
} from "@/components/IndustrialGeometry";

import { Label, Node, Trace } from "./DeploymentGeometry";

interface VesselProps {
  x: number;
  y: number;
  length: number;
  beam: number;
  variant: "patrol" | "support" | "uncrewed";
  label: string;
}

const VESSELS: readonly VesselProps[] = [
  { x: 46, y: 62, length: 220, beam: 52, variant: "patrol", label: "M.01" },
  { x: 409, y: 34, length: 124, beam: 34, variant: "uncrewed", label: "M.02" },
  { x: 165.75, y: 226, length: 256, beam: 56, variant: "support", label: "M.03" },
  { x: 597, y: 209, length: 117, beam: 40, variant: "uncrewed", label: "M.04" },
  { x: 65, y: 390, length: 126, beam: 40, variant: "uncrewed", label: "M.05" },
  { x: 418, y: 451, length: 118, beam: 32, variant: "uncrewed", label: "M.06" },
];

const deckNode = ({ x, y, length, beam }: VesselProps): Point => [
  x + length * 0.65,
  y + beam / 2,
  18,
];

const Radome = ({ x, y, z }: { x: number; y: number; z: number }): ReactElement => {
  const ring = (radius: number, height: number): Point[] =>
    Array.from({ length: 12 }, (_, i) => [
      x + Math.cos((i * Math.PI) / 6) * radius,
      y + Math.sin((i * Math.PI) / 6) * radius,
      z + height,
    ]);
  const lower = ring(5, 2);
  const upper = ring(3.5, 6);
  return (
    <g className="scene-solid">
      <Cylinder x={x} y={y} z={z} radius={3} height={2} dark />
      {Array.from({ length: 12 }, (_, i) => (
        <Face
          key={i}
          vertices={[lower[i], lower[(i + 1) % 12], upper[(i + 1) % 12], upper[i]]}
          fill={i < 6 ? LIGHT.left : METAL.top}
        />
      ))}
      <Face vertices={upper} fill={LIGHT.top} />
    </g>
  );
};

const Vessel = ({ x, y, length, beam, variant, label }: VesselProps): ReactElement => {
  const forwardBridge = variant === "support";
  const autonomous = variant === "uncrewed";
  const large = !autonomous;
  const shoulder = large ? 0.84 : 0.73;
  const deck: Point[] = [
    [x, y + 5, 18],
    [x + length * shoulder, y, 18],
    [x + length, y + beam / 2, large ? 25 : 22],
    [x + length * shoulder, y + beam, 18],
    [x, y + beam - 5, 18],
    [x - 4, y + beam / 2, 18],
  ];
  const waterline: Point[] = deck.map(([px, py]) => [
    x + length / 2 + (px - x - length / 2) * 0.94,
    y + beam / 2 + (py - y - beam / 2) * 0.76,
    1,
  ]);
  const cabinX = x + length * (forwardBridge ? 0.73 : large ? 0.28 : 0.19);
  const cabinY = y + beam * 0.23;
  const cabinWidth = length * (forwardBridge ? 0.17 : large ? 0.23 : 0.24);
  const cabinDepth = beam * 0.54;
  const cabinHeight = autonomous ? 13 : 25;
  const lowerRoof = 18 + (autonomous ? 4 : 8);
  const upperRoof = 18 + cabinHeight;
  const rake = autonomous ? 8 : 7;
  const [nodeX, nodeY, nodeZ] = deckNode({ x, y, length, beam, label, variant });
  const [nodePX, nodePY] = project([nodeX, nodeY, nodeZ + 7]);
  const windowBottom = lowerRoof + 4;
  const windowTop = upperRoof - 3;
  const sideY = (z: number): number =>
    cabinY + cabinDepth - (3 * (z - lowerRoof)) / (upperRoof - lowerRoof);
  const frontX = (z: number): number =>
    cabinX + cabinWidth - (rake * (z - lowerRoof)) / (upperRoof - lowerRoof);
  return (
    <g>
      <g className="scene-solid">
        {[0, 5, 4, 1, 2, 3].map((i) => (
          <Face
            key={i}
            vertices={[
              waterline[i],
              waterline[(i + 1) % 6],
              deck[(i + 1) % 6],
              deck[i],
            ]}
            fill={i === 1 || i === 2 ? CONCRETE.right : CONCRETE.left}
          />
        ))}
        <Face vertices={deck} fill={METAL.top} />
        <Wire
          vertices={[deck[2], deck[3], deck[4], deck[5]]}
          className="scene-railing"
        />
      </g>
      <Wire
        vertices={[
          [x + 8, y + beam - 6, 8],
          [x + length * shoulder, y + beam - 5, 8],
          [x + length - 7, y + beam / 2 + 2, 13],
        ]}
        className="scene-detail"
      />
      {large && (
        <g>
          <Face
            vertices={[
              [x + 10, y + 8, 18.2],
              [cabinX - 7, y + 8, 18.2],
              [cabinX - 7, y + beam - 8, 18.2],
              [x + 10, y + beam - 8, 18.2],
            ]}
            fill={CONCRETE.left}
          />
          <Wire
            vertices={[
              [x + 13, y + 10, 18.5],
              [cabinX - 10, y + 10, 18.5],
              [cabinX - 10, y + beam - 10, 18.5],
              [x + 13, y + beam - 10, 18.5],
            ]}
            className="scene-detail"
          />
          {forwardBridge ? (
            <g>
              <Wire
                vertices={[
                  [x + 38, y + 14, 18.6],
                  [x + 49, y + 7, 18.6],
                  [x + 96, y + 7, 18.6],
                  [x + 107, y + 14, 18.6],
                  [x + 107, y + beam - 14, 18.6],
                  [x + 96, y + beam - 7, 18.6],
                  [x + 49, y + beam - 7, 18.6],
                  [x + 38, y + beam - 14, 18.6],
                  [x + 38, y + 14, 18.6],
                ]}
                className="scene-detail"
              />
              <Wire
                vertices={[
                  [x + 66, y + 18, 18.7],
                  [x + 66, y + beam - 18, 18.7],
                ]}
                className="scene-railing"
              />
              <Wire
                vertices={[
                  [x + 80, y + 18, 18.7],
                  [x + 80, y + beam - 18, 18.7],
                ]}
                className="scene-railing"
              />
              <Wire
                vertices={[
                  [x + 66, y + beam / 2, 18.7],
                  [x + 80, y + beam / 2, 18.7],
                ]}
                className="scene-railing"
              />
              {[0, 1].map((i) => (
                <g key={i}>
                  <Box
                    x={x + 124 + i * 30}
                    y={y + 10}
                    z={18}
                    width={23}
                    depth={beam - 20}
                    height={2}
                    material={DARK}
                  />
                  <Wire
                    vertices={[
                      [x + 127 + i * 30, y + 13, 20.3],
                      [x + 144 + i * 30, y + 13, 20.3],
                      [x + 144 + i * 30, y + beam - 13, 20.3],
                    ]}
                    className="scene-detail"
                  />
                </g>
              ))}
            </g>
          ) : (
            <g>
              <Face
                vertices={[
                  [x + 13, y + beam / 2 - 7, 19],
                  [x + 43, y + beam / 2 - 7, 19],
                  [x + 50, y + beam / 2, 21],
                  [x + 43, y + beam / 2 + 7, 22],
                  [x + 13, y + beam / 2 + 7, 22],
                  [x + 9, y + beam / 2, 21],
                ]}
                fill={DARK.top}
                className="scene-panel"
              />
              <Face
                vertices={[
                  [x + 18, y + beam / 2 - 4, 22],
                  [x + 39, y + beam / 2 - 4, 22],
                  [x + 44, y + beam / 2, 23],
                  [x + 39, y + beam / 2 + 4, 23],
                  [x + 18, y + beam / 2 + 4, 23],
                ]}
                fill={METAL.left}
              />
              <Box
                x={x + length * 0.79}
                y={y + beam / 2 - 10}
                z={19}
                width={17}
                depth={20}
                height={2}
                material={CONCRETE}
              />
            </g>
          )}
          {[y + 9, y + beam - 9].map((bollardY) => (
            <g key={bollardY}>
              <Cylinder x={x + 8} y={bollardY} z={18} radius={2} height={3} dark />
              <Cylinder x={x + 14} y={bollardY} z={18} radius={2} height={3} dark />
            </g>
          ))}
        </g>
      )}
      <Box
        x={cabinX}
        y={cabinY}
        z={18}
        width={cabinWidth}
        depth={cabinDepth}
        height={lowerRoof - 18}
        material={CONCRETE}
      />
      <g className="scene-solid">
        <Face
          vertices={[
            [cabinX, cabinY + cabinDepth, lowerRoof],
            [cabinX + cabinWidth, cabinY + cabinDepth, lowerRoof],
            [cabinX + cabinWidth - rake, cabinY + cabinDepth - 3, upperRoof],
            [cabinX + 4, cabinY + cabinDepth - 3, upperRoof],
          ]}
          fill={LIGHT.left}
        />
        <Face
          vertices={[
            [cabinX + cabinWidth, cabinY, lowerRoof],
            [cabinX + cabinWidth, cabinY + cabinDepth, lowerRoof],
            [cabinX + cabinWidth - rake, cabinY + cabinDepth - 3, upperRoof],
            [cabinX + cabinWidth - rake, cabinY + 3, upperRoof],
          ]}
          fill={LIGHT.right}
        />
        <Face
          vertices={[
            [cabinX + 4, cabinY + 3, upperRoof],
            [cabinX + cabinWidth - rake, cabinY + 3, upperRoof],
            [cabinX + cabinWidth - rake, cabinY + cabinDepth - 3, upperRoof],
            [cabinX + 4, cabinY + cabinDepth - 3, upperRoof],
          ]}
          fill={LIGHT.top}
        />
      </g>
      {Array.from({ length: large ? 4 : 2 }, (_, i) => {
        const count = large ? 4 : 2;
        const wx = cabinX + 6 + (i * (cabinWidth - 18)) / count;
        const ww = (cabinWidth - 18) / count - 2;
        return (
          <Face
            key={i}
            vertices={[
              [wx, sideY(windowBottom) + 0.2, windowBottom],
              [wx + ww, sideY(windowBottom) + 0.2, windowBottom],
              [wx + ww, sideY(windowTop) + 0.2, windowTop],
              [wx, sideY(windowTop) + 0.2, windowTop],
            ]}
            fill={DARK.right}
            className="scene-panel"
          />
        );
      })}
      <Face
        vertices={[
          [frontX(windowBottom) + 0.2, cabinY + 5, windowBottom],
          [frontX(windowBottom) + 0.2, cabinY + cabinDepth - 5, windowBottom],
          [frontX(windowTop) + 0.2, cabinY + cabinDepth - 5, windowTop],
          [frontX(windowTop) + 0.2, cabinY + 5, windowTop],
        ]}
        fill={DARK.top}
        className="scene-panel"
      />
      <Wire
        vertices={[
          [cabinX + cabinWidth * 0.5, y + beam / 2, upperRoof],
          [cabinX + cabinWidth * 0.5, y + beam / 2, upperRoof + (autonomous ? 23 : 35)],
        ]}
        className="scene-railing"
      />
      <Wire
        vertices={[
          [cabinX + cabinWidth * 0.5 - 8, y + beam / 2, upperRoof + 22],
          [cabinX + cabinWidth * 0.5 + 8, y + beam / 2, upperRoof + 22],
        ]}
        className="scene-railing"
      />
      <Wire
        vertices={[
          [cabinX + 8, cabinY + 6, upperRoof],
          [cabinX + 8, cabinY + 6, upperRoof + 17],
        ]}
        className="scene-detail"
      />
      {large && (
        <g>
          <Radome x={cabinX + 12} y={y + beam / 2} z={upperRoof} />
          <g className="scene-solid">
            <Face
              vertices={[
                [cabinX + cabinWidth * 0.5 - 3, y + beam / 2 + 2, upperRoof],
                [cabinX + cabinWidth * 0.5 + 4, y + beam / 2 + 2, upperRoof],
                [cabinX + cabinWidth * 0.5 + 2, y + beam / 2 + 1, upperRoof + 22],
                [cabinX + cabinWidth * 0.5, y + beam / 2 + 1, upperRoof + 22],
              ]}
              fill={METAL.left}
            />
            <Face
              vertices={[
                [cabinX + cabinWidth * 0.5 + 4, y + beam / 2 - 2, upperRoof],
                [cabinX + cabinWidth * 0.5 + 4, y + beam / 2 + 2, upperRoof],
                [cabinX + cabinWidth * 0.5 + 2, y + beam / 2 + 1, upperRoof + 22],
                [cabinX + cabinWidth * 0.5 + 2, y + beam / 2 - 1, upperRoof + 22],
              ]}
              fill={METAL.right}
            />
          </g>
          <Box
            x={cabinX + cabinWidth * 0.5 - 8}
            y={y + beam / 2 - 1}
            z={upperRoof + 22}
            width={16}
            depth={2}
            height={2}
            material={LIGHT}
          />
        </g>
      )}
      <Cylinder
        x={x + length * 0.91}
        y={y + beam / 2}
        z={22}
        radius={3.5}
        height={4}
        dark
      />
      <Wire
        vertices={[
          [x + length * 0.87, y + beam / 2, 23],
          [x + length * 0.96, y + beam / 2, 26],
        ]}
        className="scene-railing"
      />
      {[0, 1, 2, 3, 4, 5].map((i) => (
        <g key={i}>
          <Wire
            vertices={[
              [x + 7 + i * length * 0.13, y + beam - 5 + Math.min(i, 2) * 1.5, 18],
              [
                x + 7 + i * length * 0.13,
                y + beam - 5 + Math.min(i, 2) * 1.5,
                large ? 25 : 23,
              ],
            ]}
            className="scene-detail"
          />
          {i % 2 === 0 && (
            <Box
              x={x + 7 + i * length * 0.13}
              y={y + beam - 5 + Math.min(i, 2) * 1.5}
              z={8}
              width={3}
              depth={2}
              height={8}
              material={DARK}
            />
          )}
        </g>
      ))}
      <Wire
        vertices={[
          [x + 7, y + beam - 5, large ? 25 : 23],
          [x + length * 0.74, y + beam - 2, large ? 25 : 23],
        ]}
        className="scene-detail"
      />
      {!large && (
        <g>
          <Face
            vertices={[
              [x + 4, y + 9, 18.3],
              [x + 19, y + 9, 18.3],
              [x + 19, y + beam - 9, 18.3],
              [x + 4, y + beam - 9, 18.3],
            ]}
            fill={DARK.top}
            className="scene-panel"
          />
          <Cylinder
            x={cabinX + 10}
            y={y + beam / 2}
            z={upperRoof}
            radius={3.5}
            height={3}
            dark
          />
          <Wire
            vertices={[
              [x + length * 0.5, y + 7, 19],
              [x + length * 0.5, y + beam - 7, 19],
            ]}
            className="scene-detail"
          />
        </g>
      )}
      <Trace
        muted
        vertices={[
          [forwardBridge ? cabinX : cabinX + cabinWidth, nodeY, 24],
          [nodeX, nodeY, 24],
        ]}
      />
      <g
        transform={`translate(${nodePX} ${nodePY}) scale(0.65) translate(${-nodePX} ${-nodePY})`}
      >
        <Node x={nodeX} y={nodeY} z={nodeZ} label={label} />
      </g>
    </g>
  );
};

const Quay = (): ReactElement => (
  <g>
    <Box
      x={603}
      y={370}
      z={-5}
      width={177}
      depth={130}
      height={14}
      material={CONCRETE}
    />
    <Box x={601} y={367} z={9} width={181} depth={6} height={4} material={LIGHT} />
    <Box x={600} y={370} z={9} width={6} depth={130} height={4} material={LIGHT} />
    <Box x={694} y={388} z={9} width={70} depth={78} height={46} />
    <Box x={691} y={385} z={55} width={76} depth={84} height={4} material={LIGHT} />
    <Box x={710} y={399} z={59} width={27} depth={21} height={8} material={METAL} />
    {[703, 723, 743].map((x) => (
      <Face
        key={x}
        vertices={[
          [x, 466.3, 26],
          [x + 12, 466.3, 26],
          [x + 12, 466.3, 42],
          [x, 466.3, 42],
        ]}
        fill={DARK.right}
        className="scene-panel"
      />
    ))}
    <Box x={682} y={483} z={9} width={15} depth={15} height={7} material={DARK} />
    <Box x={687} y={488} z={16} width={5} depth={5} height={74} material={METAL} />
    <Wire
      vertices={[
        [690, 490, 88],
        [634, 490, 73],
        [690, 490, 70],
        [708, 490, 88],
        [690, 490, 88],
      ]}
      className="scene-mast"
    />
    <Wire
      vertices={[
        [634, 490, 73],
        [634, 490, 29],
        [640, 490, 27],
      ]}
      className="scene-railing"
    />
    {[390, 424, 458, 492].map((y) => (
      <g key={y}>
        <Box x={598} y={y} z={-2} width={5} depth={11} height={11} material={DARK} />
        <Cylinder x={614} y={y + 5} z={9} radius={3} height={5} dark />
      </g>
    ))}
    <Wire
      vertices={[
        [674, 387, 9],
        [674, 472, 9],
        [752, 472, 9],
      ]}
      className="scene-road-line"
    />
    <Trace
      muted
      vertices={[
        [694, 427, 16],
        [669, 427, 16],
        [669, 438, 16],
        [637, 438, 16],
      ]}
    />
    <Node x={637} y={438} z={9} label="M.00" />
  </g>
);

const YellowSubmarine = (): ReactElement => {
  const x = 262;
  const y = 380;
  const sections = [
    [0, 0],
    [7, 6],
    [17, 10],
    [62, 10],
    [73, 7],
    [79, 0],
  ];
  const rings: Point[][] = sections.map(([offset, radius]) =>
    Array.from({ length: 12 }, (_, i) => {
      const angle = (i * Math.PI) / 6;
      return [x + offset, y + Math.cos(angle) * radius, 10 + Math.sin(angle) * radius];
    }),
  );
  const hull = rings.slice(1).flatMap((ring, section) =>
    ring.map((point, i) => {
      const next = (i + 1) % ring.length;
      const vertices = [rings[section][i], rings[section][next], ring[next], point];
      const light = (Math.sin(((i + 0.5) * Math.PI) / 6) + 1) / 2;
      return {
        vertices,
        fill: `rgb(${Math.round(132 + light * 88)} ${Math.round(102 + light * 87)} ${Math.round(39 + light * 50)})`,
        depth: vertices.reduce((sum, [px, py, pz]) => sum + px + py + pz * 0.94, 0),
      };
    }),
  );
  // Match the fleet's orthographic view, drawing the curved hull back to front.
  hull.sort((a, b) => a.depth - b.depth);
  return (
    <g className="deployment-submarine" transform="translate(0 24)" aria-hidden="true">
      <g opacity={0.58}>
        <Face
          vertices={[
            [x + 3, y - 17, 10],
            [x + 13, y - 14, 10],
            [x + 17, y + 14, 10],
            [x + 3, y + 17, 10],
          ]}
          fill="#ad8d42"
        />
        <Face
          vertices={[
            [x + 4, y, 10],
            [x + 4, y, 25],
            [x + 10, y, 23],
            [x + 17, y, 10],
          ]}
          fill="#bea04e"
        />
        {hull.map(({ vertices, fill }, i) => (
          <Face key={i} vertices={vertices} fill={fill} />
        ))}
        <Box
          x={x + 31}
          y={y - 4}
          z={18}
          width={17}
          depth={8}
          height={9}
          material={{ top: "#e0c66e", left: "#b39445", right: "#997937" }}
        />
        {[24, 40, 56].map((offset) => (
          <g key={offset}>
            {[3, 1.8].map((radius, i) => (
              <Face
                key={radius}
                vertices={Array.from({ length: 12 }, (_, j) => {
                  const angle = (j * Math.PI) / 6;
                  return [
                    x + offset + Math.cos(angle) * radius,
                    y + 9.7,
                    12 + Math.sin(angle) * radius,
                  ];
                })}
                fill={i === 0 ? "#dfc47a" : "#263748"}
              />
            ))}
          </g>
        ))}
      </g>
      <g fill="none" stroke="#7799b4" strokeWidth={0.8} opacity={0.32}>
        {[-4, 16, 35].map((offset) => {
          const start = project([x - 25, y + offset, 24]);
          const crest = project([x + 25, y + offset - 6, 24]);
          const end = project([x + 103, y + offset + 2, 24]);
          return (
            <path
              key={offset}
              d={`M${start.join(",")} Q${crest.join(",")} ${end.join(",")}`}
            />
          );
        })}
      </g>
      <Box
        x={x + 40}
        y={y - 1}
        z={27}
        width={2}
        depth={2}
        height={10}
        material={{ top: "#d3bc7b", left: "#a68e53", right: "#796641" }}
      />
      <Box
        x={x + 40}
        y={y - 1}
        z={35}
        width={6}
        depth={2}
        height={2}
        material={{ top: "#d3bc7b", left: "#a68e53", right: "#796641" }}
      />
    </g>
  );
};

const ROUTES: readonly (readonly Point[])[] = [
  [
    [189, 88, 25],
    [189, 158, 7],
    [560, 158, 7],
    [560, 438, 7],
    [637, 438, 16],
  ],
  [
    [489.6, 51, 25],
    [560, 51, 7],
    [560, 158, 7],
  ],
  [
    [332.15, 254, 25],
    [332.15, 320, 7],
    [560, 320, 7],
  ],
  [
    [673.05, 229, 25],
    [738, 229, 7],
    [738, 302, 7],
    [560, 302, 7],
  ],
  [
    [146.9, 410, 25],
    [146.9, 530, 7],
    [560, 530, 7],
    [560, 438, 7],
  ],
  [
    [494.7, 467, 25],
    [560, 467, 7],
  ],
];

export const MarineScene = (): ReactElement => (
  <g className="deployment-marine">
    <g className="scene-datum">
      {[0, 1, 2, 3, 4, 5, 6, 7].map((i) => (
        <Wire
          key={i}
          vertices={[
            [10, 5 + i * 72, -9],
            [131, -2 + i * 72, -9],
            [266, 17 + i * 72, -9],
            [421, 3 + i * 72, -9],
            [580, 13 + i * 72, -9],
            [770, 2 + i * 72, -9],
          ]}
        />
      ))}
      {[0, 1, 2, 3, 4].map((i) => (
        <Wire
          key={i}
          vertices={[
            [80 + i * 155, -4, -9],
            [62 + i * 155, 172, -9],
            [89 + i * 155, 301, -9],
            [73 + i * 155, 545, -9],
          ]}
        />
      ))}
    </g>
    <g opacity={0.18}>
      <Wire
        vertices={[
          [10, 194, -3],
          [130, 181, -3],
          [259, 188, -3],
          [420, 179, -3],
          [759, 182, -3],
        ]}
        className="scene-road-line"
      />
      <Wire
        vertices={[
          [21, 349, -3],
          [144, 352, -3],
          [290, 364, -3],
          [460, 350, -3],
          [777, 346, -3],
        ]}
        className="scene-road-line"
      />
    </g>
    <g className="scene-network">
      {ROUTES.map((vertices, i) => (
        <Trace key={i} vertices={vertices} delay={-i * 0.7} />
      ))}
    </g>
    <g className="scene-physical">
      {VESSELS.map((vessel) => (
        <Vessel key={vessel.label} {...vessel} />
      ))}
      <Quay />
      <YellowSubmarine />
    </g>
    <g className="scene-annotations">
      <Label
        anchor={[145, 82, 64]}
        x={272}
        y={93}
        title="ONBOARD FOUNDATION"
        detail="Sensors · navigation · machinery"
        width={281}
      />
      <Label
        anchor={[673.05, 229, 32]}
        x={1165}
        y={412}
        title="A CONNECTED FLEET"
        detail="Local context on every vessel"
        width={260}
        side="right"
      />
      <Label
        anchor={[637, 438, 23]}
        x={923}
        y={703}
        title="SHORE OPERATIONS"
        detail="Fleetwide telemetry aggregation"
        width={280}
        side="right"
      />
    </g>
  </g>
);
