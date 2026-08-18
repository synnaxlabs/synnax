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

export interface Props extends Primitive.DivProps, Primitive.SVGBasedProps {}

const DIMENSIONS = { width: 36, height: 72 };

export const IsoBurstDisc = ({
  className,
  color,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Primitive.Div {...rest} className={CSS.cx(CSS.B("symbol"), className)}>
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={11.1111}
        top={50}
        id="1"
      />
    </Handle.Boundary>
    <Primitive.SVG
      dimensions={DIMENSIONS} // Reduced to ~2/3 of original size (50x108)
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Rect x="4" y="4" width="28" height="64" rx="2" strokeWidth="2" />
      <Primitive.Path
        d="M13 68V47C13 46.4477 13.4489 45.9892 13.9928 45.8933C16.1351 45.5152 21 43.7981 21 36C21 28.2019 16.1351 26.4848 13.9928 26.1068C13.4489 26.0108 13 25.5523 13 25V4"
        strokeWidth="2"
      />
    </Primitive.SVG>
  </Primitive.Div>
);
