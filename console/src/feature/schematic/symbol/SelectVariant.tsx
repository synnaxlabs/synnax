// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { Form, Icon, type Input, Select } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";

export interface SelectVariantProps extends Input.Control<string> {}

const VARIANT_DATA: Select.StaticEntry<string>[] = [
  { key: "static", name: "Static", icon: <Icon.Auto /> },
  { key: "actuator", name: "Actuator", icon: <Icon.Channel /> },
];

const SelectVariant = ({ value, onChange }: SelectVariantProps) => (
  <Select.Static
    data={VARIANT_DATA}
    onChange={onChange}
    value={value}
    resourceName="variant"
  />
);

export interface SelectVariantFieldProps {
  onSelectState: (state: string) => void;
}

export const SelectVariantField = ({ onSelectState }: SelectVariantFieldProps) => (
  <Form.Field<string>
    path="data.variant"
    showLabel={false}
    onChange={(next, { get, set }) => {
      const prev = get("data.variant").value;
      if (prev === next) return;
      const prevStates = get<schematic.symbol.State[]>("data.states").value;
      if (next === "actuator") {
        const baseState = prevStates.find((s) => s.key === "base");
        const baseRegions = baseState?.regions ?? [];
        set("data.states", [
          ...prevStates,
          {
            key: "active",
            name: "Active",
            regions: [...deep.copy(baseRegions)],
          },
        ]);
      } else if (next === "static") {
        onSelectState("base");
        set("data.states", [...prevStates.filter((s) => s.key !== "active")]);
      }
    }}
  >
    {({ onChange, value }) => <SelectVariant value={value} onChange={onChange} />}
  </Form.Field>
);
