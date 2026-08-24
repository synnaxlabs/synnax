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

const DIMENSIONS = { width: 64, height: 64 };

export const Piston = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Toggle.Button
    {...rest}
    className={CSS.cls(CSS.B("piston-pump"), className)}
    orientation={orientation}
  >
    <Handle.Rectangle
      orientation={orientation}
      left={3.125}
      top={3.125}
      right={96.875}
      bottom={96.875}
    />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Circle cx="32" cy="32" r="30" />
      <Primitive.Line
        x1="32"
        y1="2"
        x2="62"
        y2="32"
        className={CSS.cls(CSS.M("detail"), className)}
      />
      <Primitive.Line
        x1="32"
        y1="62"
        x2="62"
        y2="32"
        className={CSS.cls(CSS.M("detail"), className)}
      />
      <Primitive.Path
        d="M 23 32, h 16, m 0 -8, v 16"
        className={CSS.cls(CSS.M("detail"), className)}
        strokeLinecap="round"
      />
    </Primitive.SVG>
  </Toggle.Button>
);
