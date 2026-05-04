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
import {
  Handle,
  HandleBoundary,
  InternalSVG,
  Line,
  Path,
  type SVGBasedPrimitiveProps,
  Toggle,
  type ToggleProps,
} from "@/schematic/node/common/symbol/primitives";
export interface Props extends ToggleProps, SVGBasedPrimitiveProps {}

const DIMENSIONS = { width: 87, height: 48 };

export const Primitive = ({
  className,
  orientation = "left",
  color,
  scale,
  enabled = false,
  ...rest
}: Props): ReactElement => (
  <Toggle
    {...rest}
    orientation={orientation}
    className={CSS(CSS.B("manual-valve"), className)}
    enabled={enabled}
  >
    <HandleBoundary orientation={orientation}>
      <Handle
        location="left"
        orientation={orientation}
        left={2.2989}
        top={56.25}
        id="1"
      />
      <Handle
        location="right"
        orientation={orientation}
        left={97.7011}
        top={56.25}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Line x1="43.5" y1="27" x2="43.5" y2="1" />
      <Path d="M19.64 1 L66.68 1" strokeLinecap="round" />
      <Path d="M43.5 27L6.35453 8.20349C4.35901 7.19372 2 8.64384 2 10.8803V43.1197C2 45.3562 4.35901 46.8063 6.35453 45.7965L43.5 27ZM43.5 27L80.6455 8.20349C82.641 7.19372 85 8.64384 85 10.8803V43.1197C85 45.3562 82.641 46.8063 80.6455 45.7965L43.5 27Z" />
    </InternalSVG>
  </Toggle>
);
