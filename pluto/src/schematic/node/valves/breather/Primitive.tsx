// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import {
  Circle,
  Handle,
  HandleBoundary,
  InternalSVG,
  Path,
  type SVGBasedPrimitiveProps,
  Toggle,
  type ToggleProps,
} from "@/schematic/node/common/symbol/primitives";
export interface Props extends ToggleProps, SVGBasedPrimitiveProps {}

const DIMENSIONS = { width: 99, height: 57 };

export const Primitive = ({
  color: colorVal,
  className,
  orientation = "left",
  scale,
  enabled = false,
  ...rest
}: Props): ReactElement => (
  <Toggle
    {...rest}
    orientation={orientation}
    className={CSS(CSS.B("breather-valve"), className)}
    enabled={enabled}
  >
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={8.081} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={91.919}
        top={50}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={DIMENSIONS}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Circle cx="91" cy="49.5" r="6" fill={color.cssString(colorVal)} />
      <Circle cx="8" cy="7.5" r="6" fill={color.cssString(colorVal)} />
      <Path d="M49.5 28.5L12.3545 9.70349C10.359 8.69372 8 10.1438 8 12.3803V44.6197C8 46.8562 10.359 48.3063 12.3545 47.2965L49.5 28.5ZM49.5 28.5L86.6455 9.70349C88.641 8.69372 91 10.1438 91 12.3803V44.6197C91 46.8562 88.641 48.3063 86.6455 47.2965L49.5 28.5Z" />
    </InternalSVG>
  </Toggle>
);
