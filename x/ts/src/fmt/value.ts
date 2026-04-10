// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { narrow } from "@/narrow";

export interface Options {
  maxStringLength?: number;
  maxArrayLength?: number;
  maxDepth?: number;
  redactKeys?: Iterable<string>;
}

const DEFAULT_MAX_STRING_LENGTH = 200;
const DEFAULT_MAX_ARRAY_LENGTH = 10;
const DEFAULT_MAX_DEPTH = 8;
const REDACTED = "[REDACTED]";

export const DEFAULT_REDACT_KEYS: ReadonlySet<string> = new Set([
  "password",
  "passwd",
  "secret",
  "token",
  "apikey",
  "api_key",
  "accesskey",
  "access_key",
  "privatekey",
  "private_key",
  "authorization",
  "auth",
  "credentials",
]);

const normalizeRedactKeys = (keys?: Iterable<string>): ReadonlySet<string> => {
  if (keys == null) return DEFAULT_REDACT_KEYS;
  const out = new Set<string>();
  for (const k of keys) out.add(k.toLowerCase());
  return out;
};

/**
 * Produces a safe, JSON-friendly representation of an arbitrary value for logging and
 * display. Strings, arrays, and object depth are capped; sensitive keys (passwords,
 * tokens, credentials) are replaced with `[REDACTED]`; non-plain values such as `Date`,
 * `Error`, functions, and symbols are rendered as short bracketed tags.
 *
 * Intended for user-facing error messages, debug output, and any situation where the
 * input may be untrusted or unbounded in size. The output is always structurally
 * cloneable and safe to JSON.stringify.
 */
export const value = (input: unknown, options: Options = {}): unknown => {
  const maxStringLength = options.maxStringLength ?? DEFAULT_MAX_STRING_LENGTH;
  const maxArrayLength = options.maxArrayLength ?? DEFAULT_MAX_ARRAY_LENGTH;
  const maxDepth = options.maxDepth ?? DEFAULT_MAX_DEPTH;
  const redactKeys = normalizeRedactKeys(options.redactKeys);

  const walk = (v: unknown, depth: number): unknown => {
    if (v === null) return null;
    if (v === undefined) return "[undefined]";
    const t = typeof v;
    if (t === "string") {
      const s = v as string;
      if (s.length > maxStringLength)
        return `${s.slice(0, maxStringLength)}…(+${s.length - maxStringLength} chars)`;
      return s;
    }
    if (t === "number" || t === "boolean") return v;
    if (t === "bigint") return `${(v as bigint).toString()}n`;
    if (t === "symbol" || t === "function") return `[${t}]`;
    if (Array.isArray(v)) {
      if (depth >= maxDepth) return `[Array(${v.length})]`;
      const items: unknown[] = v
        .slice(0, maxArrayLength)
        .map((item) => walk(item, depth + 1));
      if (v.length > maxArrayLength)
        items.push(`…(+${v.length - maxArrayLength} more)`);
      return items;
    }
    if (narrow.isPlainObject(v)) {
      if (depth >= maxDepth) return "[Object]";
      const out: Record<string, unknown> = {};
      for (const [k, val] of Object.entries(v))
        out[k] = redactKeys.has(k.toLowerCase()) ? REDACTED : walk(val, depth + 1);
      return out;
    }
    if (v instanceof Date) return v.toISOString();
    if (v instanceof Error) return `[Error: ${v.message}]`;
    try {
      return `[${Object.prototype.toString.call(v)}]`;
    } catch {
      return "[unknown]";
    }
  };

  return walk(input, 0);
};

/**
 * Returns a pretty-printed JSON rendering of `input` after passing it through
 * {@link value}. Safe for untrusted or unbounded data.
 */
export const stringify = (input: unknown, options: Options = {}): string => {
  try {
    return JSON.stringify(value(input, options), null, 2);
  } catch {
    return String(input);
  }
};
