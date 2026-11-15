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
    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 48">' +
    '<path d="M24 24C24 35.1852 15.2546 44.4725 3.87626 45.8297C2.90571 45.9455 2 45.1407 2 44V4C2 2.85926 2.90571 2.0545 3.87626 2.17027C15.2546 3.52755 24 12.8148 24 24Z" stroke="var(--pluto-symbol-display)"/>' +
    '<path d="M37.9706 23.2076C38.4906 23.6079 38.4906 24.3921 37.9706 24.7924L33.86 27.9568C33.2024 28.463 32.25 27.9942 32.25 27.1644V20.8356C32.25 20.0058 33.2024 19.537 33.86 20.0432L37.9706 23.2076Z" fill="var(--pluto-symbol-display)" stroke="var(--pluto-symbol-display)"/>' +
    '<path d="M33 24H2" stroke-linecap="round" stroke="var(--pluto-symbol-display)"/>' +
    "</svg>",
  handles: [{ key: "1", position: { x: 0.05, y: 0.5 }, orientation: "left" }],
});

export const BurstDisc = ({
  className,
  strokeColor,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Primitive.Div {...rest} className={CSS.cls(CSS.B("symbol"), className)}>
    <Custom.Drawing
      spec={SPEC}
      strokeColor={strokeColor}
      orientation={orientation}
      scale={scale}
    />
  </Primitive.Div>
);
