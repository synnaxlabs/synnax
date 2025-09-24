// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { id } from "@/id";
import { type Optional } from "@/optional";
import { TimeStamp } from "@/telem";

export const variantZ = z.enum([
  "success",
  "info",
  "warning",
  "error",
  "loading",
  "disabled",
]);

// Represents one of the possible variants of a status message.
export type Variant = z.infer<typeof variantZ>;

type StatusZodObject<DetailsSchema extends z.ZodType> = z.ZodObject<
  {
    key: z.ZodString;
    variant: typeof variantZ;
    message: z.ZodString;
    description: z.ZodOptional<z.ZodString>;
    time: typeof TimeStamp.z;
  } & ([DetailsSchema] extends [z.ZodNever] ? {} : { details: DetailsSchema })
>;

interface StatusZFunction {
  (): StatusZodObject<z.ZodNever>;
  <DetailsSchema extends z.ZodType>(
    details: DetailsSchema,
  ): StatusZodObject<DetailsSchema>;
}

export const statusZ: StatusZFunction = <DetailsSchema extends z.ZodType>(
  details?: DetailsSchema,
) =>
  z.object({
    key: z.string(),
    variant: variantZ,
    message: z.string(),
    description: z.string().optional(),
    time: TimeStamp.z,
    details: details ?? z.unknown().optional(),
  });

type Base<V extends Variant> = {
  key: string;
  variant: V;
  message: string;
  description?: string;
  time: TimeStamp;
};

export type Status<Details = never, V extends Variant = Variant> = Base<V> &
  ([Details] extends [never] ? {} : { details: Details });

export type Crude<Details = never, V extends Variant = Variant> = Optional<
  Base<V>,
  "key" | "time"
> &
  ([Details] extends [never] ? {} : { details: Details });

export interface ExceptionDetails {
  stack: string;
}

export const fromException = (
  exc: unknown,
  message?: string,
): Status<ExceptionDetails, "error"> => {
  if (!(exc instanceof Error)) throw exc;
  return create<ExceptionDetails, "error">({
    variant: "error",
    message: message ?? exc.message,
    description: message != null ? exc.message : undefined,
    details: {
      stack: exc.stack ?? "",
    },
  });
};

export const create = <Details = never, V extends Variant = Variant>(
  spec: Crude<Details, V>,
): Status<Details, V> =>
  ({
    key: id.create(),
    time: TimeStamp.now(),
    ...spec,
  }) as Status<Details, V>;

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
