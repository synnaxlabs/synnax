// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Header } from "@synnaxlabs/lyra/header";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { Schematic } from "@synnaxlabs/pluto";
import { strings } from "@synnaxlabs/x";
import pluralize from "pluralize";
import type { ReactElement } from "react";

import { Spec } from "@/feature/schematic/symbol/spec";
import { Modals } from "@/platform/modals";

const path = Schematic.Symbol.formPath;

const TYPE_NAMES: Record<schematic.symbol.StateType, string> = {
  enumeration: "Enumeration",
  number: "Number",
};

const describeUses = ({ looks, commands, interactions }: Spec.Removed): string =>
  strings.naturalLanguageJoin(
    (
      [
        ["look", looks],
        ["command", commands],
        ["click", interactions],
      ] as const
    )
      .filter(([, keys]) => keys.length > 0)
      .map(([noun, keys]) => pluralize(noun, keys.length, true)),
  );

export interface StateListProps {
  /** onRemove is called with the key of a state after it goes. */
  onRemove: (key: string) => void;
}

/** StateList lists the states of the symbol, and adds, renames, and removes them. */
export const StateList = ({ onRemove }: StateListProps): ReactElement => {
  const states = Form.useFieldValue(path.data.states);
  const { get, set } = Form.useContext();
  const confirm = Modals.useConfirm();
  const handleRemove = async ({ key, name }: schematic.symbol.State) => {
    const [next, removed] = Spec.removeState(get(path.data).value, key);
    const uses = describeUses(removed);
    if (
      uses.length > 0 &&
      (await confirm({
        message: `Remove ${name}?`,
        description: `Its uses go with it: ${uses}.`,
        confirm: { label: "Remove", variant: "error" },
      })) !== true
    )
      return;
    set(path.data, next);
    onRemove(key);
  };
  return (
    <Flex.Box y gap={1}>
      <Header.Header level="p" bordered={false}>
        <Header.Title level="p" weight={500}>
          States
        </Header.Title>
        <Header.Actions>
          <Button.Button
            onClick={() => set(path.data, Spec.addState(get(path.data).value))}
            size="small"
            variant="text"
            tooltip="Add state"
          >
            <Icon.Add />
          </Button.Button>
        </Header.Actions>
      </Header.Header>
      {states.map((s) => (
        <Flex.Box key={s.key} x justify="between" align="center">
          <Flex.Box y gap={0}>
            <Form.Field
              path={path.data.states[s.key].name}
              showLabel={false}
              padHelpText={false}
            >
              {({ onChange, value }) => (
                <Text.MaybeEditable level="small" value={value} onChange={onChange} />
              )}
            </Form.Field>
            <Text.Text level="small" color={9}>
              {TYPE_NAMES[s.type]}
            </Text.Text>
          </Flex.Box>
          <Button.Button
            onClick={() => void handleRemove(s)}
            size="small"
            variant="text"
            tooltip={`Remove ${s.name}`}
            aria-label={`Remove ${s.name}`}
            reveal
          >
            <Icon.Close />
          </Button.Button>
        </Flex.Box>
      ))}
    </Flex.Box>
  );
};
