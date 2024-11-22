// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type CrudeDirection,
  crudeDirection,
  type Dimension,
  type Direction,
  direction,
  DIRECTIONS,
  type Location,
  type SignedDimension,
  Y_LOCATIONS,
  type YLocation,
} from "@/spatial/base";

export { Direction, direction };

export const crude = crudeDirection;

export type Crude = CrudeDirection;

export const construct = (c: Crude): Direction => {
  if (DIRECTIONS.includes(c as Direction)) return c as Direction;
  if (Y_LOCATIONS.includes(c as YLocation)) return "y";
  return "x";
};

export const swap = (direction: CrudeDirection): Direction =>
  construct(direction) === "x" ? "y" : "x";

export const dimension = (direction: CrudeDirection): Dimension =>
  construct(direction) === "x" ? "width" : "height";

export const location = (direction: CrudeDirection): Location =>
  construct(direction) === "x" ? "left" : "top";

export const isDirection = (c: unknown): c is Direction => crude.safeParse(c).success;

export const signedDimension = (direction: CrudeDirection): SignedDimension =>
  construct(direction) === "x" ? "signedWidth" : "signedHeight";
