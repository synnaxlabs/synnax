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

import { useValues } from "@/feature/schematic/symbol/edit/useValues";
import { CSS } from "@/platform/css";

const path = Schematic.Symbol.formPath;

// NONE is the key of no state, which a select cannot hold as undefined.
const NONE = "";

const ASK_NEEDS_STATE = "A command that asks for a value needs a state.";
const TOGGLED = "A toggle sends this command, so it keeps a fixed value on its state.";
const TAKEN = "The other command of its toggle sends this value.";

export interface CommandInspectorProps {
  /** command is the key of the command to edit. */
  command: string;
}

/**
 * CommandInspector edits the state of one command, and the value that it sends. A
 * fixed value on an enumeration state picks one of its values.
 */
export const CommandInspector = ({
  command: key,
}: CommandInspectorProps): ReactElement | null => {
  const states = Form.useFieldValue(path.data.states);
  const commands = Form.useFieldValue(path.data.commands);
  const command = commands.find((c) => c.key === key);
  const interactions = Form.useFieldValue(path.data.interactions);
  const { get, set } = Form.useContext();
  const values = useValues(command?.state ?? NONE);
  if (command == null) return null;
  // A late commit of the number field reads the command as it is then.
  const update = (next: Partial<schematic.symbol.Command>) => {
    const current = get(path.data.commands[key], { optional: true })?.value;
    if (current != null) set(path.data.commands[key], { ...current, ...next });
  };
  // A fixed value starts at the default of an enumeration state, an entry of it.
  const start = (state?: string): number => {
    const found = states.find((s) => s.key === state);
    return found?.type === "enumeration" ? found.default : 0;
  };
  const handleStateChange = (state: string) => {
    const next = state === NONE ? undefined : state;
    update({
      state: next,
      value:
        command.value.type === "fixed" || next == null
          ? { type: "fixed", value: start(next) }
          : command.value,
    });
  };
  const handleValueChange = (value: number) =>
    update({ value: { type: "fixed", value } });
  // The Core refuses a toggle unless its commands send other fixed values on one state.
  const partners = interactions.flatMap((i) =>
    i.type === "next" && i.commands.includes(key)
      ? i.commands.filter((c) => c !== key)
      : [],
  );
  const toggled = partners.length > 0;
  // The Core compares the values of a toggle after the match of their state.
  const state = states.find((s) => s.key === command.state);
  const match = (v: number): number =>
    state == null ? v : Schematic.Node.Custom.matched(state, v);
  const taken = commands.flatMap((c) =>
    partners.includes(c.key) && c.value.type === "fixed" ? [match(c.value.value)] : [],
  );
  const isTaken = (v: number): boolean => taken.includes(match(v));
  const sendsTaken = command.value.type === "fixed" && isTaken(command.value.value);
  // useValues lists no values for a number state, with no state, or while one loads.
  const listed = values.list.length > 0;
  return (
    <Flex.Box y gap="medium" className={CSS.B("schematic-inspector")}>
      <Text.Text level="h5">{command.name}</Text.Text>
      <Divider.Divider x />
      <Input.Item label="State">
        <Select.Buttons<string>
          value={command.state ?? NONE}
          onChange={handleStateChange}
          size="small"
          variant="outlined"
        >
          <Select.Item<string>
            itemKey={NONE}
            size="small"
            disabled={toggled}
            tooltip={toggled ? TOGGLED : undefined}
          >
            None
          </Select.Item>
          {states.map((s) => (
            <Select.Item<string>
              key={s.key}
              itemKey={s.key}
              size="small"
              disabled={toggled && s.key !== command.state}
              tooltip={toggled && s.key !== command.state ? TOGGLED : undefined}
            >
              {s.name}
            </Select.Item>
          ))}
        </Select.Buttons>
      </Input.Item>
      <Input.Item
        label="Value"
        status={sendsTaken ? "error" : undefined}
        helpText={sendsTaken ? TAKEN : undefined}
      >
        <Select.Buttons<schematic.symbol.CommandValueType>
          value={command.value.type}
          onChange={(type: schematic.symbol.CommandValueType) =>
            update({
              value: type === "ask" ? { type } : { type, value: start(command.state) },
            })
          }
          size="small"
          variant="outlined"
        >
          <Select.Item<schematic.symbol.CommandValueType> itemKey="fixed" size="small">
            Fixed
          </Select.Item>
          <Select.Item<schematic.symbol.CommandValueType>
            itemKey="ask"
            size="small"
            disabled={command.state == null || toggled}
            tooltip={
              toggled ? TOGGLED : command.state == null ? ASK_NEEDS_STATE : undefined
            }
          >
            Ask
          </Select.Item>
        </Select.Buttons>
        {command.value.type === "fixed" &&
          (listed ? (
            <Select.Buttons<number>
              value={command.value.value}
              onChange={handleValueChange}
              size="small"
              variant="outlined"
            >
              {values.list.map((v) => (
                <Select.Item<number>
                  key={v.value}
                  itemKey={v.value}
                  size="small"
                  disabled={isTaken(v.value)}
                  tooltip={isTaken(v.value) ? TAKEN : undefined}
                >
                  {v.name}
                </Select.Item>
              ))}
            </Select.Buttons>
          ) : (
            <Input.Numeric value={command.value.value} onChange={handleValueChange} />
          ))}
      </Input.Item>
    </Flex.Box>
  );
};
