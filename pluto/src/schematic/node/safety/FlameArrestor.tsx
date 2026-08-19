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

const DIMENSIONS = { width: 33, height: 69 };

export const FlameArrestor = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Primitive.Div {...rest} className={CSS.cls(CSS.B("flame-arrestor"), className)}>
    <Handle.Linear orientation={orientation} left={7.575} right={92.425} />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Path
        d="M16.5 2.5L16.5 66.5M2.5 34.5H30.5M2.5 18.9848H30.5M2.5 50.0152H30.5M5.3 66.5H27.7C29.2464 66.5 30.5 65.1976 30.5 63.5909V5.40909C30.5 3.80244 29.2464 2.5 27.7 2.5H5.3C3.7536 2.5 2.5 3.80245 2.5 5.40909V63.5909C2.5 65.1976 3.7536 66.5 5.3 66.5Z"
        strokeLinecap="round"
      />
    </Primitive.SVG>
  </Primitive.Div>
);
