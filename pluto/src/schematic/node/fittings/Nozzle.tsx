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

export interface NozzleProps extends Primitive.DivProps, Primitive.SVGBasedProps {}

const DIMENSIONS = { width: 64, height: 126 };

export const Nozzle = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: NozzleProps): ReactElement => (
  <Primitive.Div {...rest} className={CSS.cls(CSS.B("nozzle"), className)}>
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={18.75}
        top={17.46}
        id="1"
      />
      <Handle.Handle
        location="top"
        orientation={orientation}
        left={50}
        top={1.5873}
        id="2"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={81.25}
        top={17.46}
        id="3"
      />
    </Handle.Boundary>
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Rect x="12" y="1.6667" width="40" height="40" rx="2" />
      <Primitive.Path d="M1.3333 124H62.6667" strokeLinecap="round" />
      <Primitive.Path
        d="M50.1883 41.6667C41.7748 41.6667 35.0143 46.3333 42.6923 59.3333C59.3476 87.5333 63.6327 119 62.855 124"
        strokeLinecap="round"
      />
      <Primitive.Path
        d="M14 41.6667C22.4135 41.6667 29.174 46.3333 21.496 59.3333C4.84066 87.5333 0.555555 119 1.33333 124"
        strokeLinecap="round"
      />
    </Primitive.SVG>
  </Primitive.Div>
);
