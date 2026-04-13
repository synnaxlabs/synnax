// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const stringFilterZ = z.object({
  eq: z.string().optional(),
  in: z.string().array().optional(),
  contains: z.string().optional(),
  regex: z.string().optional(),
});
export interface StringFilter extends z.infer<typeof stringFilterZ> {}

export const boolFilterZ = z.object({
  eq: z.boolean().optional(),
});
export interface BoolFilter extends z.infer<typeof boolFilterZ> {}

export const numericFilterZ = z.object({
  eq: z.number().optional(),
  in: z.number().array().optional(),
  gt: z.number().optional(),
  lt: z.number().optional(),
  gte: z.number().optional(),
  lte: z.number().optional(),
});
export interface NumericFilter extends z.infer<typeof numericFilterZ> {}

export const timeFilterZ = z.object({
  eq: z.bigint().optional(),
  gt: z.bigint().optional(),
  lt: z.bigint().optional(),
  gte: z.bigint().optional(),
  lte: z.bigint().optional(),
});
export interface TimeFilter extends z.infer<typeof timeFilterZ> {}

export const stringOrderByZ = z.object({
  direction: z.enum(["asc", "desc"]),
  after: z.string().optional(),
});
export interface StringOrderBy extends z.infer<typeof stringOrderByZ> {}

export const numericOrderByZ = z.object({
  direction: z.enum(["asc", "desc"]),
  after: z.number().optional(),
});
export interface NumericOrderBy extends z.infer<typeof numericOrderByZ> {}

export const timestampOrderByZ = z.object({
  direction: z.enum(["asc", "desc"]),
  after: z.bigint().optional(),
});
export interface TimestampOrderBy extends z.infer<typeof timestampOrderByZ> {}
