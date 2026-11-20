// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/View.css";

import { Color } from "@synnaxlabs/lyra/color";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import type { theme } from "@synnaxlabs/lyra/theme";
import { color as xcolor } from "@synnaxlabs/x";
import { type KeyboardEvent, type ReactElement, type ReactNode, useId } from "react";

import { formPath } from "@/color/queries";
import { CSS } from "@/css";

interface RowProps {
  label: string;
  labelID?: string;
  children: ReactNode;
}

const labelClass = CSS.BE("color-view", "label");

const Row = ({ label, labelID, children }: RowProps): ReactElement => (
  <Flex.Box x gap="small" align="start">
    <Input.Label id={labelID} className={labelClass}>
      {label}
    </Input.Label>
    {children}
  </Flex.Box>
);

const NameField = ({ onEnter }: { onEnter?: () => void }): ReactElement => {
  const id = useId();
  return (
    <Row label="Name" labelID={id}>
      <Form.Field<string>
        path={formPath.name}
        showLabel={false}
        padHelpText={false}
        grow
      >
        {(p) => (
          <Input.Text
            {...p}
            aria-labelledby={id}
            size="small"
            autoFocus
            onKeyDown={(e: KeyboardEvent) => {
              if (e.key !== "Enter" || onEnter == null) return;
              e.preventDefault();
              onEnter();
            }}
          />
        )}
      </Form.Field>
    </Row>
  );
};

const ValuesField = (): ReactElement => {
  const light = Form.useField<xcolor.Crude, xcolor.Color>(formPath.light);
  const dark = Form.useField<xcolor.Crude | undefined, xcolor.Color | undefined>(
    formPath.dark,
  );
  return (
    <Color.PairEditor
      label={<Input.Label className={labelClass}>Color</Input.Label>}
      value={{ light: light.value, dark: dark.value }}
      onChange={(next: theme.Pair) => {
        if (next.light !== light.value) light.onChange(xcolor.construct(next.light));
        if (next.dark !== dark.value)
          dark.onChange(next.dark == null ? undefined : xcolor.construct(next.dark));
      }}
    />
  );
};

export interface FieldsProps {
  /** Called on Enter in the name. */
  onEnter?: () => void;
  /** Shown above the fields. */
  note?: string;
  /** The control of a Palette field, between the name and the color. */
  palette?: ReactNode;
  /** Shown at the trailing edge, under a hairline across the view. */
  footer?: ReactNode;
}

/**
 * The fields of the named color of the enclosing form, as a picker view: the name, an
 * optional palette, and the color, with their labels in one column.
 */
export const Fields = ({
  onEnter,
  note,
  palette,
  footer,
}: FieldsProps): ReactElement => (
  <Flex.Box y gap="medium" className={CSS.B("color-view")}>
    {note != null && (
      <Text.Text level="small" color={9}>
        {note}
      </Text.Text>
    )}
    <NameField onEnter={onEnter} />
    {palette != null && <Row label="Palette">{palette}</Row>}
    <ValuesField />
    {footer != null && (
      <Flex.Box x justify="end" className={CSS.BE("color-view", "footer")}>
        {footer}
      </Flex.Box>
    )}
  </Flex.Box>
);
