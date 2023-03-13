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

import { Annotation } from "@/vis/Annotation";

export interface AnnotationProps {
  position: number;
  values: Record<string, string>;
}

export interface RuleProps {
  position: number;
  size: Bound;
  direction: Direction;
  annotations?: AnnotationProps[];
}

export const Rule = ({
  direction,
  position,
  size,
  annotations = [],
}: RuleProps): JSX.Element => {
  return (
    <g {...gProps(direction, position)}>
      <line {...lineProps(direction, size)} />
      {annotations.map(({ position, values }) => (
        <Annotation
          key={position}
          position={
            {
              [direction]: position,
              [swapDir(direction)]: 0,
            } as const as XY
          }
          dimensions={{ height: Object.keys(values).length * 20, width: 100 }}
        >
          {Object.entries(values).map(([key, value]) => (
            <text key={key} x={0} y={0}>
              {key}: {value}
            </text>
          ))}
        </Annotation>
      ))}
    </g>
  );
};

const gProps = (
  direction: Direction,
  position: number
): ComponentPropsWithoutRef<"g"> =>
  direction === "y"
    ? {
        transform: `translate(${position}, 0)`,
      }
    : {
        transform: `translate(0, ${position})`,
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
