// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/light/light.css";

import { type schematic } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Custom } from "@/schematic/node/common/custom";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import svg from "@/schematic/node/general/light/Light.svg?raw";

interface RenderProps extends Partial<
  Pick<schematic.LightNodeConfig, "strokeColor" | "onColor" | "orientation" | "scale">
> {
  className?: string;
  enabled?: boolean;
}

const SPEC = Custom.createBuiltIn({ svg, handles: [] });

const ON_COLOR = CSS.variable("light-on-color");

export const WIDTH_PER_SCALE = 64 * Primitive.BASE_SCALE;

export const Light = ({
  className,
  strokeColor,
  onColor,
  orientation = "left",
  enabled,
  scale,
}: RenderProps): ReactElement => (
  <Primitive.Div
    orientation={orientation}
    className={CSS.cls(
      CSS.B("light"),
      enabled && CSS.M("enabled"),
      onColor != null && CSS.BM("light", "on-colored"),
      className,
    )}
    style={onColor == null ? undefined : { [ON_COLOR]: color.rgbaString(onColor) }}
  >
    <Handle.Rectangle
      orientation={orientation}
      left={3.125}
      top={3.125}
      right={96.875}
      bottom={96.75}
    />
    <Custom.Drawing
      spec={SPEC}
      builtIn
      strokeColor={strokeColor}
      orientation={orientation}
      scale={scale}
    />
  </Primitive.Div>
);
