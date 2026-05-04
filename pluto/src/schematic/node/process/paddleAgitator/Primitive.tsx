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
import {
  Handle,
  HandleBoundary,
  InternalSVG,
  Line,
  Rect,
  Toggle,
} from "@/schematic/node/common/symbol/primitives";
import { type Props as AgitatorProps } from "@/schematic/node/process/agitator/Primitive";
export interface Props extends AgitatorProps {}

const DIMENSIONS = { width: 87, height: 87 };

export const Primitive = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: Props): ReactElement => (
  <Toggle {...rest} className={CSS(CSS.B("agitator"))}>
    <HandleBoundary orientation={orientation}>
      <Handle location="top" orientation={orientation} left={51} top={2} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Line x1="43" y1="1" x2="43" y2="49" />
      <Rect x="3" y="49" width="80" height="34" rx="3" />
      <Line
        x1="3.8"
        y1="82.1"
        x2="43"
        y2="49"
        className={CSS(CSS.M("detail"), className)}
        strokeLinecap="round"
      />
      <Line
        x1="43"
        y1="49"
        x2="43"
        y2="83"
        className={CSS(CSS.M("detail"), className)}
        strokeLinecap="round"
      />
      <Line
        x1="43"
        y1="83"
        x2="82.2"
        y2="49.9"
        className={CSS(CSS.M("detail"), className)}
        strokeLinecap="round"
      />
    </InternalSVG>
  </Toggle>
);
