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
import svg from "@/schematic/node/valves/ElectricRegulator.svg?raw";

export interface Props extends Primitive.DivProps, Primitive.SVGBasedProps {}

const SPEC = Custom.createBuiltIn({
  svg,
  handles: [
    { key: "1", position: { x: 0.0229885, y: 0.69565 }, orientation: "left" },
    { key: "2", position: { x: 0.9770115, y: 0.69565 }, orientation: "right" },
    { key: "3", position: { x: 0.231479, y: 0.2361 }, orientation: "left" },
    { key: "4", position: { x: 0.5, y: 0.1111 }, orientation: "top" },
    { key: "5", position: { x: 0.768521, y: 0.2361 }, orientation: "right" },
  ],
});

export const ElectricRegulator = ({
  className,
  orientation = "left",
  strokeColor,
  scale,
  ...rest
}: Props): ReactElement => (
  <Primitive.Div className={CSS.cls(className, CSS.B("regulator"))} {...rest}>
    <Custom.Drawing
      spec={SPEC}
      builtIn
      strokeColor={strokeColor}
      orientation={orientation}
      scale={scale}
    />
  </Primitive.Div>
);
