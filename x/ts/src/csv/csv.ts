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

/**
 * Parses a headerless delimited block, the form a spreadsheet puts on the clipboard,
 * into rows of fields. A line holding a tab splits on tabs, otherwise on commas, so a
 * spreadsheet's tab-delimited fields survive the commas inside them. Quoted fields keep
 * their delimiters and unescape doubled quotes.
 * @returns One entry per line, each holding the line's fields. Ragged lines stay
 * ragged; empty lines are dropped.
 */
export const parseBlock = (data: string): string[][] =>
  data
    .split(/\r?\n/)
    .filter((line) => line.trim().length > 0)
    .map((line) => parseLine(line, line.includes(TAB) ? TAB : ","));

const parseLine = (line: string, delimiter: string): string[] => {
  const fields: string[] = [];
  let field = "";
  let quoted = false;
  for (let i = 0; i < line.length; i++) {
    const char = line[i];
    if (quoted && char === '"' && line[i + 1] === '"') {
      field += '"';
      i++;
    } else if (char === '"') quoted = !quoted;
    else if (char === delimiter && !quoted) {
      fields.push(field.trim());
      field = "";
    } else field += char;
  }
  fields.push(field.trim());
  return fields;
};

/**
 * Formats rows of values into a tab delimited block, the form a spreadsheet reads from
 * the clipboard. Fields holding a delimiter, a quote, or a newline are quoted.
 */
export const formatBlock = (rows: unknown[][]): string =>
  rows.map((row) => row.map(formatValue).join(TAB)).join("\n");
