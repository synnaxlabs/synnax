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

export const PropellerAgitator = ({
  orientation = "left",
  color,
  scale,
  ...rest
}: Props): ReactElement => (
  <Toggle.Button {...rest} className={CSS.cls(CSS.B("agitator"))}>
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
      <Primitive.Path d="M43.5 69.573L14.9534 55.6147C8.97428 52.6911 2 57.0443 2 63.6999V75.4462C2 82.1018 8.97429 86.455 14.9534 83.5314L43.5 69.573ZM43.5 69.573L72.0466 55.6147C78.0257 52.6911 85 57.0443 85 63.6999V75.4462C85 82.1018 78.0257 86.455 72.0466 83.5314L43.5 69.573Z" />
      <Primitive.Path d="M43.5 69.6L43.5 2" strokeLinecap="round" />
    </Primitive.SVG>
  </Toggle.Button>
);
