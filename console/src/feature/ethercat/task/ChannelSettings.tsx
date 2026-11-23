// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ethercat } from "@synnaxlabs/client";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Text } from "@synnaxlabs/lyra/text";
import { deep, primitive, strings } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { Device } from "@/feature/ethercat/device";
import type { EntryReadChannel, ReadSchemas } from "@/feature/ethercat/task/types";
import type { Task } from "@/platform/task";

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
 * its settings. For a slave that other software sets up, shows the values from the
 * scan, read only. Renders nothing for an entry that the catalog does not describe.
 */
export const ChannelSettings = ({
  path,
}: ChannelSettingsProps): ReactElement | null => {
  const ch = PForm.useFieldValue(path);
  const { value: held, onChange } = PForm.useField(PARAMETERS_PATH);
  const { data: slave } = Device.useResultSlave(
    primitive.isZero(ch.device) ? null : { key: ch.device },
  );
  const [moved, setMoved] = useState<{
    device: string;
    notes: Device.Settings.Note[];
  }>();
  const input = Device.Settings.findInput(slave?.properties.settings, ch);
  if (input == null || slave == null) return null;
  const reader = Device.Setup.setUpBy(slave.properties.settings);
  const fields = input.settings;
  const prior = held.find((s) => s.device === ch.device)?.parameters ?? [];
  const filled =
    reader == null ? Device.Settings.fill(fields, prior) : { values: [], notes: [] };
  const { values } = filled;
  const problems = reader == null ? Device.Settings.check(input, values) : [];
  const notes = [...filled.notes, ...(moved?.device === ch.device ? moved.notes : [])];
  return (
    <PForm.Section title="Settings">
      {reader != null && (
        <Text.Text level="small" color={9}>
          {strings.capitalize(reader)} sets these values. Synnax does not change them.
        </Text.Text>
      )}
      {Device.Settings.visibleFields(fields, values).map((field) => (
        <Device.SettingInput
          key={`${field.index}_${field.subIndex}_${field.name}`}
          field={field}
          value={
            reader == null ? Device.Settings.valueAt(fields, values, field) : undefined
          }
          problem={Device.Settings.problemAt(problems, field)?.message}
          note={notes.find((n) => Device.Settings.sameAddress(n, field))?.message}
          onChange={(value) => {
            const change = Device.Settings.set(fields, prior, field, value);
            setMoved({ device: ch.device, notes: change.notes });
            onChange(withParameters(held, ch.device, change.values));
          }}
          disabled={reader != null}
        />
      ))}
    </PForm.Section>
  );
};
