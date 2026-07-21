// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  errors,
  id,
  narrow,
  type optional,
  primitive,
  record,
  TimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type Status, type Variant } from "@/status/types.gen";

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

/**
 * Interface that errors may optionally implement to provide richer rendering when
 * passed to {@link fromException}. Implementers return a partial {@link Crude} spec
 * whose fields override the defaults derived from the underlying `Error`.
 *
 * This is a duck-typed contract: `fromException` checks for the presence of a
 * `toStatus` method via the `in` operator, so there is no need to import this
 * interface to use it.
 */
export interface Custom {
  toStatus(): Partial<Crude<z.ZodRecord, "error">>;
}

const customReturnZ = z.object({
  message: z.string().optional(),
  description: z.string().optional(),
  details: record.unknownZ().optional(),
});

const safeToStatus = (exc: unknown): z.infer<typeof customReturnZ> | undefined => {
  if (
    exc == null ||
    typeof exc !== "object" ||
    !("toStatus" in exc) ||
    typeof exc.toStatus !== "function"
  )
    return undefined;
  let raw: unknown;
  try {
    raw = exc.toStatus();
  } catch {
    return undefined;
  }
  const parsed = customReturnZ.safeParse(raw);
  return parsed.success ? parsed.data : undefined;
};

export const exceptionDetailsSchema = z
  .object({
    stack: z.string(),
    error: z.instanceof(Error),
  })
  .and(record.unknownZ());

export const fromException = (
  exc: unknown,
  message?: string,
): Status<typeof exceptionDetailsSchema, z.ZodLiteral<"error">> => {
  const err = errors.fromUnknown(exc);
  const crude: Crude<typeof exceptionDetailsSchema, "error"> = {
    variant: "error",
    message: message ?? err.message,
    description: message != null ? err.message : undefined,
    details: { stack: err.stack ?? "", error: err },
  };
  // Probe the original (pre-coercion) value so a non-Error throwable with a custom
  // `toStatus()` method still contributes its status fields.
  const custom = safeToStatus(exc);
  if (custom != null) {
    if (message != null && custom.message != null)
      crude.message = `${message}: ${custom.message}`;
    else if (custom.message != null) crude.message = custom.message;
    if (custom.description != null) crude.description = custom.description;
    if (custom.details != null && crude.details != null)
      crude.details = { ...crude.details, ...custom.details };
  }
  return create<typeof exceptionDetailsSchema, "error">(crude);
};

/**
 * Converts an exception-shaped status (one built via {@link fromException}) back
 * into a thrown-shaped {@link Error}. The returned error carries the status's
 * wrapped message, copies `name` and `stack` from the inner error preserved on
 * `details.error`, and stashes the full status on `cause` for callers that need
 * the rich shape.
 *
 * Use this when bridging the status pipeline back into a context that expects
 * a real Error — typically before `throw`-ing across an error boundary.
 */
export const toError = (
  s: Status<typeof exceptionDetailsSchema, z.ZodLiteral<"error">>,
): Error => {
  const inner = s.details.error;
  const err = new Error(s.message, { cause: s });
  err.name = inner.name;
  err.stack = inner.stack;
  return err;
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
    description: spec.description ?? "",
  }) as Status<DetailsSchema, z.ZodType<V>>;

/**
 * Reads the details of a status whose schema is not statically known, as when
 * scanning the generically-typed status table for rows belonging to a domain.
 * @returns The details, or undefined when the status carries none.
 */
export const detailsOf = (status: Status): record.Unknown | undefined =>
  (status as { details?: record.Unknown }).details;

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

const renderDescription = (description: string): string => {
  if (description.includes("\n")) return `Description:\n${description}`;
  try {
    const parsed = JSON.parse(description);
    return `Description:\n${JSON.stringify(parsed, null, 2)}`;
  } catch {
    return `Description: ${description}`;
  }
};

export const toString = <Details extends z.ZodType = z.ZodNever>(
  stat: Status<Details>,
  options: ToStringOptions = {},
): string => {
  const opts = { ...DEFAULT_TO_STRING_OPTIONS, ...options };
  const parts: string[] = [];
  let header = stat.variant.toUpperCase();
  if (opts.includeName && primitive.isNonZero(stat.name)) header += ` [${stat.name}]`;
  header += `: ${stat.message}`;
  if (opts.includeTimestamp) header += ` (${stat.time.toString("dateTime", "local")})`;
  parts.push(header);
  if (stat.description != null) parts.push(renderDescription(stat.description));
  if ("details" in stat && narrow.isObject(stat.details)) {
    const details = stat.details as Record<string, unknown>;
    if ("stack" in details && typeof details.stack === "string" && details.stack !== "")
      parts.push(`Stack Trace:\n${details.stack}`);
    const extraDetails = Object.fromEntries(
      Object.entries(details).filter(([k]) => k !== "stack" && k !== "error"),
    );
    if (Object.keys(extraDetails).length > 0)
      parts.push(`Details:\n${JSON.stringify(extraDetails, null, 2)}`);
  }
  return parts.join("\n\n");
};
