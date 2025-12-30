// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { array } from "@/array";
import { id } from "@/id";
import { label } from "@/label";
import { type optional } from "@/optional";
import { type Variant, variantZ } from "@/status/types.gen";
import { TimeStamp } from "@/telem";

export type StatusZodObject<DetailsSchema extends z.ZodType = z.ZodNever> = z.ZodObject<
  {
    key: z.ZodString;
    name: z.ZodDefault<z.ZodString>;
    variant: typeof variantZ;
    message: z.ZodString;
    description: z.ZodOptional<z.ZodString>;
    labels: z.ZodOptional<ReturnType<typeof array.nullableZ<typeof label.labelZ>>>;
    time: typeof TimeStamp.z;
  } & ([DetailsSchema] extends [z.ZodNever] ? {} : { details: DetailsSchema })
>;

export interface StatusZFunction {
  <DetailsSchema extends z.ZodType>(
    details: DetailsSchema,
  ): StatusZodObject<DetailsSchema>;
  <DetailsSchema extends z.ZodType = z.ZodNever>(
    details?: DetailsSchema,
  ): StatusZodObject<DetailsSchema>;
}

export const statusZ: StatusZFunction = <DetailsSchema extends z.ZodType>(
  details?: DetailsSchema,
) =>
  z.object({
    key: z.string(),
    name: z.string().default(""),
    variant: variantZ,
    message: z.string(),
    description: z.string().optional(),
    time: TimeStamp.z,
    labels: array.nullableZ(label.labelZ).optional(),
    details: details ?? z.unknown().optional(),
  });

export const newZ = <DetailsSchema extends z.ZodType>(details?: DetailsSchema) =>
  z.object({
    key: z.string().optional(),
    name: z.string().optional(),
    variant: variantZ,
    message: z.string(),
    description: z.string().optional(),
    time: TimeStamp.z,
    labels: array.nullableZ(label.labelZ).optional(),
    details: details ?? z.unknown().optional(),
  });

export type New<DetailsSchema = z.ZodNever, V extends Variant = Variant> = Partial<
  Pick<Base<V>, "key" | "name">
> &
  Omit<Base<V>, "key" | "name"> &
  ([DetailsSchema] extends [z.ZodNever] ? {} : { details: z.output<DetailsSchema> });

type Base<V extends Variant> = {
  key: string;
  name: string;
  variant: V;
  message: string;
  description?: string;
  time: TimeStamp;
  labels?: label.Label[];
};

export type Status<DetailsSchema = z.ZodNever, V extends Variant = Variant> = Base<V> &
  ([DetailsSchema] extends [z.ZodNever] ? {} : { details: z.output<DetailsSchema> });

export type Crude<
  DetailsSchema = z.ZodNever,
  V extends Variant = Variant,
> = optional.Optional<Base<V>, "key" | "time" | "name"> &
  ([DetailsSchema] extends [z.ZodNever] ? {} : { details: z.output<DetailsSchema> });

export const exceptionDetailsSchema = z.object({
  stack: z.string(),
  error: z.instanceof(Error),
});

export const fromException = (
  exc: unknown,
  message?: string,
): Status<typeof exceptionDetailsSchema, "error"> => {
  if (!(exc instanceof Error)) throw exc;
  return create<typeof exceptionDetailsSchema, "error">({
    variant: "error",
    message: message ?? exc.message,
    description: message != null ? exc.message : undefined,
    details: { stack: exc.stack ?? "", error: exc },
  });
};

export const create = <DetailsSchema = z.ZodNever, V extends Variant = Variant>(
  spec: Crude<DetailsSchema, V>,
): Status<DetailsSchema, V> =>
  ({
    key: id.create(),
    time: TimeStamp.now(),
    name: "",
    ...spec,
  }) as Status<DetailsSchema, V>;

export const keepVariants = (
  variant?: Variant,
  keep: Variant | Variant[] = [],
): Variant | undefined => {
  if (variant == null) return undefined;
  if (Array.isArray(keep)) {
    if (keep.includes(variant)) return variant;
    return undefined;
  }
  return keep === variant ? variant : undefined;
};

export const removeVariants = (
  variant?: Variant,
  remove: Variant | Variant[] = [],
): Variant | undefined => {
  if (variant == null) return undefined;
  if (Array.isArray(remove)) {
    if (remove.includes(variant)) return undefined;
    return variant;
  }
  return remove === variant ? undefined : variant;
};
