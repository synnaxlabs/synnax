// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { errors, id, narrow, type optional, primitive, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

export const VARIANTS = [
  "success",
  "info",
  "warning",
  "error",
  "loading",
  "disabled",
] as const;
export const variantZ = z.enum(VARIANTS);
export type Variant = z.infer<typeof variantZ>;

interface Base<V extends Variant> {
  key: string;
  name: string;
  variant: V;
  message: string;
  description: string;
  time: TimeStamp;
}

/** A status carrying optional structured details. */
export type Status<
  Details extends z.ZodType = z.ZodNever,
  V extends Variant = Variant,
> = Base<V> & ([Details] extends [z.ZodNever] ? {} : { details: z.output<Details> });

/** A status before it is stamped with a key, a time, a name, and a description. */
export type Crude<
  Details extends z.ZodType = z.ZodNever,
  V extends Variant = Variant,
> = optional.Optional<Base<V>, "key" | "time" | "name" | "description"> &
  ([Details] extends [z.ZodNever] ? {} : { details: z.output<Details> });

/** Stamps a {@link Crude} status with a fresh key and the current time. */
export const create = <
  Details extends z.ZodType = z.ZodNever,
  V extends Variant = Variant,
>(
  spec: Crude<Details, V>,
): Status<Details, V> =>
  ({
    key: id.create(),
    time: TimeStamp.now(),
    name: "",
    description: "",
    ...spec,
  }) as Status<Details, V>;

/** Details a status built by {@link fromException} carries. */
export const exceptionDetailsZ = z.object({
  stack: z.string(),
  error: z.instanceof(Error),
});

const causeChain = (err: Error): string | undefined => {
  const parts: string[] = [];
  let cause: unknown = err.cause;
  for (let i = 0; i < 8 && cause instanceof Error; i++) {
    if (cause.message.length > 0) parts.push(cause.message);
    cause = cause.cause;
  }
  return parts.length === 0 ? undefined : parts.join(": ");
};

/**
 * Turns any thrown value into an error status. The message argument becomes the
 * status message and pushes the error's own message into the description, ahead of
 * its flattened cause chain; the error and its stack ride on the details.
 */
export const fromException = (
  exc: unknown,
  message?: string,
): Status<typeof exceptionDetailsZ, "error"> => {
  const err = errors.fromUnknown(exc);
  const chain = causeChain(err);
  return create<typeof exceptionDetailsZ, "error">({
    variant: "error",
    message: message ?? err.message,
    description:
      message != null
        ? [err.message, chain].filter((p) => p != null).join(": ")
        : chain,
    details: { stack: err.stack ?? "", error: err },
  });
};

/** @returns the variant unless it is one of the removed ones. */
export const removeVariants = (
  variant?: Variant,
  remove: Variant | Variant[] = [],
): Variant | undefined => {
  if (variant == null) return undefined;
  if (Array.isArray(remove)) return remove.includes(variant) ? undefined : variant;
  return remove === variant ? undefined : variant;
};

const renderDescription = (description: string): string => {
  if (description.includes("\n")) return `Description:\n${description}`;
  try {
    return `Description:\n${JSON.stringify(JSON.parse(description), null, 2)}`;
  } catch {
    return `Description: ${description}`;
  }
};

/** Renders a status as readable text, pretty-printing a JSON description. */
export const toString = <Details extends z.ZodType = z.ZodNever>(
  stat: Status<Details>,
): string => {
  const parts: string[] = [];
  let header = stat.variant.toUpperCase();
  if (primitive.isNonZero(stat.name)) header += ` [${stat.name}]`;
  parts.push(`${header}: ${stat.message}`);
  if (primitive.isNonZero(stat.description))
    parts.push(renderDescription(stat.description));
  if ("details" in stat && narrow.isObject(stat.details)) {
    const details = stat.details as Record<string, unknown>;
    if (typeof details.stack === "string" && details.stack !== "")
      parts.push(`Stack Trace:\n${details.stack}`);
    const extra = Object.fromEntries(
      Object.entries(details).filter(([k]) => k !== "stack" && k !== "error"),
    );
    if (Object.keys(extra).length > 0)
      parts.push(`Details:\n${JSON.stringify(extra, null, 2)}`);
  }
  return parts.join("\n\n");
};
