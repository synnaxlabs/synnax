// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS } from "@synnaxlabs/charon";
import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Handle } from "@/schematic/node/common/handle";
import { Primitive as Base } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";
export interface Props extends Toggle.ButtonProps, Base.SVGBasedProps {}

const DIMENSIONS = { width: 87, height: 42 };

export const Primitive = ({
  color: colorVal,
  className,
  orientation = "left",
  scale,
  enabled = false,
  ...rest
}: Props): ReactElement => (
  <Toggle.Button
    {...rest}
    orientation={orientation}
    className={CSS(CSS.B("butterfly-valve-two"), className)}
    enabled={enabled}
  >
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={2.2989}
        top={50}
        id="1"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={97.7011}
        top={50}
        id="2"
      />
    </Handle.Boundary>
    <Base.SVG
      dimensions={DIMENSIONS}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Base.Circle cx="43.5" cy="21" r="10" fill={color.cssString(colorVal)} />
      <Base.Rect x="2" y="2" width="83" height="38" rx="1" />
      <Base.Path d="M2.29001 2.29004L84.7069 39.676" />
    </Base.SVG>
  </Toggle.Button>
);
