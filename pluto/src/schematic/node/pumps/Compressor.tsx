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
import svg from "@/schematic/node/pumps/Compressor.svg?raw";

export interface CompressorProps extends Toggle.ButtonProps, Primitive.SVGBasedProps {}

const SPEC = Custom.createBuiltIn({
  variant: "actuator",
  svg,
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
