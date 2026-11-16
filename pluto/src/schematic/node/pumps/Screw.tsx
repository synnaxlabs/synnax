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

export interface Props extends Toggle.ButtonProps, Primitive.SVGBasedProps {}

const SPEC = Custom.createBuiltIn({
  variant: "actuator",
  svg:
    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">' +
    '<circle cx="32" cy="32" r="30"/>' +
    '<line x1="32" y1="2" x2="62" y2="32" class="pluto--detail"/>' +
    '<line x1="32" y1="62" x2="62" y2="32" class="pluto--detail"/>' +
    '<path d="M 0 0, L -10 -10, M 0 0, L -10 10" transform="translate(32, 32)" stroke-linecap="round" class="pluto--detail"/>' +
    '<path d="M 0 0, L -10 -10, M 0 0, L -10 10" transform="translate(42, 32)" stroke-linecap="round" class="pluto--detail"/>' +
    "</svg>",
  handles: [
    { key: "1", position: { x: 0.03125, y: 0.5 }, orientation: "left" },
    { key: "2", position: { x: 0.96875, y: 0.5 }, orientation: "right" },
    { key: "3", position: { x: 0.5, y: 0.03125 }, orientation: "top" },
    { key: "4", position: { x: 0.5, y: 0.96875 }, orientation: "bottom" },
  ],
});

export const Screw = ({
  className,
  strokeColor,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Toggle.Button
    {...rest}
    className={CSS.cls(CSS.B("screw-pump"), className)}
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
