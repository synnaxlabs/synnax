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
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { Label } from "@/schematic/node/flowmeters/Label";
import { type Props } from "@/schematic/node/flowmeters/props";

const DIMENSIONS = { width: 72, height: 36 };

export const PositiveDisplacement = ({
  id,
  className,
  orientation = "right",
  color,
  scale = 1,
  ...rest
}: Props): ReactElement => (
  <Primitive.Div
    {...rest}
    className={CSS.cx(CSS.B("flowmeter-PositiveDisplacement"), className)}
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
      <Primitive.Path
        d="M41 13C41 15.4853 38.9853 17.5 36.5 17.5C34.0147 17.5 32 15.4853 32 13C32 10.5147 34.0147 8.5 36.5 8.5C38.9853 8.5 41 10.5147 41 13Z"
        strokeWidth="2"
      />
      <Primitive.Path
        d="M41 22C41 24.4853 38.9853 26.5 36.5 26.5C34.0147 26.5 32 24.4853 32 22C32 19.5147 34.0147 17.5 36.5 17.5C38.9853 17.5 41 19.5147 41 22Z"
        strokeWidth="2"
      />
      <Label color={color} />
    </Primitive.SVG>
  </Primitive.Div>
);
