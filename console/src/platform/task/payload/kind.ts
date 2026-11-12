// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeDataType, DataType, type payload } from "@synnaxlabs/x";

/** Returns the kind of the read field that writes to a channel of the data type. */
export const kindOf = (dataType: CrudeDataType): payload.Kind => {
  const dt = new DataType(dataType);
  if (dt.isFloat) return "float";
  if (dt.isSignedInteger) return "integer";
  if (dt.isUnsignedInteger) return "unsigned";
  if (dt.equals(DataType.BOOLEAN)) return "boolean";
  if (dt.equals(DataType.TIMESTAMP)) return "timestamp";
  if (dt.equals(DataType.JSON)) return "json";
  if (dt.equals(DataType.UUID)) return "uuid";
  if (dt.equals(DataType.BYTES)) return "bytes";
  return "string";
};

/** A read field of any kind, as each integration declares it. */
export interface ReadField {
  type: payload.Kind;
  dataType?: CrudeDataType;
}

/** The data type of a new channel for each kind, unless the field states its own. */
const DATA_TYPES: Record<payload.Kind, DataType> = {
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

/** Returns true when a field of the kind states the data type of a new channel. */
export const isNumeric = (kind: payload.Kind): boolean =>
  kind === "float" || kind === "integer" || kind === "unsigned";

/** Returns the data type of the channel that deploying a read field creates. */
export const dataTypeOf = ({ type, dataType }: ReadField): string =>
  new DataType(dataType ?? DATA_TYPES[type]).toString();
