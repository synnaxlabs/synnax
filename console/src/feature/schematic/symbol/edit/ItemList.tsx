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
import { type deep, strings } from "@synnaxlabs/x";
import pluralize from "pluralize";
import type { ReactElement } from "react";

import type { Spec } from "@/feature/schematic/symbol/spec";
import { Modals } from "@/platform/modals";

const path = Schematic.Symbol.formPath;

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

interface Item {
  key: string;
  name: string;
}

export interface ItemListProps<T extends Item> {
  /** title names the items, such as States. */
  title: string;
  /** noun names one item in its tooltips, such as state. */
  noun: string;
  items: T[];
  /** name returns the path of the name of an item. */
  name: (key: string) => deep.Target<string>;
  /** detail returns the second line of an item. */
  detail: (item: T) => string;
  /** add returns spec with a new item, and the key of the item. */
  add: (spec: schematic.symbol.Spec) => [schematic.symbol.Spec, string];
  /** remove returns spec without an item, and what goes with it. */
  remove: (
    spec: schematic.symbol.Spec,
    key: string,
  ) => [schematic.symbol.Spec, Spec.Removed];
}

/**
 * ItemList lists items of the symbol, and adds, renames, and removes them. A remove
 * that takes other items with it asks the user first.
 */
export const ItemList = <T extends Item>({
  title,
  noun,
  items,
  name,
  detail,
  add,
  remove,
}: ItemListProps<T>): ReactElement => {
  const { get, set } = Form.useContext();
  const confirm = Modals.useConfirm();
  const handleRemove = async (item: T) => {
    const [next, removed] = remove(get(path.data).value, item.key);
    const uses = describeUses(removed);
    if (
      uses.length > 0 &&
      (await confirm({
        message: `Remove ${item.name}?`,
        description: `Its uses go with it: ${uses}.`,
        confirm: { label: "Remove", variant: "error" },
      })) !== true
    )
      return;
    set(path.data, next);
  };
  const nameID = (key: string) => `symbol-${noun}-${key}`;
  const handleAdd = () => {
    const [next, key] = add(get(path.data).value);
    set(path.data, next);
    Text.edit(nameID(key));
  };
  return (
    <Flex.Box y gap={1}>
      <Header.Header level="p" bordered={false}>
        <Header.Title level="p" weight={500}>
          {title}
        </Header.Title>
        <Header.Actions>
          <Button.Button
            onClick={handleAdd}
            size="small"
            variant="text"
            tooltip={`Add ${noun}`}
          >
            <Icon.Add />
          </Button.Button>
        </Header.Actions>
      </Header.Header>
      {items.map((item) => (
        <Flex.Box key={item.key} x justify="between" align="center">
          <Flex.Box y gap={0}>
            <Form.Field path={name(item.key)} showLabel={false} padHelpText={false}>
              {({ onChange, value }) => (
                <Text.MaybeEditable
                  id={nameID(item.key)}
                  level="small"
                  value={value}
                  onChange={onChange}
                />
              )}
            </Form.Field>
            <Text.Text level="small" color={9}>
              {detail(item)}
            </Text.Text>
          </Flex.Box>
          <Button.Button
            onClick={() => void handleRemove(item)}
            size="small"
            variant="text"
            tooltip={`Remove ${item.name}`}
            aria-label={`Remove ${item.name}`}
            reveal
          >
            <Icon.Close />
          </Button.Button>
        </Flex.Box>
      ))}
    </Flex.Box>
  );
};
