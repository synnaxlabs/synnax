// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, payload } from "@synnaxlabs/x";

import {
  type BrowsedField,
  createReadField,
  type ReadField,
} from "@/feature/mqtt/task/types";
import { Task } from "@/platform/task";

const DATA_TYPES: Record<payload.JSONType, string> = {
  number: DataType.FLOAT64.toString(),
  boolean: DataType.UINT8.toString(),
  string: DataType.STRING.toString(),
};

/**
 * Creates one read field for each scalar of a browsed payload. String fields come back
 * disabled when the payload also holds numbers, because the fields of a topic share one
 * index and a variable-length channel has none.
 */
export const createReadFields = (scalars: BrowsedField[]): ReadField[] => {
  const fields = scalars.map(({ pointer, type }) =>
    createReadField(Task.Payload.readOptionsOf(DATA_TYPES[type]), { pointer }),
  );
  const isVariable = (f: ReadField) =>
    new DataType(Task.Payload.dataTypeOf(f)).isVariable;
  if (fields.every(isVariable)) return fields;
  return fields.map((f) => ({ ...f, disabled: isVariable(f) }));
};

/**
 * @returns the JSON type of the sample that a field came from.
 * @throws {Error} if no JSON sample gives a field of its data type.
 */
export const jsonTypeOf = (field: ReadField): payload.JSONType => {
  const dataType = Task.Payload.dataTypeOf(field);
  const type = payload.JSON_TYPES.find((t) => DATA_TYPES[t] === dataType);
  if (type == null) throw new Error(`No JSON sample gives a ${dataType} field`);
  return type;
};
