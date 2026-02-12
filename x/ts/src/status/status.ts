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
import { label } from "@/label";
import { narrow } from "@/narrow";
import { type optional } from "@/optional";
import { type Status, type Variant } from "@/status/types.gen";
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
): Status<typeof exceptionDetailsSchema, z.ZodLiteral<"error">> => {
  if (!(exc instanceof Error)) throw exc;
  return create<typeof exceptionDetailsSchema, "error">({
    variant: "error",
    message: message ?? exc.message,
    description: message != null ? exc.message : undefined,
    details: { stack: exc.stack ?? "", error: exc },
  });
};

export const create = <
  DetailsSchema extends z.ZodType = z.ZodNever,
  V extends Variant = Variant,
>(
  spec: Crude<DetailsSchema, V>,
): Status<DetailsSchema, z.ZodType<V>> =>
  ({
    key: id.create(),
    time: TimeStamp.now(),
    name: "",
    ...spec,
  }) as unknown as Status<DetailsSchema, z.ZodType<V>>;

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

export interface ToStringOptions {
  includeTimestamp?: boolean;
  includeName?: boolean;
}

const DEFAULT_TO_STRING_OPTIONS: ToStringOptions = {
  includeTimestamp: false,
  includeName: true,
};

export const toString = <Details = never>(
  stat: Status<Details>,
  options: ToStringOptions = {},
): string => {
  const opts = { ...DEFAULT_TO_STRING_OPTIONS, ...options };
  const parts: string[] = [];
  let header = stat.variant.toUpperCase();
  if (opts.includeName && stat.name.length > 0) header += ` [${stat.name}]`;
  header += `: ${stat.message}`;
  if (opts.includeTimestamp) header += ` (${stat.time.toString("dateTime", "local")})`;
  parts.push(header);
  if (stat.description != null) {
    let descriptionText: string;
    try {
      const parsed = JSON.parse(stat.description);
      descriptionText = `Description:\n${JSON.stringify(parsed, null, 2)}`;
    } catch {
      descriptionText = `Description: ${stat.description}`;
    }
    parts.push(descriptionText);
  }
  if ("details" in stat && narrow.isObject(stat.details)) {
    const details = stat.details as Record<string, unknown>;
    // Extract stack trace separately for special formatting
    if ("stack" in details) parts.push(`Stack Trace:\n${String(details.stack)}`);
    // Include other details (excluding stack and error which don't serialize well)
    const extraDetails = Object.fromEntries(
      Object.entries(details).filter(([k]) => k !== "stack" && k !== "error"),
    );
    if (Object.keys(extraDetails).length > 0)
      parts.push(`Details:\n${JSON.stringify(extraDetails, null, 2)}`);
  }
  return parts.join("\n\n");
};
