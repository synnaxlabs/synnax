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

const DIMENSIONS = { width: 88, height: 86 };

export const Agitator = ({
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
        left={50}
        top={1}
        id="4"
      />
    </Handle.Boundary>
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Path d="M1 85V48.8541C1 46.624 3.34694 45.1735 5.34164 46.1708L80.6584 83.8292 C82.6531 84.8265 85 83.376 85 81.1459 V44" />
      <Primitive.Path d="M43 1L45 65" />
    </Primitive.SVG>
  </Toggle.Button>
);
