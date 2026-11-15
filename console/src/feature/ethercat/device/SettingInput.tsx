// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";

import { type Field, valueText } from "@/feature/ethercat/device/settings";

interface BaseProps {
  field: Field;
  /** Undefined shows the field as not set. */
  value: number | undefined;
  problem?: string;
  /** Says why the value changed. A problem hides it. */
  note?: string;
}

interface RequiredProps extends BaseProps {
  allowNone?: false;
  onChange: (value: number) => void;
}

/** A field that the user can leave not set. onChange gets null when they clear it. */
interface OptionalProps extends BaseProps {
  allowNone: true;
  onChange: (value: number | null) => void;
}

/** @returns the name with a lowercase first letter, unless it starts an acronym. */
const inSentence = (name: string): string =>
  /^[A-Z][a-z]/.test(name) ? name[0].toLowerCase() + name.slice(1) : name;

/** @returns the value that the slave held at the scan, or undefined for no option. */
const heldText = (field: Field): string | undefined => {
  const { value, options } = field;
  if (value == null) return undefined;
  if (options == null) return valueText(field, value);
  return options.find((o) => o.value === value)?.name;
};

export type SettingInputProps = RequiredProps | OptionalProps;

/**
 * A select for a field with options, or a number input with its unit. A field not set
 * shows the value that the slave held at the scan as its placeholder.
 */
export const SettingInput = (props: SettingInputProps): ReactElement => {
  const { field, value, problem, note, onChange } = props;
  const status = problem == null ? undefined : "error";
  const held = heldText(field);
  const clear = props.allowNone === true ? () => props.onChange(null) : undefined;
  return (
    <Input.Item
      label={field.name}
      helpText={problem ?? note}
      status={status}
      padHelpText={false}
    >
      {field.options == null ? (
        <Input.Numeric
          value={value}
          onChange={onChange}
          onClear={clear}
          endContent={field.unit === "" ? undefined : field.unit}
          status={status}
          placeholder={held}
          aria-label={field.name}
        />
      ) : (
        <Select.Simple<number>
          value={value}
          onChange={(next: number | null) =>
            next == null ? clear?.() : onChange(next)
          }
          resourceName={inSentence(field.name)}
          searchHidden
          allowNone={clear != null}
          triggerProps={held == null ? { status } : { status, placeholder: held }}
        >
          {field.options.map(({ value, name }) => (
            <Select.Item<number>
              key={value}
              itemKey={value}
              label={name}
              textValue={name}
            >
              {name}
            </Select.Item>
          ))}
        </Select.Simple>
      )}
    </Input.Item>
  );
};
