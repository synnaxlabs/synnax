// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";

interface RenderProps extends Omit<schematic.NodeConfigLight, "variant"> {
  className?: string;
  enabled?: boolean;
}

const DIMENSIONS = { width: 64, height: 64 };

// The SVG renders the base width scaled by BASE_SCALE, not at full size.
export const WIDTH_PER_SCALE = DIMENSIONS.width * Primitive.BASE_SCALE;

export const Light = ({
  className,
  color,
  orientation = "left",
  enabled,
  scale,
}: RenderProps): ReactElement => (
  <Primitive.Div
    orientation={orientation}
    className={CSS(CSS.B("light"), enabled && CSS.M("enabled"), className)}
  >
    <Handle.Rectangle
      orientation={orientation}
      left={3.125}
      top={3.125}
      right={96.875}
      bottom={96.75}
    />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Circle cx="32" cy="32" r="30" />
    </Primitive.SVG>
  </Primitive.Div>
);
