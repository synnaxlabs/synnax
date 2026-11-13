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

/**
 * @returns the field with the kind of the data type. A field of another kind keeps only
 * its base fields.
 */
export const retype = (field: ReadField, dataType: DataType): ReadField => {
  const type = kindOf(dataType);
  if (type === field.type) return "dataType" in field ? { ...field, dataType } : field;
  const { key, name, disabled, channel, pointer } = field;
  return http.READ_FIELD_SCHEMAS[type].parse({
    key,
    name,
    disabled,
    channel,
    pointer,
    type,
    dataType: dataType.toString(),
    timeFormat: "iso8601",
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

const NON_FINITE = new Set(["NaN", "Infinity", "-Infinity"]);

/**
 * @returns the field with the fill that the text states, or null when the text does not
 * fit the kind. Empty text is a NaN fill on a float and no fill on other kinds. A
 * string fill keeps its text as typed, so only empty text clears it.
 */
export const parseFill = (field: FillField, text: string): FillField | null => {
  const trimmed = text.trim();
  if (field.type === "float") {
    const fill = trimmed === "" ? NaN : Number(trimmed);
    return Number.isFinite(fill) || trimmed === "" || NON_FINITE.has(trimmed)
      ? { ...field, fill }
      : null;
  }
  if (field.type === "string")
    return { ...field, fill: text === "" ? undefined : text };
  if (trimmed === "") return { ...field, fill: undefined };
  switch (field.type) {
    case "integer":
      return payload.integerZ.safeParse(trimmed).success
        ? { ...field, fill: trimmed }
        : null;
    case "unsigned":
      return payload.unsignedZ.safeParse(trimmed).success
        ? { ...field, fill: trimmed }
        : null;
    case "boolean":
      return trimmed === "true" || trimmed === "false"
        ? { ...field, fill: trimmed === "true" }
        : null;
  }
};
