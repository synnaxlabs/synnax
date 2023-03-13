// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PropsWithChildren } from "react";

import { Dimensions, XY } from "@synnaxlabs/x";

import { CSS } from "@/css";
import { Theming } from "@/theming";

export interface AnnotationProps extends PropsWithChildren {
  position: XY;
  dimensions: Dimensions;
  stroke?: string;
}

export const Annotation = ({
  children,
  position,
  dimensions,
  stroke = "var(--pluto-gray-m3)",
}: AnnotationProps): JSX.Element => {
  const { theme } = Theming.useContext();
  return (
    <g transform={`translate(${position.x}, ${position.y})`}>
      <rect
        className={CSS.B("annotation")}
        x={0}
        y={0}
        rx={theme.sizes.border.radius}
        ry={theme.sizes.border.radius}
        stroke={stroke}
        {...dimensions}
      />
      {children}
    </g>
  );
};
