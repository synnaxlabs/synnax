// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ReactElement } from "react";

import { CSS } from "@/css";
import { Custom } from "@/schematic/node/common/custom";
import type { Primitive } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";
import svg from "@/schematic/node/valves/AngledSpringLoadedRelief.svg?raw";

export interface Props extends Toggle.ButtonProps, Primitive.SVGBasedProps {}

const SPEC = Custom.createBuiltIn({
  variant: "actuator",
  svg,
  handles: [
    { key: "1", position: { x: 0.318182, y: 0.98 }, orientation: "bottom" },
    { key: "2", position: { x: 0.956061, y: 0.555185 }, orientation: "right" },
  ],
});

export const AngledSpringLoadedRelief = ({
  className,
  orientation = "left",
  strokeColor,
  scale,
  enabled = false,
  ...rest
}: Props): ReactElement => (
  <Toggle.Button
    {...rest}
    orientation={orientation}
    className={CSS.cls(CSS.B("spring-loaded-relief-valve"), className)}
    enabled={enabled}
  >
    <Custom.Drawing
      spec={SPEC}
      builtIn
      strokeColor={strokeColor}
      orientation={orientation}
      scale={scale}
    />
  </Toggle.Button>
);
