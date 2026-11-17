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
import { type ReactElement, useState } from "react";

import { useSlavesByKeys } from "@/feature/ethercat/device/queries";
import { SettingInput } from "@/feature/ethercat/device/SettingInput";
import {
  check,
  type Field,
  fill,
  findInput,
  type Input,
  type Note,
  problemAt,
  sameAddress,
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
 * setting that the inputs hold different values for shows as mixed. Renders nothing
 * when the catalogs describe none of the inputs, and a note for slaves that other
 * software sets up.
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
  const [moved, setMoved] = useState<{ selection: string; notes: Note[] }>();
  const targets: Target[] = [];
  const readers = new Set<string>();
  for (const ch of channels) {
    const slave = slaves?.find((s) => s.key === ch.device);
    const input = findInput(slave?.properties.settings, ch);
    if (slave == null || input == null) continue;
    const reader = setUpBy(slave.properties);
    if (reader != null) readers.add(reader);
    else if (!targets.some((t) => t.device === ch.device && t.input === input))
      targets.push({ device: ch.device, input });
  }
  if (targets.length === 0 && readers.size === 0) return null;
  const [reader] = readers;
  const note = reader != null && (
    <Text.Text level="small" color={9}>
      {caseconv.capitalize(reader)} sets the settings of{" "}
      {targets.length === 0 ? "this device" : "some selected devices"}.
    </Text.Text>
  );
  // A note on a moved value shows until the selection changes.
  const selection = keys.join(",");
  const filled = targets.map(({ input, device }) =>
    fill(input.settings, parametersOf(held, device)),
  );
  const notes = [
    ...filled.flatMap((f) => f.notes),
    ...(moved?.selection === selection ? moved.notes : []),
  ];
  const [lead, ...others] = targets;
  const rows =
    lead == null
      ? []
      : visibleFields(lead.input.settings, filled[0].values).flatMap((field) => {
          const matches = others.map(({ input }, i) =>
            visibleFields(input.settings, filled[i + 1].values).find((f) =>
              sameSetting(f, field),
            ),
          );
          return matches.every((m) => m != null) ? [[field, ...matches]] : [];
        });
  return (
    <PForm.Section title="Settings">
      {note}
      {rows.map((fields) => {
        const values = targets.map(({ input }, i) =>
          valueAt(input.settings, filled[i].values, fields[i]),
        );
        const problem = targets
          .map(({ input }, i) => problemAt(check(input, filled[i].values), fields[i]))
          .find((p) => p != null);
        const handleChange = (value: number) => {
          let next = held;
          const changed: Note[] = [];
          targets.forEach(({ input, device }, i) => {
            const change = set(
              input.settings,
              parametersOf(next, device),
              fields[i],
              value,
            );
            changed.push(...change.notes);
            next = withParameters(next, device, change.values);
          });
          setMoved({ selection, notes: changed });
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
            note={notes.find((n) => fields.some((f) => sameAddress(n, f)))?.message}
            onChange={handleChange}
          />
        );
      })}
    </PForm.Section>
  );
};
