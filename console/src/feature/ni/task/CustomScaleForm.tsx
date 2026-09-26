// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { deep } from "@synnaxlabs/x";
import { type FC } from "react";

import { CoefficientsField } from "@/feature/ni/task/CoefficientsField";
import { TableScaleForm } from "@/feature/ni/task/TableScaleForm";
import {
  createScale,
  type Scale,
  SCALE_SCHEMAS,
  type ScaleType,
  type Units,
} from "@/feature/ni/task/types";

const SelectCustomScaleTypeField = Form.buildSelectField<ScaleType>({
  fieldKey: "type",
  fieldProps: {
    label: "Custom scaling",
    onChange: (value, { get, set, path }) => {
      const prevType = get<ScaleType>(path).value;
      if (prevType === value) return;
      const next = createScale(value);
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
        <Select.Item itemKey="linear">
          <Icon.Linear />
          Linear
        </Select.Item>
        <Select.Item itemKey="map">
          <Icon.Map />
          Map
        </Select.Item>
        <Select.Item itemKey="polynomial">
          <Icon.Function />
          Polynomial
        </Select.Item>
        <Select.Item itemKey="table">
          <Icon.Table />
          Table
        </Select.Item>
        <Select.Item itemKey="none">
          <Icon.None />
          None
        </Select.Item>
      </>
    ),
  },
});

const UnitsField = Form.buildSelectField<Units>({
  fieldKey: "units",
  fieldProps: { label: "Units", style: { width: "19rem" } },
  inputProps: {
    resourceName: "units",
    allowNone: false,
    children: (
      <>
        <Select.Item itemKey="Volts">V</Select.Item>
        <Select.Item itemKey="Amps">A</Select.Item>
        <Select.Item itemKey="DegF">°F</Select.Item>
        <Select.Item itemKey="DegC">°C</Select.Item>
        <Select.Item itemKey="DegR">R</Select.Item>
        <Select.Item itemKey="Kelvins">K</Select.Item>
        <Select.Item itemKey="Strain">strain</Select.Item>
        <Select.Item itemKey="Ohms">Ω</Select.Item>
        <Select.Item itemKey="Hz">Hz</Select.Item>
        <Select.Item itemKey="Seconds">s</Select.Item>
        <Select.Item itemKey="Meters">m</Select.Item>
        <Select.Item itemKey="Inches">in</Select.Item>
        <Select.Item itemKey="Degrees">°</Select.Item>
        <Select.Item itemKey="Radians">rad</Select.Item>
        <Select.Item itemKey="g">g</Select.Item>
        <Select.Item itemKey="MetersPerSecondSquared">m/s^2</Select.Item>
        <Select.Item itemKey="Newtons">N</Select.Item>
        <Select.Item itemKey="Pounds">lb</Select.Item>
        <Select.Item itemKey="KilogramForce">kgf</Select.Item>
        <Select.Item itemKey="PoundsPerSquareInch">psi</Select.Item>
        <Select.Item itemKey="Bar">bar</Select.Item>
        <Select.Item itemKey="Pascals">Pa</Select.Item>
        <Select.Item itemKey="VoltsPerVolt">V/V</Select.Item>
        <Select.Item itemKey="mVoltsPerVolt">mV/V</Select.Item>
        <Select.Item itemKey="NewtonMeters">N·m</Select.Item>
        <Select.Item itemKey="InchOunces">in·oz</Select.Item>
        <Select.Item itemKey="InchPounds">in·lb</Select.Item>
        <Select.Item itemKey="FootPounds">ft·lb</Select.Item>
      </>
    ),
  },
});

export interface CustomScaleFormProps {
  prefix: string;
}

const CustomScaleUnitsFields = ({ prefix }: { prefix: string }) => (
  <Flex.Box x>
    <UnitsField fieldKey="preScaledUnits" label="Prescaled units" path={prefix} grow />
    <Form.TextField fieldKey="scaledUnits" label="Scaled units" path={prefix} grow />
  </Flex.Box>
);

const SCALE_FORMS: Record<ScaleType, FC<CustomScaleFormProps>> = {
  linear: ({ prefix }) => (
    <>
      <CustomScaleUnitsFields prefix={prefix} />
      <Flex.Box x>
        <Form.NumericField fieldKey="slope" label="Slope" path={prefix} grow />
        <Form.NumericField
          fieldKey="yIntercept"
          label="Y-Intercept"
          path={prefix}
          grow
        />
      </Flex.Box>
    </>
  ),
  map: ({ prefix }) => (
    <>
      <CustomScaleUnitsFields prefix={prefix} />
      <Flex.Box x>
        <Form.NumericField
          fieldKey="preScaledMin"
          label="Pre-scaled min"
          path={prefix}
          grow
        />
        <Form.NumericField
          fieldKey="preScaledMax"
          label="Pre-scaled max"
          path={prefix}
        />
      </Flex.Box>
      <Flex.Box x>
        <Form.NumericField fieldKey="scaledMin" label="Scaled min" path={prefix} grow />
        <Form.NumericField fieldKey="scaledMax" label="Scaled max" path={prefix} />
      </Flex.Box>
    </>
  ),
  polynomial: ({ prefix }) => (
    <>
      <CustomScaleUnitsFields prefix={prefix} />
      <CoefficientsField
        path={`${prefix}.forwardCoeffs`}
        label="Forward coefficients"
      />
      <CoefficientsField
        path={`${prefix}.reverseCoeffs`}
        label="Reverse coefficients"
      />
    </>
  ),
  table: ({ prefix }) => (
    <>
      <CustomScaleUnitsFields prefix={prefix} />
      <TableScaleForm prefix={prefix} />
    </>
  ),
  none: () => null,
};

export const CustomScaleForm = ({ prefix }: CustomScaleFormProps) => {
  const path = `${prefix}.customScale`;
  const type = Form.useFieldValue<ScaleType>(`${path}.type`);
  const FormComponent = SCALE_FORMS[type];
  return (
    <>
      <SelectCustomScaleTypeField path={path} />
      <FormComponent prefix={path} />
    </>
  );
};
