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
  Cylinder,
  DARK,
  Face,
  LIGHT,
  METAL,
  SIGNAL,
  Wire,
} from "@/components/IndustrialGeometry";

import { Label, Node, Pad, Rack, Trace } from "./DeploymentGeometry";

const BAYS = [
  { x: 90, y: 66, label: "Q.01", open: false },
  { x: 310, y: 66, label: "Q.02", open: false },
  { x: 530, y: 66, label: "Q.03", open: false },
  { x: 175, y: 305, label: "Q.04", open: true },
  { x: 395, y: 305, label: "Q.05", open: false },
] as const;

// The cryostat hangs from its room-temperature flange. Four columns carry the
// support frame, while measurement electronics and the compressor sit beside it.
const Cryostat = ({
  x,
  y,
  open,
}: {
  x: number;
  y: number;
  open: boolean;
}): ReactElement => (
  <g className="scene-physical">
    <Box
      x={x - 51}
      y={y - 46}
      width={153}
      depth={116}
      height={4}
      material={{ top: "#28384a", left: "#1a293a", right: "#132131" }}
    />
    {[-39, 35].map((dx) => (
      <g key={dx}>
        <Box
          x={x + dx - 5}
          y={y - 42}
          z={4}
          width={15}
          depth={13}
          height={4}
          material={LIGHT}
        />
        <Box
          x={x + dx}
          y={y - 38}
          z={8}
          width={5}
          depth={5}
          height={153}
          material={METAL}
        />
      </g>
    ))}
    <Box
      x={x - 42}
      y={y - 41}
      z={153}
      width={85}
      depth={7}
      height={8}
      material={LIGHT}
    />
    {open ? (
      <g>
        {[-15, 15].map((dx) => (
          <Wire
            key={dx}
            vertices={[
              [x + dx, y - 10, 40],
              [x + dx, y - 10, 153],
            ]}
            className="scene-railing"
          />
        ))}
        {[
          { z: 35, radius: 15 },
          { z: 58, radius: 20 },
          { z: 84, radius: 24 },
          { z: 112, radius: 27 },
          { z: 140, radius: 30 },
        ].map(({ z, radius }) => (
          <Cylinder key={z} x={x} y={y} z={z} radius={radius} height={5} />
        ))}
        {[-14, -7, 0, 7, 14].map((dx) => (
          <Wire
            key={dx}
            vertices={[
              [x + dx, y + 17, 153],
              [x + dx, y + 17, 117],
              [x + dx * 0.8, y + 15, 89],
              [x + dx * 0.6, y + 11, 63],
              [x + dx * 0.5, y + 8, 40],
            ]}
            className="scene-detail"
          />
        ))}
        <Box
          x={x - 7}
          y={y - 6}
          z={24}
          width={14}
          depth={12}
          height={10}
          material={METAL}
        />
      </g>
    ) : (
      <g>
        <Cylinder x={x} y={y} z={29} radius={28} height={126} />
        <Cylinder x={x} y={y} z={26} radius={30} height={6} dark />
        <Cylinder x={x} y={y} z={89} radius={29} height={3} />
      </g>
    )}
    {[-39, 35].map((dx) => (
      <g key={dx}>
        <Box
          x={x + dx - 5}
          y={y + 30}
          z={4}
          width={15}
          depth={13}
          height={4}
          material={LIGHT}
        />
        <Box
          x={x + dx}
          y={y + 34}
          z={8}
          width={5}
          depth={5}
          height={153}
          material={METAL}
        />
      </g>
    ))}
    <Box
      x={x - 42}
      y={y + 33}
      z={153}
      width={85}
      depth={7}
      height={8}
      material={METAL}
    />
    {[-42, 36].map((dx) => (
      <Box
        key={dx}
        x={x + dx}
        y={y - 34}
        z={153}
        width={7}
        depth={67}
        height={8}
        material={LIGHT}
      />
    ))}
    <Cylinder x={x} y={y} z={157} radius={34} height={7} />
    <Cylinder x={x - 11} y={y - 7} z={164} radius={7} height={23} dark />
    <Cylinder x={x + 12} y={y + 6} z={164} radius={5} height={13} />
    <g className="deployment-detail">
      {[-20, -10, 0, 10, 20].map((dx) => (
        <Cylinder key={dx} x={x + dx} y={y - 20} z={164} radius={2} height={4} dark />
      ))}
    </g>
    <Rack x={x + 64} y={y - 22} z={4} width={29} depth={32} height={82} />
    <Box
      x={x + 62}
      y={y + 28}
      z={4}
      width={32}
      depth={30}
      height={33}
      material={METAL}
    />
    {[12, 18, 24, 30].map((z) => (
      <Wire
        key={z}
        vertices={[
          [x + 67, y + 58, z],
          [x + 89, y + 58, z],
        ]}
        className="scene-rack-slot"
      />
    ))}
    <Wire
      vertices={[
        [x - 11, y - 7, 187],
        [x - 11, y - 28, 191],
        [x + 48, y - 28, 191],
        [x + 48, y + 43, 191],
        [x + 48, y + 43, 26],
        [x + 62, y + 43, 26],
      ]}
      className="scene-railing"
    />
    <Wire
      vertices={[
        [x + 19, y - 13, 165],
        [x + 50, y - 13, 165],
        [x + 50, y - 13, 78],
        [x + 64, y - 13, 78],
      ]}
      className="scene-detail"
    />
  </g>
);

const ExperimentControl = (): ReactElement => (
  <g className="scene-physical">
    <Box x={534} y={315} width={109} depth={109} height={4} material={DARK} />
    <Rack x={540} y={320} z={4} width={38} depth={37} height={103} />
    <Rack x={589} y={320} z={4} width={38} depth={37} height={103} />
    <Box x={551} y={388} z={4} width={5} depth={27} height={36} material={METAL} />
    <Box x={621} y={388} z={4} width={5} depth={27} height={36} material={METAL} />
    <Box x={545} y={382} z={40} width={87} depth={38} height={4} material={LIGHT} />
    <Box x={579} y={391} z={44} width={19} depth={10} height={3} material={DARK} />
    <Box x={586} y={391} z={47} width={4} depth={4} height={9} material={METAL} />
    <Box x={562} y={388} z={55} width={54} depth={4} height={30} material={DARK} />
    <Face
      vertices={[
        [565, 392.5, 59],
        [613, 392.5, 59],
        [613, 392.5, 81],
        [565, 392.5, 81],
      ]}
      fill="#233d5a"
    />
    <Wire
      vertices={[
        [568, 393, 65],
        [573, 393, 65],
        [577, 393, 74],
        [582, 393, 63],
        [589, 393, 73],
        [594, 393, 65],
        [608, 393, 65],
      ]}
      className="scene-signal-path"
    />
    <Box x={576} y={409} z={44} width={25} depth={7} height={1} material={METAL} />
    <Box x={609} y={408} z={44} width={4} depth={5} height={1} material={SIGNAL} />
  </g>
);

export const QuantumScene = (): ReactElement => (
  <g>
    <Pad x={20} y={10} width={704} depth={492} />
    <g className="deployment-detail">
      {[218, 438, 658].map((x) => (
        <Wire
          key={x}
          vertices={[
            [x, 20, 0],
            [x, 487, 0],
          ]}
          className="scene-site-edge"
        />
      ))}
      {[220, 459].map((y) => (
        <Wire
          key={y}
          vertices={[
            [33, y, 0],
            [710, y, 0],
          ]}
          className="scene-site-edge"
        />
      ))}
      {BAYS.map(({ x, y, label }) => (
        <Wire
          key={label}
          vertices={[
            [x - 59, y - 52, 0],
            [x + 116, y - 52, 0],
            [x + 116, y + 137, 0],
            [x - 59, y + 137, 0],
            [x - 59, y - 52, 0],
          ]}
          className="scene-site-edge"
        />
      ))}
    </g>
    <g className="scene-local-links">
      {BAYS.map(({ x, y, label }) => (
        <Trace
          key={label}
          muted
          vertices={[
            [x + 78, y + 12, 8],
            [x + 105, y + 12, 8],
            [x + 105, y + 118, 8],
            [x + 48, y + 118, 8],
          ]}
        />
      ))}
    </g>
    <g>
      {BAYS.map(({ x, y, label }, index) => (
        <Trace
          key={label}
          delay={index * 0.7}
          vertices={
            index < 2
              ? [
                  [x + 48, y + 118, 7],
                  [x + 128, y + 118, 7],
                  [x + 128, 228, 7],
                  [x + 213, 228, 7],
                  [x + 213, 470, 7],
                  [580, 470, 7],
                ]
              : [
                  [x + 48, y + 118, 7],
                  [x + 128, y + 118, 7],
                  [x + 128, 470, 7],
                  [580, 470, 7],
                ]
          }
        />
      ))}
    </g>
    {BAYS.map((bay) => (
      <Cryostat key={bay.label} {...bay} />
    ))}
    <ExperimentControl />
    <Trace
      vertices={[
        [580, 470, 7],
        [680, 470, 7],
        [680, 372, 7],
        [608, 372, 7],
        [608, 357, 7],
      ]}
      reverse
    />
    {BAYS.map(({ x, y, label }) => (
      <Node key={label} x={x + 48} y={y + 118} label={label} />
    ))}
    <Node x={580} y={470} label="F.01" />
    <g className="scene-annotations">
      <Label
        anchor={[90, 66, 164]}
        x={282}
        y={-18}
        title="DILUTION REFRIGERATOR FLEET"
        detail="Cryostats · instruments · local nodes"
        width={266}
      />
      <Label
        anchor={[175, 305, 84]}
        x={58}
        y={220}
        title="CRYOGENIC MEASUREMENT"
        detail="Thermometry · vacuum · RF"
        width={235}
      />
      <Label
        anchor={[608, 338, 107]}
        x={1096}
        y={564}
        title="ONE EXPERIMENT LAYER"
        detail="Acquisition · control · analysis"
        width={270}
        side="right"
      />
    </g>
  </g>
);
