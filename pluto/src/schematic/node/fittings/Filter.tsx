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
import svg from "@/schematic/node/fittings/Filter.svg?raw";

export interface FilterProps extends Primitive.SVGBasedProps, Primitive.DivProps {}

const SPEC = Custom.createBuiltIn({
  svg,
  handles: [
    { key: "1", position: { x: 0.115385, y: 0.5 }, orientation: "left" },
    { key: "2", position: { x: 0.884615, y: 0.5 }, orientation: "right" },
  ],
});

export const Filter = ({
  className,
  orientation = "left",
  strokeColor,
  scale,
  ...rest
}: FilterProps): ReactElement => (
  <Primitive.Div className={CSS.cls(CSS.B("filter"), className)} {...rest}>
    <Custom.Drawing
      spec={SPEC}
      builtIn
      strokeColor={strokeColor}
      orientation={orientation}
      scale={scale}
    />
  </Primitive.Div>
);
