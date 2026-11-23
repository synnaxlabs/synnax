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

/** The data type of the channel that a field of each kind takes by default. */
export const DEFAULT_DATA_TYPES: Record<payload.Kind, string> = {
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

const isScaled = ({
  scale,
  offset,
}: Pick<icd.JSONField, "scale" | "offset">): boolean => scale !== 1 || offset !== 0;

/**
 * @returns whether the Core writes the values of field to a channel of the data type. A
 * scaled field writes only to a float. Else a number or a boolean also writes to a
 * float, and an unsigned number or a boolean to any integer.
 */
export const converts = (
  field: Pick<icd.JSONField, "type" | "scale" | "offset">,
  dataType: string,
): boolean => {
  const to = payload.kindOf(dataType);
  if (isScaled(field)) return to === "float";
  const kind = field.type;
  if (to === kind) return true;
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
export const unbind = <B extends { field: icd.FieldKey; channel?: channel.Key }>(
  bindings: B[],
  fields: icd.JSONField[],
  channels: channel.Channel[],
  takes: (field: icd.JSONField, dataType: string) => boolean,
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
export const defaultDataType = (field: icd.JSONField): string =>
  isScaled(field) ? DataType.FLOAT64.toString() : DEFAULT_DATA_TYPES[field.type];

/**
 * @returns the data type of the channel that deploying creates for a binding of field.
 * A binding with an empty or absent data type takes the default of field.
 */
export const bindingDataType = (
  { dataType = "" }: { dataType?: string },
  field: icd.JSONField,
): string => (dataType === "" ? defaultDataType(field) : dataType);
