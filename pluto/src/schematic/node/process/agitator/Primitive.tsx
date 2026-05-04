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

const DIMENSIONS = { width: 88, height: 86 };

export const Primitive = ({
  orientation = "left",
  color,
  scale,
  ...rest
}: Props): ReactElement => (
  <Toggle {...rest} className={CSS(CSS.B("agitator"))}>
    <HandleBoundary orientation={orientation}>
      <Handle location="top" orientation={orientation} left={50} top={1} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M1 85V48.8541C1 46.624 3.34694 45.1735 5.34164 46.1708L80.6584 83.8292 C82.6531 84.8265 85 83.376 85 81.1459 V44" />
      <Path d="M43 1L45 65" />
    </InternalSVG>
  </Toggle>
);
