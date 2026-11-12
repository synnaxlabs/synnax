// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, payload } from "@synnaxlabs/x";

/** The kind and the options of a read field. */
export type ReadOptions =
  | ({ type: "float" } & payload.FloatRead)
  | ({ type: "integer" } & payload.IntegerRead)
  | ({ type: "unsigned" } & payload.UnsignedRead)
  | ({ type: "string" } & payload.StringRead)
  | ({ type: "boolean" } & payload.BooleanRead)
  | ({ type: "timestamp" } & payload.TimestampRead)
  | { type: "json" | "uuid" | "bytes" };

type NumericOptions = Extract<ReadOptions, { type: "float" | "integer" | "unsigned" }>;

interface NumericEntry {
  label: string;
  value: number | string;
}

const isNumeric = (options?: ReadOptions): options is NumericOptions =>
  options?.type === "float" ||
  options?.type === "integer" ||
  options?.type === "unsigned";

const DECIMAL = /^-?[0-9]+$/;

const toFloat = (value: number | string): number | null => {
  if (typeof value === "number") return value;
  if (!DECIMAL.test(value)) return null;
  const n = BigInt(value);
  const f = Number(n);
  return BigInt(f) === n ? f : null;
};

const RANGES: Record<"integer" | "unsigned", readonly [bigint, bigint]> = {
  integer: [-(2n ** 63n), 2n ** 63n - 1n],
  unsigned: [0n, 2n ** 64n - 1n],
};

const toDecimal = (
  value: number | string,
  kind: "integer" | "unsigned",
): string | null => {
  if (typeof value === "number" ? !Number.isInteger(value) : !DECIMAL.test(value))
    return null;
  const n = BigInt(value);
  const [min, max] = RANGES[kind];
  return n >= min && n <= max ? n.toString() : null;
};

const convertEntries = <V>(
  entries: NumericEntry[],
  convert: (value: number | string) => V | null,
): { label: string; value: V }[] =>
  entries.flatMap(({ label, value }) => {
    const v = convert(value);
    return v == null ? [] : [{ label, value: v }];
  });

// Converts the entries and fill of previous to kind, keeping each one that kind holds
// exactly, as the Core does when it reconciles a field with its channel.
const convertNumeric = (
  kind: NumericOptions["type"],
  dataType: DataType,
  previous: NumericOptions,
): NumericOptions => {
  const entries: NumericEntry[] = previous.enumValues;
  const fill = previous.fill;
  if (kind === "float")
    return {
      type: kind,
      dataType,
      enumValues: convertEntries(entries, toFloat),
      fill: fill == null ? undefined : (toFloat(fill) ?? undefined),
    };
  const convert = (v: number | string) => toDecimal(v, kind);
  return {
    type: kind,
    dataType,
    enumValues: convertEntries(entries, convert),
    fill: fill == null ? undefined : (convert(fill) ?? undefined),
  };
};

/**
 * @returns the options of a read field into a new channel of dataType. A field of the
 * same kind keeps its options. Between numeric kinds, each entry and the fill that the
 * new kind holds exactly carry over.
 * @throws {Error} if dataType is not a known data type.
 */
export const readOptionsOf = (
  dataType: string,
  previous?: ReadOptions,
): ReadOptions => {
  const kind = payload.kindOf(dataType);
  if (kind == null) throw new Error(`${dataType} is not a known data type`);
  if (previous?.type === kind) {
    if (
      previous.type === "float" ||
      previous.type === "integer" ||
      previous.type === "unsigned"
    )
      return { ...previous, dataType: new DataType(dataType) };
    return previous;
  }
  switch (kind) {
    case "float":
    case "integer":
    case "unsigned":
      if (isNumeric(previous))
        return convertNumeric(kind, new DataType(dataType), previous);
      return { type: kind, dataType: new DataType(dataType), enumValues: [] };
    case "string":
    case "boolean":
      return { type: kind, enumValues: [] };
    case "timestamp":
      return { type: kind, timeFormat: "iso8601" };
    default:
      return { type: kind };
  }
};
