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
  Path,
  type SVGBasedPrimitiveProps,
  Toggle,
  type ToggleProps,
} from "@/schematic/node/common/symbol/primitives";
export interface Props extends ToggleProps, SVGBasedPrimitiveProps {}

const DIMENSIONS = { width: 87, height: 58 };

export const Primitive = ({
  className,
  orientation = "left",
  color,
  scale,
  enabled = false,
  ...rest
}: Props): ReactElement => (
  <Toggle className={CSS(CSS.B("relief-valve"), className)} enabled={enabled} {...rest}>
    <HandleBoundary orientation={orientation}>
      <Handle
        location="left"
        orientation={orientation}
        left={2.2989}
        top={63.7931}
        id="1"
      />
      <Handle
        location="right"
        orientation={orientation}
        left={97.7011}
        top={63.7931}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M43.5 37L6.35453 18.2035C4.35901 17.1937 2 18.6438 2 20.8803V53.1197C2 55.3562 4.35901 56.8063 6.35453 55.7965L43.5 37ZM43.5 37L80.6455 18.2035C82.641 17.1937 85 18.6438 85 20.8803V53.1197C85 55.3562 82.641 56.8063 80.6455 55.7965L43.5 37Z" />
      <Path d="M43.5 2 L43.5 37" strokeLinecap="round" />
      <Path d="M31.8011 14.0802L55.1773 4.29611" strokeLinecap="round" />
      <Path d="M31.8011 20.0802L55.1773 10.2961" strokeLinecap="round" />
      <Path d="M31.8011 26.0802L55.1773 16.2961" strokeLinecap="round" />
    </InternalSVG>
  </Toggle>
);
