// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Divider } from "@synnaxlabs/lyra/divider";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { deep, type record } from "@synnaxlabs/x";
import { type FC } from "react";

import * as Device from "@/feature/labjack/device/types";
import {
  AIR_CJC_SOURCE,
  DEVICE_CJC_SOURCE,
  type ReadChannelType,
  type Scale,
  SCALE_SCHEMAS,
  type ScaleType,
  type TemperatureUnits,
  type ThermocoupleType,
} from "@/feature/labjack/task/types";

const MaxVoltageField = PForm.buildNumericField({
  fieldKey: "range",
  fieldProps: { label: "Max voltage" },
  inputProps: { endContent: "V" },
});

const SelectScaleTypeField = PForm.buildSelectField<ScaleType>({
  fieldKey: "type",
  fieldProps: {
    label: "Scale",
    onChange: (value, { get, set, path }) => {
      const prevType = get<ScaleType>(path).value;
      if (prevType === value) return;
      const next = SCALE_SCHEMAS[value].parse({ type: value });
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
    children: (
      <>
        <Select.Item itemKey="none">
          <Icon.None />
          None
        </Select.Item>
        <Select.Item itemKey="linear">
          <Icon.Linear />
          Linear
        </Select.Item>
        <Select.Item itemKey="map">
          <Icon.Map />
          Map
        </Select.Item>
      </>
    ),
  },
});

const SCALE_FORMS: Record<ScaleType, FC<CustomScaleFormProps>> = {
  linear: ({ prefix }) => (
    <Flex.Box x>
      <PForm.NumericField path={`${prefix}.slope`} label="Slope" grow />
      <PForm.NumericField path={`${prefix}.offset`} label="Offset" grow />
    </Flex.Box>
  ),
  map: ({ prefix }) => (
    <>
      <Flex.Box x>
        <PForm.NumericField
          path={`${prefix}.preScaledMin`}
          label="Pre-scaled min"
          grow
        />
        <PForm.NumericField
          path={`${prefix}.preScaledMax`}
          label="Pre-scaled max"
          grow
        />
      </Flex.Box>
      <Flex.Box x>
        <PForm.NumericField path={`${prefix}.scaledMin`} label="Scaled min" grow />
        <PForm.NumericField path={`${prefix}.scaledMax`} label="Scaled max" grow />
      </Flex.Box>
    </>
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

const ThermocoupleTypeField = PForm.buildSelectField<ThermocoupleType>({
  fieldKey: "thermocoupleType",
  fieldProps: { label: "Thermocouple type" },
  inputProps: {
    resourceName: "thermocouple type",
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
  fieldKey: "units",
  fieldProps: { label: "Temperature units" },
  inputProps: {
    resourceName: "temperature units",
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
  Select.SimpleProps<string>,
  "children" | "resourceName"
> {
  model: Device.Model;
}

const SelectCJCSourceField = ({ model, ...rest }: SelectCJCSourceFieldProps) => {
  const ports: record.KeyedNamed[] = Device.PORTS[model][Device.AI_PORT_TYPE];
  return (
    <Select.Simple<string> allowNone={false} {...rest} resourceName="CJC source">
      <Select.Item itemKey={DEVICE_CJC_SOURCE}>Device</Select.Item>
      <Select.Item itemKey={AIR_CJC_SOURCE}>Air</Select.Item>
      {ports.map(({ key, name }) => (
        <Select.Item key={key} itemKey={key}>
          {name}
        </Select.Item>
      ))}
    </Select.Simple>
  );
};

interface FormProps {
  path: string;
  deviceModel: Device.Model;
}

export const FORMS: Record<ReadChannelType, FC<FormProps>> = {
  analog: ({ path }) => (
    <>
      <Divider.Divider x padded="bottom" />
      <MaxVoltageField path={path} />
      <CustomScaleForm prefix={path} />
    </>
  ),
  digital: () => null,
  thermocouple: ({ path, deviceModel }) => (
    <>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <ThermocoupleTypeField path={path} grow />
        <TemperatureUnitsField path={path} grow />
        <PForm.NumericField
          fieldKey="negChan"
          path={path}
          label="Negative channel"
          grow
        />
      </Flex.Box>
      <Flex.Box x>
        <PForm.Field<string>
          path={`${path}.cjcSource`}
          grow
          hideIfNull
          label="CJC source"
        >
          {({ value, onChange, preview }) => (
            <SelectCJCSourceField
              value={value}
              onChange={onChange}
              preview={preview}
              model={deviceModel}
            />
          )}
        </PForm.Field>
        <PForm.NumericField fieldKey="cjcSlope" path={path} label="CJC slope" grow />
        <PForm.NumericField fieldKey="cjcOffset" path={path} label="CJC offset" grow />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={path} />
    </>
  ),
};
