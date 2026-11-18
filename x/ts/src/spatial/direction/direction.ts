// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type AngularDirection,
  type Dimension,
  type Direction,
  directionZ,
  type Location,
  Y_LOCATIONS,
} from "@/spatial/base";

export { type Direction, directionZ };

export type Angular = AngularDirection;
export type Crude = Direction | Location;
type CrudeX = "x" | "left" | "right";
type CrudeY = "y" | "top" | "bottom";

const Y_LOCATION_SET = new Set<string>(Y_LOCATIONS);

export const construct = (c: Crude): Direction => {
  if (c === "x" || c === "y") return c;
  return Y_LOCATION_SET.has(c) ? "y" : "x";
};

export const swap = (direction: Crude): Direction =>
  construct(direction) === "x" ? "y" : "x";

export const dimension = (direction: Crude): Dimension =>
  construct(direction) === "x" ? "width" : "height";

export const location = (direction: Crude): Location =>
  construct(direction) === "x" ? "left" : "top";

export const isX = (direction: Crude): direction is CrudeX => {
  if (direction === "center") return false;
  return construct(direction) === "x";
};

export const isY = (direction: Crude): direction is CrudeY =>
  construct(direction) === "y";
