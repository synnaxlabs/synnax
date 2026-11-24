// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { Divider } from "@synnaxlabs/lyra/divider";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Schematic } from "@synnaxlabs/pluto";
import type { ReactElement } from "react";

import { Spec } from "@/feature/schematic/symbol/spec";
import { CSS } from "@/platform/css";

const path = Schematic.Symbol.formPath;

type Interaction = schematic.symbol.Interaction;

// WHOLE_DRAWING is the key of the whole drawing, which a select cannot hold as
// undefined.
const WHOLE_DRAWING = "";

const TOGGLE_NEEDS_PAIR =
  "A toggle needs two commands on one state that send different fixed values.";
const TARGET_TAKEN = "Another click is on this target.";
const NOT_PARTNER = "It sends no other fixed value on the state of the first command.";

/** @returns the name of the target of a click: a part, or the whole drawing. */
export const targetName = (parts: schematic.symbol.Part[], part?: string): string =>
  part == null
    ? "Whole drawing"
    : (parts.find((p) => p.key === part)?.name ?? "Missing part");

/** @returns the target of a click in a sentence: a part, or "the whole drawing". */
export const targetPhrase = (parts: schematic.symbol.Part[], part?: string): string =>
  part == null ? "the whole drawing" : targetName(parts, part);

interface CommandSelectProps {
  label: string;
  value: string;
  onChange: (key: string) => void;
  commands: schematic.symbol.Command[];
  /** reason returns why a command cannot be picked, or undefined when it can. */
  reason?: (command: schematic.symbol.Command) => string | undefined;
}

const CommandSelect = ({
  label,
  value,
  onChange,
  commands,
  reason,
}: CommandSelectProps): ReactElement => (
  <Select.Simple<string>
    resourceName="command"
    value={value}
    onChange={onChange}
    allowNone={false}
    triggerProps={{ "aria-label": label }}
  >
    {commands.map((c) => {
      const why = reason?.(c);
      return (
        <Select.Item<string>
          key={c.key}
          itemKey={c.key}
          disabled={why != null}
          tooltip={why}
        >
          {c.name}
        </Select.Item>
      );
    })}
  </Select.Simple>
);

export interface ClickInspectorProps {
  /** click is the key of the click to edit. */
  click: string;
}

/**
 * ClickInspector edits the target of one click, and whether it sends one command or
 * toggles between two. It turns off the targets and commands that the Core refuses.
 */
export const ClickInspector = ({
  click: key,
}: ClickInspectorProps): ReactElement | null => {
  const click = Form.useFieldValue(path.data.interactions[key], { optional: true });
  const parts = Form.useFieldValue(path.data.parts);
  const states = Form.useFieldValue(path.data.states);
  const commands = Form.useFieldValue(path.data.commands);
  const interactions = Form.useFieldValue(path.data.interactions);
  const { set } = Form.useContext();
  if (click == null) return null;
  const update = (next: Interaction) => set(path.data.interactions[key], next);
  const first = click.type === "fire" ? click.command : click.commands[0];
  const partners = Spec.nextPartners({ states, commands }, first);
  const toggle = (command: string, second?: string): Interaction => {
    const pairs = Spec.nextPartners({ states, commands }, command);
    const partner = pairs.find((p) => p.key === second) ?? pairs.at(0);
    if (partner == null) throw new Error(`command ${command} has no toggle partner`);
    return { type: "next", key, part: click.part, commands: [command, partner.key] };
  };
  const handleTypeChange = (type: schematic.symbol.InteractionType) =>
    update(
      type === "next" ? toggle(first) : { type, key, part: click.part, command: first },
    );
  const targets: Spec.Target[] = [undefined, ...parts.map((p) => p.key)];
  const free = Spec.freeTargets({ parts, interactions });
  const taken = (t: Spec.Target) => t !== click.part && !free.includes(t);
  return (
    <Flex.Box y gap="medium" className={CSS.B("schematic-inspector")}>
      <Text.Text level="h5">Click on {targetPhrase(parts, click.part)}</Text.Text>
      <Divider.Divider x />
      <Input.Item label="Target">
        <Select.Simple<string>
          resourceName="target"
          value={click.part ?? WHOLE_DRAWING}
          onChange={(t: string) =>
            update({ ...click, part: t === WHOLE_DRAWING ? undefined : t })
          }
          allowNone={false}
          triggerProps={{ "aria-label": "Target" }}
        >
          {targets.map((t) => (
            <Select.Item<string>
              key={t ?? WHOLE_DRAWING}
              itemKey={t ?? WHOLE_DRAWING}
              disabled={taken(t)}
              tooltip={taken(t) ? TARGET_TAKEN : undefined}
            >
              {targetName(parts, t)}
            </Select.Item>
          ))}
        </Select.Simple>
      </Input.Item>
      <Input.Item label="Kind">
        <Select.Buttons<schematic.symbol.InteractionType>
          value={click.type}
          onChange={handleTypeChange}
          size="small"
          variant="outlined"
        >
          <Select.Item<schematic.symbol.InteractionType> itemKey="fire" size="small">
            Send
          </Select.Item>
          <Select.Item<schematic.symbol.InteractionType>
            itemKey="next"
            size="small"
            disabled={partners.length === 0}
            tooltip={partners.length === 0 ? TOGGLE_NEEDS_PAIR : undefined}
          >
            Toggle
          </Select.Item>
        </Select.Buttons>
      </Input.Item>
      {click.type === "fire" ? (
        <Input.Item label="Command">
          <CommandSelect
            label="Command"
            value={click.command}
            onChange={(command) => update({ ...click, command })}
            commands={commands}
          />
        </Input.Item>
      ) : (
        <>
          <Input.Item label="First command">
            <CommandSelect
              label="First command"
              value={first}
              onChange={(command) => update(toggle(command, click.commands[1]))}
              commands={commands}
              reason={(c) =>
                Spec.nextPartners({ states, commands }, c.key).length > 0
                  ? undefined
                  : TOGGLE_NEEDS_PAIR
              }
            />
          </Input.Item>
          <Input.Item label="Second command">
            <CommandSelect
              label="Second command"
              value={click.commands[1]}
              onChange={(command) => update({ ...click, commands: [first, command] })}
              commands={commands}
              reason={(c) => (partners.includes(c) ? undefined : NOT_PARTNER)}
            />
          </Input.Item>
        </>
      )}
    </Flex.Box>
  );
};
