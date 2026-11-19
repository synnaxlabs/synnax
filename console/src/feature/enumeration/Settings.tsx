// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/enumeration/Settings.css";

import { enumeration, schematic } from "@synnaxlabs/client";
import { Badge } from "@synnaxlabs/lyra/badge";
import { Caret } from "@synnaxlabs/lyra/caret";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Header } from "@synnaxlabs/lyra/header";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Access, Enumeration, Flux } from "@synnaxlabs/pluto";
import { deep, uuid } from "@synnaxlabs/x";
import { createContext, type ReactElement, use, useState } from "react";

import { useDeleteModal } from "@/feature/enumeration/Delete";
import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Settings as PlatformSettings } from "@/platform/settings";

/** @returns whether a stored built-in differs from its shipped name or entries. */
const edited = (
  stored: enumeration.Enumeration,
  shipped: enumeration.Enumeration,
): boolean =>
  stored.name !== shipped.name || !deep.equal(stored.entries, shipped.entries);

/** The built-in enumerations as Synnax ships them, by key. */
const ShippedContext = createContext<Map<enumeration.Key, enumeration.Enumeration>>(
  new Map(),
);
ShippedContext.displayName = "Enumeration.Shipped";

const COLUMNS: List.Column[] = [
  { name: "Name", width: "1fr" },
  { name: "Entries", width: "12rem" },
  { name: "Used by", width: "20rem" },
];

const UsedBy = ({ enumeration: key }: { enumeration: enumeration.Key }) => {
  const res = Enumeration.useResultDependents({ key });
  const n = (res.data ?? []).filter(
    ({ type }) => type === schematic.TYPE_RESOURCE_ID.type,
  ).length;
  if (n === 0) return null;
  return (
    <Text.Text level="small" color={9}>
      <Icon.Schematic />
      {n} {n === 1 ? "schematic" : "schematics"}
    </Text.Text>
  );
};

interface ItemProps extends List.ItemProps<enumeration.Key> {
  enumeration: enumeration.Enumeration;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/** An enumeration's row, and its entries under it while it is open. */
const Item = ({
  enumeration: e,
  open,
  onOpenChange,
  ...props
}: ItemProps): ReactElement => {
  const builtin = use(ShippedContext).get(e.key);
  const canUpdate = Access.useUpdateGranted(enumeration.resourceID(e.key));
  const { update: rename } = Enumeration.useRename();
  const itemClass = PlatformSettings.useItemClass(e.key);
  return (
    <div className={CSS.BE("enumeration-settings", "record")}>
      <Select.Item<enumeration.Key>
        {...props}
        checkable
        align="center"
        aria-label={e.name}
        aria-expanded={open}
        onActivate={() => onOpenChange(!open)}
        className={CSS.cls(CSS.BE("enumeration-settings", "item"), itemClass)}
      >
        <List.Cells>
          <List.Cell>
            <Flex.Box x align="center" gap="small">
              <Caret.Animated enabled={open} enabledLoc="bottom" disabledLoc="right" />
              <Text.MaybeEditable
                id={List.itemNameID(e.key)}
                value={e.name}
                onChange={canUpdate && ((name) => rename({ key: e.key, name }))}
                allowDoubleClick={false}
                overflow="fade"
              />
              {builtin != null && <Badge.Badge>Built-in</Badge.Badge>}
              {builtin != null && edited(e, builtin) && (
                <Tooltip.Dialog location="bottom">
                  <Text.Text level="small">
                    This built-in enumeration differs from the one that Synnax ships.
                  </Text.Text>
                  <Badge.Badge status="warning">Edited</Badge.Badge>
                </Tooltip.Dialog>
              )}
            </Flex.Box>
          </List.Cell>
          <List.Cell>
            <Text.Text level="small" color={9}>
              {e.entries.length}
            </Text.Text>
          </List.Cell>
          <List.Cell>
            <UsedBy enumeration={e.key} />
          </List.Cell>
        </List.Cells>
      </Select.Item>
      {open && (
        <div className={CSS.BE("enumeration-settings", "entries")}>
          <Enumeration.Entries enumeration={e.key} />
        </div>
      )}
    </div>
  );
};

const CreateForm = (): ReactElement => {
  const { close } = PlatformSettings.useCreate();
  const { updateAsync, variant } = Enumeration.useCreate();
  const [name, setName] = useState("");
  const handleSave = (): void => {
    const trimmed = name.trim();
    if (trimmed === "" || variant === "loading") return;
    void (async () => {
      const created = await updateAsync({ key: uuid.create(), name: trimmed });
      if (created !== false) close();
    })();
  };
  const onKeyDown = PlatformSettings.useCreateKeyDown(handleSave);
  return (
    <PlatformSettings.CreateFrame>
      <Flex.Box x align="center" gap="medium">
        <Input.Text
          value={name}
          onChange={setName}
          placeholder="Enumeration name"
          aria-label="Enumeration name"
          autoFocus
          grow
          onKeyDown={onKeyDown}
        />
        <PlatformSettings.CreateActions onSave={handleSave} status={variant} />
      </Flex.Box>
    </PlatformSettings.CreateFrame>
  );
};

interface SectionMenuProps {
  enumeration: enumeration.Enumeration;
}

const SectionMenu = ({ enumeration: e }: SectionMenuProps): ReactElement => {
  const id = enumeration.resourceID(e.key);
  const canUpdate = Access.useUpdateGranted(id);
  const canDelete = Access.useDeleteGranted(id);
  const builtin = use(ShippedContext).get(e.key);
  const openDelete = useDeleteModal();
  const { update: restore } = Enumeration.useRestore();
  return (
    <ContextMenu.Menu>
      {canUpdate && (
        <ContextMenu.RenameItem onClick={() => Text.edit(List.itemNameID(e.key))} />
      )}
      {builtin != null && edited(e, builtin) && canUpdate && (
        <Menu.Item itemKey="restore" onClick={() => restore(builtin)}>
          <Icon.Reset />
          Restore default
        </Menu.Item>
      )}
      <Menu.Divider />
      {builtin == null && canDelete && (
        <ContextMenu.DeleteItem onClick={() => openDelete({ enumeration: e })} />
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const SettingsMenu = ({ keys }: Menu.ContextMenuMenuProps): ReactElement | null => {
  const { getItem } = List.useUtilContext<enumeration.Key, enumeration.Enumeration>();
  const e = keys[0] == null ? undefined : getItem?.(keys[0]);
  return e == null ? null : <SectionMenu enumeration={e} />;
};

const menu = Component.renderProp(SettingsMenu);

/**
 * Every enumeration, by name, with its entries under it. The entries edit in place, and
 * the menu renames, restores built-ins, or deletes other enumerations.
 */
export const Settings = (): ReactElement => {
  const list = Flux.useList((client) => client.enumerations, {
    name: "enumerations",
    query: {},
    pageSize: 1000,
  });
  const records = Enumeration.useResultList({});
  const builtins = Enumeration.useResultBuiltins({});
  const canCreate = Access.useCreateGranted(enumeration.TYPE_RESOURCE_ID);
  const [opened, setOpened] = useState<ReadonlySet<enumeration.Key>>(() => new Set());
  const byKey = new Map(
    (records.variant === "success" ? records.data : []).map((e) => [e.key, e]),
  );
  const shipped = new Map(
    (builtins.variant === "success" ? builtins.data : []).map((e) => [e.key, e]),
  );
  const sorted = list.data
    .flatMap((k) => {
      const e = byKey.get(k);
      return e == null ? [] : [e];
    })
    .sort((a, b) => a.name.localeCompare(b.name));
  const setOpen = (key: enumeration.Key, open: boolean): void =>
    setOpened((prev) => {
      const next = new Set(prev);
      if (open) next.add(key);
      else next.delete(key);
      return next;
    });
  return (
    <ShippedContext value={shipped}>
      <PlatformSettings.Table<enumeration.Key, enumeration.Enumeration>
        list={{ ...list, data: sorted.map((e) => e.key) }}
        columns={COLUMNS}
        noun="enumerations"
        menu={menu}
        createLabel={canCreate ? "Create enumeration" : undefined}
        create={<CreateForm />}
      >
        {({ key, ...p }) => {
          const e = byKey.get(p.itemKey);
          if (e == null) return null;
          return (
            <Item
              key={key}
              {...p}
              enumeration={e}
              open={opened.has(p.itemKey)}
              onOpenChange={(open) => setOpen(p.itemKey, open)}
            />
          );
        }}
      </PlatformSettings.Table>
    </ShippedContext>
  );
};
