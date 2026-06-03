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
import { useColor } from "@/schematic/node/common/color";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
export interface Props extends Primitive.DivProps, Primitive.SVGBasedProps {}

const DIMENSIONS = { width: 93, height: 57 };

export const CheckWithArrow = ({
  className,
  orientation = "left",
  color: colorVal,
  scale,
  ...rest
}: Props): ReactElement => {
  const resolved = useColor(colorVal);
  const colorStr = color.cssString(resolved);
  return (
    <Primitive.Div
      orientation={orientation}
      className={CSS(CSS.B("check-valve-with-arrow"), className)}
      {...rest}
    >
      <Handle.Linear
        orientation={orientation}
        left={8.602}
        right={96.775}
        top={60.65}
      />
      <Primitive.SVG
        dimensions={DIMENSIONS}
        color={resolved}
        orientation={orientation}
        scale={scale}
      >
        <Primitive.Path
          d="M67.4706 5.20759C67.9906 5.6079 67.9906 6.3921 67.4706 6.79241L63.36 9.95678C62.7024 10.463 61.75 9.99421 61.75 9.16437V2.83563C61.75 2.00579 62.7024 1.53702 63.36 2.04322L67.4706 5.20759Z"
          fill={colorStr}
        />
        <Primitive.Path d="M62.5 6H32.5" strokeLinecap="round" />
        <Primitive.Circle cx="7.5" cy="13.5" r="6" fill={colorStr} />
        <Primitive.Path d="M49 34.5L11.8545 15.7035C9.85901 14.6937 7.5 16.1438 7.5 18.3803V50.6197C7.5 52.8562 9.85901 54.3063 11.8545 53.2965L49 34.5ZM49 34.5L86.1455 15.7035C88.141 14.6937 90.5 16.1438 90.5 18.3803V50.6197C90.5 52.8562 88.141 54.3063 86.1455 53.2965L49 34.5Z" />
      </Primitive.SVG>
    </Primitive.Div>
  );
};
