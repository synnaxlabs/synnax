// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, payload } from "@synnaxlabs/x";

/** The data type of the channel that a field of each kind reads into by default. */
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

/**
 * @returns whether the Core writes the values of a field kind to a channel of the data
 * type. A number or a boolean also writes to a float, and an unsigned number or a
 * boolean to any integer.
 */
export const converts = (kind: payload.Kind, dataType: string): boolean => {
  const to = payload.kindOf(dataType);
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

/** @returns the data type of the channel that deploying creates for a binding. */
export const bindingDataType = (
  { dataType }: { dataType: string },
  kind: payload.Kind,
): string => (dataType === "" ? DEFAULT_DATA_TYPES[kind] : dataType);
