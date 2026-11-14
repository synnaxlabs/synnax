// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/label/Settings.css";

import { label } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Color } from "@synnaxlabs/lyra/color";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Flux, Label } from "@synnaxlabs/pluto";
import { type color } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Label as PlatformLabel } from "@/platform/label";
import { Modals } from "@/platform/modals";
import { Settings as PlatformSettings } from "@/platform/settings";

const COLUMNS: List.Column[] = [{ name: "Name", width: "1fr" }];

const path = Label.createFormPath;

const Item = (props: List.ItemProps<label.Key>): ReactElement => {
  const { itemKey } = props;
  const { form } = Label.useForm({ query: { key: itemKey } });
  const name = Form.useField<string>(path.name, { ctx: form });
  const swatch = Form.useField<color.Crude, color.Color>(path.color, { ctx: form });
  const canUpdate = Access.useUpdateGranted(label.resourceID(itemKey));
  return (
    <Select.Item<label.Key>
      {...props}
      checkable
      align="center"
      aria-label={name.value}
      className={CSS.BE("label-settings", "item")}
    >
      <List.Cells>
        <List.Cell>
          <Flex.Box x align="center" gap="medium">
            <Color.Swatch
              value={swatch.value}
              onChange={canUpdate ? swatch.onChange : undefined}
              size="small"
              aria-label="Color"
            />
            <Text.MaybeEditable
              id={List.itemNameID(itemKey)}
              value={name.value}
              onChange={canUpdate && name.onChange}
              allowDoubleClick={false}
              overflow="fade"
            />
          </Flex.Box>
        </List.Cell>
      </List.Cells>
    </Select.Item>
  );
};

const LabelsMenu = ({ keys }: Menu.ContextMenuMenuProps): ReactElement => {
  const { getItem } = List.useUtilContext<label.Key, label.Label>();
  const labels = getItem?.(keys) ?? [];
  const ids = label.resourceID(keys);
  const canUpdate = Access.useUpdateGranted(ids);
  const canDelete = Access.useDeleteGranted(ids);
  const confirm = Modals.useConfirmDelete({ type: "Label" });
  const { update: remove } = Label.useDelete({
    beforeUpdate: async () => await confirm(labels),
  });
  const single = labels.length === 1 ? labels[0] : undefined;
  return (
    <ContextMenu.Menu>
      {single != null && canUpdate && (
        <ContextMenu.RenameItem
          onClick={() => Text.edit(List.itemNameID(single.key))}
        />
      )}
      <Menu.Divider />
      {labels.length > 0 && canDelete && (
        <ContextMenu.DeleteItem onClick={() => remove(labels.map((l) => l.key))} />
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const menu = Component.renderProp(LabelsMenu);

const EMPTY_CONTENT = (
  <Flex.Box center className={CSS.BE("label-settings", "empty")}>
    <Text.Text color={9}>No labels</Text.Text>
  </Flex.Box>
);

/**
 * Every label, with its color. Each line renames and recolors its label in place, and
 * the menu of the selected labels renames or deletes them.
 */
export const Settings = (): ReactElement => {
  const { data, getItem, subscribe, more, search, answered } = Flux.useList(
    (client) => client.labels,
    { name: "labels", query: { order: { field: "name" } }, pageSize: 100 },
  );
  const [term, setTerm] = useState("");
  const [selected, setSelected] = useState<label.Key[]>([]);
  const menuProps = Menu.useContextMenu();
  const create = PlatformLabel.useCreateModal();
  const canCreate = Access.useCreateGranted(label.TYPE_RESOURCE_ID);
  const handleSearch = (next: string): void => {
    setTerm(next);
    search(next);
  };
  return (
    <Select.Frame<label.Key, label.Label>
      multiple
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={more}
      value={selected}
      onChange={setSelected}
    >
      <Menu.ContextMenu menu={menu} {...menuProps} />
      <Flex.Box y gap="medium">
        <Flex.Box x align="center" justify="between" gap="medium">
          <Input.Text
            value={term}
            onChange={handleSearch}
            placeholder="Search labels"
            startContent={<Icon.Search />}
            className={CSS.BE("label-settings", "search")}
          />
          {canCreate && (
            <Button.Button variant="outlined" onClick={() => void create()}>
              <Icon.Add />
              Create label
            </Button.Button>
          )}
        </Flex.Box>
        <PlatformSettings.Card flush>
          <List.Columns columns={COLUMNS}>
            <List.Headings checkable />
            <List.Scroll onContextMenu={menuProps.open}>
              <List.Items<label.Key> emptyContent={answered && EMPTY_CONTENT}>
                {({ key, ...p }) => <Item key={key} {...p} />}
              </List.Items>
            </List.Scroll>
          </List.Columns>
        </PlatformSettings.Card>
      </Flex.Box>
    </Select.Frame>
  );
};
