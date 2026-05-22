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

const DIMENSIONS = { width: 72, height: 36 };

export const OrificePlate = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: Props): ReactElement => (
  <Base.Div className={CSS(CSS.B("orifice_plate"), className)} {...rest}>
    <Handle.Linear orientation={orientation} left={3.125} right={96.875} />
    <Base.SVG
      color={color}
      dimensions={DIMENSIONS}
      orientation={orientation}
      scale={scale}
    >
      <Base.Rect x="2.5" y="2" width="67" height="31" rx="2" />
      <Base.Path d="M24.5 2V7.5V13" strokeLinecap="round" />
      <Base.Path d="M24.5 33V22" strokeLinecap="round" />
    </Base.SVG>
  </Base.Div>
);
