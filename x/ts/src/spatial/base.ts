// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

// Tuples

export const numberCouple = z.tuple([z.number(), z.number()]);
export type NumberCouple<T extends number | bigint = number> = [T, T];

// Direction

export const DIRECTIONS = ["x", "y"] as const;
export const directionZ = z.enum(DIRECTIONS);
export type Direction = z.infer<typeof directionZ>;

// Location

export const OUTER_LOCATIONS = ["top", "right", "bottom", "left"] as const;
export const outerLocationZ = z.enum(OUTER_LOCATIONS);
export type OuterLocation = z.infer<typeof outerLocationZ>;

export const X_LOCATIONS = ["left", "right"] as const;
export const xLocationZ = z.enum(X_LOCATIONS);
export type XLocation = z.infer<typeof xLocationZ>;

export const Y_LOCATIONS = ["top", "bottom"] as const;
export const yLocationZ = z.enum(Y_LOCATIONS);
export type YLocation = z.infer<typeof yLocationZ>;

export const CENTER_LOCATIONS = ["center"] as const;
export const centerLocationZ = z.enum(CENTER_LOCATIONS);
export type CenterLocation = z.infer<typeof centerLocationZ>;

export const LOCATIONS = ["top", "right", "bottom", "left", "center"] as const;
export const locationZ = z.enum(LOCATIONS);
export type Location = z.infer<typeof locationZ>;

// Alignment

export const ALIGNMENTS = ["start", "center", "end"] as const;
export const alignmentZ = z.enum(ALIGNMENTS);
export type Alignment = z.infer<typeof alignmentZ>;

// Order

export const ORDERS = ["first", "last"] as const;
export const orderZ = z.enum(ORDERS);
export type Order = z.infer<typeof orderZ>;

// XY

export const xyZ = z.object({ x: z.number(), y: z.number() });
export type XY = z.infer<typeof xyZ>;

export const clientXyZ = z.object({ clientX: z.number(), clientY: z.number() });
export type ClientXY = z.infer<typeof clientXyZ>;

// Dimensions

export const dimensionsZ = z.object({ width: z.number(), height: z.number() });
export type Dimensions = z.infer<typeof dimensionsZ>;

export const signedDimensionsZ = z.object({
  signedWidth: z.number(),
  signedHeight: z.number(),
});
export type SignedDimensions = z.infer<typeof signedDimensionsZ>;

export type Dimension = "width" | "height";
export type SignedDimension = "signedWidth" | "signedHeight";

// Bounds

export const boundsZ = z.object({ lower: z.number(), upper: z.number() });

// Generic bounds interface (supports bigint)
export interface Bounds<T extends number | bigint = number> {
  lower: T;
  upper: T;
}

export type CrudeBounds<T extends number | bigint = number> =
  | Bounds<T>
  | NumberCouple<T>;

// Derived/complex types

export const crudeDirection = z.enum([
  "x",
  "y",
  ...OUTER_LOCATIONS,
  ...CENTER_LOCATIONS,
]);
export type CrudeDirection = z.infer<typeof crudeDirection>;
export type CrudeXDirection = "x" | "left" | "right";
export type CrudeYDirection = "y" | "top" | "bottom";
export type AngularDirection = "clockwise" | "counterclockwise";
export const crudeLocation = z.union([
  directionZ,
  z.enum([...OUTER_LOCATIONS, ...CENTER_LOCATIONS]),
  z.instanceof(String),
]);
export type CrudeLocation = z.infer<typeof crudeLocation>;
