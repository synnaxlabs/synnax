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

const DIMENSIONS = { width: 66, height: 30 };

export const StaticMixer = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Primitive.Div
    {...rest}
    className={CSS.cx(CSS.B("static-mixer"), className)}
    orientation={orientation}
  >
    <Handle.Rectangle
      orientation={orientation}
      left={1.5152}
      top={3.3333}
      right={98.4848}
      bottom={96.6667}
    />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Rect x="1" y="1" width="64" height="28" rx="2" ry="2" />
      <Primitive.Path
        d="M17 10C23 10 27 20 33 20C39 20 43 10 49 10"
        strokeLinecap="round"
      />
      <Primitive.Path
        d="M17 20C23 20 27 10 33 10C39 10 43 20 49 20"
        strokeLinecap="round"
      />
    </Primitive.SVG>
  </Primitive.Div>
);
