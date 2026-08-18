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

const DIMENSIONS = { width: 64, height: 64 };

export const Angled = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Toggle.Button
    {...rest}
    orientation={orientation}
    className={CSS.cx(CSS.B("angled-valve"), className)}
  >
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="bottom"
        orientation={orientation}
        left={32.8125}
        top={97.0278}
        id="1"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={97.0278}
        top={32.8125}
        id="2"
      />
    </Handle.Boundary>
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Path d="M22.3611 20.1077C21.6298 20.4778 21.6298 21.5222 22.3611 21.8923L57.7433 39.7965C59.7388 40.8063 62.0978 39.3562 62.0978 37.1197L62.0978 4.88029C62.0978 2.64384 59.7388 1.19372 57.7433 2.2035L22.3611 20.1077Z" />
      <Primitive.Path d="M21.8923 22.3611C21.5222 21.6298 20.4778 21.6298 20.1077 22.3611L2.20349 57.7433C1.19372 59.7388 2.64384 62.0978 4.8803 62.0978L37.1197 62.0978C39.3562 62.0978 40.8063 59.7388 39.7965 57.7433L21.8923 22.3611Z" />
    </Primitive.SVG>
  </Toggle.Button>
);
