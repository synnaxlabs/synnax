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

const NUMERIC_DATA_TYPES = {
  float: DataType.FLOAT64.toString(),
  integer: DataType.INT64.toString(),
  unsigned: DataType.UINT64.toString(),
};

/**
 * @returns the data type of the channel that configure creates for a read field. A
 * numeric field without one takes the widest data type of its kind.
 */
export const dataTypeOf = (options: ReadOptions): string => {
  switch (options.type) {
    case "float":
    case "integer":
    case "unsigned":
      return options.dataType?.toString() ?? NUMERIC_DATA_TYPES[options.type];
    default:
      return options.type;
  }
};

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

const MAX_FLOAT32 = 3.4028234663852886e38;

const toFloat = (value: number | string, dataType: DataType): number | null => {
  let f: number;
  if (typeof value === "number") f = value;
  else {
    if (!DECIMAL.test(value)) return null;
    const n = BigInt(value);
    f = Number(n);
    if (BigInt(f) !== n) return null;
  }
  const fits =
    !dataType.equals(DataType.FLOAT32) ||
    !Number.isFinite(f) ||
    Math.abs(f) <= MAX_FLOAT32;
  return fits ? f : null;
};

const toDecimal = (
  value: number | string,
  kind: "integer" | "unsigned",
  dataType: DataType,
): string | null => {
  if (typeof value === "number" ? !Number.isInteger(value) : !DECIMAL.test(value))
    return null;
  const n = BigInt(value);
  const bits = BigInt(dataType.density.valueOf() * 8);
  const [min, max] =
    kind === "integer"
      ? [-(2n ** (bits - 1n)), 2n ** (bits - 1n) - 1n]
      : [0n, 2n ** bits - 1n];
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
// exactly and that fits dataType, as the Core does when it reconciles a field with its
// channel.
const convertNumeric = (
  kind: NumericOptions["type"],
  dataType: DataType,
  previous: NumericOptions,
): NumericOptions => {
  const entries: NumericEntry[] = previous.enumValues;
  const fill = previous.fill;
  if (kind === "float") {
    const convert = (v: number | string) => toFloat(v, dataType);
    return {
      type: kind,
      dataType,
      enumValues: convertEntries(entries, convert),
      fill: fill == null ? undefined : (convert(fill) ?? undefined),
    };
  }
  const convert = (v: number | string) => toDecimal(v, kind, dataType);
  return {
    type: kind,
    dataType,
    enumValues: convertEntries(entries, convert),
    fill: fill == null ? undefined : (convert(fill) ?? undefined),
  };
};

// Drops each entry and the fill of previous that parse but do not fit dataType. A value
// that does not parse yet, such as a half-typed row, stays for the deploy check.
const narrowNumeric = (
  dataType: DataType,
  previous: NumericOptions,
): NumericOptions => {
  const fits = (v: number | string): boolean => {
    if (typeof v === "string" && !DECIMAL.test(v)) return true;
    if (previous.type === "float") return toFloat(v, dataType) != null;
    return toDecimal(v, previous.type, dataType) != null;
  };
  const keep = <V extends number | string>(fill?: V): V | undefined =>
    fill == null || fits(fill) ? fill : undefined;
  switch (previous.type) {
    case "float":
      return {
        ...previous,
        dataType,
        enumValues: previous.enumValues.filter(({ value }) => fits(value)),
        fill: keep(previous.fill),
      };
    case "integer":
      return {
        ...previous,
        dataType,
        enumValues: previous.enumValues.filter(({ value }) => fits(value)),
        fill: keep(previous.fill),
      };
    case "unsigned":
      return {
        ...previous,
        dataType,
        enumValues: previous.enumValues.filter(({ value }) => fits(value)),
        fill: keep(previous.fill),
      };
  }
};

/**
 * @returns the options of a read field into a new channel of dataType. A field of the
 * same kind keeps its options, but a numeric one drops each entry and the fill that
 * does not fit dataType. Between numeric kinds, each entry and the fill that the new
 * kind holds exactly, and that fit dataType, carry over.
 * @throws {Error} if dataType is not a known data type.
 */
export const readOptionsOf = (
  dataType: string,
  previous?: ReadOptions,
): ReadOptions => {
  const kind = payload.kindOf(dataType);
  if (kind == null) throw new Error(`${dataType} is not a known data type`);
  if (previous?.type === kind)
    return isNumeric(previous)
      ? narrowNumeric(new DataType(dataType), previous)
      : previous;
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
