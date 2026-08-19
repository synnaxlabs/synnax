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

export interface ISOFilterProps extends Primitive.SVGBasedProps, Primitive.DivProps {}

const DIMENSIONS = { width: 60, height: 42 };

export const ISOFilter = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: ISOFilterProps): ReactElement => (
  <Primitive.Div className={CSS.cls(CSS.B("iso-filter"), className)} {...rest}>
    <Handle.Linear orientation={orientation} left={5} right={95} />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Rect x="3" y="3" width="54" height="36" rx="3" ry="3" />
      <Primitive.Line x1="30" y1="3" x2="30" y2="13" strokeLinecap="round" />
      <Primitive.Line x1="30" y1="17" x2="30" y2="25" strokeLinecap="round" />
      <Primitive.Line x1="30" y1="29" x2="30" y2="39" strokeLinecap="round" />
    </Primitive.SVG>
  </Primitive.Div>
);
