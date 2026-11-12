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
import { caseconv, deep, primitive, unique } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { useSlavesByKeys } from "@/feature/ethercat/device/queries";
import { SettingInput } from "@/feature/ethercat/device/SettingInput";
import {
  check,
  type Field,
  findInput,
  type Input,
  problemAt,
  set,
  valueAt,
  visibleFields,
} from "@/feature/ethercat/device/settings";
import { setUpBy } from "@/feature/ethercat/device/setup";
import { hasEntry, type ReadSchemas } from "@/feature/ethercat/task/types";
import { type Task } from "@/platform/task";

const ROOT = deep.path<Task.Values<ReadSchemas>>();

/** @returns the startup parameters with those of the slave replaced. */
const withParameters = (
  held: ethercat.SlaveStartupParameters[],
  device: string,
  parameters: ethercat.StartupParameter[],
): ethercat.SlaveStartupParameters[] => [
  ...held.filter((s) => s.device !== device),
  { device, parameters },
];

const parametersOf = (
  held: ethercat.SlaveStartupParameters[],
  device: string,
): ethercat.StartupParameter[] =>
  held.find((s) => s.device === device)?.parameters ?? [];

/** One input of a slave that the selected channels read. */
interface Target {
  device: string;
  input: Input;
}

/** @returns whether the two fields set the same thing in the same way. */
const sameSetting = (a: Field, b: Field): boolean =>
  a.name === b.name && a.unit === b.unit && deep.equal(a.options, b.options);

export interface ChannelSettingsProps {
  /** The keys of the channels whose inputs to edit. */
  keys: string[];
}

/**
 * Edits the settings that the read task writes to the inputs that the channels read, as
 * the catalogs of their slaves describe them. Channels that read one input share its
 * settings. With several inputs, it edits the settings that every input shows, and a
 * setting that the inputs hold different values for shows as mixed. When other software
 * sets up every input, shows the values that the inputs held at the scan, and the user
 * cannot change them. Renders nothing when the catalogs describe none of the inputs.
 */
export const ChannelSettings = ({
  keys,
}: ChannelSettingsProps): ReactElement | null => {
  const channels = PForm.useFieldValue(ROOT.config.channels)
    .filter(hasEntry)
    .filter((c) => keys.includes(c.key));
  const { value: held, onChange } = PForm.useField(ROOT.config.startupParameters);
  const slaves = useSlavesByKeys(
    unique.unique(channels.map((c) => c.device).filter(primitive.isNonZero)),
  );
  const targets: Target[] = [];
  const shared: Target[] = [];
  const readers = new Set<string>();
  for (const ch of channels) {
    const slave = slaves?.find((s) => s.key === ch.device);
    const input = findInput(slave?.properties.settings, ch);
    if (slave == null || input == null) continue;
    const reader = setUpBy(slave.properties);
    if (reader != null) readers.add(reader);
    const list = reader == null ? targets : shared;
    if (!list.some((t) => t.device === ch.device && t.input === input))
      list.push({ device: ch.device, input });
  }
  const disabled = targets.length === 0;
  const shown = disabled ? shared : targets;
  if (shown.length === 0) return null;
  const valuesOf = (device: string) => (disabled ? [] : parametersOf(held, device));
  const [reader] = readers;
  const note =
    reader != null &&
    (disabled ? (
      <Text.Text level="small" color={9}>
        {caseconv.capitalize(reader)} sets these values. Synnax does not change them.
      </Text.Text>
    ) : (
      <Text.Text level="small" color={9}>
        {caseconv.capitalize(reader)} sets the settings of some selected devices.
      </Text.Text>
    ));
  const [lead, ...others] = shown;
  const rows = visibleFields(lead.input.settings, valuesOf(lead.device)).flatMap(
    (field) => {
      const matches = others.map(({ input, device }) =>
        visibleFields(input.settings, valuesOf(device)).find((f) =>
          sameSetting(f, field),
        ),
      );
      return matches.every((m) => m != null) ? [[field, ...matches]] : [];
    },
  );
  return (
    <PForm.Section title="Settings">
      {note}
      {rows.map((fields) => {
        const values = shown.map(({ input, device }, i) =>
          valueAt(input.settings, valuesOf(device), fields[i]),
        );
        const problem = disabled
          ? undefined
          : targets
              .map(({ input, device }, i) =>
                problemAt(check(input, parametersOf(held, device)), fields[i]),
              )
              .find((p) => p != null);
        const handleChange = (value: number) => {
          let next = held;
          targets.forEach(({ input, device }, i) => {
            const parameters = set(
              input.settings,
              parametersOf(next, device),
              fields[i],
              value,
            );
            next = withParameters(next, device, parameters);
          });
          onChange(next);
        };
        const [field] = fields;
        return (
          <SettingInput
            key={`${field.index}_${field.subIndex}_${field.name}`}
            field={field}
            value={values[0]}
            mixed={values.some((v) => v !== values[0])}
            problem={problem?.message}
            onChange={handleChange}
            disabled={disabled}
          />
        );
      })}
    </PForm.Section>
  );
};
