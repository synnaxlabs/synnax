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
import { type ReactElement } from "react";

export interface OptionalFieldProps<V> {
  path: string;
  label: string;
  /** Shown while the field is absent. */
  placeholder: string;
  format: (value: V) => string;
  /** @returns the value the text holds, or null when it holds none. */
  parse: (text: string) => V | null;
}

/**
 * A text input for a value the schema lets be absent. Clearing the text removes the
 * value, and text that does not parse leaves the value as it was.
 */
export const OptionalField = <V,>({
  path,
  label,
  placeholder,
  format,
  parse,
}: OptionalFieldProps<V>): ReactElement => {
  const { set } = Form.useContext();
  const state = Form.useFieldState<V>(path, { optional: true });
  const value = state?.value;
  const handleChange = (text: string) => {
    if (text.trim() === "") return set(path, undefined);
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
      />
    </Input.Item>
  );
};

/** Parses an integer written in decimal or, with a 0x prefix, in hexadecimal. */
export const parseInteger = (text: string): number | null => {
  const value = Number(text.trim());
  return Number.isInteger(value) ? value : null;
};
