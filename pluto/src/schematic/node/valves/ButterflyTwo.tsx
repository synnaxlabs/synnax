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

export const ButterflyTwo = ({
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
      className={CSS(CSS.B("butterfly-valve-two"), className)}
      enabled={enabled}
    >
      <Handle.Linear orientation={orientation} left={2.2989} right={97.7011} />
      <Primitive.SVG
        dimensions={DIMENSIONS}
        color={resolved}
        orientation={orientation}
        scale={scale}
      >
        <Primitive.Circle cx="43.5" cy="21" r="10" fill={color.cssString(resolved)} />
        <Primitive.Rect x="2" y="2" width="83" height="38" rx="1" />
        <Primitive.Path d="M2.29001 2.29004L84.7069 39.676" />
      </Primitive.SVG>
    </Toggle.Button>
  );
};
