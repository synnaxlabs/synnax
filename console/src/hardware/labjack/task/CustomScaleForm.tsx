// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align, Form } from "@synnaxlabs/pluto";
import { deep, type KeyedNamed } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";

import {
  type ChannelType,
  type Scale,
  SCALE_SCHEMAS,
  type ScaleType,
  ZERO_SCALES,
} from "@/hardware/labjack/task/types";

const SelectScaleTypeField = Form.buildDropdownButtonSelectField<
  ScaleType,
  KeyedNamed<ScaleType>
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
    entryRenderKey: "name",
    columns: [{ key: "name", name: "Name" }],
    data: [
      { key: "none", name: "None" },
      { key: "linear", name: "Linear" },
    ],
  },
});

export interface CustomScaleFormProps {
  prefix: string;
  fieldKey?: string;
  label?: string;
}

const SCALE_FORMS: Record<ScaleType, FC<CustomScaleFormProps>> = {
  linear: ({ prefix }) => (
    <Align.Space direction="x" grow>
      <Form.NumericField path={`${prefix}.slope`} label="Slope" grow />
      <Form.NumericField path={`${prefix}.offset`} label="Offset" grow />
    </Align.Space>
  ),
  none: () => <></>,
};

export const CustomScaleForm = ({
  prefix,
}: CustomScaleFormProps): ReactElement | null => {
  const path = `${prefix}.scale`;
  const channelType = Form.useFieldValue<ChannelType>(`${prefix}.type`, true);
  const scaleType = Form.useFieldValue<ScaleType>(`${path}.type`, true);
  if (channelType !== "AI" || scaleType == null) return null;
  const FormComponent = SCALE_FORMS[scaleType];
  return (
    <>
      <SelectScaleTypeField path={path} />
      <FormComponent prefix={path} />
    </>
  );
};
