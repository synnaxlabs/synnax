// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { color } from "@synnaxlabs/client";
import { Color } from "@synnaxlabs/lyra/color";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Text } from "@synnaxlabs/lyra/text";
import { theme } from "@synnaxlabs/lyra/theme";
import { color as xcolor } from "@synnaxlabs/x";
import type { ReactElement, ReactNode } from "react";

import { formPath, type formSchema, useForm } from "@/color/queries";

export interface EditorProps {
  /** The key of the named color to edit. */
  itemKey: color.Key;
  children: ReactNode;
}

/**
 * Binds the inputs inside it to a named color. Each edit saves at once and reaches
 * every user of the color.
 */
export const Editor = ({ itemKey, children }: EditorProps): ReactElement => {
  const { form } = useForm({ query: { key: itemKey } });
  return <Form.Form<typeof formSchema> {...form}>{children}</Form.Form>;
};

export interface ValueInputProps {
  /** If true, the input shows the value and offers no picker. */
  disabled?: boolean;
}

interface HexProps {
  value?: xcolor.Crude;
}

const Hex = ({ value }: HexProps): ReactElement => (
  <Text.Text level="small" variant="code" color={value == null ? 8 : 10}>
    {value == null ? "Auto" : xcolor.hex(value).toUpperCase()}
  </Text.Text>
);

/** A swatch and the hex code of the light value of the enclosing form's color. */
export const LightInput = ({ disabled = false }: ValueInputProps): ReactElement => {
  const { value, onChange } = Form.useField<xcolor.Crude, xcolor.Color>(formPath.light);
  return (
    <Flex.Box x align="center" gap="small">
      <Color.Swatch
        value={value}
        onChange={disabled ? undefined : onChange}
        size="small"
        aria-label="Light theme color"
      />
      <Hex value={value} />
    </Flex.Box>
  );
};

const DARK = theme.themeZ.parse(theme.SYNNAX_DARK);

/**
 * A swatch and the hex code of the dark value of the enclosing form's color. While the
 * value is absent, it shows Auto and the color that the dark theme derives.
 */
export const DarkInput = ({ disabled = false }: ValueInputProps): ReactElement => {
  const light = Form.useFieldValue<xcolor.Crude>(formPath.light);
  const { value, onChange } = Form.useField<
    xcolor.Crude | undefined,
    xcolor.Color | undefined
  >(formPath.dark);
  return (
    <Flex.Box x align="center" gap="small">
      <Color.Swatch
        value={value}
        fallback={theme.legible(light, DARK)}
        onChange={disabled ? undefined : onChange}
        size="small"
        aria-label="Dark theme color"
      />
      <Hex value={value} />
    </Flex.Box>
  );
};
