// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { http } from "@synnaxlabs/client";
import { DataType, payload } from "@synnaxlabs/x";

import { type ReadField } from "@/feature/http/task/types";

const KIND_DATA_TYPES: Record<payload.Kind, DataType> = {
  float: DataType.FLOAT64,
  integer: DataType.INT64,
  unsigned: DataType.UINT64,
  string: DataType.STRING,
  boolean: DataType.BOOLEAN,
  timestamp: DataType.TIMESTAMP,
  json: DataType.JSON,
  uuid: DataType.UUID,
  bytes: DataType.BYTES,
};

const kindOf = (dataType: DataType): payload.Kind => {
  if (dataType.isFloat) return "float";
  if (dataType.isSignedInteger) return "integer";
  if (dataType.isUnsignedInteger) return "unsigned";
  return payload.kindZ.parse(dataType.toString());
};

/** @returns the data type of a new channel for the field. */
export const dataTypeOf = (field: ReadField): DataType =>
  "dataType" in field && field.dataType != null
    ? new DataType(field.dataType)
    : KIND_DATA_TYPES[field.type];

type NumericKind = "float" | "integer" | "unsigned";

type NumericField = Extract<ReadField, { type: NumericKind }>;

const isNumericKind = (kind: payload.Kind): kind is NumericKind =>
  kind === "float" || kind === "integer" || kind === "unsigned";

const isNumeric = (field: ReadField): field is NumericField =>
  isNumericKind(field.type);

const MAX_FLOAT32 = 3.4028234663852886e38;

const NON_FINITE = new Set(["NaN", "Infinity", "-Infinity"]);

const INTEGER_TEXT = /^-?\d+$/;

/**
 * @returns v, a float or the decimal text of an integer, as a value of the kind of the
 * data type, or undefined when a channel of the data type cannot hold it exactly.
 */
const convertNumber = (
  v: number | string,
  kind: NumericKind,
  dataType: DataType,
): number | string | undefined => {
  // Text is an integer, or the text of a non-finite float that the Core sent.
  if (typeof v === "string" && !INTEGER_TEXT.test(v))
    return kind === "float" && NON_FINITE.has(v) ? Number(v) : undefined;
  const f = typeof v === "number" ? v : Number(v);
  if (kind === "float") {
    if (typeof v === "string" && Number.isFinite(f) && BigInt(f) !== BigInt(v))
      return undefined;
    if (
      dataType.equals(DataType.FLOAT32) &&
      Number.isFinite(f) &&
      Math.abs(f) > MAX_FLOAT32
    )
      return undefined;
    return f;
  }
  if (!Number.isInteger(f)) return undefined;
  const n = BigInt(v);
  const bits = BigInt(dataType.density.valueOf() * 8);
  const [min, max] =
    kind === "integer"
      ? [-(2n ** (bits - 1n)), 2n ** (bits - 1n) - 1n]
      : [0n, 2n ** bits - 1n];
  return n >= min && n <= max ? n.toString() : undefined;
};

/**
 * @returns v as a boolean that is true when v is not 0, or undefined for NaN or text
 * that is neither an integer nor a non-finite float.
 */
const toBoolean = (v: number | string): boolean | undefined => {
  if (typeof v === "string" && !INTEGER_TEXT.test(v) && !NON_FINITE.has(v))
    return undefined;
  const f = typeof v === "number" ? v : Number(v);
  return Number.isNaN(f) ? undefined : f !== 0;
};

const convertValues = <V>(
  field: NumericField,
  convert: (v: number | string) => V | undefined,
) => ({
  fill: field.fill == null ? undefined : convert(field.fill),
  enumValues: field.enumValues.flatMap(({ label, value }) => {
    const v = convert(value);
    return v == null ? [] : [{ label, value: v }];
  }),
});

/**
 * @returns the field with the kind of the data type, converting the fill and enum
 * entries as the Core does. On float, integer, and unsigned kinds, they convert when a
 * channel of the data type holds them exactly, and on a boolean kind a number is true
 * when it is not 0. A field of another kind keeps only its base fields.
 */
export const retype = (field: ReadField, dataType: DataType): ReadField => {
  const type = kindOf(dataType);
  if (type === field.type && !isNumeric(field)) return field;
  const { key, name, disabled, channel, pointer } = field;
  const convert:
    ((v: number | string) => number | string | boolean | undefined) | null =
    type === "boolean"
      ? toBoolean
      : isNumericKind(type)
        ? (v: number | string) => convertNumber(v, type, dataType)
        : null;
  const converted =
    isNumeric(field) && convert != null ? convertValues(field, convert) : {};
  return http.READ_FIELD_SCHEMAS[type].parse({
    key,
    name,
    disabled,
    channel,
    pointer,
    type,
    dataType: dataType.toString(),
    timeFormat: "iso8601",
    ...converted,
  });
};

/** @returns the field, retyped to the kind of the data type when they differ. */
export const matchKind = (field: ReadField, dataType?: DataType): ReadField =>
  dataType == null || kindOf(dataType) === field.type ? field : retype(field, dataType);

/** The fields that take a fill from the form. */
export type FillField = Extract<
  ReadField,
  { type: "float" | "integer" | "unsigned" | "string" | "boolean" }
>;

/** @returns whether the form takes a fill for the field. */
export const hasFill = (field: ReadField): field is FillField =>
  field.type === "float" ||
  field.type === "integer" ||
  field.type === "unsigned" ||
  field.type === "string" ||
  field.type === "boolean";

/**
 * @returns the text of the fill. A NaN float fill is empty, as is no fill. The cache
 * holds a non-finite float that the Core sent as its text.
 */
export const formatFill = (field: FillField): string =>
  field.fill == null || (field.type === "float" && Number.isNaN(Number(field.fill)))
    ? ""
    : String(field.fill);

/**
 * @returns the field with the fill that the text states, or null when the text does not
 * fit the kind. Empty text is a NaN fill on a float and no fill on other kinds. A
 * string fill keeps its text as typed, so only empty text clears it.
 */
export const parseFill = (
  field: FillField,
  text: string,
  dataType: DataType,
): FillField | null => {
  const trimmed = text.trim();
  if (field.type === "float") {
    const fill = trimmed === "" ? NaN : Number(trimmed);
    const valid = Number.isFinite(fill) || trimmed === "" || NON_FINITE.has(trimmed);
    return valid && convertNumber(fill, "float", dataType) != null
      ? { ...field, fill }
      : null;
  }
  if (field.type === "string")
    return { ...field, fill: text === "" ? undefined : text };
  if (trimmed === "") return { ...field, fill: undefined };
  switch (field.type) {
    case "integer":
      return payload.integerZ.safeParse(trimmed).success &&
        convertNumber(trimmed, "integer", dataType) != null
        ? { ...field, fill: trimmed }
        : null;
    case "unsigned":
      return payload.unsignedZ.safeParse(trimmed).success &&
        convertNumber(trimmed, "unsigned", dataType) != null
        ? { ...field, fill: trimmed }
        : null;
    case "boolean":
      return trimmed === "true" || trimmed === "false"
        ? { ...field, fill: trimmed === "true" }
        : null;
  }
};
