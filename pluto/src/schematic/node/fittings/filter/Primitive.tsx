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
import {
  Div,
  type DivProps,
  Handle,
  HandleBoundary,
  InternalSVG,
  Path,
  type SVGBasedPrimitiveProps,
} from "@/schematic/node/common/symbol/primitives";
export interface Props extends SVGBasedPrimitiveProps, DivProps {}

const DIMENSIONS = { width: 52, height: 34 };

export const Primitive = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: Props): ReactElement => (
  <Div className={CSS(CSS.B("filter"), className)} {...rest}>
    <HandleBoundary orientation={orientation}>
      <Handle
        location="left"
        orientation={orientation}
        left={11.5385}
        top={50}
        id="1"
      />
      <Handle
        location="right"
        orientation={orientation}
        left={88.4615}
        top={50}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={DIMENSIONS}
      orientation={orientation}
      color={color}
      scale={scale}
    >
      <Path d="M6 17L24.8 2.9C25.5111 2.36667 26.4889 2.36667 27.2 2.9L46 17M6 17L24.8 31.1C25.5111 31.6333 26.4889 31.6333 27.2 31.1L46 17" />
    </InternalSVG>
  </Div>
);
