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
  const { set } = Form.useContext();
  const values = useValues(command?.state);
  if (command == null) return null;
  const update = (next: Partial<schematic.symbol.Command>) =>
    set(path.data.commands[key], { ...command, ...next });
  const handleStateChange = (state: string) =>
    update({
      state: state === NONE ? undefined : state,
      value:
        state === NONE && command.value.type === "ask"
          ? { type: "fixed", value: 0 }
          : command.value,
    });
  const fixed = (value: number) => update({ value: { type: "fixed", value } });
  // useValues falls back to another state when the command's state lists no values.
  const listed = command.state != null && values.state === command.state;
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
          <Select.Item<string> itemKey={NONE} size="small">
            None
          </Select.Item>
          {states.map((s) => (
            <Select.Item<string> key={s.key} itemKey={s.key} size="small">
              {s.name}
            </Select.Item>
          ))}
        </Select.Buttons>
      </Input.Item>
      <Input.Item label="Value">
        <Select.Buttons<schematic.symbol.CommandValueType>
          value={command.value.type}
          onChange={(type: schematic.symbol.CommandValueType) =>
            update({ value: type === "ask" ? { type } : { type, value: 0 } })
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
            disabled={command.state == null}
            tooltip={command.state == null ? ASK_NEEDS_STATE : undefined}
          >
            Ask
          </Select.Item>
        </Select.Buttons>
        {command.value.type === "fixed" &&
          (listed ? (
            <Select.Buttons<number>
              value={command.value.value}
              onChange={fixed}
              size="small"
              variant="outlined"
            >
              {values.list.map((v) => (
                <Select.Item<number> key={v.value} itemKey={v.value} size="small">
                  {v.name}
                </Select.Item>
              ))}
            </Select.Buttons>
          ) : (
            <Input.Numeric value={command.value.value} onChange={fixed} />
          ))}
      </Input.Item>
    </Flex.Box>
  );
};
