// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS } from "@synnaxlabs/charon/css";
import { type ReactElement } from "react";

import { Handle } from "@/schematic/node/common/handle";
import { Primitive as Base } from "@/schematic/node/common/primitive";
export interface Props extends Base.DivProps, Base.SVGBasedProps {}

const DIMENSIONS = { width: 70, height: 34 };

export const Primitive = ({
  className,
  orientation = "left",
  scale,
  color,
  ...rest
}: Props): ReactElement => (
  <Base.Div className={CSS(CSS.B("orifice"), className)} {...rest}>
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={2.8571}
        top={50}
        id="1"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={97.1429}
        top={50}
        id="2"
      />
    </Handle.Boundary>
    <Base.SVG
      dimensions={DIMENSIONS}
      orientation={orientation}
      color={color}
      scale={scale}
    >
      <Base.Path
        d="M2 2.5C27.451 21.3469 60.0915 12.0132 68 2.5"
        strokeLinecap="round"
      />
      <Base.Path
        d="M2 32.5C27.451 13.6531 60.0915 22.9868 68 32.5"
        strokeLinecap="round"
      />
    </Base.SVG>
  </Base.Div>
);
