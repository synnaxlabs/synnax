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
export interface Props extends Base.DivProps, Base.SVGBasedProps {}

const DIMENSIONS = { width: 33, height: 69 };

export const StrainerCone = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Base.Div {...rest} className={CSS(CSS.B("strainer"), className)}>
    <Handle.Linear orientation={orientation} left={6.06} right={93.04} />
    <Base.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Base.Rect x="2" y="2" width="29" height="65" rx="1" />
      <Base.Path d="M31 34.5L2.30611 2.33992" strokeDasharray="6 6" />
      <Base.Path d="M31 34.5L2.30611 66.6601" strokeDasharray="6 6" />
    </Base.SVG>
  </Base.Div>
);
