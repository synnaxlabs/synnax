// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color, label, UnexpectedError } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Color, Flux, Label } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Settings as PlatformSettings } from "@/platform/settings";

const COLUMNS: List.Column[] = [{ name: "Name", width: "minmax(24rem, 1fr)" }];

const path = Label.createFormPath;

const Item = (props: List.ItemProps<label.Key>): ReactElement => {
  const { itemKey } = props;
  const { form } = Label.useForm({ query: { key: itemKey } });
  const name = Form.useField<string>(path.name, { ctx: form });
  const swatch = Form.useField<color.Ref>(path.color, { ctx: form });
  const canUpdate = Access.useUpdateGranted(label.resourceID(itemKey));
  const itemClass = PlatformSettings.useItemClass(itemKey);
  return (
    <Select.Item<label.Key>
      {...props}
      checkable
      align="center"
      aria-label={name.value}
      className={CSS.cls(CSS.BE("label-settings", "item"), itemClass)}
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

const CreateLabel = (): ReactElement => {
  const { created } = PlatformSettings.useCreate();
  const { form, save, variant } = Label.useCreateForm({
    query: null,
    afterSave: ({ value }) => {
      const { key } = value();
      if (key == null) throw new UnexpectedError("Label key is null");
      created(key);
    },
  });
  const handleSave = (): void => {
    if (variant !== "loading") save();
  };
  const onKeyDown = PlatformSettings.useCreateKeyDown(handleSave);
  return (
    <PlatformSettings.CreateFrame>
      <List.Cell>
        <Form.Form<typeof Label.createFormSchema> {...form}>
          <Flex.Box x align="center" gap="medium">
            <Form.Field path={path.color} showLabel={false} padHelpText={false}>
              {(p) => <Color.Swatch aria-label="Label color" size="small" {...p} />}
            </Form.Field>
            <Form.Field path={path.name} showLabel={false} padHelpText={false} grow>
              {(p) => (
                <Input.Text
                  placeholder="Label name"
                  aria-label="Label name"
                  autoFocus
                  full="x"
                  onKeyDown={onKeyDown}
                  {...p}
                />
              )}
            </Form.Field>
            <PlatformSettings.CreateActions onSave={handleSave} status={variant} />
          </Flex.Box>
        </Form.Form>
      </List.Cell>
    </PlatformSettings.CreateFrame>
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

/**
 * Every label, with its color. Each line renames and recolors its label in place, and
 * the menu of the selected labels renames or deletes them.
 */
export const Settings = (): ReactElement => {
  const list = Flux.useList((client) => client.labels, {
    name: "labels",
    query: { order: { field: "name" } },
    pageSize: 100,
  });
  const canCreate = Access.useCreateGranted(label.TYPE_RESOURCE_ID);
  return (
    <PlatformSettings.Table<label.Key, label.Label>
      list={list}
      columns={COLUMNS}
      noun="labels"
      menu={menu}
      createLabel={canCreate ? "Create label" : undefined}
      create={<CreateLabel />}
    >
      {({ key, ...p }) => <Item key={key} {...p} />}
    </PlatformSettings.Table>
  );
};
