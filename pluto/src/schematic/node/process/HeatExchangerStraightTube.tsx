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
import svg from "@/schematic/node/process/HeatExchangerStraightTube.svg?raw";

export interface Props extends Primitive.DivProps, Primitive.SVGBasedProps {}

const SPEC = Custom.createBuiltIn({
  svg,
  handles: [
    { key: "1", position: { x: 0.09, y: 0.0625 }, orientation: "top" },
    { key: "2", position: { x: 0.24, y: 0.9375 }, orientation: "bottom" },
    { key: "3", position: { x: 0.76, y: 0.0625 }, orientation: "top" },
    { key: "4", position: { x: 0.91, y: 0.9375 }, orientation: "bottom" },
    { key: "5", position: { x: 0.01508, y: 0.5 }, orientation: "left" },
    { key: "6", position: { x: 0.98492, y: 0.5 }, orientation: "right" },
  ],
});

export const HeatExchangerStraightTube = ({
  id,
  className,
  orientation = "right",
  strokeColor,
  scale = 1,
  ...rest
}: Props): ReactElement => (
  <Primitive.Div {...rest} className={CSS.cls(CSS.B("heat-exchanger-M"), className)}>
    <Custom.Drawing
      spec={SPEC}
      builtIn
      strokeColor={strokeColor}
      orientation={orientation}
      scale={scale}
    />
  </Primitive.Div>
);
