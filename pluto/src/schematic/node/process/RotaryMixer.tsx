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

const DIMENSIONS = { width: 50, height: 33 };

export const RotaryMixer = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Toggle.Button
    {...rest}
    className={CSS.cx(CSS.B("rotary-mixer"), className)}
    orientation={orientation}
  >
    <Handle.Linear orientation={orientation} left={2} right={97.5} top={48.4849} />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Path d="M1 30V2C1 1.44772 1.44772 1 2 1H35.4545C35.7434 1 36.0181 1.12487 36.208 1.34247L48.4262 15.3425C48.7549 15.7192 48.7549 16.2808 48.4262 16.6575L36.208 30.6575C36.0181 30.8751 35.7434 31 35.4545 31H2C1.44772 31 1 30.5523 1 30Z" />
      <Primitive.Line
        x1="32"
        y1="16"
        x2="40"
        y2="16"
        strokeLinecap="round"
        className={CSS.cx(CSS.M("detail"), className)}
      />
      <Primitive.Line
        x1="32"
        y1="16"
        x2="28"
        y2="22.9282"
        strokeLinecap="round"
        className={CSS.cx(CSS.M("detail"), className)}
      />
      <Primitive.Line
        x1="32"
        y1="16"
        x2="28"
        y2="9.0717"
        strokeLinecap="round"
        className={CSS.cx(CSS.M("detail"), className)}
      />
    </Primitive.SVG>
  </Toggle.Button>
);
