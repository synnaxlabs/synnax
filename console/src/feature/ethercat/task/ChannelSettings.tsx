// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ethercat } from "@synnaxlabs/client";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Text } from "@synnaxlabs/lyra/text";
import { caseconv, deep, primitive } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { useResultSlave } from "@/feature/ethercat/device/queries";
import { SettingInput } from "@/feature/ethercat/device/SettingInput";
import {
  check,
  findInput,
  problemAt,
  set,
  valueAt,
  visibleFields,
} from "@/feature/ethercat/device/settings";
import { setUpBy } from "@/feature/ethercat/device/setup";
import { type EntryReadChannel, type ReadSchemas } from "@/feature/ethercat/task/types";
import { type Task } from "@/platform/task";

const PARAMETERS_PATH = deep.path<Task.Values<ReadSchemas>>().config.startupParameters;

/** @returns the startup parameters with those of the slave replaced. */
const withParameters = (
  held: ethercat.SlaveStartupParameters[],
  device: string,
  parameters: ethercat.StartupParameter[],
): ethercat.SlaveStartupParameters[] => [
  ...held.filter((s) => s.device !== device),
  { device, parameters },
];

export interface ChannelSettingsProps {
  path: deep.Path<EntryReadChannel>;
}

/**
 * Edits the settings that the read task writes to the input that the channel reads, as
 * the catalog of the channel's slave describes them. Channels that read one input share
 * its settings. Renders nothing for an entry that the catalog does not describe, and a
 * note for a slave that other software sets up.
 */
export const ChannelSettings = ({
  path,
}: ChannelSettingsProps): ReactElement | null => {
  const ch = PForm.useFieldValue(path);
  const { value: held, onChange } = PForm.useField(PARAMETERS_PATH);
  const { data: slave } = useResultSlave(
    primitive.isZero(ch.device) ? null : { key: ch.device },
  );
  const input = findInput(slave?.properties.settings, ch);
  if (input == null || slave == null) return null;
  const reader = setUpBy(slave.properties);
  if (reader != null)
    return (
      <PForm.Section title="Settings">
        <Text.Text level="small" color={9}>
          {caseconv.capitalize(reader)} sets the settings of this device.
        </Text.Text>
      </PForm.Section>
    );
  const fields = input.settings;
  const values = held.find((s) => s.device === ch.device)?.parameters ?? [];
  const problems = check(input, values);
  return (
    <PForm.Section title="Settings">
      {visibleFields(fields, values).map((field) => (
        <SettingInput
          key={`${field.index}_${field.subIndex}_${field.name}`}
          field={field}
          value={valueAt(fields, values, field)}
          problem={problemAt(problems, field)?.message}
          onChange={(value) =>
            onChange(withParameters(held, ch.device, set(fields, values, field, value)))
          }
        />
      ))}
    </PForm.Section>
  );
};
