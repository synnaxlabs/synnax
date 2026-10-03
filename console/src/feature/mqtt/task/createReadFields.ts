// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { mqtt } from "@synnaxlabs/client";
import { DataType } from "@synnaxlabs/x";

import { type ReadField } from "@/feature/mqtt/task/types";

const escapeToken = (token: string): string =>
  token.replaceAll("~", "~0").replaceAll("/", "~1");

const DATA_TYPES: Partial<Record<string, string>> = {
  number: DataType.FLOAT64.toString(),
  boolean: DataType.UINT8.toString(),
  string: DataType.STRING.toString(),
};

const flatten = (value: unknown, pointer: string): ReadField[] => {
  const dataType = DATA_TYPES[typeof value];
  if (dataType != null) return [{ ...mqtt.readFieldZ.parse({}), pointer, dataType }];
  if (value == null || typeof value !== "object" || Array.isArray(value)) return [];
  return Object.entries(value).flatMap(([token, child]) =>
    flatten(child, `${pointer}/${escapeToken(token)}`),
  );
};

/**
 * Creates one read field for each scalar in a JSON payload sample. Arrays, nulls, and a
 * sample that is not JSON give no field. String fields come back disabled when the
 * payload also holds numbers, because the fields of a topic share one index and a
 * variable-length channel has none.
 */
export const createReadFields = (payload: string): ReadField[] => {
  let value: unknown;
  try {
    value = JSON.parse(payload);
  } catch {
    return [];
  }
  const fields = flatten(value, "");
  const isVariable = (f: ReadField) => new DataType(f.dataType).isVariable;
  if (fields.every(isVariable)) return fields;
  return fields.map((f) => ({ ...f, disabled: isVariable(f) }));
};
