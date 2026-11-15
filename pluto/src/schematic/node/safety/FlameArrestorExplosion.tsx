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
    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 60 69">' +
    '<rect x="2" y="2.5" width="56" height="64" rx="3" stroke="var(--pluto-symbol-display)"/>' +
    '<path d="M30 2.5L30 66.5" stroke="var(--pluto-symbol-display)"/>' +
    '<path d="M16 2.5L16 66.5" stroke="var(--pluto-symbol-display)"/>' +
    '<path d="M2 34.5H30" stroke="var(--pluto-symbol-display)"/>' +
    '<path d="M2 19H30" stroke="var(--pluto-symbol-display)"/>' +
    '<path d="M2 50H30" stroke="var(--pluto-symbol-display)"/>' +
    "</svg>",
  handles: [
    { key: "1", position: { x: 0.03333, y: 0.5 }, orientation: "left" },
    { key: "2", position: { x: 0.96667, y: 0.5 }, orientation: "right" },
  ],
});

export const FlameArrestorExplosion = ({
  className,
  strokeColor,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Primitive.Div {...rest} className={CSS.cls(CSS.B("flame-arrestor"), className)}>
    <Custom.Drawing
      spec={SPEC}
      strokeColor={strokeColor}
      orientation={orientation}
      scale={scale}
    />
  </Primitive.Div>
);
