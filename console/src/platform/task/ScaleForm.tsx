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
import { Select } from "@synnaxlabs/lyra/select";
import { deep } from "@synnaxlabs/x";
import { type FC } from "react";
import { type z } from "zod";

export type ScaleType = "linear" | "map" | "none";

interface Scale {
  type: ScaleType;
}

/** The schema of each scale type, used to fill defaults when the type changes. */
export type ScaleSchemas = Record<ScaleType, z.ZodType<Scale>>;

const SelectTypeField = Form.buildSelectField<ScaleType>({
  fieldKey: "type",
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
      </>
    ),
  },
});

interface ParamsProps {
  path: string;
}

const PARAMS: Record<ScaleType, FC<ParamsProps>> = {
  linear: ({ path }) => (
    <>
      <Form.NumericField path={`${path}.slope`} label="Slope" />
      <Form.NumericField path={`${path}.offset`} label="Offset" />
    </>
  ),
  map: ({ path }) => (
    <>
      <Form.NumericField path={`${path}.preScaledMin`} label="Pre-scaled min" />
      <Form.NumericField path={`${path}.preScaledMax`} label="Pre-scaled max" />
      <Form.NumericField path={`${path}.scaledMin`} label="Scaled min" />
      <Form.NumericField path={`${path}.scaledMax`} label="Scaled max" />
    </>
  ),
  none: () => null,
};

export interface ScaleFormProps {
  /** The path of the scale field. */
  path: string;
  /** The schema of each scale type the field accepts. */
  schemas: ScaleSchemas;
}

/**
 * Edits a scale that is none, linear (slope and offset), or map (one range onto
 * another). Changing the type keeps the parameters the new type shares with the old.
 */
export const ScaleForm = ({ path, schemas }: ScaleFormProps) => {
  const type = Form.useFieldValue<ScaleType>(`${path}.type`);
  const handleTypeChange: Form.FieldProps<ScaleType>["onChange"] = (
    value,
    { get, set },
  ) => {
    if (type === value) return;
    const next = schemas[value].parse({ type: value });
    set(path, {
      ...deep.overrideValidItems(next, get<Scale>(path).value, schemas[value]),
      type: next.type,
    });
  };
  const Params = PARAMS[type];
  return (
    <>
      <SelectTypeField path={path} onChange={handleTypeChange} />
      <Params path={path} />
    </>
  );
};
