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
import { DataType } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { type Field } from "@/feature/ethercat/device/settings";

interface BaseProps {
  field: Field;
  /** Undefined shows the field as not set. */
  value: number | undefined;
  /** Whether the inputs that the field sets hold different values. */
  mixed?: boolean;
  problem?: string;
  /** Shows the value read only. */
  disabled?: boolean;
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

/** @returns the shortest text that reads back as the float32 value. */
const float32Text = (value: number): string => {
  let digits = 1;
  while (Math.fround(Number(value.toPrecision(digits))) !== value) digits++;
  return Number(value.toPrecision(digits)).toString();
};

/** @returns the value that the slave held at the scan, or undefined for no option. */
const heldText = ({ value, options, dataType }: Field): string | undefined => {
  if (value == null) return undefined;
  if (options == null)
    return DataType.FLOAT32.equals(dataType) ? float32Text(value) : value.toString();
  return options.find((o) => o.value === value)?.name;
};

export type SettingInputProps = RequiredProps | OptionalProps;

/**
 * A select for a field with options, or a number input with its unit. A field not set
 * shows the value that the slave held at the scan as its placeholder. A mixed field
 * shows "Mixed" until the user sets it.
 */
export const SettingInput = (props: SettingInputProps): ReactElement => {
  const { field, mixed = false, problem, disabled, onChange } = props;
  const value = mixed ? undefined : props.value;
  const status = problem == null ? undefined : "error";
  const held = mixed ? "Mixed" : heldText(field);
  const clear = props.allowNone === true ? () => props.onChange(null) : undefined;
  return (
    <Input.Item
      label={field.name}
      helpText={problem}
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
          disabled={disabled}
        />
      ) : (
        <Select.Simple<number>
          value={value}
          onChange={(next: number | null) =>
            next == null ? clear?.() : onChange(next)
          }
          resourceName={inSentence(field.name)}
          disabled={disabled}
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
