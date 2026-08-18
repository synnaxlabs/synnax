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

export const CrossBeamAgitator = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: Props): ReactElement => (
  <Toggle.Button {...rest} className={CSS.cx(CSS.B("agitator"))}>
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
      <Primitive.Line x1="43" y1="1" x2="43" y2="49" strokeLinecap="round" />
      <Primitive.Line x1="3" y1="49" x2="83" y2="49" strokeLinecap="round" />
      <Primitive.Line x1="3" y1="83" x2="83" y2="83" strokeLinecap="round" />
      <Primitive.Line x1="43" y1="49" x2="43" y2="83" strokeLinecap="round" />
      {/* We need this rectangle here because so that when a user hovers above to click the agitator, the rectangle changes color */}
      <Primitive.Rect x="3" y="49" width="80" height="34" strokeWidth={0} />
    </Primitive.SVG>
  </Toggle.Button>
);
