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

export interface VentProps extends Primitive.SVGBasedProps, Primitive.DivProps {}

const DIMENSIONS = { width: 22, height: 32 };

export const Vent = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: VentProps): ReactElement => (
  <Primitive.Div className={CSS.cls(CSS.B("vent"), className)} {...rest}>
    <Handle.Linear orientation={orientation} left={22.7273} right={80} />
    <Primitive.SVG
      color={color}
      dimensions={DIMENSIONS}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Path
        d="M5 3L16.6325 13.8016C17.9107 14.9885 17.9107 17.0115 16.6325 18.1984L5 29"
        strokeLinecap="round"
      />
    </Primitive.SVG>
  </Primitive.Div>
);
