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
  /** The name of what the status is about. Empty when it is not about a named thing. */
  name: string;
  variant: V;
  /**
   * What happened, in one short line, such as "Failed to save file" or "Saved file". A
   * status that reports a state names the state instead, such as "Disconnected".
   */
  message: string;
  /**
   * Why it happened and what to do now, in full sentences. Raw error text goes here,
   * never in the message. Empty when the message says enough.
   */
  description: string;
  time: TimeStamp;
  /**
   * Identifies the problem that a warning or error reports. A warning or error with the
   * same key and problem as an earlier one joins its notification.
   */
  problem?: string;
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
 * The text a status built by {@link fromException} leads with: a message, or a message
 * and the name of what the status is about.
 */
export type Heading = string | Pick<Base<Variant>, "message" | "name">;

/**
 * Turns any thrown value into an error status. The heading sets the message and name.
 * The error's own message and its flattened cause chain form the description, and the
 * details keep the error and its stack.
 */
export const fromException = (
  exc: unknown,
  heading: Heading,
): Status<typeof exceptionDetailsZ, "error"> => {
  const err = errors.fromUnknown(exc);
  const { message, name } =
    typeof heading === "string" ? { message: heading, name: "" } : heading;
  return create<typeof exceptionDetailsZ, "error">({
    variant: "error",
    name,
    message,
    description: [err.message, causeChain(err)].filter((p) => p != null).join(": "),
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
