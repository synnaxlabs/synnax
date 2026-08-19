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

export interface OrificeProps extends Primitive.DivProps, Primitive.SVGBasedProps {}

const DIMENSIONS = { width: 70, height: 34 };

export const Orifice = ({
  className,
  orientation = "left",
  scale,
  color,
  ...rest
}: OrificeProps): ReactElement => (
  <Primitive.Div className={CSS.cls(CSS.B("orifice"), className)} {...rest}>
    <Handle.Linear orientation={orientation} left={2.8571} right={97.1429} />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      orientation={orientation}
      color={color}
      scale={scale}
    >
      <Primitive.Path
        d="M2 2.5C27.451 21.3469 60.0915 12.0132 68 2.5"
        strokeLinecap="round"
      />
      <Primitive.Path
        d="M2 32.5C27.451 13.6531 60.0915 22.9868 68 32.5"
        strokeLinecap="round"
      />
    </Primitive.SVG>
  </Primitive.Div>
);
