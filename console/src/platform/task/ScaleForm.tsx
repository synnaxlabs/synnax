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

interface LinearScale {
  type: "linear";
  slope: number;
  offset: number;
}

interface MapScale {
  type: "map";
  preScaledMin: number;
  preScaledMax: number;
  scaledMin: number;
  scaledMax: number;
}

type Scale = LinearScale | MapScale | { type: "none" };

type ScaleType = Scale["type"];

/** The schema of each scale type, used to fill defaults when the type changes. */
type ScaleSchemas = Record<ScaleType, z.ZodType<Scale>>;

const SelectType = Form.buildSelectField<ScaleType>({
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
  path: deep.Path<Scale>;
}

const PARAMS: Record<ScaleType, FC<ParamsProps>> = {
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
  none: () => null,
};

export interface ScaleFormProps {
  /** The path of the scale field. */
  path: deep.Target<Scale>;
  /** The schema of each scale type the field accepts. */
  schemas: ScaleSchemas;
}

/**
 * Edits a scale that is none, linear (slope and offset), or map (one range onto
 * another).
 */
export const ScaleForm = ({ path, schemas }: ScaleFormProps) => {
  const scale = deep.widen(path);
  const type = Form.useFieldValue(scale.type);
  const Params = PARAMS[type];
  return (
    <>
      <SelectType
        path={scale.type}
        onChange={(value, { set }) => {
          if (type !== value) set(scale, schemas[value].parse({ type: value }));
        }}
      />
      <Params path={scale} />
    </>
  );
};
