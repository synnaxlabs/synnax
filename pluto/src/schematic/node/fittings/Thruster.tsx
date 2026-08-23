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

export interface ThrusterProps extends Primitive.DivProps, Primitive.SVGBasedProps {}

const DIMENSIONS = { width: 81, height: 42 };

export const Thruster = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: ThrusterProps): ReactElement => (
  <Primitive.Div {...rest} className={CSS.cls(CSS.B("thruster"), className)}>
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={3.125}
        top={50}
        id="1"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={96.875}
        top={50}
        id="2"
      />
      <Handle.Handle
        location="top"
        orientation={orientation}
        left={25.3}
        top={4.76}
        id="3"
      />
      <Handle.Handle
        location="bottom"
        orientation={orientation}
        left={25.3}
        top={95.24}
        id="4"
      />
    </Handle.Boundary>
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Rect x="2.5" y="2" width="38" height="38" rx="3" />
      <Primitive.Path d="M78.5 37.5117V4.51172" />
      <Primitive.Path d="M40.5 11.5L76.0072 2.6232" />
      <Primitive.Path d="M40.5 30.5L76.0072 39.3768" />
      <Primitive.Path d="M75.6192 2.71597C75.9231 2.56695 76.2597 2.49745 76.5977 2.51399C76.9357 2.53052 77.264 2.63256 77.5518 2.81053C77.8397 2.98851 78.0776 3.23661 78.2435 3.53161C78.4093 3.82661 78.4975 4.15886 78.4999 4.49726" />
      <Primitive.Path d="M78.4994 37.5101C78.4914 37.8382 78.4028 38.1592 78.2414 38.445C78.0801 38.7307 77.8509 38.9723 77.574 39.1486C77.2972 39.3248 76.9813 39.4302 76.6541 39.4555C76.3269 39.4808 75.9986 39.4252 75.698 39.2936" />
    </Primitive.SVG>
  </Primitive.Div>
);
