// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { CSS } from "@synnaxlabs/charon";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive as Base } from "@/schematic/node/common/primitive";
export interface Props extends Base.SVGBasedProps, Base.DivProps {}

const DIMENSIONS = { width: 36, height: 48 };

export const Primitive = ({
  className,
  orientation = "left",
  color,
  scale = 1,
  ...rest
}: Props): ReactElement => (
  <Base.Div className={CSS(CSS.B("cap"), className)} {...rest}>
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={50}
        top={50}
        id="1"
      />
    </Handle.Boundary>
    <Base.SVG
      color={color}
      dimensions={DIMENSIONS}
      orientation={orientation}
      scale={scale * 0.6}
    >
      <Base.Path
        d="M3 3H30C31.6569 3 33 4.34315 33 6V42C33 43.6569 31.6569 45 30 45H3"
        strokeLinecap="round"
      />
    </Base.SVG>
  </Base.Div>
);
