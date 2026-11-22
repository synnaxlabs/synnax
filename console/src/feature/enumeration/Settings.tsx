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
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Header } from "@synnaxlabs/lyra/header";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Access, Enumeration, Flux } from "@synnaxlabs/pluto";
import { deep, uuid } from "@synnaxlabs/x";
import { createContext, type ReactElement, use, useState } from "react";

import { useDeleteModal } from "@/feature/enumeration/Delete";
import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Settings as PlatformSettings } from "@/platform/settings";

const headingNameID = (key: enumeration.Key): string => List.itemNameID(key);

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

const HeadingTitle = ({ enumeration: e }: { enumeration: enumeration.Enumeration }) => {
  const canUpdate = Access.useUpdateGranted(enumeration.resourceID(e.key));
  const { update } = Enumeration.useRename();
  return (
    // A span takes the size and weight of the heading's label.
    <Text.MaybeEditable
      id={headingNameID(e.key)}
      defaultEl="span"
      value={e.name}
      onChange={canUpdate && ((name) => update({ key: e.key, name }))}
      allowDoubleClick={false}
      overflow="fade"
    />
  );
};

interface SectionProps {
  enumeration: enumeration.Enumeration;
  first: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/** An enumeration's heading, and its entries under it while it is open. */
const Section = ({
  enumeration: e,
  first,
  open,
  onOpenChange,
}: SectionProps): ReactElement => {
  const builtin = use(ShippedContext).get(e.key);
  return (
    <section
      aria-labelledby={headingNameID(e.key)}
      className={CSS.cls(
        CSS.BE("enumeration-settings", "section"),
        first && CSS.BEM("enumeration-settings", "section", "first"),
        !open && CSS.BEM("enumeration-settings", "section", "closed"),
      )}
    >
      <div
        className={CSS.cls(
          CSS.BE("enumeration-settings", "heading"),
          Menu.CONTEXT_TARGET,
        )}
        data-menu-key={e.key}
      >
        <Header.Group
          size="large"
          open={open}
          onOpenChange={onOpenChange}
          className={CSS.BE("enumeration-settings", "bar")}
        >
          <HeadingTitle enumeration={e} />
          <Text.Text level="tiny" color={9}>
            {e.entries.length}
          </Text.Text>
          {builtin != null && <Badge.Badge>Built-in</Badge.Badge>}
          {builtin != null && edited(e, builtin) && (
            <Tooltip.Dialog location="bottom">
              <Text.Text level="small">
                This built-in enumeration differs from the one that Synnax ships.
              </Text.Text>
              <Badge.Badge status="warning">Edited</Badge.Badge>
            </Tooltip.Dialog>
          )}
        </Header.Group>
      </div>
      {open && (
        // A right click in the grid keeps the browser's menu, not the settings menu.
        <div
          className={CSS.BE("enumeration-settings", "entries")}
          onContextMenu={(ev) => ev.stopPropagation()}
        >
          <Enumeration.Entries enumeration={e.key} />
        </div>
      )}
    </section>
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
        <ContextMenu.RenameItem onClick={() => Text.edit(headingNameID(e.key))} />
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
  const [closed, setClosed] = useState<ReadonlySet<enumeration.Key>>(() => new Set());
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
  const first = sorted[0]?.key;
  const setOpen = (key: enumeration.Key, open: boolean): void =>
    setClosed((prev) => {
      const next = new Set(prev);
      if (open) next.delete(key);
      else next.add(key);
      return next;
    });
  return (
    <ShippedContext value={shipped}>
      <PlatformSettings.Table<enumeration.Key, enumeration.Enumeration>
        list={{ ...list, data: sorted.map((e) => e.key) }}
        noun="enumerations"
        menu={menu}
        createLabel={canCreate ? "Create enumeration" : undefined}
        create={<CreateForm />}
      >
        {({ key, itemKey }) => {
          const e = byKey.get(itemKey);
          if (e == null) return null;
          return (
            <Section
              key={key}
              enumeration={e}
              first={itemKey === first}
              open={!closed.has(itemKey)}
              onOpenChange={(open) => setOpen(itemKey, open)}
            />
          );
        }}
      </PlatformSettings.Table>
    </ShippedContext>
  );
};
