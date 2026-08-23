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
import { Label } from "@/schematic/node/flowmeters/Label";
import { type Props } from "@/schematic/node/flowmeters/props";

const DIMENSIONS = { width: 71, height: 36 };

export const Turbine = ({
  id,
  className,
  orientation = "right",
  color,
  scale = 1,
  ...rest
}: Props): ReactElement => (
  <Primitive.Div {...rest} className={CSS.cls(CSS.B("flowmeter-Turbine"), className)}>
    <Handle.Rectangle
      orientation={orientation}
      left={4}
      top={6}
      right={98}
      bottom={91}
    />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Rect x="2" y="2" width="67" height="31" rx="2" />
      <Primitive.Path d="M16.5 17.5H54.5" strokeLinecap="round" />
      <Primitive.Path d="M32.5 9L35.4756 17.1753" strokeLinecap="round" />
      <Primitive.Path d="M38.5 26L35.5244 17.8247" strokeLinecap="round" />
      <Primitive.Path d="M32.5 26L35.4756 17.8247" strokeLinecap="round" />
      <Primitive.Path d="M38.5 9L35.5244 17.1753" strokeLinecap="round" />
      <Primitive.Path
        d="M32.5 9C32.5 8.20435 32.8161 7.44129 33.3787 6.87868C33.9413 6.31607 34.7044 6 35.5 6C36.2956 6 37.0587 6.31607 37.6213 6.87868C38.1839 7.44129 38.5 8.20435 38.5 9"
        strokeWidth="2"
      />
      <Primitive.Path
        d="M38.5 26C38.5 26.7956 38.1839 27.5587 37.6213 28.1213C37.0587 28.6839 36.2956 29 35.5 29C34.7044 29 33.9413 28.6839 33.3787 28.1213C32.8161 27.5587 32.5 26.7956 32.5 26"
        strokeWidth="2"
      />
      <Label color={color} />
    </Primitive.SVG>
  </Primitive.Div>
);
