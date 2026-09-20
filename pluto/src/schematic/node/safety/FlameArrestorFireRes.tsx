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

export interface Props extends Primitive.DivProps, Primitive.SVGBasedProps {}

const DIMENSIONS = { width: 63, height: 69 };

export const FlameArrestorFireRes = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Primitive.Div {...rest} className={CSS.cls(CSS.B("flame-arrestor"), className)}>
    <Handle.Linear orientation={orientation} left={3.333} right={96.667} />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Rect x="2" y="2.5" width="28" height="64" rx="3" />
      <Primitive.Path d="M16 2.5L16 66.5" />
      <Primitive.Path d="M2 34.5H30" />
      <Primitive.Path d="M2 19H30" />
      <Primitive.Path d="M2 50H30" />
      <Primitive.Path
        d="M29 2.5C33.2023 2.5 37.3635 3.3277 41.2459 4.93586C45.1283 6.54401 48.656 8.90111 51.6274 11.8726C54.5989 14.8441 56.956 18.3717 58.5642 22.2541C60.1723 26.1366 61 30.2977 61 34.5C61 38.7023 60.1723 42.8635 58.5642 46.7459C56.956 50.6283 54.5989 54.1559 51.6274 57.1274C48.6559 60.0989 45.1283 62.456 41.2459 64.0641C37.3635 65.6723 33.2023 66.5 29 66.5"
        strokeLinecap="square"
      />
      <Primitive.Line x1="27" y1="66.5" x2="29" y2="66.5" />
      <Primitive.Line x1="27" y1="2.5" x2="29" y2="2.5" />
    </Primitive.SVG>
  </Primitive.Div>
);
