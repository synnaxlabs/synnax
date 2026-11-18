// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { narrow } from "@/narrow";

/**
 * Renders an object path as a human-readable string.
 *
 * - An empty path becomes `<root>`.
 * - String segments are joined with `.`.
 * - Numeric segments are rendered as bracket indices (`items[0]`).
 *
 * @example
 * formatPath([]) // "<root>"
 * formatPath(["config", "channels", 0, "port"]) // "config.channels[0].port"
 */
export const formatPath = (segments: ReadonlyArray<PropertyKey>): string => {
  if (segments.length === 0) return "<root>";
  let out = "";
  segments.forEach((p, i) => {
    if (typeof p === "number") out += `[${p}]`;
    else if (i === 0) out += String(p);
    else out += `.${String(p)}`;
  });
  return out;
};

export interface FormatOptions {
  maxStringLength?: number;
  maxArrayLength?: number;
  maxDepth?: number;
}

const DEFAULT_MAX_STRING_LENGTH = 200;
const DEFAULT_MAX_ARRAY_LENGTH = 10;
const DEFAULT_MAX_DEPTH = 8;

/**
 * Produces a safe, JSON-friendly representation of an arbitrary value for logging and
 * display. Strings, arrays, and object depth are capped; non-plain values such as
 * `Date`, `Error`, functions, and symbols are rendered as short bracketed tags.
 * Intended for user-facing error messages, debug output, and any situation where the
 * input may be untrusted or unbounded in size. The output is always structurally
 * cloneable and safe to JSON.stringify.
 */
export const formatValue = (input: unknown, options: FormatOptions = {}): unknown => {
  const maxStringLength = options.maxStringLength ?? DEFAULT_MAX_STRING_LENGTH;
  const maxArrayLength = options.maxArrayLength ?? DEFAULT_MAX_ARRAY_LENGTH;
  const maxDepth = options.maxDepth ?? DEFAULT_MAX_DEPTH;

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
      for (const [k, val] of Object.entries(v)) out[k] = walk(val, depth + 1);
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
