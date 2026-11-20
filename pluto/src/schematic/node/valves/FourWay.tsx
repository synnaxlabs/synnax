// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ReactElement } from "react";

import { CSS } from "@/css";
import { Custom } from "@/schematic/node/common/custom";
import type { Primitive } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";
import svg from "@/schematic/node/valves/FourWay.svg?raw";

export interface Props extends Toggle.ButtonProps, Primitive.SVGBasedProps {}

const SPEC = Custom.createBuiltIn({
  variant: "actuator",
  svg,
  handles: [
    { key: "1", position: { x: 0.12037, y: 0.5 }, orientation: "left" },
    { key: "2", position: { x: 0.87963, y: 0.5 }, orientation: "right" },
    { key: "3", position: { x: 0.5, y: 0.072916 }, orientation: "top" },
    { key: "4", position: { x: 0.5, y: 0.926084 }, orientation: "bottom" },
  ],
});

export const FourWay = ({
  className,
  orientation = "left",
  scale,
  strokeColor,
  ...rest
}: Props): ReactElement => (
  <Toggle.Button
    {...rest}
    orientation={orientation}
    className={CSS.cls(CSS.B("four-way-valve"), className)}
  >
    <Custom.Drawing
      spec={SPEC}
      builtIn
      strokeColor={strokeColor}
      orientation={orientation}
      scale={scale}
    />
  </Toggle.Button>
);
