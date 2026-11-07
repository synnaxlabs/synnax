// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { deep } from "@synnaxlabs/x";
import { type FC } from "react";

import { CoefficientsField } from "@/platform/task/CoefficientsField";
import { TableScaleForm } from "@/platform/task/TableScaleForm";

const SelectType = Form.buildSelectField<task.ScaleType>({
  fieldProps: { label: "Scale" },
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
        <Select.Item itemKey="polynomial">
          <Icon.Function />
          Polynomial
        </Select.Item>
        <Select.Item itemKey="table">
          <Icon.Table />
          Table
        </Select.Item>
      </>
    ),
  },
});

interface ParamsProps {
  path: deep.Path<task.Scale>;
}

const PARAMS: Record<task.ScaleType, FC<ParamsProps>> = {
  linear: ({ path }) => {
    const { offset, slope } = deep.variant(path, "type", "linear");
    return (
      <>
        <Form.NumericField path={slope} label="Slope" />
        <Form.NumericField path={offset} label="Offset" />
      </>
    );
  },
  map: ({ path }) => {
    const scale = deep.variant(path, "type", "map");
    return (
      <>
        <Form.NumericField path={scale.preScaledMin} label="Pre-scaled min" />
        <Form.NumericField path={scale.preScaledMax} label="Pre-scaled max" />
        <Form.NumericField path={scale.scaledMin} label="Scaled min" />
        <Form.NumericField path={scale.scaledMax} label="Scaled max" />
      </>
    );
  },
  polynomial: ({ path }) => (
    <CoefficientsField
      path={deep.variant(path, "type", "polynomial").coeffs}
      label="Coefficients"
    />
  ),
  table: ({ path }) => <TableScaleForm path={deep.variant(path, "type", "table")} />,
  none: () => null,
};

export interface ScaleFormProps {
  /** The path of the scale field. */
  path: deep.Target<task.Scale>;
}

/**
 * Edits a scale that is none, linear (slope and offset), map (one range onto another),
 * polynomial (coefficients), or table (pairs of raw and scaled values).
 */
export const ScaleForm = ({ path }: ScaleFormProps) => {
  const scale = deep.widen(path);
  const type = Form.useFieldValue(scale.type);
  const Params = PARAMS[type];
  return (
    <>
      <SelectType
        path={scale.type}
        onChange={(value, { set }) => {
          if (type !== value)
            set(scale, task.SCALE_SCHEMAS[value].parse({ type: value }));
        }}
      />
      <Params path={scale} />
    </>
  );
};
