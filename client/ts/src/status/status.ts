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

/** A status before it is stamped with a key, a time, and a name. */
export type Crude<
  DetailsSchema extends z.ZodType = z.ZodNever,
  V extends Variant = Variant,
> = optional.Optional<Base<V>, "key" | "time" | "name"> &
  ([DetailsSchema] extends [z.ZodNever] ? {} : { details: z.output<DetailsSchema> });

/**
 * An error that implements this gives {@link fromException} richer text. Its message
 * leads the status description, and its description and details replace the defaults
 * built from the error. fromException finds the method by duck typing, so implementers
 * need not import this interface.
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

/** Details a status built by {@link fromException} carries. */
export const exceptionDetailsSchema = z
  .object({
    stack: z.string(),
    error: z.instanceof(Error),
  })
  .and(record.unknownZ());

// Bounds the clone check so it stays cheap even when errors are created in a tight
// loop. An exhausted budget reports un-cloneable, which safely over-rebuilds.
const CLONE_CHECK_BUDGET = 64;

const isCloneableValue = (
  v: unknown,
  seen: Set<object>,
  budget: { left: number },
): boolean => {
  if (--budget.left < 0) return false;
  if (v == null) return true;
  const t = typeof v;
  if (t === "function" || t === "symbol") return false;
  if (t !== "object") return true;
  const o = v;
  if (seen.has(o)) return true;
  seen.add(o);
  const values = (vals: unknown[]): boolean =>
    vals.every((e) => isCloneableValue(e, seen, budget));
  if (o instanceof Error)
    return isCloneableValue(o.cause, seen, budget) && values(Object.values(o));
  if (Array.isArray(o)) return values(o);
  const proto = Object.getPrototypeOf(o);
  if (proto === Object.prototype || proto === null) return values(Object.values(o));
  if (o instanceof Date || o instanceof ArrayBuffer || ArrayBuffer.isView(o))
    return true;
  // Host objects (an Event, a DOM node) and exotic containers land here.
  return false;
};

// Object.values can trip an enumerable getter that throws. Treat a throwing value
// as not cloneable rather than throwing while already handling an error.
const isCloneable = (v: unknown): boolean => {
  try {
    return isCloneableValue(v, new Set(), { left: CLONE_CHECK_BUDGET });
  } catch {
    return false;
  }
};

// Returns a cloneable stand-in for an un-cloneable value.
const cloneSafeValue = (v: unknown, depth: number): unknown => {
  if (isCloneable(v)) return v;
  if (v instanceof Error) return cloneSafeError(v, depth);
  if (depth <= 0) return errors.fromUnknown(v).message;
  try {
    if (Array.isArray(v)) return v.map((e) => cloneSafeValue(e, depth - 1));
    if (typeof v === "object" && v != null) {
      const out: Record<string, unknown> = {};
      for (const [key, value] of Object.entries(v))
        if (typeof value !== "function" && typeof value !== "symbol")
          out[key] = cloneSafeValue(value, depth - 1);
      return out;
    }
  } catch {
    // Tolerate throwing enumerable getters.
  }
  return errors.fromUnknown(v).message;
};

/**
 * Returns err unchanged when it survives structured cloning. Otherwise rebuilds it
 * with the same name, message, stack, and `type`, sanitizing un-cloneable fields and
 * causes. Statuses cross the worker boundary, where an un-cloneable error kills the
 * worker.
 */
const cloneSafeError = (err: Error, depth: number = 8): Error => {
  if (isCloneable(err)) return err;
  const safe = new Error(err.message);
  safe.name = err.name;
  safe.stack = err.stack;
  const { type } = err as { type?: unknown };
  if (typeof type === "string") (safe as { type?: string }).type = type;
  try {
    for (const [key, value] of Object.entries(err))
      if (key !== "cause" && typeof value !== "function" && typeof value !== "symbol")
        (safe as unknown as Record<string, unknown>)[key] = cloneSafeValue(
          value,
          depth - 1,
        );
  } catch {
    // Tolerate throwing enumerable getters.
  }
  const { cause } = err;
  if (cause !== undefined && depth > 0)
    if (cause instanceof Error) safe.cause = cloneSafeError(cause, depth - 1);
    else safe.cause = cloneSafeValue(cause, depth - 1);
  return safe;
};

/** Flattens an error's cause chain into one readable line. */
const causeChain = (err: Error): string | undefined => {
  const parts: string[] = [];
  let cause: unknown = err.cause;
  for (let i = 0; i < 8 && cause instanceof Error; i++) {
    if (cause.message.length > 0) parts.push(cause.message);
    cause = cause.cause;
  }
  if (parts.length === 0) return undefined;
  return parts.join(": ");
};

/**
 * The text a status built by {@link fromException} leads with: a message, or a message
 * and the name of what the status is about.
 */
export type Heading = string | Pick<Base<Variant>, "message" | "name">;

const joinParts = (parts: Array<string | undefined>, separator: string): string =>
  parts.filter((part) => part != null && part.length > 0).join(separator);

/**
 * Turns any thrown value into an error status. The heading sets the message and name.
 * The error's own message and its flattened cause chain form the description, and the
 * details keep the error and its stack.
 */
export const fromException = (
  exc: unknown,
  heading: Heading,
): Status<typeof exceptionDetailsSchema, z.ZodLiteral<"error">> => {
  const err = errors.fromUnknown(exc);
  const { message, name } =
    typeof heading === "string" ? { message: heading, name: "" } : heading;
  // Probe the original (pre-coercion) value so a non-Error throwable with a custom
  // `toStatus()` method still contributes its status fields.
  const custom = safeToStatus(exc);
  const reason = custom?.description ?? joinParts([err.message, causeChain(err)], ": ");
  return create<typeof exceptionDetailsSchema, "error">({
    variant: "error",
    name,
    message,
    description: joinParts([custom?.message, reason], ". "),
    details: { stack: err.stack ?? "", error: cloneSafeError(err), ...custom?.details },
  });
};

class ThrownStatus extends Error implements Custom {
  readonly description: string;

  constructor(s: Status) {
    super(s.message, { cause: s });
    this.description = s.description;
  }

  toStatus(): Pick<Status, "message" | "description"> {
    return { message: this.message, description: this.description };
  }
}

/**
 * Converts a status into an {@link Error} to throw. {@link fromException} keeps the
 * status message and description from it, and the error's cause is the status. A status
 * built by {@link fromException} also passes on its inner error's name and stack.
 */
export const toError = (s: Status): Error & Custom => {
  const err = new ThrownStatus(s);
  const inner = detailsOf(s)?.error;
  if (inner instanceof Error) {
    err.name = inner.name;
    err.stack = inner.stack;
  }
  return err;
};

/** Stamps a {@link Crude} status with a fresh key and the current time. */
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

/** @returns the variant when it is one of the kept ones, else undefined. */
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

/** @returns the variant unless it is one of the removed ones. */
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

/** Rank of each variant by severity, most severe lowest. */
const SEVERITY: Record<Variant, number> = {
  error: 0,
  warning: 1,
  loading: 2,
  success: 3,
  info: 4,
  disabled: 5,
};

/** @returns true when a is more severe than b. */
export const moreSevere = (a: Variant, b: Variant): boolean =>
  SEVERITY[a] < SEVERITY[b];

/**
 * @returns the most severe of the statuses, the first of equally severe ones, or
 * undefined when every status is null or undefined.
 */
export const worst = <S extends { variant: Variant }>(
  ...statuses: Array<S | null | undefined>
): S | undefined => {
  let out: S | undefined;
  for (const s of statuses)
    if (s != null && (out == null || moreSevere(s.variant, out.variant))) out = s;
  return out;
};

/** Options for {@link toString}. */
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

/** Renders a status as readable text, pretty-printing a JSON description. */
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
  if (primitive.isNonZero(stat.description))
    parts.push(renderDescription(stat.description));
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
