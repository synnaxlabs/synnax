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

import { type Field } from "@/feature/ethercat/device/settings";

export interface SettingInputProps {
  field: Field;
  /** Undefined shows the field as not set. */
  value: number | undefined;
  /** Called with null when the user clears a field that allows none. */
  onChange: (value: number | null) => void;
  problem?: string;
  /** Whether the user can leave the field not set. */
  allowNone?: boolean;
}

/** A select for a field with options, or a number input with its unit. */
export const SettingInput = ({
  field,
  value,
  onChange,
  problem,
  allowNone = false,
}: SettingInputProps): ReactElement => {
  const status = problem == null ? undefined : "error";
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
          onClear={allowNone ? () => onChange(null) : undefined}
          endContent={field.unit === "" ? undefined : field.unit}
          status={status}
          aria-label={field.name}
        />
      ) : (
        <Select.Simple<number>
          value={value}
          onChange={(next: number | null) => {
            if (next != null || allowNone) onChange(next);
          }}
          resourceName={field.name.toLowerCase()}
          searchHidden
          allowNone={allowNone}
          triggerProps={{ status }}
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
