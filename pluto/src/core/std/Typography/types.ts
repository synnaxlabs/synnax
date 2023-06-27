// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { ComponentSize } from "@/util/component";

export const TypographyLevels = ["h1", "h2", "h3", "h4", "h5", "p", "small"] as const;
export const typographyLevel = z.enum(TypographyLevels);

/* Level of typography i.e paragraph and heading */
export type TypographyLevel = z.infer<typeof typographyLevel>;

export const typographySpec = z.object({
  size: z.number(),
  weight: z.union([z.number(), z.string()]),
  lineHeight: z.number(),
  textTransform: z.string().optional(),
});

/* Defines a particular typography style */
export type TypographySpec = z.infer<typeof typographySpec>;

/* Standardizes the typography levels for components of different sizes */
export const ComponentSizeTypographyLevels: Record<ComponentSize, TypographyLevel> = {
  small: "small",
  medium: "p",
  large: "h4",
};

export const TypographyLevelComponentSizes: Record<TypographyLevel, ComponentSize> = {
  h1: "large",
  h2: "large",
  h3: "medium",
  h4: "medium",
  h5: "small",
  p: "medium",
  small: "small",
};
