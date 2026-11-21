// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/enumeration/Settings.css";

import { enumeration } from "@synnaxlabs/client";
import { Badge } from "@synnaxlabs/lyra/badge";
import { Caret } from "@synnaxlabs/lyra/caret";
import { Component } from "@synnaxlabs/lyra/component";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Access, Enumeration, Flux } from "@synnaxlabs/pluto";
import { deep, uuid } from "@synnaxlabs/x";
import {
  createContext,
  type KeyboardEvent,
  type ReactElement,
  use,
  useState,
} from "react";

import { useDeleteModal } from "@/feature/enumeration/Delete";
import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
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
  { name: "Name", width: "minmax(24rem, 1fr)" },
  { name: "Entries", width: "12rem" },
  { name: "Used by", width: "14rem" },
];

/** Every enumeration with its uses, read in one request. */
const QUERY: Enumeration.ListQuery = { includeUses: true };

interface ItemProps extends List.ItemProps<enumeration.Key> {
  enumeration: enumeration.Enumeration;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Focuses the entries when they first show. */
  autoFocus: boolean;
  onEntriesFocus: () => void;
}

/** An enumeration in the table, with its entries under it while it is open. */
const Item = ({
  enumeration: e,
  open,
  onOpenChange,
  autoFocus,
  onEntriesFocus,
  ...props
}: ItemProps): ReactElement => {
  const builtin = use(ShippedContext).get(e.key);
  const canUpdate = Access.useUpdateGranted(enumeration.resourceID(e.key));
  const { update: rename } = Enumeration.useRename();
  const itemClass = PlatformSettings.useItemClass(e.key);
  const [element, setElement] = useState<HTMLElement | null>(null);
  const handleKeyDown = (ev: KeyboardEvent<HTMLElement>): void => {
    if (ev.target !== ev.currentTarget) return;
    if (ev.key === "ArrowRight") onOpenChange(true);
    else if (ev.key === "ArrowLeft") onOpenChange(false);
    else return;
    ev.preventDefault();
  };
  // No wrapper, so the checked items share the parent that the context menu searches.
  return (
    <>
      <Select.Item<enumeration.Key>
        {...props}
        ref={setElement}
        checkable
        align="center"
        aria-label={e.name}
        aria-expanded={open}
        tabIndex={0}
        onKeyDown={handleKeyDown}
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
            <PlatformSettings.UsedBy
              itemKey={e.key}
              kind="enumeration"
              uses={e.uses}
              anchor={element}
            />
          </List.Cell>
        </List.Cells>
      </Select.Item>
      {open && (
        // A right click in the grid keeps the browser's menu, not the settings menu.
        <div
          role="region"
          aria-label={`${e.name} entries`}
          className={CSS.BE("enumeration-settings", "entries")}
          onFocus={onEntriesFocus}
          onContextMenu={(ev) => ev.stopPropagation()}
        >
          <Enumeration.Entries enumeration={e.key} autoFocus={autoFocus} />
        </div>
      )}
    </>
  );
};

interface CreateFormProps {
  onCreated: (key: enumeration.Key) => void;
}

const CreateForm = ({ onCreated }: CreateFormProps): ReactElement => {
  const { close, created } = PlatformSettings.useCreate();
  const { updateAsync, variant } = Enumeration.useCreate();
  const [name, setName] = useState("");
  const handleSave = (): void => {
    const trimmed = name.trim();
    if (trimmed === "" || variant === "loading") return;
    const key = uuid.create();
    void (async () => {
      if ((await updateAsync({ key, name: trimmed })) === false) return;
      // The new enumeration lists at the top, so its entries show to take focus.
      created(key);
      close();
      onCreated(key);
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

/** The menu of the selected enumerations. Built-ins are never deleted. */
const SettingsMenu = ({ keys }: Menu.ContextMenuMenuProps): ReactElement => {
  const { getItem } = List.useUtilContext<enumeration.Key, enumeration.Enumeration>();
  const enumerations = getItem?.(keys) ?? [];
  const ids = enumeration.resourceID(keys);
  const canUpdate = Access.useUpdateGranted(ids);
  const canDelete = Access.useDeleteGranted(ids);
  const shipped = use(ShippedContext);
  const deletable = enumerations.filter((e) => shipped.get(e.key) == null);
  const openDelete = useDeleteModal();
  const confirm = Modals.useConfirmDelete({ type: "Enumeration" });
  const { update: remove } = Enumeration.useDelete({
    beforeUpdate: async () => await confirm(deletable),
  });
  const { update: restore } = Enumeration.useRestore();
  const single = enumerations.length === 1 ? enumerations[0] : undefined;
  const builtin = single == null ? undefined : shipped.get(single.key);
  // One enumeration gets the dialog that can replace it where it is in use.
  const handleDelete = (): void => {
    if (deletable.length === 1) openDelete({ enumeration: deletable[0] });
    else remove(deletable.map((e) => e.key));
  };
  return (
    <ContextMenu.Menu>
      {single != null && (
        <Menu.Item
          itemKey="usedBy"
          onClick={() => Dialog.open(PlatformSettings.usedByDialogID(single.key))}
        >
          <Icon.Link />
          Used by
        </Menu.Item>
      )}
      <Menu.Divider />
      {single != null && canUpdate && (
        <ContextMenu.RenameItem
          onClick={() => Text.edit(List.itemNameID(single.key))}
        />
      )}
      {single != null && builtin != null && edited(single, builtin) && canUpdate && (
        <Menu.Item itemKey="restore" onClick={() => restore(builtin)}>
          <Icon.Reset />
          Restore default
        </Menu.Item>
      )}
      <Menu.Divider />
      {deletable.length > 0 && canDelete && (
        <ContextMenu.DeleteItem onClick={handleDelete} />
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const menu = Component.renderProp(SettingsMenu);

/**
 * Every enumeration, by name, with its entry count and uses. A click shows its entries
 * under it to edit in place. The menu renames, restores built-ins, or deletes the rest.
 */
export const Settings = (): ReactElement => {
  const list = Flux.useList((client) => client.enumerations, {
    name: "enumerations",
    query: QUERY,
    pageSize: 1000,
  });
  const records = Enumeration.useResultList(QUERY);
  const builtins = Enumeration.useResultBuiltins({});
  const canCreate = Access.useCreateGranted(enumeration.TYPE_RESOURCE_ID);
  const [opened, setOpened] = useState<ReadonlySet<enumeration.Key>>(() => new Set());
  // The enumeration just created here, whose entries take focus when they first show.
  // Focus in any entries or a search clears it, so a later mount takes no focus.
  const [created, setCreated] = useState<enumeration.Key>();
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
  const setOpen = (key: enumeration.Key, open: boolean): void => {
    setCreated(undefined);
    setOpened((prev) => {
      const next = new Set(prev);
      if (open) next.add(key);
      else next.delete(key);
      return next;
    });
  };
  const handleCreated = (key: enumeration.Key): void => {
    setOpen(key, true);
    setCreated(key);
  };
  const handleSearch = (term: string): void => {
    setCreated(undefined);
    list.search(term);
  };
  return (
    <ShippedContext value={shipped}>
      <PlatformSettings.Table<enumeration.Key, enumeration.Enumeration>
        list={{ ...list, search: handleSearch, data: sorted.map((e) => e.key) }}
        columns={COLUMNS}
        noun="enumerations"
        menu={menu}
        createLabel={canCreate ? "Create enumeration" : undefined}
        create={<CreateForm onCreated={handleCreated} />}
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
              autoFocus={created === p.itemKey}
              onEntriesFocus={() => setCreated(undefined)}
            />
          );
        }}
      </PlatformSettings.Table>
    </ShippedContext>
  );
};
