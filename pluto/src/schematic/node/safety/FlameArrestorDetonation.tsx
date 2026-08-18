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

const DIMENSIONS = { width: 63, height: 69 };

export const FlameArrestorDetonation = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Primitive.Div {...rest} className={CSS.cx(CSS.B("flame-arrestor"), className)}>
    <Handle.Linear orientation={orientation} left={3.333} right={96.667} />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Rect x="2" y="2.5" width="28" height="64" rx="3" />
      <Primitive.Path d="M16 2.5L16 66.5" />
      <Primitive.Path d="M2 34.5H30" />
      <Primitive.Path d="M2 19H30" />
      <Primitive.Path d="M2 50H30" />
      <Primitive.Path d="M29.121 3.37903L61 34.5" />
      <Primitive.Path d="M29.12 65.62L61 34.5" />
    </Primitive.SVG>
  </Primitive.Div>
);
