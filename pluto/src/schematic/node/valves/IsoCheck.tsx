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

import { Handle } from "@/schematic/node/common/handle";
import { Primitive as Base } from "@/schematic/node/common/primitive";
export interface Props extends Base.DivProps, Base.SVGBasedProps {}

const DIMENSIONS = { width: 84, height: 42 };

export const IsoCheck = ({
  className,
  orientation = "left",
  color: colorVal,
  scale,
  ...rest
}: Props): ReactElement => {
  const colorStr = color.cssString(colorVal);
  return (
    <Base.Div {...rest} orientation={orientation}>
      <Handle.Boundary orientation={orientation}>
        <Handle.Handle
          location="left"
          orientation={orientation}
          left={8.3333}
          top={50}
          id="1"
        />
        <Handle.Handle
          location="right"
          orientation={orientation}
          left={96.4286}
          top={50}
          id="2"
        />
      </Handle.Boundary>
      <Base.SVG
        dimensions={DIMENSIONS}
        color={colorVal}
        orientation={orientation}
        scale={scale}
      >
        <Base.Circle cx="7" cy="7" r="4" fill={colorStr} />
        <Base.Path
          d="M7 39.5V11.5941C7 9.42886 9.22384 7.97669 11.2063 8.84738L76.7937 37.6526C78.7762 38.5233 81 37.0711 81 34.9059V6"
          strokeLinecap="round"
        />
      </Base.SVG>
    </Base.Div>
  );
};
