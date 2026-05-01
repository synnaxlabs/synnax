// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import z from "zod";

import {
  type AngularDirection,
  CENTER_LOCATIONS,
  type Dimension,
  type Direction,
  DIRECTIONS,
  directionZ,
  type Location,
  OUTER_LOCATIONS,
  type SignedDimension,
  Y_LOCATIONS,
  type YLocation,
} from "@/spatial/types.gen";

export { type Direction, DIRECTIONS, directionZ };

export type Angular = AngularDirection;
export const crudeZ = z.enum(["x", "y", ...OUTER_LOCATIONS, ...CENTER_LOCATIONS]);
export type Crude = z.infer<typeof crudeZ>;
export type CrudeX = "x" | "left" | "right";
export type CrudeY = "y" | "top" | "bottom";

export const construct = (c: Crude): Direction => {
  if (DIRECTIONS.includes(c as Direction)) return c as Direction;
  if (Y_LOCATIONS.includes(c as YLocation)) return "y";
  return "x";
};

export const swap = (direction: Crude): Direction =>
  construct(direction) === "x" ? "y" : "x";

export const dimension = (direction: Crude): Dimension =>
  construct(direction) === "x" ? "width" : "height";

export const location = (direction: Crude): Location =>
  construct(direction) === "x" ? "left" : "top";

export const isDirection = (c: unknown): c is Direction => crudeZ.safeParse(c).success;

export const signedDimension = (direction: Crude): SignedDimension =>
  construct(direction) === "x" ? "signedWidth" : "signedHeight";

export const isX = (direction: Crude): direction is CrudeX => {
  if (direction === "center") return false;
  return construct(direction) === "x";
};

export const isY = (direction: Crude): direction is CrudeY =>
  construct(direction) === "y";
