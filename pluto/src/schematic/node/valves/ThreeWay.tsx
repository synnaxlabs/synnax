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

const DIMENSIONS = { width: 87, height: 64 };

export const ThreeWay = ({
  color,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Toggle.Button
    {...rest}
    className={CSS.cx(CSS.B("three-way-valve"))}
    orientation={orientation}
  >
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={97.0278}
        id="1"
      />
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={2.2989}
        top={33.1308}
        id="2"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={97.7011}
        top={33.1308}
        id="3"
      />
    </Handle.Boundary>
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Path d="M43.5 21L6.35453 2.20349C4.35901 1.19372 2 2.64384 2 4.88029V37.1197C2 39.3562 4.35901 40.8063 6.35453 39.7965L43.5 21ZM43.5 21L80.6455 2.20349C82.641 1.19372 85 2.64384 85 4.8803V37.1197C85 39.3562 82.641 40.8063 80.6455 39.7965L43.5 21Z" />
      <Primitive.Path d="M44.3923 22.3611C44.0222 21.6298 42.9778 21.6298 42.6077 22.3611L24.7035 57.7433C23.6937 59.7388 25.1438 62.0978 27.3803 62.0978L59.6197 62.0978C61.8562 62.0978 63.3063 59.7388 62.2965 57.7433L44.3923 22.3611Z" />
    </Primitive.SVG>
  </Toggle.Button>
);
