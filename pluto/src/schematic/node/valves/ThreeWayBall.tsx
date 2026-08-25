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
import { Toggle } from "@/schematic/node/common/toggle";
export interface Props extends Toggle.ButtonProps, Primitive.SVGBasedProps {}

const DIMENSIONS = { width: 87, height: 66 };

export const ThreeWayBall = ({
  color,
  className,
  orientation = "left",
  scale,
  enabled = false,
  ...rest
}: Props): ReactElement => (
  <Toggle.Button
    {...rest}
    orientation={orientation}
    className={CSS.cls(CSS.B("three-way-ball-valve"), className)}
    enabled={enabled}
  >
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={95.8}
        id="1"
      />
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={2.2989}
        top={33.1308}
        id="2"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={97.7011}
        top={33.1308}
        id="3"
      />
    </Handle.Boundary>
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Circle cx="43.5978" cy="21.722" r="19" />
      <Primitive.Path d="M26.5 13.5972L6.35452 2.92563C4.35901 1.91585 2 3.36598 2 5.60243V37.8418C2 40.0783 4.35901 41.5284 6.35453 40.5186L26.5 30.0972" />
      <Primitive.Path d="M60.5 29.5986L80.6455 40.2702C82.641 41.2799 85 39.8298 85 37.5934V5.35396C85 3.11751 82.641 1.66738 80.6455 2.67716L60.5 13.0986" />
      <Primitive.Path d="M35.3737 38.7499L24.7021 58.8954C23.6923 60.8909 25.1425 63.2499 27.3789 63.2499H59.6183C61.8548 63.2499 63.3049 60.8909 62.2951 58.8954L51.8737 38.7499" />
    </Primitive.SVG>
  </Toggle.Button>
);
