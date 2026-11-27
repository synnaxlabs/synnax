// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { channel, icd } from "@synnaxlabs/client";
import { DataType, payload } from "@synnaxlabs/x";

import type { Field } from "@/platform/icd/read";

/** The data type of the channel that a field of each kind takes by default. */
const DEFAULT_DATA_TYPES: Record<payload.Kind, string> = {
  float: DataType.FLOAT64.toString(),
  integer: DataType.INT64.toString(),
  unsigned: DataType.UINT64.toString(),
  boolean: DataType.UINT8.toString(),
  string: DataType.STRING.toString(),
  timestamp: DataType.TIMESTAMP.toString(),
  json: DataType.JSON.toString(),
  uuid: DataType.UUID.toString(),
  bytes: DataType.BYTES.toString(),
};

/** The type that a picker lists for a value of each kind. */
export const VALUE_TYPES: Record<payload.Kind, string> = {
  ...DEFAULT_DATA_TYPES,
  boolean: DataType.BOOLEAN.toString(),
};

const isScaled = ({
  scale,
  offset,
}: Pick<icd.BaseField, "scale" | "offset">): boolean => scale !== 1 || offset !== 0;

/** @returns the kind of the values that a read takes from field. Text reads floats. */
const kindOf = (field: Field): payload.Kind => {
  if ("type" in field) return field.type;
  if ("kind" in field) return field.kind;
  return "float";
};

/** @returns whether a value of the kind also writes to a channel of kind to. */
const widens = (kind: payload.Kind, to: payload.Kind): boolean => {
  switch (to) {
    case "float":
      return kind === "integer" || kind === "unsigned" || kind === "boolean";
    case "integer":
      return kind === "unsigned" || kind === "boolean";
    case "unsigned":
      return kind === "boolean";
    default:
      return false;
  }
};

/**
 * @returns whether an integer channel of the data type holds the bits of field, and one
 * more bit for the sign of an unsigned field in a signed channel.
 */
const holds = (field: icd.BinaryField, to: payload.Kind, dataType: string): boolean => {
  if (to !== "integer" && to !== "unsigned") return true;
  const bits =
    field.bitLength + (to === "integer" && field.kind === "unsigned" ? 1 : 0);
  return bits <= new DataType(dataType).density.valueOf() * 8;
};

/**
 * @returns whether the Core writes the values of field to a channel of the data type. A
 * scaled field writes only to a float. Else a number or a boolean also writes to a
 * float, and an unsigned number or a boolean to any integer. A binary field reads into
 * an integer only when the integer holds its bits.
 */
export const converts = (field: Field, dataType: string): boolean => {
  const to = payload.kindOf(dataType);
  if (to == null) return false;
  if (isScaled(field)) return to === "float";
  const kind = kindOf(field);
  if (to !== kind && !widens(kind, to)) return false;
  return !("bitLength" in field) || holds(field, to, dataType);
};

const NUMERIC: ReadonlySet<payload.Kind> = new Set([
  "float",
  "integer",
  "unsigned",
  "boolean",
]);

/**
 * @returns whether the Core writes the samples of a channel of the data type to field.
 * A scaled field takes only a float. Else a number or a boolean writes to any number or
 * boolean field.
 */
export const encodes = (
  field: Pick<icd.JSONField, "type" | "scale" | "offset">,
  dataType: string,
): boolean => {
  const from = payload.kindOf(dataType);
  if (isScaled(field)) return from === "float";
  if (from === field.type) return true;
  return from != null && NUMERIC.has(from) && NUMERIC.has(field.type);
};

/**
 * @returns bindings, with each channel that takes refuses for its field unbound. A
 * binding with no channel, or whose field or channel is not found, stays.
 */
export const unbind = <
  B extends { field: icd.FieldKey; channel?: channel.Key },
  F extends Field,
>(
  bindings: B[],
  fields: F[],
  channels: channel.Channel[],
  takes: (field: F, dataType: string) => boolean,
): B[] => {
  const dataTypes = new Map(channels.map((c) => [c.key, c.dataType.toString()]));
  return bindings.map((b) => {
    const field = fields.find(({ key }) => key === b.field);
    const dataType = b.channel == null ? undefined : dataTypes.get(b.channel);
    if (field == null || dataType == null || takes(field, dataType)) return b;
    return { ...b, channel: 0 };
  });
};

/**
 * @returns the data type of the channel that a field takes by default. A scaled field
 * takes float64, because the Core scales only float channels.
 */
export const defaultDataType = (field: Field): string =>
  isScaled(field) ? DataType.FLOAT64.toString() : DEFAULT_DATA_TYPES[kindOf(field)];

/**
 * @returns the data type of the channel that deploying creates for a binding of field.
 * A binding with an empty or absent data type takes the default of field.
 */
export const bindingDataType = (
  { dataType = "" }: { dataType?: string },
  field: Field,
): string => (dataType === "" ? defaultDataType(field) : dataType);
