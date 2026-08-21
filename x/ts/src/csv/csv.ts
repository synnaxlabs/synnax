// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * formatValue correctly formats the given value into a string that is safe to use as a
 * field in a CSV file.
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

const QUOTE_REGEX = /["\t,\n]/;

const maybeEscapeField = (field: string): string => {
  if (!QUOTE_REGEX.test(field)) return field;
  const escaped = field.replace(/"/g, '""');
  return `"${escaped}"`;
};

const TAB = "\t";

const delimiterOf = (data: string): string => {
  let quoted = false;
  for (const char of data)
    if (char === '"') quoted = !quoted;
    else if (char === TAB && !quoted) return TAB;
  return ",";
};

/**
 * Parses a headerless delimited block, the form a spreadsheet puts on the clipboard,
 * into rows of fields. The block splits on tabs when it holds one outside a quoted
 * field, otherwise on commas, so a spreadsheet's tab-delimited fields survive the
 * commas inside them. A quoted field keeps its delimiters and its newlines, and
 * doubled quotes unescape.
 * @returns One entry per row, each holding the row's fields. Ragged rows stay ragged;
 * rows holding nothing but empty fields are dropped.
 */
export const parseBlock = (data: string): string[][] => {
  const delimiter = delimiterOf(data);
  const rows: string[][] = [];
  let fields: string[] = [];
  let field = "";
  let quoted = false;
  const endField = () => {
    fields.push(field.trim());
    field = "";
  };
  const endRow = () => {
    endField();
    if (fields.some((f) => f.length > 0)) rows.push(fields);
    fields = [];
  };
  for (let i = 0; i < data.length; i++) {
    const char = data[i];
    if (quoted && char === '"' && data[i + 1] === '"') {
      field += '"';
      i++;
    } else if (char === '"') quoted = !quoted;
    else if (quoted) field += char;
    else if (char === delimiter) endField();
    else if (char === "\n") endRow();
    else if (char !== "\r") field += char;
  }
  endRow();
  return rows;
};

/**
 * Formats rows of values into a tab delimited block, the form a spreadsheet reads from
 * the clipboard. Fields holding a delimiter, a quote, or a newline are quoted.
 */
export const formatBlock = (rows: unknown[][]): string =>
  rows.map((row) => row.map(formatValue).join(TAB)).join("\n");
