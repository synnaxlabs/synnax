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

const DIMENSIONS = { width: 87, height: 48 };

export const Manual = ({
  className,
  orientation = "left",
  color,
  scale,
  enabled = false,
  ...rest
}: Props): ReactElement => (
  <Toggle.Button
    {...rest}
    orientation={orientation}
    className={CSS.cx(CSS.B("manual-valve"), className)}
    enabled={enabled}
  >
    <Handle.Linear
      orientation={orientation}
      left={2.2989}
      right={97.7011}
      top={56.25}
    />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Line x1="43.5" y1="27" x2="43.5" y2="1" />
      <Primitive.Path d="M19.64 1 L66.68 1" strokeLinecap="round" />
      <Primitive.Path d="M43.5 27L6.35453 8.20349C4.35901 7.19372 2 8.64384 2 10.8803V43.1197C2 45.3562 4.35901 46.8063 6.35453 45.7965L43.5 27ZM43.5 27L80.6455 8.20349C82.641 7.19372 85 8.64384 85 10.8803V43.1197C85 45.3562 82.641 46.8063 80.6455 45.7965L43.5 27Z" />
    </Primitive.SVG>
  </Toggle.Button>
);
