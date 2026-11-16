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
import { type Primitive } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";

export interface CavityProps extends Toggle.ButtonProps, Primitive.SVGBasedProps {}

const SPEC = Custom.createBuiltIn({
  variant: "actuator",
  svg:
    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">' +
    '<circle cx="32" cy="32" r="30"/>' +
    '<line x1="32" y1="2" x2="62" y2="32" class="pluto--detail"/>' +
    '<line x1="32" y1="62" x2="62" y2="32" class="pluto--detail"/>' +
    '<path d="M 17 26 C 17 20.6667 23 20.6667 23 26 C 23 31.3333 29 31.3333 29 26 C 29 20.6667 35 20.6667 35 26" stroke-linecap="round" transform="translate(6, 6)" class="pluto--detail"/>' +
    "</svg>",
  handles: [
    { key: "1", position: { x: 0.03125, y: 0.5 }, orientation: "left" },
    { key: "2", position: { x: 0.96875, y: 0.5 }, orientation: "right" },
    { key: "3", position: { x: 0.5, y: 0.03125 }, orientation: "top" },
    { key: "4", position: { x: 0.5, y: 0.96875 }, orientation: "bottom" },
  ],
});

export const Cavity = ({
  className,
  strokeColor,
  orientation = "left",
  scale,
  ...rest
}: CavityProps): ReactElement => (
  <Toggle.Button
    {...rest}
    className={CSS.cls(CSS.B("cavity-pump"), className)}
    orientation={orientation}
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
