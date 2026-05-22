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
import { Primitive as Base } from "@/schematic/node/common/primitive";
export interface Props extends Base.SVGBasedProps, Base.DivProps {}

const DIMENSIONS = { width: 60, height: 42 };

export const IsoFilter = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: Props): ReactElement => (
  <Base.Div className={CSS(CSS.B("iso-filter"), className)} {...rest}>
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={95}
        top={50}
        id="1"
      />
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={5}
        top={50}
        id="2"
      />
    </Handle.Boundary>
    <Base.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Base.Rect x="3" y="3" width="54" height="36" rx="3" ry="3" />
      <Base.Line x1="30" y1="3" x2="30" y2="13" strokeLinecap="round" />
      <Base.Line x1="30" y1="17" x2="30" y2="25" strokeLinecap="round" />
      <Base.Line x1="30" y1="29" x2="30" y2="39" strokeLinecap="round" />
    </Base.SVG>
  </Base.Div>
);
