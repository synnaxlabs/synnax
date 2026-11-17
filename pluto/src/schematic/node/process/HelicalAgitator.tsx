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
import { Custom } from "@/schematic/node/common/custom";
import { Toggle } from "@/schematic/node/common/toggle";
import { type Props as AgitatorProps } from "@/schematic/node/process/Agitator";
import svg from "@/schematic/node/process/HelicalAgitator.svg?raw";

export interface Props extends AgitatorProps {}

const SPEC = Custom.createBuiltIn({
  variant: "actuator",
  svg,
  handles: [{ key: "4", position: { x: 0.51, y: 0.02 }, orientation: "top" }],
});

export const HelicalAgitator = ({
  className,
  orientation = "left",
  strokeColor,
  scale,
  ...rest
}: Props): ReactElement => (
  <Toggle.Button {...rest} className={CSS.cls(CSS.B("agitator"))}>
    <Custom.Drawing
      spec={SPEC}
      builtIn
      strokeColor={strokeColor}
      orientation={orientation}
      scale={scale}
    />
  </Toggle.Button>
);
