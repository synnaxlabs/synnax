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
import { Primitive } from "@/schematic/node/common/primitive";
import svg from "@/schematic/node/fittings/Cap.svg?raw";

export interface CapProps extends Primitive.SVGBasedProps, Primitive.DivProps {}

const SPEC = Custom.createBuiltIn({
  svg,
  handles: [{ key: "1", position: { x: 0.076923, y: 0.5 }, orientation: "left" }],
});

export const Cap = ({
  className,
  orientation = "left",
  strokeColor,
  scale,
  ...rest
}: CapProps): ReactElement => (
  <Primitive.Div className={CSS.cls(CSS.B("cap"), className)} {...rest}>
    <Custom.Drawing
      spec={SPEC}
      builtIn
      strokeColor={strokeColor}
      orientation={orientation}
      scale={scale}
    />
  </Primitive.Div>
);
