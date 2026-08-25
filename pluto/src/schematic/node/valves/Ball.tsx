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

const DIMENSIONS = { width: 87, height: 42 };

export const Ball = ({
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
    className={CSS.cls(CSS.B("ball-valve"), className)}
    enabled={enabled}
  >
    <Handle.Linear orientation={orientation} left={2.2989} right={97.7011} />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Circle cx="43.5978" cy="20.972" r="19" />
      <Primitive.Path d="M26.5 12.8472L6.35452 2.17563C4.35901 1.16585 2 2.61598 2 4.85243V37.0918C2 39.3283 4.35901 40.7784 6.35453 39.7686L26.5 29.3472" />
      <Primitive.Path d="M60.5 28.8486L80.6455 39.5202C82.641 40.5299 85 39.0798 85 36.8434V4.60396C85 2.36751 82.641 0.917381 80.6455 1.92716L60.5 12.3486" />
    </Primitive.SVG>
  </Toggle.Button>
);
