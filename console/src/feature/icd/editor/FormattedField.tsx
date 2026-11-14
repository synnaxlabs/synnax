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
import { type deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

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
