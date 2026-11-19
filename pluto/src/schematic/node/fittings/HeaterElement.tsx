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
import svg from "@/schematic/node/fittings/HeaterElement.svg?raw";

export interface HeaterElementProps
  extends Primitive.DivProps, Primitive.SVGBasedProps {}

const SPEC = Custom.createBuiltIn({
  svg,
  handles: [
    { key: "3", position: { x: 0.119, y: 0.0555555556 }, orientation: "top" },
    { key: "4", position: { x: 0.119, y: 0.94 }, orientation: "bottom" },
  ],
});

export const HeaterElement = ({
  className,
  orientation = "left",
  strokeColor,
  scale,
  ...rest
}: HeaterElementProps): ReactElement => (
  <Primitive.Div className={CSS.cls(CSS.B("heater-element"), className)} {...rest}>
    <Custom.Drawing
      spec={SPEC}
      builtIn
      strokeColor={strokeColor}
      orientation={orientation}
      scale={scale}
    />
  </Primitive.Div>
);
