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
import svg from "@/schematic/node/safety/FlameArrestorExplosion.svg?raw";

export interface Props extends Primitive.DivProps, Primitive.SVGBasedProps {}

const SPEC = Custom.createBuiltIn({
  svg,
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
      builtIn
      strokeColor={strokeColor}
      orientation={orientation}
      scale={scale}
    />
  </Primitive.Div>
);
