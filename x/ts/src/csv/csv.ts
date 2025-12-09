// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export type RecordDelimiter = "\r\n" | "\n";

/**
 * formatValue correctly formats the given value into a string that is safe to use as a
 * field in a CSV file.
 *
 * @param value - The value to format.
 * @returns The string to use as a field in a CSV file.
 */
export const formatValue = (value: unknown): string => {
  switch (typeof value) {
    case "bigint":
    case "number":
      return value.toString();
    case "boolean":
      return value ? "1" : "0";
    case "undefined":
      return "";
    case "string":
    case "symbol":
    case "function":
      return maybeEscapeField(value.toString());
    case "object":
      if (value == null) return "";
      return maybeEscapeField(JSON.stringify(value));
  }
};

const QUOTE_REGEX = /[",\n]/;

const maybeEscapeField = (field: string): string => {
  if (!QUOTE_REGEX.test(field)) return field;
  const escaped = field.replace(/"/g, '""');
  return `"${escaped}"`;
};
