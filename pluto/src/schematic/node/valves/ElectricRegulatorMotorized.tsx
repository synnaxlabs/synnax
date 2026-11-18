// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Custom } from "@/schematic/node/common/custom";
import { Primitive } from "@/schematic/node/common/primitive";
import svg from "@/schematic/node/valves/ElectricRegulatorMotorized.svg?raw";

export interface Props extends Primitive.DivProps, Primitive.SVGBasedProps {}

const SPEC = Custom.createBuiltIn({
  svg,
  handles: [
    { key: "2", position: { x: 0.356321, y: 0.1372549 }, orientation: "left" },
    { key: "4", position: { x: 0.5, y: 0.0196078 }, orientation: "top" },
    { key: "3", position: { x: 0.643679, y: 0.1372549 }, orientation: "right" },
    { key: "1", position: { x: 0.241379, y: 0.46078 }, orientation: "left" },
    { key: "5", position: { x: 0.758621, y: 0.46078 }, orientation: "right" },
    { key: "6", position: { x: 0.022989, y: 0.7843137 }, orientation: "left" },
    { key: "7", position: { x: 0.977011, y: 0.7843137 }, orientation: "right" },
  ],
});

export const ElectricRegulatorMotorized = ({
  className,
  orientation = "left",
  strokeColor,
  scale,
  ...rest
}: Props): ReactElement => (
  <Primitive.Div
    className={CSS.cls(className, CSS.B("regulator-motorized"))}
    {...rest}
    orientation={orientation}
  >
    <Custom.Drawing
      spec={SPEC}
      builtIn
      strokeColor={strokeColor}
      orientation={orientation}
      scale={scale}
    />
  </Primitive.Div>
);
