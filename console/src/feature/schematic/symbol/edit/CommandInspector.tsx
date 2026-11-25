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
import { Icon } from "@synnaxlabs/lyra/icon";
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
  const command = Form.useFieldValue(path.data.commands[key], { optional: true });
  const states = Form.useFieldValue(path.data.states);
  const commands = Form.useFieldValue(path.data.commands);
  const interactions = Form.useFieldValue(path.data.interactions);
  const { get, set } = Form.useContext();
  const values = useValues(command?.state);
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
  const fixed = (value: number) => update({ value: { type: "fixed", value } });
  // The Core refuses a toggle unless its commands send other fixed values on one state.
  const partners = interactions.flatMap((i) =>
    i.type === "next" && i.commands.includes(key)
      ? i.commands.filter((c) => c !== key)
      : [],
  );
  const toggled = partners.length > 0;
  const taken = commands.flatMap((c) =>
    partners.includes(c.key) && c.value.type === "fixed" ? [c.value.value] : [],
  );
  // useValues falls back to another state when the command's state lists no values.
  const listed =
    command.state != null && values.state === command.state && values.list.length > 0;
  return (
    <Flex.Box y gap="medium" className={CSS.B("schematic-inspector")}>
      <Text.Text level="h5">{command.name}</Text.Text>
      <Divider.Divider x />
      <Input.Item label="State">
        <Select.Simple<string>
          resourceName="state"
          value={command.state ?? NONE}
          onChange={handleStateChange}
          allowNone={false}
          triggerProps={{ "aria-label": "State" }}
          footer={
            <>
              <Divider.Divider x />
              <Select.Item<string>
                itemKey={NONE}
                textValue="No state"
                disabled={toggled}
                tooltip={toggled ? TOGGLED : undefined}
              >
                <Icon.None />
                No state
              </Select.Item>
            </>
          }
        >
          {states.map((s) => (
            <Select.Item<string>
              key={s.key}
              itemKey={s.key}
              disabled={toggled && s.key !== command.state}
              tooltip={toggled && s.key !== command.state ? TOGGLED : undefined}
            >
              {s.name}
            </Select.Item>
          ))}
        </Select.Simple>
      </Input.Item>
      <Input.Item label="Value">
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
            <Select.Simple<number>
              resourceName="value"
              value={command.value.value}
              onChange={fixed}
              allowNone={false}
              triggerProps={{ "aria-label": "Fixed value" }}
            >
              {values.list.map((v) => (
                <Select.Item<number>
                  key={v.value}
                  itemKey={v.value}
                  disabled={taken.includes(v.value)}
                  tooltip={taken.includes(v.value) ? TAKEN : undefined}
                >
                  {v.name}
                </Select.Item>
              ))}
            </Select.Simple>
          ) : (
            <Input.Numeric value={command.value.value} onChange={fixed} />
          ))}
      </Input.Item>
    </Flex.Box>
  );
};
