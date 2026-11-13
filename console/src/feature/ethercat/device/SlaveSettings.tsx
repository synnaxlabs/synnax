// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ethercat } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Text } from "@synnaxlabs/lyra/text";
import { caseconv, deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { SettingInput } from "@/feature/ethercat/device/SettingInput";
import {
  type Field,
  sameAddress,
  visibleFields,
} from "@/feature/ethercat/device/settings";
import { setUpBy } from "@/feature/ethercat/device/setup";
import { type SlaveDevice } from "@/feature/ethercat/device/types";

const PATH = deep.path<SlaveDevice>().properties;

/**
 * @returns the parameters after the field takes the value, or loses it on null. A new
 * parameter has no name, so the table shows the name of the field.
 */
const setParameter = (
  parameters: ethercat.StartupParameter[],
  { index, subIndex, dataType }: Field,
  value: number | null,
): ethercat.StartupParameter[] => {
  const i = parameters.findIndex((p) => sameAddress(p, { index, subIndex }));
  if (value == null) return parameters.filter((_, j) => j !== i);
  const name = i === -1 ? "" : parameters[i].name;
  const parameter = { index, subIndex, dataType, value, name };
  if (i === -1) return [...parameters, parameter];
  return parameters.map((p, j) => (j === i ? parameter : p));
};

/**
 * Edits the fields of the whole slave that its catalog describes. Each field is a
 * startup parameter. For a slave that other software sets up, shows the values that the
 * slave held at the scan, read only. Renders nothing for a slave whose catalog
 * describes none.
 */
export const SlaveSettings = (): ReactElement | null => {
  const settings = Form.useFieldValue(PATH.settings, { optional: true });
  const { value: stored, onChange } = Form.useField(PATH.startupParameters);
  const reader = setUpBy(settings);
  const fields = settings?.device ?? [];
  if (fields.length === 0) return null;
  const parameters = reader == null ? stored : [];
  return (
    <Form.Section title="Settings">
      {visibleFields(fields, parameters).map((field) => (
        <SettingInput
          key={`${field.index}_${field.subIndex}_${field.name}`}
          field={field}
          value={
            reader == null
              ? parameters.find((p) => sameAddress(p, field))?.value
              : field.value
          }
          onChange={(value) => onChange(setParameter(parameters, field, value))}
          disabled={reader != null}
          allowNone
        />
      ))}
      <Text.Text level="small" color={9}>
        {reader == null
          ? "The Driver writes each set value as a startup parameter. A field not " +
            "set keeps the value on the device."
          : `${caseconv.capitalize(reader)} sets these values. Synnax does not ` +
            "change them."}
      </Text.Text>
    </Form.Section>
  );
};
