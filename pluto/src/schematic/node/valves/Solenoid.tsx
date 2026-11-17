// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/valves/Solenoid.css";

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Custom } from "@/schematic/node/common/custom";
import { type Primitive } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";
import svg from "@/schematic/node/valves/Solenoid.svg?raw";

export interface Props extends Toggle.ButtonProps, Primitive.SVGBasedProps {
  normallyOpen?: boolean;
}

const SPEC = Custom.createBuiltIn({
  variant: "actuator",
  svg,
  handles: [
    { key: "1", position: { x: 0.022989, y: 0.695652 }, orientation: "left" },
    { key: "2", position: { x: 0.977011, y: 0.695652 }, orientation: "right" },
    { key: "3", position: { x: 0.333333, y: 0.177778 }, orientation: "left" },
    { key: "4", position: { x: 0.5, y: 0.028986 }, orientation: "top" },
    { key: "5", position: { x: 0.666667, y: 0.177778 }, orientation: "right" },
  ],
});

export const Solenoid = ({
  className,
  strokeColor,
  orientation = "left",
  normallyOpen = false,
  scale,
  ...rest
}: Props): ReactElement => (
  <Toggle.Button
    className={CSS.cls(
      CSS.B("solenoid-valve"),
      normallyOpen && CSS.M("normally-open"),
      className,
    )}
    orientation={orientation}
    {...rest}
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
