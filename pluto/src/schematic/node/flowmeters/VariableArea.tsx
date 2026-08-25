// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS } from "@/css";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { Label } from "@/schematic/node/flowmeters/Label";
import { type Props } from "@/schematic/node/flowmeters/props";

const DIMENSIONS = { width: 71, height: 36 };

export const VariableArea = ({
  id,
  className,
  orientation = "right",
  color,
  scale = 1,
  ...rest
}: Props) => (
  <Primitive.Div
    {...rest}
    className={CSS.cls(CSS.B("flowmeter-VariableArea"), className)}
  >
    <Handle.Rectangle
      orientation={orientation}
      left={4}
      top={6}
      right={98}
      bottom={91}
    />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Rect x="2" y="2" width="67" height="31" rx="2" />
      <Primitive.Path d="M46 10V25" />
      <Primitive.Path d="M23 13V22" />
      <Primitive.Path d="M23 13L46 10" />
      <Primitive.Path d="M23 22L46 25" />
      <Label color={color} />
    </Primitive.SVG>
  </Primitive.Div>
);
