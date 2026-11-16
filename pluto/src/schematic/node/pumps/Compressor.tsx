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

export interface CompressorProps extends Toggle.ButtonProps, Primitive.SVGBasedProps {}

const SPEC = Custom.createBuiltIn({
  variant: "actuator",
  svg:
    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 66 66">' +
    '<path d="M17 7.59998L59.5394 18.9984M17 58.3984L59.5394 47M63 33C63 36.9397 62.224 40.8407 60.7164 44.4805C59.2087 48.1203 56.999 51.4274 54.2132 54.2132C51.4274 56.999 48.1203 59.2087 44.4805 60.7164C40.8407 62.224 36.9397 63 33 63C29.0603 63 25.1593 62.224 21.5195 60.7164C17.8797 59.2087 14.5726 56.999 11.7868 54.2132C9.00104 51.4274 6.79125 48.1203 5.28361 44.4805C3.77597 40.8407 3 36.9397 3 33C3 29.0603 3.77597 25.1593 5.28362 21.5195C6.79126 17.8797 9.00104 14.5726 11.7868 11.7868C14.5726 9.00104 17.8797 6.79125 21.5195 5.28361C25.1593 3.77597 29.0603 3 33 3C36.9397 3 40.8407 3.77597 44.4805 5.28362C48.1203 6.79126 51.4274 9.00104 54.2132 11.7868C56.999 14.5726 59.2087 17.8797 60.7164 21.5195C62.224 25.1593 63 29.0603 63 33Z" stroke-linecap="round"/>' +
    "</svg>",
  handles: [
    { key: "1", position: { x: 0.0455, y: 0.5 }, orientation: "left" },
    { key: "2", position: { x: 0.9545, y: 0.5 }, orientation: "right" },
    { key: "3", position: { x: 0.5, y: 0.0455 }, orientation: "top" },
    { key: "4", position: { x: 0.5, y: 0.9545 }, orientation: "bottom" },
  ],
});

export const Compressor = ({
  className,
  strokeColor,
  orientation = "left",
  scale,
  ...rest
}: CompressorProps): ReactElement => (
  <Toggle.Button
    {...rest}
    className={CSS.cls(CSS.B("compressor"), className)}
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
