// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const regex = /[",\n]/;

/**
 * Escapes a CSV field by wrapping it in double quotes if it contains a comma, double
 * quote, or newline. Also escapes any internal double quotes by doubling them. For
 * example, the field foo"bar,baz becomes "foo""bar,baz"
 *
 * @param field -  The string field to potentially escape for CSV output.
 * @returns The escaped CSV-safe field.
 */
export const maybeEscapeField = (field: string): string => {
  if (!regex.test(field)) return field;
  const escaped = field.replace(/"/g, '""');
  return `"${escaped}"`;
};

export type RecordDelimiter = "\r\n" | "\n";
