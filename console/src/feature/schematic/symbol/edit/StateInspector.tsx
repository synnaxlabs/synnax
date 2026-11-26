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
import { Enumeration } from "@/platform/enumeration";

const path = Schematic.Symbol.formPath;

interface EnumerationFieldsProps {
  state: schematic.symbol.EnumerationState;
  update: (next: Partial<schematic.symbol.EnumerationState>) => void;
}

const EnumerationFields = ({ state, update }: EnumerationFieldsProps): ReactElement => {
  const { list } = useValues(state.key);
  return (
    <>
      <Input.Item label="Enumeration" padHelpText={false}>
        <Schematic.Node.Form.EnumerationSelect
          value={state.enumeration}
          onChange={(next) => update({ enumeration: next })}
          draft={() => ({ name: state.name, entries: [] })}
        />
      </Input.Item>
      <Input.Item label="Default" padHelpText={false}>
        <Select.Simple<number>
          resourceName="value"
          value={state.default}
          onChange={(value: number) => update({ default: value })}
          allowNone={false}
          triggerProps={{ "aria-label": "Default" }}
        >
          {list.map((v) => (
            <Select.Item<number> key={v.value} itemKey={v.value}>
              {v.name}
            </Select.Item>
          ))}
        </Select.Simple>
      </Input.Item>
      <Input.Item label="Match" padHelpText={false}>
        <Schematic.Node.Form.MatchInput
          value={state.match}
          onChange={(match) => update({ match })}
        />
      </Input.Item>
    </>
  );
};

export interface StateInspectorProps {
  /** The key of the state to edit. */
  state: string;
}

/**
 * Edits the values of one state: the enumeration that names and colors them, the value
 * that a node draws with no data, and how the state reads a value. A number state edits
 * its range.
 */
export const StateInspector = ({
  state: key,
}: StateInspectorProps): ReactElement | null => {
  const states = Form.useFieldValue(path.data.states);
  const state = states.find((s) => s.key === key);
  const { get, set } = Form.useContext();
  if (state == null) return null;
  // A late commit of a number field reads the state as it is then.
  const update = (next: Partial<schematic.symbol.State>) => {
    const current = get(path.data.states[key], { optional: true })?.value;
    if (current != null)
      set(path.data.states[key], { ...current, ...next } as schematic.symbol.State);
  };
  return (
    <Flex.Box y gap="medium" className={CSS.B("schematic-inspector")}>
      <Text.Text level="h5">{state.name}</Text.Text>
      <Divider.Divider x />
      {state.type === "number" ? (
        <Input.Item label="Range" padHelpText={false}>
          <Schematic.Node.Form.RangeInput
            label="Range"
            value={state.bounds}
            onChange={(b) => update({ bounds: b })}
          />
        </Input.Item>
      ) : (
        <Enumeration.CreatePromptProvider>
          <EnumerationFields state={state} update={update} />
        </Enumeration.CreatePromptProvider>
      )}
    </Flex.Box>
  );
};
