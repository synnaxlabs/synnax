// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Dimension,
  type Direction,
  type Location,
  type direction,
  DIRECTIONS,
  Y_LOCATIONS,
  type YLocation,
  type SignedDimension,
  crudeDirection,
  type CrudeDirection,
} from "@/spatial/base";

export type { Direction, direction };

export const crude = crudeDirection;

export type Crude = CrudeDirection;

export const construct = (c: Crude): Direction => {
  if (DIRECTIONS.includes(c as Direction)) return c as Direction;
  if (Y_LOCATIONS.includes(c as YLocation)) return "y";
  else return "x";
};

export const swap = (direction: Direction): Direction =>
  direction === "x" ? "y" : "x";

export const dimension = (direction: Direction): Dimension =>
  direction === "x" ? "width" : "height";

export const location = (direction: Direction): Location =>
  direction === "x" ? "left" : "top";

export const isDirection = (c: unknown): c is Direction => crude.safeParse(c).success;

export const signedDimension = (direction: Direction): SignedDimension =>
  direction === "x" ? "signedWidth" : "signedHeight";
