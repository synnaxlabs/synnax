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

export interface Props extends Primitive.DivProps, Primitive.SVGBasedProps {}

const SPEC = Custom.createBuiltIn({
  svg:
    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 33 69">' +
    '<path d="M16.5 2.5L16.5 66.5M2.5 34.5H30.5M2.5 18.9848H30.5M2.5 50.0152H30.5M5.3 66.5H27.7C29.2464 66.5 30.5 65.1976 30.5 63.5909V5.40909C30.5 3.80244 29.2464 2.5 27.7 2.5H5.3C3.7536 2.5 2.5 3.80245 2.5 5.40909V63.5909C2.5 65.1976 3.7536 66.5 5.3 66.5Z" stroke-linecap="round"/>' +
    "</svg>",
  handles: [
    { key: "1", position: { x: 0.07575, y: 0.5 }, orientation: "left" },
    { key: "2", position: { x: 0.92425, y: 0.5 }, orientation: "right" },
  ],
});

export const FlameArrestor = ({
  className,
  strokeColor,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Primitive.Div {...rest} className={CSS.cls(CSS.B("flame-arrestor"), className)}>
    <Custom.Drawing
      spec={SPEC}
      builtIn
      strokeColor={strokeColor}
      orientation={orientation}
      scale={scale}
    />
  </Primitive.Div>
);
