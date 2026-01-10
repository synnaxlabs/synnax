// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { id } from "@/id";
import { type optional } from "@/optional";
import { type Status, type Variant, type variantZ } from "@/status/types.gen";
import { TimeStamp } from "@/telem";

// Input type for creating statuses - uses conditional typing for optional details
type Base<V extends Variant> = {
  key: string;
  name: string;
  variant: V;
  message: string;
  description?: string;
  time: TimeStamp;
};

export type Crude<
  DetailsSchema extends z.ZodType = z.ZodNever,
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
): Status<typeof exceptionDetailsSchema> => {
  if (!(exc instanceof Error)) throw exc;
  return create<typeof exceptionDetailsSchema>({
    variant: "error",
    message: message ?? exc.message,
    description: message != null ? exc.message : undefined,
    details: { stack: exc.stack ?? "", error: exc },
  });
};

export const create = <
  DetailsSchema extends z.ZodType = z.ZodNever,
  V extends typeof variantZ = typeof variantZ,
>(
  spec: Crude<DetailsSchema, z.infer<V>>,
): Status<DetailsSchema, V> =>
  ({
    key: id.create(),
    time: TimeStamp.now(),
    name: "",
    ...spec,
  }) as unknown as Status<DetailsSchema, V>;

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
