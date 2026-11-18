// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Schematic } from "@synnaxlabs/pluto";

import { setVariant } from "@/feature/schematic/symbol/edit/spec";

const path = Schematic.Symbol.formPath;

export interface SelectVariantProps extends Input.Control<string> {}

const SelectVariant = ({ value, onChange }: SelectVariantProps) => (
  <Select.Buttons<string> onChange={onChange} value={value}>
    <Select.Item itemKey="static">
      <Icon.Auto />
      Static
    </Select.Item>
    <Select.Item itemKey="actuator">
      <Icon.Channel />
      Actuator
    </Select.Item>
  </Select.Buttons>
);

export interface SelectVariantFieldProps {
  /** onSelectValue selects a value of the state, such as Base when Active goes. */
  onSelectValue: (value: number) => void;
}

export const SelectVariantField = ({ onSelectValue }: SelectVariantFieldProps) => (
  <Form.Field
    path={path.data.variant}
    showLabel={false}
    showHelpText={false}
    onChange={(next, { get, set }) => {
      if (get(path.data.variant).value === next) return;
      if (next === "static") onSelectValue(0);
      set(path.data, setVariant(get(path.data).value, next));
    }}
  >
    {({ onChange, value }) => <SelectVariant value={value} onChange={onChange} />}
  </Form.Field>
);
