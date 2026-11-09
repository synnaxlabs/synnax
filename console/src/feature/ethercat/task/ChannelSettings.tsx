// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as PForm } from "@synnaxlabs/lyra/form";
import { type deep, primitive } from "@synnaxlabs/x";
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
import { type EntryReadChannel } from "@/feature/ethercat/task/types";

export interface ChannelSettingsProps {
  path: deep.Path<EntryReadChannel>;
}

/**
 * Edits the settings that the read task writes for the channel when it starts, as the
 * catalog of the channel's slave describes them. Renders nothing for an entry that the
 * catalog does not describe.
 */
export const ChannelSettings = ({
  path,
}: ChannelSettingsProps): ReactElement | null => {
  const ch = PForm.useFieldValue(path);
  const { value: values, onChange } = PForm.useField(path.settings);
  const { data: slave } = useResultSlave(
    primitive.isZero(ch.device) ? null : { key: ch.device },
  );
  const channel = findChannel(slave?.properties.settings, ch);
  if (channel == null) return null;
  const fields = channel.settings;
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
            if (value != null) onChange(set(fields, values, field, value));
          }}
        />
      ))}
    </PForm.Section>
  );
};
