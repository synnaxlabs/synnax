// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import type { deep } from "@synnaxlabs/x";
import type { ReactElement } from "react";

export interface FormattedFieldProps<V> {
  path: deep.Target<V | undefined>;
  label: string;
  /** Shown while the text is empty. */
  placeholder?: string;
  /** Shown after the text, such as "ms". */
  unit?: string;
  /** Whether clearing the text removes the value. */
  optional?: boolean;
  format: (value: V) => string;
  /** @returns the value the text holds, or null when it holds none. */
  parse: (text: string) => V | null;
}

/**
 * A text input for a value the user writes in a format, such as a hexadecimal ID. The
 * text commits on blur or Enter, and text that does not parse leaves the value as it
 * was.
 */
export const FormattedField = <V,>({
  path,
  label,
  placeholder,
  unit,
  optional = false,
  format,
  parse,
}: FormattedFieldProps<V>): ReactElement => {
  const { set, mode } = Form.useContext();
  const state = Form.useFieldState(path, { optional });
  const value = state?.value;
  const handleChange = (text: string) => {
    if (optional && text.trim() === "") return set(path, undefined);
    const parsed = parse(text);
    if (parsed != null) set(path, parsed);
  };
  return (
    <Input.Item
      label={label}
      padHelpText
      helpText={state?.status.message}
      status={state?.status.variant}
    >
      <Input.Text
        value={value == null ? "" : format(value)}
        onChange={handleChange}
        placeholder={placeholder}
        endContent={unit}
        onlyChangeOnBlur
        preview={mode === "preview"}
      />
    </Input.Item>
  );
};

/** Parses an integer written in decimal or, with a 0x prefix, in hexadecimal. */
export const parseInteger = (text: string): number | null => {
  const trimmed = text.trim();
  if (trimmed === "") return null;
  const value = Number(trimmed);
  return Number.isInteger(value) ? value : null;
};

/** Formats an integer in hexadecimal with a 0x prefix. */
export const formatHex = (value: number): string =>
  `0x${value.toString(16).toUpperCase()}`;

/** Formats bytes as pairs of uppercase hexadecimal digits split by spaces. */
export const formatHexBytes = (bytes: Uint8Array): string =>
  Array.from(bytes, (b) => b.toString(16).toUpperCase().padStart(2, "0")).join(" ");

/**
 * Parses bytes written as pairs of hexadecimal digits. Spaces and 0x prefixes are
 * optional.
 * @returns null when the text holds no bytes or a digit is missing.
 */
export const parseHexBytes = (text: string): Uint8Array | null => {
  const digits = text.replace(/0x/gi, "").replace(/\s/g, "");
  if (digits.length === 0 || digits.length % 2 !== 0 || /[^0-9a-f]/i.test(digits))
    return null;
  return Uint8Array.from(digits.match(/../g) ?? [], (pair) => parseInt(pair, 16));
};

const ESCAPES: Record<string, string> = { "\\": "\\", n: "\n", r: "\r", t: "\t" };

const ESCAPED = Object.fromEntries(
  Object.entries(ESCAPES).map(([code, char]) => [char, `\\${code}`]),
);

/**
 * Formats text so that each control character shows: a backslash becomes \\, a new
 * line \n, a carriage return \r, a tab \t, and any other control character \xNN.
 */
export const formatEscaped = (text: string): string =>
  Array.from(text, (char) => {
    if (char in ESCAPED) return ESCAPED[char];
    const code = char.charCodeAt(0);
    if (code < 0x20 || code === 0x7f)
      return `\\x${code.toString(16).toUpperCase().padStart(2, "0")}`;
    return char;
  }).join("");

/**
 * Parses text written with the escapes of {@link formatEscaped}. \xNN takes an ASCII
 * code, from 00 to 7F.
 * @returns null when the text is empty or holds an escape it does not know.
 */
export const parseEscaped = (text: string): string | null => {
  let out = "";
  for (let i = 0; i < text.length; i++) {
    if (text[i] !== "\\") {
      out += text[i];
      continue;
    }
    const code = text[i + 1];
    if (code in ESCAPES) {
      out += ESCAPES[code];
      i++;
      continue;
    }
    const hex = text.slice(i + 2, i + 4);
    if (code !== "x" || !/^[0-7][0-9a-f]$/i.test(hex)) return null;
    out += String.fromCharCode(parseInt(hex, 16));
    i += 3;
  }
  return out.length === 0 ? null : out;
};
