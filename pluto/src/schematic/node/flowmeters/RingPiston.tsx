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
import { Label } from "@/schematic/node/flowmeters/Label";
export interface Props extends Base.DivProps, Base.SVGBasedProps {}

const DIMENSIONS = { width: 71, height: 36 };

export const RingPiston = ({
  id,
  className,
  orientation = "right",
  color: colorVal,
  scale = 1,
  ...rest
}: Props): ReactElement => (
  <Base.Div {...rest} className={CSS(CSS.B("flowmeter-RingPiston"), className)}>
    <Handle.Rectangle
      orientation={orientation}
      left={1.6667}
      top={5.714}
      right={98}
      bottom={94.386}
    />
    <Base.SVG
      dimensions={DIMENSIONS}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Base.Rect x="2" y="2" width="67" height="31" rx="2" />
      <Base.Circle cx="36.5" cy="17.5" r="10.5" strokeWidth="2" />
      <Base.Circle cx="36.5" cy="21.5" r="6.5" strokeWidth="2" />
      <Label position={{ x: 56, y: 29 }} color={colorVal} />
    </Base.SVG>
  </Base.Div>
);
