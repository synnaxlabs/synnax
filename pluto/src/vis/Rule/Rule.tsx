// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ComponentPropsWithoutRef } from "react";

import { Bound, Direction, swapDir, XY } from "@synnaxlabs/x";

import { Annotation, AnnotationProps } from "@/vis/Annotation";

export interface RuleAnnotationProps extends Omit<AnnotationProps, "position"> {
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
}: RuleProps): JSX.Element => {
  return (
    <g {...gProps(direction, position)}>
      <line
        {...lineProps(direction, size)}
        stroke="var(--pluto-gray-m1)"
        strokeWidth={2}
      />
      {annotations.map(({ position, ...rest }) => (
        <Annotation
          key={position}
          position={
            {
              [direction]: position,
              [swapDir(direction)]: 0,
            } as const as XY
          }
          {...rest}
        />
      ))}
    </g>
  );
};

const gProps = (
  direction: Direction,
  position: number
): ComponentPropsWithoutRef<"g"> =>
  direction === "x"
    ? {
        transform: `translate(0, ${position})`,
      }
    : {
        transform: `translate(${position}, 0)`,
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
