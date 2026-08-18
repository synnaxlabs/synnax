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

export interface FilterProps extends Primitive.SVGBasedProps, Primitive.DivProps {}

const DIMENSIONS = { width: 52, height: 34 };

export const Filter = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: FilterProps): ReactElement => (
  <Primitive.Div className={CSS.cx(CSS.B("filter"), className)} {...rest}>
    <Handle.Linear orientation={orientation} left={11.5385} right={88.4615} />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      orientation={orientation}
      color={color}
      scale={scale}
    >
      <Primitive.Path d="M6 17L24.8 2.9C25.5111 2.36667 26.4889 2.36667 27.2 2.9L46 17M6 17L24.8 31.1C25.5111 31.6333 26.4889 31.6333 27.2 31.1L46 17" />
    </Primitive.SVG>
  </Primitive.Div>
);
