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
export interface Props extends Primitive.DivProps, Primitive.SVGBasedProps {}

const DIMENSIONS = { width: 87, height: 102 };

export const ElectricRegulatorMotorized = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: Props): ReactElement => (
  <Primitive.Div
    className={CSS.cx(className, CSS.B("regulator-motorized"))}
    {...rest}
    orientation={orientation}
  >
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={35.6321}
        top={13.72549}
        id="2"
      />
      <Handle.Handle
        location="top"
        orientation={orientation}
        left={50}
        top={1.96078}
        id="2"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={64.3679}
        top={13.72549}
        id="3"
      />
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={24.1379}
        top={46.078}
        id="1"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={75.8621}
        top={46.078}
        id="5"
      />
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={2.2989}
        top={78.43137}
        id="6"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={97.7011}
        top={78.43137}
        id="7"
      />
    </Handle.Boundary>
    <Primitive.SVG
      dimensions={DIMENSIONS}
      orientation={orientation}
      color={color}
      scale={scale}
    >
      <Primitive.Path d="M43.5 80.5L6.35453 61.7035C4.35901 60.6937 2 62.1438 2 64.3803V96.6197C2 98.8562 4.35901 100.306 6.35453 99.2965L43.5 80.5ZM43.5 80.5L80.6455 61.7035C82.641 60.6937 85 62.1438 85 64.3803V96.6197C85 98.8562 82.641 100.306 80.6455 99.2965L43.5 80.5Z" />
      <Primitive.Line x1="43" y1="80.5" x2="43" y2="56" />
      <Primitive.Path d="M43 38V35.5C43 33.8431 44.3431 32.5 46 32.5H79C80.6569 32.5 82 33.8431 82 35.5V51.9281C82 52.9126 81.517 53.8344 80.7076 54.3947L43 80.5" />
      <Primitive.Rect x="21" y="38" width="44" height="18" rx="2" />
      <Primitive.Path d="M43 38V26" strokeLinecap="round" />
      <Primitive.Circle cx="43" cy="14" r="12" />
    </Primitive.SVG>
  </Primitive.Div>
);
