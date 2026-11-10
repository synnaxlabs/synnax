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
import { deep, primitive } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { useResultSlave } from "@/feature/ethercat/device/queries";
import { SettingInput } from "@/feature/ethercat/device/SettingInput";
import {
  check,
  findChannel,
  problemAt,
  set,
  valueAt,
  visibleFields,
} from "@/feature/ethercat/device/settings";
import { isShared } from "@/feature/ethercat/device/setup";
import { type EntryReadChannel, type ReadSchemas } from "@/feature/ethercat/task/types";
import { type Task } from "@/platform/task";

const SETTINGS_PATH = deep.path<Task.Values<ReadSchemas>>().config.settings;

/** @returns settings with the values of the slave replaced. */
const withValues = (
  settings: ethercat.SlaveSettings[],
  device: string,
  values: ethercat.Setting[],
): ethercat.SlaveSettings[] => [
  ...settings.filter((s) => s.device !== device),
  { device, values },
];

export interface ChannelSettingsProps {
  path: deep.Path<EntryReadChannel>;
}

/**
 * Edits the settings that the read task writes to the input that the channel reads,
 * as the catalog of the channel's slave describes them. Channels that read one input
 * share its settings. Renders nothing for an entry that the catalog does not describe,
 * or for a slave that other software sets up.
 */
export const ChannelSettings = ({
  path,
}: ChannelSettingsProps): ReactElement | null => {
  const ch = PForm.useFieldValue(path);
  const { value: settings, onChange } = PForm.useField(SETTINGS_PATH);
  const { data: slave } = useResultSlave(
    primitive.isZero(ch.device) ? null : { key: ch.device },
  );
  const channel = findChannel(slave?.properties.settings, ch);
  if (channel == null || slave == null || isShared(slave.properties)) return null;
  const fields = channel.settings;
  const values = settings.find((s) => s.device === ch.device)?.values ?? [];
  const problems = check(channel, values);
  return (
    <PForm.Section title="Settings">
      {visibleFields(fields, values).map((field) => (
        <SettingInput
          key={`${field.index}_${field.subIndex}_${field.name}`}
          field={field}
          value={valueAt(fields, values, field)}
          problem={problemAt(problems, field)?.message}
          onChange={(value) => {
            if (value == null) return;
            onChange(
              withValues(settings, ch.device, set(fields, values, field, value)),
            );
          }}
        />
      ))}
    </PForm.Section>
  );
};
