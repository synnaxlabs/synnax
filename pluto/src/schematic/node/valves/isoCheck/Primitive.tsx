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

import {
  Circle,
  Div,
  type DivProps,
  Handle,
  HandleBoundary,
  InternalSVG,
  Path,
  type SVGBasedPrimitiveProps,
} from "@/schematic/node/common/symbol/primitives";
export interface Props extends DivProps, SVGBasedPrimitiveProps {}

const DIMENSIONS = { width: 84, height: 42 };

export const Primitive = ({
  className,
  orientation = "left",
  color: colorVal,
  scale,
  ...rest
}: Props): ReactElement => {
  const colorStr = color.cssString(colorVal);
  return (
    <Div {...rest} orientation={orientation}>
      <HandleBoundary orientation={orientation}>
        <Handle
          location="left"
          orientation={orientation}
          left={8.3333}
          top={50}
          id="1"
        />
        <Handle
          location="right"
          orientation={orientation}
          left={96.4286}
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
        <Circle cx="7" cy="7" r="4" fill={colorStr} />
        <Path
          d="M7 39.5V11.5941C7 9.42886 9.22384 7.97669 11.2063 8.84738L76.7937 37.6526C78.7762 38.5233 81 37.0711 81 34.9059V6"
          strokeLinecap="round"
        />
      </InternalSVG>
    </Div>
  );
};
