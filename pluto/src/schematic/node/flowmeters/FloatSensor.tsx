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

export const FloatSensor = ({
  id,
  className,
  orientation = "right",
  color: colorVal,
  scale = 1,
  ...rest
}: Props): ReactElement => (
  <Base.Div {...rest} className={CSS(CSS.B("flowmeter-FloatSensor"), className)}>
    <Handle.Rectangle
      orientation={orientation}
      left={4}
      top={6}
      right={98}
      bottom={91}
    />
    <Base.SVG
      dimensions={DIMENSIONS}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Base.Rect x="2" y="2" width="67" height="31" rx="2" />
      <Base.Path d="M25 8H46" strokeLinecap="round" />
      <Base.Path d="M31 27H40" strokeLinecap="round" />
      <Base.Path d="M31 27L25.046 8.11641" strokeLinecap="round" />
      <Base.Path d="M40 27L45.954 8.11641" strokeLinecap="round" />
      <Label color={colorVal} />
    </Base.SVG>
  </Base.Div>
);
