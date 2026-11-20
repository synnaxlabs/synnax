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
import svg from "@/schematic/node/flowmeters/PositiveDisplacement.svg?raw";
import type { Props } from "@/schematic/node/flowmeters/props";

const SPEC = Custom.createBuiltIn({
  svg,
  handles: [
    { key: "1", position: { x: 0.04, y: 0.5 }, orientation: "left" },
    { key: "2", position: { x: 0.98, y: 0.5 }, orientation: "right" },
    { key: "3", position: { x: 0.5, y: 0.06 }, orientation: "top" },
    { key: "4", position: { x: 0.5, y: 0.91 }, orientation: "bottom" },
  ],
});

export const PositiveDisplacement = ({
  id,
  className,
  orientation = "right",
  strokeColor,
  scale = 1,
  ...rest
}: Props): ReactElement => (
  <Primitive.Div
    {...rest}
    className={CSS.cls(CSS.B("flowmeter-PositiveDisplacement"), className)}
  >
    <Custom.Drawing
      spec={SPEC}
      builtIn
      strokeColor={strokeColor}
      orientation={orientation}
      scale={scale}
    />
  </Primitive.Div>
);
