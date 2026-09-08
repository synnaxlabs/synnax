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
import { type Props as AgitatorProps } from "@/schematic/node/process/Agitator";
export interface Props extends AgitatorProps {}

const DIMENSIONS = { width: 87, height: 87 };

export const HelicalAgitator = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: Props): ReactElement => (
  <Toggle.Button {...rest} className={CSS.cls(CSS.B("agitator"))}>
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="top"
        orientation={orientation}
        left={51}
        top={2}
        id="4"
      />
    </Handle.Boundary>
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Line x1={43} y1={1} x2={43} y2={60} strokeLinecap="round" />
      <Primitive.Path
        d="M5.375 36L70.8204 48.7138C74.0584 49.3428 74.0573 53.9765 70.8189 54.6039L14.7952 65.4584C11.5729 66.0827 11.5494 70.6856 14.7651 71.3429L81.5208 84.9873"
        strokeLinecap="round"
      />
    </Primitive.SVG>
  </Toggle.Button>
);
