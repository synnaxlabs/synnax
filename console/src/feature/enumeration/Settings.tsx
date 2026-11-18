// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/enumeration/Settings.css";

import { type color, enumeration } from "@synnaxlabs/client";
import { Badge } from "@synnaxlabs/lyra/badge";
import { Button } from "@synnaxlabs/lyra/button";
import { Color } from "@synnaxlabs/lyra/color";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Header } from "@synnaxlabs/lyra/header";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { theme } from "@synnaxlabs/lyra/theme";
import { Theming } from "@synnaxlabs/lyra/theming";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Access, Color as PColor, Enumeration, Flux } from "@synnaxlabs/pluto";
import { color as xcolor, deep, id, uuid } from "@synnaxlabs/x";
import {
  createContext,
  type DragEvent,
  type KeyboardEvent,
  type ReactElement,
  use,
  useState,
} from "react";

import { check, type ResolveColor } from "@/feature/enumeration/check";
import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Settings as PlatformSettings } from "@/platform/settings";

const COLUMNS: List.Column[] = [
  { name: "Value", width: "1fr" },
  { name: "Name", width: "2fr" },
  { name: "Color", width: "1fr" },
  { name: "Abnormal", width: "1fr" },
];

const HEADING_PREFIX = "enumeration:";
const NEW_PREFIX = "new:";

/** The item key of an entry: the key of its enumeration, a slash, and its own key. */
const entryItemKey = (e: enumeration.Key, entry: string): string => `${e}/${entry}`;

/** Splits an entry item key. A UUID holds no slash, so the first one divides them. */
const parseEntryItemKey = (key: string): [enumeration.Key, string] => {
  const i = key.indexOf("/");
  return [key.slice(0, i), key.slice(i + 1)];
};

const headingNameID = (key: enumeration.Key): string =>
  List.itemNameID(`${HEADING_PREFIX}${key}`);

/** @returns whether a stored built-in differs from its shipped name or entries. */
const edited = (
  stored: enumeration.Enumeration,
  shipped: enumeration.Enumeration,
): boolean =>
  stored.name !== shipped.name || !deep.equal(stored.entries, shipped.entries);

interface ContextValue {
  shipped: Map<enumeration.Key, enumeration.Enumeration>;
  resolve: ResolveColor;
  /** The named colors, by key. */
  named: Map<color.Key, color.Named>;
}

const Context = createContext<ContextValue>({
  shipped: new Map(),
  resolve: () => undefined,
  named: new Map(),
});
Context.displayName = "Enumeration.Settings";

const HAUL_TYPE = "enumeration_entry";

interface HeadingProps {
  enumeration: enumeration.Enumeration;
  first: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Opens a form for a new entry at the end of the enumeration. */
  onCreate: () => void;
}

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

const Heading = ({
  enumeration: e,
  first,
  open,
  onOpenChange,
  onCreate,
}: HeadingProps): ReactElement => {
  const { shipped } = use(Context);
  const canUpdate = Access.useUpdateGranted(enumeration.resourceID(e.key));
  const builtin = shipped.get(e.key);
  return (
    <div
      className={CSS.cls(
        CSS.BE("enumeration-settings", "heading"),
        first && CSS.BEM("enumeration-settings", "heading", "first"),
        !open && CSS.BEM("enumeration-settings", "heading", "closed"),
        Menu.CONTEXT_TARGET,
      )}
      data-menu-key={`${HEADING_PREFIX}${e.key}`}
    >
      <Header.Group
        size="large"
        open={open}
        onOpenChange={onOpenChange}
        className={CSS.BE("enumeration-settings", "bar")}
        actions={
          !canUpdate ? undefined : (
            <Button.Button
              variant="text"
              size="small"
              textColor={9}
              tooltip="Add entry"
              aria-label={`Add an entry to ${e.name}`}
              onClick={onCreate}
            >
              <Icon.Add />
            </Button.Button>
          )
        }
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
  );
};

interface EntryColorProps {
  value?: color.Ref;
  onChange?: (value?: color.Ref) => void;
}

interface ShownColor {
  /** The named color that the entry names, if any. */
  named?: color.Named;
  /** The color the entry shows, or undefined when it shows Auto. */
  shown?: xcolor.Color;
}

const useShownColor = (value?: color.Ref): ShownColor => {
  const { named } = use(Context);
  const t = Theming.use();
  const n = value?.type === "named" ? named.get(value.key) : undefined;
  return {
    named: n,
    shown: value?.type === "fixed" ? value.value : n && theme.resolve(n, t),
  };
};

const EntryColor = ({ value, onChange }: EntryColorProps): ReactElement => {
  const t = Theming.use();
  const { named: n, shown } = useShownColor(value);
  return (
    <Flex.Box x align="center" gap="small">
      <Color.Swatch
        value={shown}
        fallback={t.colors.text}
        onChange={
          onChange == null
            ? undefined
            : (c) => onChange(c == null ? undefined : { type: "fixed", value: c })
        }
        size="small"
        aria-label="Entry color"
      />
      <Text.Text level="small" variant="code" color={shown == null ? 8 : 10}>
        {shown == null
          ? "Auto"
          : value?.type === "named"
            ? n?.name
            : xcolor.hex(shown).toUpperCase()}
      </Text.Text>
    </Flex.Box>
  );
};

const path = Enumeration.formPath;

interface WarningsProps {
  warnings: string[];
}

const Warnings = ({ warnings }: WarningsProps): ReactElement => (
  <Tooltip.Dialog location="bottom">
    <Flex.Box y gap="tiny">
      {warnings.map((w) => (
        <Text.Text key={w} level="small">
          {w}
        </Text.Text>
      ))}
    </Flex.Box>
    <Icon.Warning
      className={CSS.BE("enumeration-settings", "warning")}
      aria-label={warnings.join(" ")}
    />
  </Tooltip.Dialog>
);

const OWN_VALUE = "Each entry needs its own value.";

/** Returns a check for a value that another entry holds, which warns when one does. */
const useValueTaken = (): ((
  entries: enumeration.Entry[],
  name: string,
  value: number,
  except?: string,
) => boolean) => {
  const addStatus = Status.useAdder();
  return (entries, name, value, except) => {
    const holder = entries.find((e) => e.key !== except && e.value === value);
    if (holder == null) return false;
    const holderName = holder.name.trim() === "" ? "Another entry" : holder.name;
    addStatus({
      variant: "warning",
      name,
      message: "Value in use",
      description: `${holderName} has the value ${value}. ${OWN_VALUE}`,
    });
    return true;
  };
};

const Entry = (props: List.ItemProps<string>): ReactElement | null => {
  const { itemKey } = props;
  const [key, entryKey] = parseEntryItemKey(itemKey);
  const { resolve } = use(Context);
  const { form } = Enumeration.useForm({ query: { key } });
  const entries = Form.useFieldValue<enumeration.Entry[]>(path.entries, { ctx: form });
  const canUpdate = Access.useUpdateGranted(enumeration.resourceID(key));
  const itemClass = PlatformSettings.useItemClass(itemKey);
  const valueTaken = useValueTaken();
  const [over, setOver] = useState(false);
  const entry = entries.find((e) => e.key === entryKey);
  const { shown } = useShownColor(entry?.color);
  const { startDrag, onDragEnd, onDragOver, onDrop } = Haul.useDragAndDrop({
    type: HAUL_TYPE,
    key: itemKey,
    canDrop: ({ items }) =>
      items.some(
        (item) =>
          item.type === HAUL_TYPE && parseEntryItemKey(String(item.key))[0] === key,
      ),
    onDragOver: () => setOver(true),
    onDrop: ({ items }) => {
      setOver(false);
      const dragged = items.find(({ type }) => type === HAUL_TYPE);
      if (dragged == null || dragged.key === itemKey) return items;
      const draggedEntry = parseEntryItemKey(String(dragged.key))[1];
      const others = entries.filter((en) => en.key !== draggedEntry);
      const i = others.findIndex((en) => en.key === entryKey);
      form.move(path.entries, draggedEntry, i > 0 ? others[i - 1].key : undefined);
      return items;
    },
  });
  if (entry == null) return null;
  const at = path.entries[entryKey];
  const warnings = check(entry, entries, resolve);
  const handleDragStart = (event: DragEvent): void =>
    startDrag({
      event,
      items: [{ type: HAUL_TYPE, key: itemKey }],
      preview: (
        <Haul.Preview
          icon={<Icon.Enumeration />}
          swatch={shown == null ? undefined : xcolor.cssString(shown)}
          name={entry.name}
          facts={[`Value ${entry.value}`]}
        />
      ),
    });
  return (
    <Select.Item<string>
      {...props}
      checkable
      align="center"
      aria-label={entry.name}
      draggable={canUpdate}
      onDragStart={handleDragStart}
      onDragEnd={onDragEnd}
      onDragOver={onDragOver}
      onDragLeave={() => setOver(false)}
      onDrop={onDrop}
      className={CSS.cls(
        CSS.BE("enumeration-settings", "item"),
        over && CSS.BEM("enumeration-settings", "item", "over"),
        itemClass,
      )}
    >
      <List.Cells>
        <List.Cell>
          <Input.Numeric
            value={entry.value}
            onChange={(v) => {
              if (!valueTaken(entries, entry.name, v, entryKey)) form.set(at.value, v);
            }}
            disabled={!canUpdate}
            showDragHandle={false}
            variant="text"
            size="small"
            aria-label="Value"
          />
        </List.Cell>
        <List.Cell>
          <Flex.Box x align="center" gap="medium">
            <Text.MaybeEditable
              id={List.itemNameID(itemKey)}
              value={entry.name}
              onChange={canUpdate && ((name) => form.set(at.name, name))}
              allowDoubleClick={false}
              overflow="fade"
            />
            {warnings.length > 0 && <Warnings warnings={warnings} />}
          </Flex.Box>
        </List.Cell>
        <List.Cell>
          <EntryColor
            value={entry.color}
            onChange={
              canUpdate
                ? (c) => (c == null ? form.remove(at.color) : form.set(at.color, c))
                : undefined
            }
          />
        </List.Cell>
        <List.Cell>
          <Input.Switch
            value={entry.abnormal}
            onChange={(v) => form.set(at.abnormal, v)}
            disabled={!canUpdate}
            aria-label="Abnormal"
          />
        </List.Cell>
      </List.Cells>
    </Select.Item>
  );
};

/** @returns the least whole number above every value of the entries, or 0. */
const nextValue = (entries: enumeration.Entry[]): number =>
  entries.length === 0 ? 0 : Math.max(...entries.map((e) => Math.floor(e.value))) + 1;

interface NewEntryProps {
  enumeration: enumeration.Key;
  onClose: () => void;
}

const NewEntry = ({ enumeration: key, onClose }: NewEntryProps): ReactElement => {
  const { form } = Enumeration.useForm({ query: { key } });
  const entries = Form.useFieldValue<enumeration.Entry[]>(path.entries, { ctx: form });
  const valueTaken = useValueTaken();
  const [name, setName] = useState("");
  const [value, setValue] = useState(() => nextValue(entries));
  const handleSave = (): void => {
    const trimmed = name.trim();
    if (trimmed === "" || valueTaken(entries, trimmed, value)) return;
    const entry = { key: id.create(), value, name: trimmed, abnormal: false };
    form.insert(path.entries, entry, entries.at(-1)?.key);
    setName("");
    setValue(nextValue([...entries, entry]));
  };
  const handleKeyDown = (e: KeyboardEvent): void => {
    if (e.key === "Enter") {
      e.preventDefault();
      handleSave();
    } else if (e.key === "Escape") {
      e.preventDefault();
      onClose();
    }
  };
  return (
    <div className={CSS.BE("enumeration-settings", "form")}>
      <span className={CSS.BE("enumeration-settings", "form-check")} />
      <List.Cells>
        <List.Cell>
          <Input.Numeric
            value={value}
            onChange={setValue}
            showDragHandle={false}
            size="small"
            aria-label="Entry value"
          />
        </List.Cell>
        <List.Cell>
          <Input.Text
            value={name}
            onChange={setName}
            placeholder="Entry name"
            aria-label="Entry name"
            autoFocus
            full="x"
            onKeyDown={handleKeyDown}
          />
        </List.Cell>
        <List.Cell>
          <Text.Text level="small" variant="code" color={8}>
            Auto
          </Text.Text>
        </List.Cell>
        <List.Cell>
          <Flex.Box x gap="small" justify="end">
            <Button.Button
              variant="filled"
              size="small"
              tooltip="Add"
              aria-label="Add"
              onClick={handleSave}
            >
              <Icon.Check />
            </Button.Button>
            <Button.Button
              variant="text"
              size="small"
              tooltip="Cancel"
              aria-label="Cancel"
              onClick={onClose}
            >
              <Icon.Close />
            </Button.Button>
          </Flex.Box>
        </List.Cell>
      </List.Cells>
    </div>
  );
};

interface CreateFormProps {
  /** Called with the key of the new enumeration once the Core holds it. */
  onCreated: (key: enumeration.Key) => void;
}

const CreateForm = ({ onCreated }: CreateFormProps): ReactElement => {
  const { close } = PlatformSettings.useCreate();
  const { updateAsync, variant } = Enumeration.useCreate();
  const [name, setName] = useState("");
  const handleSave = (): void => {
    const trimmed = name.trim();
    if (trimmed === "" || variant === "loading") return;
    const key = uuid.create();
    void (async () => {
      const created = await updateAsync({ key, name: trimmed });
      if (created === false) return;
      onCreated(key);
      close();
    })();
  };
  const onKeyDown = PlatformSettings.useCreateKeyDown(handleSave);
  return (
    <PlatformSettings.CreateFrame>
      <List.Cell />
      <List.Cell>
        <Input.Text
          value={name}
          onChange={setName}
          placeholder="Enumeration name"
          aria-label="Enumeration name"
          autoFocus
          full="x"
          onKeyDown={onKeyDown}
        />
      </List.Cell>
      <List.Cell />
      <List.Cell>
        <PlatformSettings.CreateActions onSave={handleSave} status={variant} />
      </List.Cell>
    </PlatformSettings.CreateFrame>
  );
};

interface HeadingMenuProps {
  enumeration: enumeration.Enumeration;
}

const HeadingMenu = ({ enumeration: e }: HeadingMenuProps): ReactElement => {
  const { shipped } = use(Context);
  const id = enumeration.resourceID(e.key);
  const canUpdate = Access.useUpdateGranted(id);
  const canDelete = Access.useDeleteGranted(id);
  const builtin = shipped.get(e.key);
  const confirm = Modals.useConfirmDelete({ type: "Enumeration", icon: "Enumeration" });
  const { update: remove } = Enumeration.useDelete({
    beforeUpdate: async () => await confirm({ name: e.name }),
  });
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
        <ContextMenu.DeleteItem onClick={() => remove(e.key)} />
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

interface EntriesMenuProps {
  keys: string[];
}

const EntriesMenu = ({ keys }: EntriesMenuProps): ReactElement => {
  const [first] = keys;
  const parsed = keys.map(parseEntryItemKey);
  const [key] = parsed[0];
  const { form } = Enumeration.useForm({ query: { key } });
  const canUpdate = Access.useUpdateGranted(enumeration.resourceID(key));
  // A selection across enumerations gets no edits, so none applies to only part of it.
  const editable = canUpdate && parsed.every(([k]) => k === key);
  const entries = parsed.map(([, entry]) => entry);
  return (
    <ContextMenu.Menu>
      {entries.length === 1 && editable && (
        <ContextMenu.RenameItem onClick={() => Text.edit(List.itemNameID(first))} />
      )}
      <Menu.Divider />
      {editable && (
        <ContextMenu.RemoveItem
          onClick={() =>
            form.batch(() =>
              entries.forEach((entry) => form.remove(path.entries[entry])),
            )
          }
        />
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const SettingsMenu = ({ keys }: Menu.ContextMenuMenuProps): ReactElement | null => {
  const { getItem } = List.useUtilContext<enumeration.Key, enumeration.Enumeration>();
  const [first] = keys;
  if (first == null) return null;
  if (first.startsWith(HEADING_PREFIX)) {
    const e = getItem?.(first.slice(HEADING_PREFIX.length));
    return e == null ? null : <HeadingMenu enumeration={e} />;
  }
  return <EntriesMenu keys={keys} />;
};

const menu = Component.renderProp(SettingsMenu);

/**
 * Every enumeration, by name, with its entries under it in order. Each line edits its
 * entry in place and warns when the entry is hard to read, a drag moves an entry, and
 * the menu restores built-ins or deletes other enumerations.
 */
export const Settings = (): ReactElement => {
  const list = Flux.useList((client) => client.enumerations, {
    name: "enumerations",
    query: {},
    pageSize: 1000,
  });
  const records = Enumeration.useResultList({});
  const builtins = Enumeration.useResultBuiltins({});
  const colors = PColor.useResultList({});
  const canCreate = Access.useCreateGranted(enumeration.TYPE_RESOURCE_ID);
  const [closed, setClosed] = useState<ReadonlySet<string>>(() => new Set());
  const [creating, setCreating] = useState<enumeration.Key | null>(null);
  const all = records.variant === "success" ? records.data : [];
  const byKey = new Map(all.map((e) => [e.key, e]));
  const shipped = new Map(
    (builtins.variant === "success" ? builtins.data : []).map((e) => [e.key, e]),
  );
  const named = new Map(
    (colors.variant === "success" ? colors.data : []).map((c) => [c.key, c]),
  );
  const resolve: ResolveColor = (entry) => {
    const ref = entry.color;
    if (ref == null) return undefined;
    if (ref.type === "fixed") return { light: ref.value };
    const n = named.get(ref.key);
    return n == null ? undefined : { light: n.light, dark: n.dark };
  };
  const sorted = list.data
    .flatMap((k) => {
      const e = byKey.get(k);
      return e == null ? [] : [e];
    })
    .sort((a, b) => a.name.localeCompare(b.name));
  const headings = new Map<string, HeadingProps>();
  const data: string[] = [];
  const setOpen = (key: string, open: boolean): void =>
    setClosed((prev) => {
      const next = new Set(prev);
      if (open) next.delete(key);
      else next.add(key);
      return next;
    });
  for (const e of sorted) {
    const heading = `${HEADING_PREFIX}${e.key}`;
    const open = !closed.has(e.key);
    headings.set(heading, {
      enumeration: e,
      first: data.length === 0,
      open,
      onOpenChange: (next) => setOpen(e.key, next),
      onCreate: () => {
        setOpen(e.key, true);
        setCreating(e.key);
      },
    });
    data.push(heading);
    if (!open) continue;
    data.push(...e.entries.map((entry) => entryItemKey(e.key, entry.key)));
    if (creating === e.key) data.push(`${NEW_PREFIX}${e.key}`);
  }
  return (
    <Context value={{ shipped, resolve, named }}>
      <PlatformSettings.Table<string, enumeration.Enumeration>
        list={{ ...list, data }}
        columns={COLUMNS}
        noun="enumerations"
        menu={menu}
        createLabel={canCreate ? "Create enumeration" : undefined}
        create={<CreateForm onCreated={setCreating} />}
      >
        {({ key, ...p }) => {
          const heading = headings.get(p.itemKey);
          if (heading != null) return <Heading key={key} {...heading} />;
          if (p.itemKey.startsWith(NEW_PREFIX))
            return (
              <NewEntry
                key={key}
                enumeration={p.itemKey.slice(NEW_PREFIX.length)}
                onClose={() => setCreating(null)}
              />
            );
          return <Entry key={key} {...p} />;
        }}
      </PlatformSettings.Table>
    </Context>
  );
};
