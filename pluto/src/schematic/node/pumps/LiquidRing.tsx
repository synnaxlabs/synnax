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

const DIMENSIONS = { width: 66, height: 66 };

export const LiquidRing = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Toggle.Button
    {...rest}
    className={CSS.cls(CSS.B("pump"), className)}
    orientation={orientation}
  >
    <Handle.Rectangle
      orientation={orientation}
      left={4.55}
      top={4.55}
      right={95.45}
      bottom={95.45}
    />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Path d="M17 7.59998L59.5394 18.9984M17 58.3984L59.5394 47M63 33C63 36.9397 62.224 40.8407 60.7164 44.4805C59.2087 48.1203 56.999 51.4274 54.2132 54.2132C51.4274 56.999 48.1203 59.2087 44.4805 60.7164C40.8407 62.224 36.9397 63 33 63C29.0603 63 25.1593 62.224 21.5195 60.7164C17.8797 59.2087 14.5726 56.999 11.7868 54.2132C9.00104 51.4274 6.79125 48.1203 5.28361 44.4805C3.77597 40.8407 3 36.9397 3 33C3 29.0603 3.77597 25.1593 5.28362 21.5195C6.79126 17.8797 9.00104 14.5726 11.7868 11.7868C14.5726 9.00104 17.8797 6.79125 21.5195 5.28361C25.1593 3.77597 29.0603 3 33 3C36.9397 3 40.8407 3.77597 44.4805 5.28362C48.1203 6.79126 51.4274 9.00104 54.2132 11.7868C56.999 14.5726 59.2087 17.8797 60.7164 21.5195C62.224 25.1593 63 29.0603 63 33Z" />
      <Primitive.Path d="M39 33C39 33.7879 38.8448 34.5681 38.5433 35.2961C38.2417 36.0241 37.7998 36.6855 37.2426 37.2426C36.6855 37.7998 36.0241 38.2417 35.2961 38.5433C34.5681 38.8448 33.7879 39 33 39C32.2121 39 31.4319 38.8448 30.7039 38.5433C29.9759 38.2417 29.3145 37.7998 28.7574 37.2426C28.2002 36.6855 27.7583 36.0241 27.4567 35.2961C27.1552 34.5681 27 33.7879 27 33C27 32.2121 27.1552 31.4319 27.4567 30.7039C27.7583 29.9759 28.2002 29.3145 28.7574 28.7574C29.3145 28.2002 29.9759 27.7583 30.7039 27.4567C31.4319 27.1552 32.2121 27 33 27C33.7879 27 34.5681 27.1552 35.2961 27.4567C36.0241 27.7583 36.6855 28.2002 37.2426 28.7574C37.7998 29.3145 38.2417 29.9759 38.5433 30.7039C38.8448 31.4319 39 32.2121 39 33L39 33Z" />
      <Primitive.Path d="M39 33H47" />
      <Primitive.Path d="M27 33H19" />
      <Primitive.Path d="M36 27.804L40 20.8758" />
      <Primitive.Path d="M30 27.804L26 20.8758" />
      <Primitive.Path d="M30 38.196L26 45.1242" />
      <Primitive.Path d="M36 38.196L40 45.1242" />
    </Primitive.SVG>
  </Toggle.Button>
);
