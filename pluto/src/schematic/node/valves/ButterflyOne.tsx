// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { useColor } from "@/schematic/node/common/color";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";
export interface Props extends Toggle.ButtonProps, Primitive.SVGBasedProps {}

const DIMENSIONS = { width: 87, height: 42 };

export const ButterflyOne = ({
  color: colorVal,
  className,
  orientation = "left",
  scale,
  enabled = false,
  ...rest
}: Props): ReactElement => {
  const resolved = useColor(colorVal);
  return (
    <Toggle.Button
      {...rest}
      orientation={orientation}
      className={CSS(CSS.B("butterfly-valve-one"), className)}
      enabled={enabled}
    >
      <Handle.Linear orientation={orientation} left={2.2989} right={97.7011} />
      <Primitive.SVG
        dimensions={DIMENSIONS}
        color={resolved}
        orientation={orientation}
        scale={scale}
      >
        <Primitive.Path d="M43.5 21L6.35453 2.20349C4.35901 1.19372 2 2.64384 2 4.88029V37.1197C2 39.3562 4.35901 40.8063 6.35453 39.7965L43.5 21ZM43.5 21L80.6455 2.20349C82.641 1.19372 85 2.64384 85 4.8803V37.1197C85 39.3562 82.641 40.8063 80.6455 39.7965L43.5 21Z" />
        <Primitive.Path d="M43.5 2V40" />
        <Primitive.Circle cx="43.5" cy="21" r="10" fill={color.cssString(resolved)} />
      </Primitive.SVG>
    </Toggle.Button>
  );
};
