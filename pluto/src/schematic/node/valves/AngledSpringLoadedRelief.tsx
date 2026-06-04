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

const DIMENSIONS = { width: 66, height: 101 };

export const AngledSpringLoadedRelief = ({
  className,
  orientation = "left",
  color: colorVal,
  scale,
  enabled = false,
  ...rest
}: Props): ReactElement => (
    <Toggle.Button
      {...rest}
      orientation={orientation}
      className={CSS(CSS.B("spring-loaded-relief-valve"), className)}
      enabled={enabled}
    >
      <Handle.Boundary orientation={orientation}>
        <Handle.Handle
          location="bottom"
          orientation={orientation}
          left={31.8182}
          top={98}
          id="1"
        />
        <Handle.Handle
          location="right"
          orientation={orientation}
          left={95.6061}
          top={55.5185}
          id="2"
        />
      </Handle.Boundary>
      <Primitive.SVG
        dimensions={DIMENSIONS}
        color={colorVal}
        orientation={orientation}
        scale={scale}
      >
        <Primitive.Path d="M23.3625 55.6237C22.6312 55.9937 22.6311 57.0381 23.3624 57.4082L58.7435 75.3147C60.7389 76.3246 63.098 74.8747 63.0981 72.6382L63.1001 40.3988C63.1003 38.1624 60.7414 36.7121 58.7458 37.7217L23.3625 55.6237Z" />
        <Primitive.Path d="M48 38.633V72.633" strokeLinecap="round" strokeWidth={4} />
        <Primitive.Path d="M21.8923 58.4348C21.5222 57.7035 20.4778 57.7035 20.1077 58.4348L2.20349 93.817C1.19372 95.8125 2.64384 98.1715 4.8803 98.1715H37.1197C39.3562 98.1715 40.8063 95.8125 39.7965 93.817L21.8923 58.4348Z" />
        <Primitive.Circle cx="21" cy="56.0737" r="4" fill="var(--pluto-symbol-display)" />
        <Primitive.Path
          d="M21 53.0105V50.0225C21 49.3397 20.6516 48.704 20.0759 48.3366L15.6419 45.507C14.4098 44.7207 14.4098 42.9214 15.6419 42.1351L26.3581 35.2965C27.5902 34.5102 27.5902 32.7109 26.3581 31.9246L15.6419 25.0859C14.4098 24.2997 14.4098 22.5003 15.6419 21.714L26.3581 14.8754C27.5902 14.0891 27.5902 12.2898 26.3581 11.5035L21.9241 8.67393C21.3484 8.30656 21 7.67087 21 6.98798V4"
          stroke="var(--pluto-symbol-display)"
          strokeLinecap="round"
        />
      </Primitive.SVG>
    </Toggle.Button>
  );
