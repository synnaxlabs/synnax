// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type icd } from "@synnaxlabs/client";

/** The enum mapping and default of a JSON field of each number kind. */
export type NumberOptions =
  | Pick<icd.FloatJSONField, "type" | "enumValues" | "default">
  | Pick<icd.IntegerJSONField, "type" | "enumValues" | "default">
  | Pick<icd.UnsignedJSONField, "type" | "enumValues" | "default">;

type Kind = NumberOptions["type"];

type DecimalKind = Exclude<Kind, "float">;

type Value = number | string;

interface Held {
  enumValues: { label: string; value: Value }[];
  default?: Value;
}

const DECIMAL = /^-?[0-9]+$/;

const BOUNDS: Record<DecimalKind, [bigint, bigint]> = {
  integer: [-(2n ** 63n), 2n ** 63n - 1n],
  unsigned: [0n, 2n ** 64n - 1n],
};

const toFloat = (value: Value): number | null => {
  if (typeof value === "number") return value;
  if (!DECIMAL.test(value)) return null;
  const n = BigInt(value);
  const f = Number(n);
  return Number.isFinite(f) && BigInt(f) === n ? f : null;
};

const toDecimal = (value: Value, kind: DecimalKind): string | null => {
  if (typeof value === "number" ? !Number.isInteger(value) : !DECIMAL.test(value))
    return null;
  const n = BigInt(value);
  const [min, max] = BOUNDS[kind];
  return n >= min && n <= max ? n.toString() : null;
};

const carry = <V extends Value>(
  { enumValues, default: fill }: Held,
  convert: (value: Value) => V | null,
): { enumValues: { label: string; value: V }[]; default?: V } => ({
  enumValues: enumValues.flatMap(({ label, value }) => {
    const v = convert(value);
    return v == null ? [] : [{ label, value: v }];
  }),
  default: fill == null ? undefined : (convert(fill) ?? undefined),
});

/**
 * @returns the enum mapping and default of field as a JSON field of kind. Each entry
 * and the default that kind holds exactly carry over. On a field of kind, only a value
 * that parses and does not fit drops, so a half-typed one stays for the deploy check. A
 * field that holds no number starts with none.
 */
export const numberOptionsOf = (field: icd.JSONField, kind: Kind): NumberOptions => {
  const held: Held =
    field.type === "float" || field.type === "integer" || field.type === "unsigned"
      ? field
      : { enumValues: [] };
  if (kind === "float") return { type: kind, ...carry(held, toFloat) };
  const toKind = (value: Value): string | null => {
    const v = toDecimal(value, kind);
    if (field.type !== kind || typeof value !== "string") return v;
    return v != null || !DECIMAL.test(value) ? value : null;
  };
  return { type: kind, ...carry(held, toKind) };
};
