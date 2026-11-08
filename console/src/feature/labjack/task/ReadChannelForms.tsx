// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { deep, type record } from "@synnaxlabs/x";
import { type FC } from "react";

import {
  AI_PORT_TYPE,
  GROUND_CHANNEL,
  type Model,
  negativeChannelOf,
  PORTS,
  T4_MODEL,
  T7_MODEL,
  T8_MODEL,
} from "@/feature/labjack/device/types";
import {
  AIR_CJC_SOURCE,
  DEVICE_CJC_SOURCE,
  type ReadChannel,
  type ReadChannelType,
  type TemperatureUnits,
  type ThermocoupleType,
} from "@/feature/labjack/task/types";

const MaxVoltageField = PForm.buildNumericField({
  fieldProps: { label: "Max voltage" },
  inputProps: { endContent: "V" },
});

const ThermocoupleTypeField = PForm.buildSelectField<ThermocoupleType>({
  fieldProps: { label: "Thermocouple type" },
  inputProps: {
    resourceName: "thermocouple type",
    searchName: "type",
    children: (
      <>
        <Select.Item itemKey="B">B</Select.Item>
        <Select.Item itemKey="E">E</Select.Item>
        <Select.Item itemKey="J">J</Select.Item>
        <Select.Item itemKey="K">K</Select.Item>
        <Select.Item itemKey="N">N</Select.Item>
        <Select.Item itemKey="R">R</Select.Item>
        <Select.Item itemKey="S">S</Select.Item>
        <Select.Item itemKey="T">T</Select.Item>
        <Select.Item itemKey="C">C</Select.Item>
      </>
    ),
  },
});

const TemperatureUnitsField = PForm.buildSelectField<TemperatureUnits>({
  fieldProps: { label: "Temperature units" },
  inputProps: {
    resourceName: "temperature units",
    searchName: "unit",
    children: (
      <>
        <Select.Item itemKey="C">Celsius</Select.Item>
        <Select.Item itemKey="F">Fahrenheit</Select.Item>
        <Select.Item itemKey="K">Kelvin</Select.Item>
      </>
    ),
  },
});

interface SelectCJCSourceFieldProps extends Omit<
  Select.SingleSimpleProps<string>,
  "children" | "resourceName"
> {
  model: Model;
}

const SelectCJCSourceField = ({ model, ...rest }: SelectCJCSourceFieldProps) => {
  const ports: record.KeyedNamed[] = PORTS[model][AI_PORT_TYPE];
  return (
    <Select.Simple<string> allowNone={false} {...rest} resourceName="CJC source">
      {/* On a T8, the device source reads the sensor at the channel's own terminal. */}
      <Select.Item itemKey={DEVICE_CJC_SOURCE}>
        {model === T8_MODEL ? "Terminal" : "Device"}
      </Select.Item>
      {model !== T8_MODEL && <Select.Item itemKey={AIR_CJC_SOURCE}>Air</Select.Item>}
      {ports.map(({ key, name }) => (
        <Select.Item key={key} itemKey={key}>
          {name}
        </Select.Item>
      ))}
    </Select.Simple>
  );
};

interface FormProps {
  path: deep.Path<ReadChannel>;
  deviceModel: Model;
}

interface NegativeChannelFieldProps {
  port: deep.Path<string>;
  negChan: deep.Path<number>;
}

/** The negative input of a T7 channel: ground, or the port's differential pair. */
const NegativeChannelField = ({ port, negChan }: NegativeChannelFieldProps) => {
  const pair = negativeChannelOf(PForm.useFieldValue(port));
  return (
    <PForm.Field path={negChan} label="Negative channel">
      {({ value, onChange, preview }) => (
        <Select.Simple<number>
          allowNone={false}
          value={value}
          onChange={onChange}
          preview={preview}
          resourceName="negative channel"
        >
          <Select.Item itemKey={GROUND_CHANNEL}>Ground</Select.Item>
          {pair != null && <Select.Item itemKey={pair}>{`AIN${pair}`}</Select.Item>}
        </Select.Simple>
      )}
    </PForm.Field>
  );
};

const AnalogForm = ({ path, deviceModel }: FormProps) => {
  const channel = deep.variant(path, "type", "analog");
  return (
    <>
      <MaxVoltageField path={channel.range} />
      {deviceModel === T7_MODEL && (
        <NegativeChannelField port={channel.port} negChan={channel.negChan} />
      )}
    </>
  );
};

const ThermocoupleForm = ({ path, deviceModel }: FormProps) => {
  const channel = deep.variant(path, "type", "thermocouple");
  return (
    <>
      <ThermocoupleTypeField path={channel.thermocoupleType} />
      <TemperatureUnitsField path={channel.units} />
      {deviceModel === T7_MODEL && (
        <NegativeChannelField port={channel.port} negChan={channel.negChan} />
      )}
      <PForm.Field path={channel.cjcSource} label="CJC source">
        {({ value, onChange, preview }) => (
          <SelectCJCSourceField
            value={value}
            onChange={onChange}
            preview={preview}
            model={deviceModel}
          />
        )}
      </PForm.Field>
      <PForm.NumericField path={channel.cjcSlope} label="CJC slope" />
      <PForm.NumericField path={channel.cjcOffset} label="CJC offset" />
    </>
  );
};

const SIGNAL_FORMS: Record<ReadChannelType, FC<FormProps> | null> = {
  analog: AnalogForm,
  digital: null,
  thermocouple: ThermocoupleForm,
};

/**
 * The signal fields for each model and channel type, or null for a type that has none.
 * The T4 has fixed input ranges and no thermocouple inputs.
 */
export const FORMS: Record<Model, Record<ReadChannelType, FC<FormProps> | null>> = {
  [T4_MODEL]: { analog: null, digital: null, thermocouple: null },
  [T7_MODEL]: SIGNAL_FORMS,
  [T8_MODEL]: SIGNAL_FORMS,
};
