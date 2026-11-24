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

/** @returns the name of the target of a click: a part, or the whole drawing. */
export const targetName = (parts: schematic.symbol.Part[], part?: string): string =>
  part == null
    ? "Whole drawing"
    : (parts.find((p) => p.key === part)?.name ?? "Missing part");

interface CommandButtonsProps {
  value: string | undefined;
  onChange: (key: string) => void;
  commands: schematic.symbol.Command[];
}

const CommandButtons = ({
  value,
  onChange,
  commands,
}: CommandButtonsProps): ReactElement => (
  <Select.Buttons<string>
    value={value}
    onChange={onChange}
    size="small"
    variant="outlined"
  >
    {commands.map((c) => (
      <Select.Item<string> key={c.key} itemKey={c.key} size="small">
        {c.name}
      </Select.Item>
    ))}
  </Select.Buttons>
);

export interface ClickInspectorProps {
  /** click is the key of the click to edit. */
  click: string;
}

/**
 * ClickInspector edits the target of one click, and whether it sends one command or
 * toggles between two. A toggle offers only the commands that the Core accepts.
 */
export const ClickInspector = ({
  click: key,
}: ClickInspectorProps): ReactElement | null => {
  const click = Form.useFieldValue(path.data.interactions[key], { optional: true });
  const parts = Form.useFieldValue(path.data.parts);
  const commands = Form.useFieldValue(path.data.commands);
  const interactions = Form.useFieldValue(path.data.interactions);
  const { set } = Form.useContext();
  if (click == null) return null;
  const update = (next: Interaction) => set(path.data.interactions[key], next);
  const [first, second] =
    click.type === "fire" ? [click.command, undefined] : click.commands;
  const partners = Spec.nextPartners({ commands }, first);
  const toggle = (command: string): Interaction => ({
    type: "next",
    key,
    part: click.part,
    commands: [command, Spec.nextPartners({ commands }, command)[0]?.key ?? ""],
  });
  const handleTypeChange = (type: schematic.symbol.InteractionType) =>
    update(
      type === "next" ? toggle(first) : { type, key, part: click.part, command: first },
    );
  const targets = [click.part, ...Spec.freeTargets({ parts, interactions })];
  return (
    <Flex.Box y gap="medium" className={CSS.B("schematic-inspector")}>
      <Text.Text level="h5">{targetName(parts, click.part)}</Text.Text>
      <Divider.Divider x />
      <Input.Item label="Target">
        <Select.Buttons<string>
          value={click.part ?? WHOLE_DRAWING}
          onChange={(t: string) =>
            update({ ...click, part: t === WHOLE_DRAWING ? undefined : t })
          }
          size="small"
          variant="outlined"
        >
          {targets.map((t) => (
            <Select.Item<string>
              key={t ?? WHOLE_DRAWING}
              itemKey={t ?? WHOLE_DRAWING}
              size="small"
            >
              {targetName(parts, t)}
            </Select.Item>
          ))}
        </Select.Buttons>
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
            disabled={click.type === "fire" && partners.length === 0}
            tooltip={partners.length === 0 ? TOGGLE_NEEDS_PAIR : undefined}
          >
            Toggle
          </Select.Item>
        </Select.Buttons>
      </Input.Item>
      {click.type === "fire" ? (
        <Input.Item label="Command">
          <CommandButtons
            value={click.command}
            onChange={(command) => update({ ...click, command })}
            commands={commands}
          />
        </Input.Item>
      ) : (
        <>
          <Input.Item label="First command">
            <CommandButtons
              value={first}
              onChange={(command) => update(toggle(command))}
              commands={commands.filter(
                (c) => Spec.nextPartners({ commands }, c.key).length > 0,
              )}
            />
          </Input.Item>
          <Input.Item label="Second command">
            <CommandButtons
              value={second}
              onChange={(command) => update({ ...click, commands: [first, command] })}
              commands={partners}
            />
          </Input.Item>
        </>
      )}
    </Flex.Box>
  );
};
