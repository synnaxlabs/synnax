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

const DIMENSIONS = { width: 87, height: 84 };

export const RegulatorManual = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: Props): ReactElement => (
  <Primitive.Div className={CSS.cls(className, CSS.B("regulator-manual"))} {...rest}>
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="top"
        orientation={orientation}
        left={50}
        top={2.2989}
        id="1"
      />
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={2.381}
        top={75}
        id="2"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={97.619}
        top={75}
        id="3"
      />
    </Handle.Boundary>
    <Primitive.SVG
      dimensions={DIMENSIONS}
      orientation={orientation}
      color={color}
      scale={scale}
    >
      <Primitive.Path d="M43.5 20L43.5 2" strokeLinecap="round" />
      <Primitive.Path d="M19.5 2L67.5 2" strokeLinecap="round" />
      <Primitive.Path d="M43.5 63L6.35453 44.2035C4.35901 43.1937 2 44.6438 2 46.8803V79.1197C2 81.3562 4.35901 82.8063 6.35453 81.7965L43.5 63ZM43.5 63L80.6455 44.2035C82.641 43.1937 85 44.6438 85 46.8803V79.1197C85 81.3562 82.641 82.8063 80.6455 81.7965L43.5 63Z" />
      <Primitive.Path d="M60.5 40C62.1569 40 63.5231 38.6494 63.2755 37.0111C62.641 32.8129 60.681 28.8968 57.6421 25.8579C53.8914 22.1071 48.8043 20 43.5 20C38.1957 20 33.1086 22.1071 29.3579 25.8579C26.319 28.8968 24.359 32.8129 23.7245 37.0111C23.4769 38.6494 24.8431 40 26.5 40L43.5 40H60.5Z" />
      <Primitive.Path d="M60.5 40C62.1569 40 63.5231 38.6494 63.2755 37.0111C62.641 32.8129 60.681 28.8968 57.6421 25.8579C53.8914 22.1071 48.8043 20 43.5 20C38.1957 20 33.1086 22.1071 29.3579 25.8579C26.319 28.8968 24.359 32.8129 23.7245 37.0111C23.4769 38.6494 24.8431 40 26.5 40L43.5 40H60.5Z" />
      <Primitive.Line x1="43.5" y1="63" x2="43.5" y2="40" />
      <Primitive.Path d="M43.5 20V18C43.5 16.3431 44.8431 15 46.5 15H79.5C81.1569 15 82.5 16.3431 82.5 18V34.4281C82.5 35.4126 82.017 36.3344 81.2076 36.8947L43.5 63" />
    </Primitive.SVG>
  </Primitive.Div>
);
