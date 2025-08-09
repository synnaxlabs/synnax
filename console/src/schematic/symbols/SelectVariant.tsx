import { type schematic } from "@synnaxlabs/client";
import { Form, Icon, type Input, Select } from "@synnaxlabs/pluto";

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

export const SelectVariantField = () => (
  <Form.Field<string>
    path="data.variant"
    showLabel={false}
    onChange={(next, { get, set }) => {
      const prev = get("data.variant").value;
      if (prev === next) return;
      const prevStates = get<schematic.symbol.State[]>("data.states").value;
      if (next === "actuator")
        set("data.states", [
          ...prevStates,
          { key: "active", name: "Active", regions: [], color: "#000000" },
        ]);
      else if (next === "static")
        set("data.states", [...prevStates.filter((s) => s.key !== "active")]);
    }}
  >
    {({ onChange, value }) => <SelectVariant value={value} onChange={onChange} />}
  </Form.Field>
);
