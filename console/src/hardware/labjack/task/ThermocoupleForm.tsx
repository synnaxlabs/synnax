// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align, Form, Select } from "@synnaxlabs/pluto";
import { type KeyedNamed } from "@synnaxlabs/x";

import { Device } from "@/hardware/labjack/device";
import { type ChannelType, type TemperatureUnits } from "@/hardware/labjack/task/types";

const ThermocoupleTypeField = Form.buildDropdownButtonSelectField({
  fieldKey: "thermocoupleType",
  fieldProps: { label: "Thermocouple Type" },
  inputProps: {
    entryRenderKey: "name",
    columns: [{ key: "name", name: "Name" }],
    hideColumnHeader: true,
    data: [
      { key: "B", name: "B" },
      { key: "E", name: "E" },
      { key: "J", name: "J" },
      { key: "K", name: "K" },
      { key: "N", name: "N" },
      { key: "R", name: "R" },
      { key: "S", name: "S" },
      { key: "T", name: "T" },
      { key: "C", name: "C" },
    ],
  },
});

const TemperatureUnitsField = Form.buildDropdownButtonSelectField<
  TemperatureUnits,
  KeyedNamed<TemperatureUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Temperature Units" },
  inputProps: {
    entryRenderKey: "name",
    columns: [{ key: "name", name: "Name" }],
    data: [
      { key: "C", name: "Celsius" },
      { key: "F", name: "Fahrenheit" },
      { key: "K", name: "Kelvin" },
    ],
  },
});

interface CJCSourceType {
  key: string;
}

interface SelectCJCSourceProps extends Select.SingleProps<string, CJCSourceType> {
  model: Device.Model;
}

const SelectCJCSourceField = ({ model, ...rest }: SelectCJCSourceProps) => {
  const ports: CJCSourceType[] = Device.DEVICES[model].ports.AI;
  const data = [
    { key: "TEMPERATURE_DEVICE_K" },
    { key: "TEMPERATURE_AIR_K" },
    ...ports,
  ];
  return (
    <Select.Single<string, CJCSourceType>
      data={data}
      columns={[{ key: "key", name: "CJC Source" }]}
      allowNone={false}
      entryRenderKey="key"
      {...rest}
    />
  );
};

export interface ThermocoupleFormProps {
  prefix: string;
  model: Device.Model;
}

export const ThermocoupleForm = ({ prefix, model }: ThermocoupleFormProps) => {
  const channelType = Form.useFieldValue<ChannelType>(`${prefix}.type`, true);
  if (channelType !== "TC") return null;
  return (
    <Align.Space direction="y" grow>
      <Align.Space direction="x">
        <ThermocoupleTypeField path={prefix} grow />
        <TemperatureUnitsField path={prefix} grow />
      </Align.Space>
      <Align.Space direction="x">
        <Form.NumericField path={`${prefix}.posChan`} label="Positive Channel" grow />
        <Form.NumericField path={`${prefix}.negChan`} label="Negative Channel" grow />
      </Align.Space>
      <Align.Space direction="x">
        <Form.Field<string>
          path={`${prefix}.cjcSource`}
          grow
          hideIfNull
          label="CJC Source"
        >
          {(p) => <SelectCJCSourceField {...p} model={model} />}
        </Form.Field>
        <Form.NumericField path={`${prefix}.cjcSlope`} label="CJC Slope" grow />
        <Form.NumericField path={`${prefix}.cjcOffset`} label="CJC Offset" grow />
      </Align.Space>
    </Align.Space>
  );
};
