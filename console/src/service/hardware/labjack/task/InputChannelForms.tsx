// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Divider, Flex, Form as PForm, Icon, Select } from "@synnaxlabs/pluto";
import { deep, type optional, type record } from "@synnaxlabs/x";
import { type FC, useMemo } from "react";

import { Device } from "@/hardware/labjack/device";
import {
  AIR_CJC_SOURCE,
  DEVICE_CJC_SOURCE,
  type InputChannelType,
  type Scale,
  SCALE_SCHEMAS,
  type ScaleType,
  type TemperatureUnits,
  type ThermocoupleType,
  ZERO_SCALES,
} from "@/hardware/labjack/task/types";

const MaxVoltageField = PForm.buildNumericField({
  fieldKey: "range",
  fieldProps: { label: "Max Voltage" },
  inputProps: { endContent: "V" },
});

const SelectScaleTypeField = PForm.buildSelectField<
  ScaleType,
  Select.StaticEntry<ScaleType>
>({
  fieldKey: "type",
  fieldProps: {
    label: "Scale",
    onChange: (value, { get, set, path }) => {
      const prevType = get<ScaleType>(path).value;
      if (prevType === value) return;
      const next = deep.copy(ZERO_SCALES[value]);
      const parentPath = path.slice(0, path.lastIndexOf("."));
      const prevParent = get<Scale>(parentPath).value;
      set(parentPath, {
        ...deep.overrideValidItems(next, prevParent, SCALE_SCHEMAS[value]),
        type: next.type,
      });
    },
  },
  inputProps: {
    resourceName: "scale type",
    data: [
      { key: "none", name: "None", icon: <Icon.None /> },
      { key: "linear", name: "Linear", icon: <Icon.Linear /> },
    ],
  },
});

const SCALE_FORMS: Record<ScaleType, FC<CustomScaleFormProps>> = {
  linear: ({ prefix }) => (
    <Flex.Box x>
      <PForm.NumericField path={`${prefix}.slope`} label="Slope" grow />
      <PForm.NumericField path={`${prefix}.offset`} label="Offset" grow />
    </Flex.Box>
  ),
  none: () => null,
};

interface CustomScaleFormProps {
  prefix: string;
}

const CustomScaleForm = ({ prefix }: CustomScaleFormProps) => {
  const path = `${prefix}.scale`;
  const scaleType = PForm.useFieldValue<ScaleType>(`${path}.type`);
  const Form = SCALE_FORMS[scaleType];
  return (
    <>
      <SelectScaleTypeField path={path} />
      <Form prefix={path} />
    </>
  );
};

const ThermocoupleTypeField = PForm.buildSelectField<
  ThermocoupleType,
  record.KeyedNamed<ThermocoupleType>
>({
  fieldKey: "thermocoupleType",
  fieldProps: { label: "Thermocouple Type" },
  inputProps: {
    resourceName: "thermocouple type",
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

const TemperatureUnitsField = PForm.buildSelectField<
  TemperatureUnits,
  record.KeyedNamed<TemperatureUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Temperature Units" },
  inputProps: {
    resourceName: "temperature units",
    data: [
      { key: "C", name: "Celsius" },
      { key: "F", name: "Fahrenheit" },
      { key: "K", name: "Kelvin" },
    ],
  },
});

interface CJCSourceEntry extends record.KeyedNamed<string> {}

interface SelectCJCSourceFieldProps extends optional.Optional<
  Select.StaticProps<string, CJCSourceEntry>,
  "data" | "resourceName"
> {
  model: Device.Model;
}

const DEFAULT_CJC_SOURCE_ENTRIES: CJCSourceEntry[] = [
  { key: DEVICE_CJC_SOURCE, name: "Device" },
  { key: AIR_CJC_SOURCE, name: "Air" },
];

const SelectCJCSourceField = ({ model, ...rest }: SelectCJCSourceFieldProps) => {
  const data = useMemo(() => {
    const ports: CJCSourceEntry[] = Device.PORTS[model][Device.AI_PORT_TYPE];
    return [...DEFAULT_CJC_SOURCE_ENTRIES, ...ports];
  }, [model]);
  return (
    <Select.Static<string, CJCSourceEntry>
      data={data}
      allowNone={false}
      {...rest}
      resourceName="CJC source"
    />
  );
};

interface FormProps {
  path: string;
  deviceModel: Device.Model;
}

export const FORMS: Record<InputChannelType, FC<FormProps>> = {
  AI: ({ path }) => (
    <>
      <Divider.Divider x padded="bottom" />
      <MaxVoltageField path={path} />
      <CustomScaleForm prefix={path} />
    </>
  ),
  DI: () => null,
  TC: ({ path, deviceModel }) => (
    <>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <ThermocoupleTypeField path={path} grow />
        <TemperatureUnitsField path={path} grow />
      </Flex.Box>
      <Flex.Box x>
        <PForm.NumericField
          fieldKey="posChan"
          path={path}
          label="Positive Channel"
          grow
        />
        <PForm.NumericField
          fieldKey="negChan"
          path={path}
          label="Negative Channel"
          grow
        />
      </Flex.Box>
      <Flex.Box x>
        <PForm.Field<string>
          path={`${path}.cjcSource`}
          grow
          hideIfNull
          label="CJC Source"
        >
          {({ value, onChange }) => (
            <SelectCJCSourceField
              value={value}
              onChange={onChange}
              model={deviceModel}
            />
          )}
        </PForm.Field>
        <PForm.NumericField fieldKey="cjcSlope" path={path} label="CJC Slope" grow />
        <PForm.NumericField fieldKey="cjcOffset" path={path} label="CJC Offset" grow />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={path} />
    </>
  ),
};
