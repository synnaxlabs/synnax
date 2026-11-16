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
    '<path d="M62 32C62 48.5685 48.5685 62 32 62M62 32C62 15.4315 48.5685 2 32 2C15.4315 2 2 15.4315 2 32C2 48.5685 15.4315 62 32 62M62 32L32 62M32 2.00269L62.0025 32.0052" stroke-linecap="round"/>' +
    '<path d="M50.3827 20.3601C47.1902 21.7605 43.4002 22.8046 39.2752 23.4077C35.1519 24.0105 30.8103 24.1557 26.5584 23.8319C22.3057 23.508 18.2597 22.7238 14.7044 21.5418C11.1449 20.3584 8.19358 18.8149 6.03181 17.0454M6.02374 46.9613C8.18198 45.1925 11.129 43.6491 14.6841 42.465C18.2349 41.2824 22.2765 40.4968 26.5256 40.1707C30.774 39.8447 35.113 39.9871 39.2353 40.5866C43.3592 41.1864 47.15 42.2268 50.3455 43.6237" stroke-linecap="round"/>' +
    "</svg>",
  handles: [
    { key: "1", position: { x: 0.03125, y: 0.5 }, orientation: "left" },
    { key: "2", position: { x: 0.96875, y: 0.5 }, orientation: "right" },
    { key: "3", position: { x: 0.5, y: 0.03125 }, orientation: "top" },
    { key: "4", position: { x: 0.5, y: 0.96875 }, orientation: "bottom" },
  ],
});

export const Ejection = ({
  className,
  strokeColor,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Toggle.Button
    {...rest}
    className={CSS.cls(CSS.B("pump"), className)}
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
