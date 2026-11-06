// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import z from "zod";

/** Reads a field that v0.60 shipped into the fields that replace it. */
export interface Shipped<I, V> {
  /** Ends the deprecation warning, such as "Use `not: { dataType }`." */
  hint: string;
  /** Receives every value except undefined. Returns no fields for an unset value. */
  read: (value: V) => Partial<I>;
}

/** ignoreNotFoundError, which a list retrieve no longer needs. */
export const ignoreNotFoundError: Shipped<object, boolean> = {
  hint: "A list retrieve drops missing keys.",
  read: () => ({}),
};

type Renames<I> = Record<string, (keyof I & string) | Shipped<I, never>>;

type ShippedInput<I, R> = {
  [F in keyof R]?: R[F] extends keyof I
    ? Extract<I[R[F]], readonly unknown[]>
    : R[F] extends Shipped<I, infer V>
      ? V
      : never;
};

/**
 * Wraps a flat retrieve schema so it also takes the fields that v0.60 shipped. Each one
 * warns, then reads as the fields that replace it. A field mapped to a name reads its
 * list under that name, and an empty list reads as unset, as the v0.60 Core read it. A
 * replacement that the input already sets is ANDed with it. Remove in v0.62.
 */
export const shippedZ = <Z extends z.ZodType, R extends Renames<z.input<Z>>>(
  schema: Z,
  renames: R,
) =>
  z.preprocess((input: z.input<Z> & ShippedInput<z.input<Z>, R>) => {
    if (typeof input !== "object" || input === null) return input;
    let out: Record<string, unknown> = input;
    for (const [shipped, rename] of Object.entries(renames)) {
      if (!(shipped in out)) continue;
      const { [shipped]: value, ...rest } = out;
      out = rest;
      const hint = typeof rename === "string" ? `Use ${rename}.` : rename.hint;
      console.warn(`${shipped} is deprecated and will be removed in v0.62. ${hint}`);
      if (value === undefined) continue;
      let fields: Record<string, unknown>;
      if (typeof rename !== "string") fields = rename.read(value as never);
      else if (Array.isArray(value) && value.length === 0) fields = {};
      else fields = { [rename]: value };
      if (Object.keys(fields).some((field) => field in out))
        out.and = [...((out.and as unknown[] | undefined) ?? []), fields];
      else out = { ...out, ...fields };
    }
    return out;
  }, schema);
