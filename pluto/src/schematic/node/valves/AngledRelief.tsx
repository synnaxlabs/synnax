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

const DIMENSIONS = { width: 64, height: 79 };

export const AngledRelief = ({
  color,
  className,
  orientation = "left",
  scale,
  enabled = false,
  ...rest
}: Props): ReactElement => (
  <Toggle.Button
    {...rest}
    orientation={orientation}
    className={CSS.cx(CSS.B("angled-relief-valve"), className)}
    enabled={enabled}
  >
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="bottom"
        orientation={orientation}
        left={32.8125}
        top={97.5922}
        id="1"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={97.0278}
        top={45.5639}
        id="2"
      />
    </Handle.Boundary>
    <Primitive.SVG
      dimensions={DIMENSIONS}
      orientation={orientation}
      color={color}
      scale={scale}
    >
      <Primitive.Line x1={21} y1={2} x2={21} y2={36.7} strokeLinecap="round" />
      <Primitive.Path d="M9.05106 14.0802L32.4273 4.29611" strokeLinecap="round" />
      <Primitive.Path d="M9.05106 20.0802L32.4273 10.2961" strokeLinecap="round" />
      <Primitive.Path d="M9.05106 26.0802L32.4273 16.2961" strokeLinecap="round" />
      <Primitive.Path d="M22.3611 35.1077C21.6298 35.4778 21.6298 36.5222 22.3611 36.8923L57.7433 54.7965C59.7388 55.8063 62.0978 54.3562 62.0978 52.1197L62.0978 19.8803C62.0978 17.6438 59.7388 16.1937 57.7433 17.2035L22.3611 35.1077Z" />
      <Primitive.Path d="M21.8923 37.3611C21.5222 36.6298 20.4778 36.6298 20.1077 37.3611L2.20349 72.7433C1.19372 74.7388 2.64384 77.0978 4.8803 77.0978H37.1197C39.3562 77.0978 40.8063 74.7388 39.7965 72.7433L21.8923 37.3611Z" />
    </Primitive.SVG>
  </Toggle.Button>
);
