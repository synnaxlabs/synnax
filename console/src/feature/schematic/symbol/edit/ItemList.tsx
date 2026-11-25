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
import type { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Schematic } from "@synnaxlabs/pluto";
import { type deep, strings } from "@synnaxlabs/x";
import pluralize from "pluralize";
import type { ReactElement, ReactNode } from "react";

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
}

export interface ItemListProps<T extends Item> extends Input.Control<
  string | undefined
> {
  /** title names the items, such as States. */
  title: string;
  /** noun names one item in its tooltips, such as state. */
  noun: string;
  items: T[];
  /** label returns the name of an item. */
  label: (item: T) => string;
  /** name returns the path of the name of an item, which makes the name editable. */
  name?: (key: string) => deep.Target<string>;
  /**
   * removeLabel returns the label of the remove button and confirm of an item. It
   * defaults to Remove and the label.
   */
  removeLabel?: (item: T) => string;
  /** detail returns the second line of an item. */
  detail: (item: T) => ReactNode;
  /** add returns spec with a new item, and the key of the item. */
  add: (spec: schematic.symbol.Spec) => [schematic.symbol.Spec, string];
  /** addDisabled, when set, turns off the add button and says why. */
  addDisabled?: string;
  /** remove returns spec without an item, and what goes with it. */
  remove: (
    spec: schematic.symbol.Spec,
    key: string,
  ) => [schematic.symbol.Spec, Spec.Removed];
  /** onRemove is called after an item goes. */
  onRemove?: () => void;
}

/**
 * ItemList lists items of the symbol, picks one, and adds, renames, and removes them. A
 * remove that takes other items with it asks the user first.
 */
export const ItemList = <T extends Item>({
  title,
  noun,
  items,
  label,
  name,
  removeLabel,
  detail,
  add,
  addDisabled,
  remove,
  onRemove,
  value,
  onChange,
}: ItemListProps<T>): ReactElement => {
  const { get, set } = Form.useContext();
  const confirm = Modals.useConfirm();
  const removeText = (item: T) => removeLabel?.(item) ?? `Remove ${label(item)}`;
  const handleRemove = async (item: T) => {
    const [next, removed] = remove(get(path.data).value, item.key);
    const uses = describeUses(removed);
    if (
      uses.length > 0 &&
      (await confirm({
        message: `${removeText(item)}?`,
        description: `Its uses go with it: ${uses}.`,
        confirm: { label: "Remove", variant: "error" },
      })) !== true
    )
      return;
    set(path.data, next);
    onRemove?.();
  };
  const nameID = (key: string) => `symbol-${noun}-${key}`;
  const handleAdd = () => {
    const [next, key] = add(get(path.data).value);
    set(path.data, next);
    // The new item takes focus: its name opens for edit, or else the item is picked.
    if (name != null) Text.edit(nameID(key));
    else onChange(key);
  };
  const keys = items.map((item) => item.key);
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
            tooltip={addDisabled ?? `Add ${noun}`}
            disabled={addDisabled != null}
          >
            <Icon.Add />
          </Button.Button>
        </Header.Actions>
      </Header.Header>
      <Select.Frame
        value={value}
        onChange={onChange}
        data={keys}
        closeDialogOnSelect={false}
        cursor="selection"
      >
        <List.Scroll y gap={1}>
          <List.Items<string>>
            {({ key, itemKey, ...rest }) => {
              const item = items.find((i) => i.key === itemKey);
              if (item == null) return null;
              return (
                <Select.Item key={key} itemKey={itemKey} {...rest} justify="between">
                  <Flex.Box y gap={0}>
                    {name == null ? (
                      <Text.Text level="small">{label(item)}</Text.Text>
                    ) : (
                      <Form.Field
                        path={name(item.key)}
                        showLabel={false}
                        padHelpText={false}
                      >
                        {({ onChange: onRename, value: itemName }) => (
                          <Text.MaybeEditable
                            id={nameID(item.key)}
                            level="small"
                            value={itemName}
                            onChange={onRename}
                          />
                        )}
                      </Form.Field>
                    )}
                    <Text.Text level="small" color={9}>
                      {detail(item)}
                    </Text.Text>
                  </Flex.Box>
                  <Button.Button
                    onClick={(e) => {
                      e.stopPropagation();
                      void handleRemove(item);
                    }}
                    size="small"
                    variant="text"
                    tooltip={removeText(item)}
                    aria-label={removeText(item)}
                    reveal
                  >
                    <Icon.Close />
                  </Button.Button>
                </Select.Item>
              );
            }}
          </List.Items>
        </List.Scroll>
      </Select.Frame>
    </Flex.Box>
  );
};
