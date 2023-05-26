// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ComponentPropsWithoutRef, ReactElement } from "react";

import { Bound, clamp, Direction, swapDir, XY } from "@synnaxlabs/x";

import { Annotation, AnnotationProps } from "@/vis/Annotation";
import { SVG } from "@/vis/svg";

export interface RuleAnnotationProps extends Omit<AnnotationProps, "position"> {
  key: string;
  position: number;
}

export interface RuleProps {
  position: number;
  size: Bound;
  direction: Direction;
  annotations?: RuleAnnotationProps[];
}

export const Rule = ({
  direction,
  position,
  size,
  annotations = [],
}: RuleProps): ReactElement => {
  return (
    <g transform={SVG.translateIn(position, swapDir(direction))}>
      {/* Rendering the annotations first puts them below the rule. This looks cleaner */}
      {annotations.map(({ key, position, values, ...rest }) => {
        const height = Annotation.height(values);
        position = clamp(position, size.lower, size.upper - height);
        return (
          <Annotation
            key={key}
            values={values}
            position={
              {
                [direction]: position,
                [swapDir(direction)]: 0,
              } as const as XY
            }
            {...rest}
          />
        );
      })}

      <line
        {...lineProps(direction, size)}
        stroke="var(--pluto-gray-m1)"
        strokeWidth={2}
      />
    </g>
  );
};

const lineProps = (
  direction: Direction,
  size: Bound
): ComponentPropsWithoutRef<"line"> =>
  direction === "x"
    ? {
        x1: size.lower,
        x2: size.upper,
        y1: 0,
        y2: 0,
      }
    : {
        x1: 0,
        x2: 0,
        y1: size.lower,
        y2: size.upper,
      };
