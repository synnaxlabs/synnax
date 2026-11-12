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

/**
 * @returns the options of a read field into a new channel of dataType. A field of the
 * same kind keeps its options.
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
