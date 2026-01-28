// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const numberCouple = z.tuple([z.number(), z.number()]);
export type NumberCouple<T extends number | bigint = number> = [T, T];

// Dimensions

export const dimensions = z.object({ width: z.number(), height: z.number() });
export type Dimensions = z.infer<typeof dimensions>;
export const signedDimensions = z.object({
  signedWidth: z.number(),
  signedHeight: z.number(),
});
export type Dimension = "width" | "height";
export const ALIGNMENTS = ["start", "center", "end"] as const;
export type SignedDimension = "signedWidth" | "signedHeight";

// XY

export const xy = z.object({ x: z.number(), y: z.number() });
export type XY = z.infer<typeof xy>;
export const clientXY = z.object({ clientX: z.number(), clientY: z.number() });
export type ClientXY = z.infer<typeof clientXY>;

// Direction

export const DIRECTIONS = ["x", "y"] as const;
export const direction = z.enum(DIRECTIONS);
export type Direction = z.infer<typeof direction>;

// Location

export const OUTER_LOCATIONS = ["top", "right", "bottom", "left"] as const;
export const outerLocation = z.enum(OUTER_LOCATIONS);
export type OuterLocation = (typeof OUTER_LOCATIONS)[number];
export const X_LOCATIONS = ["left", "right"] as const;
export const xLocation = z.enum(X_LOCATIONS);
export type XLocation = (typeof X_LOCATIONS)[number];
export const Y_LOCATIONS = ["top", "bottom"] as const;
export const yLocation = z.enum(Y_LOCATIONS);
export type YLocation = (typeof Y_LOCATIONS)[number];
export const CENTER_LOCATIONS = ["center"] as const;
export const centerLocation = z.enum(CENTER_LOCATIONS);
export type CenterLocation = (typeof CENTER_LOCATIONS)[number];
const LOCATIONS = [...OUTER_LOCATIONS, ...CENTER_LOCATIONS] as const;
export const location = z.enum(LOCATIONS);
export type Location = z.infer<typeof location>;

// Alignment

export const alignment = z.enum(ALIGNMENTS);
export type Alignment = (typeof ALIGNMENTS)[number];
export const ORDERS = ["first", "last"] as const;
export const order = z.enum(ORDERS);
export type Order = (typeof ORDERS)[number];

// Bounds

export const bounds = z.object({ lower: z.number(), upper: z.number() });
export interface Bounds<T extends number | bigint = number> {
  lower: T;
  upper: T;
}

export type CrudeBounds<T extends number | bigint = number> =
  | Bounds<T>
  | NumberCouple<T>;
export const crudeDirection = z.enum([...direction.options, ...location.options]);
export type CrudeDirection = z.infer<typeof crudeDirection>;
export type CrudeXDirection = "x" | "left" | "right";
export type CrudeYDirection = "y" | "top" | "bottom";
export type AngularDirection = "clockwise" | "counterclockwise";
export const crudeLocation = z.union([direction, location, z.instanceof(String)]);
export type CrudeLocation = z.infer<typeof crudeLocation>;
