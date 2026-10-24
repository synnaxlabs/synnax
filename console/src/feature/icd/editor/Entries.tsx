// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type icd } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { uuid } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { formatHex } from "@/feature/icd/editor/FormattedField";
import { useNestedStatus } from "@/feature/icd/editor/useNestedStatus";
import { Button as PlatformButton } from "@/platform/button";
import { ContextMenu as PlatformContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";

const KIND_NAMES: Record<icd.EntryType, string> = {
  enum: "Enum",
  message: "Message",
};

const KIND_ICONS: Record<icd.EntryType, Icon.ReactElement> = {
  enum: <Icon.Constant />,
  message: <Icon.Binary />,
};

const uniqueName = (kind: icd.EntryType, entries: icd.Entry[]): string => {
  const names = new Set(entries.map(({ name }) => name));
  let i = entries.length + 1;
  while (names.has(`${KIND_NAMES[kind]} ${i}`)) i++;
  return `${KIND_NAMES[kind]} ${i}`;
};

const createEntry = (kind: icd.EntryType, entries: icd.Entry[]): icd.Entry => {
  const base = { key: uuid.create(), name: uniqueName(kind, entries) };
  if (kind === "enum") return { ...base, kind, values: [] };
  return { ...base, kind, payload: { format: "binary", fields: [] } };
};

const count = (n: number, noun: string): string =>
  `${n} ${n === 1 ? noun : Text.plural(noun)}`;

const describeMatch = ({ payload }: icd.MessageEntry): string | null => {
  if (payload.format === "text") return payload.prefix === "" ? null : payload.prefix;
  const id = payload.identifier;
  if (id == null) return null;
  switch (id.type) {
    case "can":
      return `CAN ${formatHex(id.id)}`;
    case "field": {
      const field = payload.fields.find(({ key }) => key === id.field);
      return field == null ? null : `${field.name} = ${id.value}`;
    }
  }
};

/** @returns a one-line summary of what an entry holds. */
const describe = (entry: icd.Entry): string => {
  if (entry.kind === "enum") return count(entry.values.length, "value");
  const fields = count(entry.payload.fields.length, "field");
  const match = describeMatch(entry);
  return match == null ? fields : `${match} · ${fields}`;
};

const Item = (props: List.ItemRenderProps<string>): ReactElement | null => {
  const path = `entries.${props.itemKey}`;
  const entry = Form.useFieldValue<icd.Entry>(path, { optional: true });
  const status = useNestedStatus(path);
  if (entry == null) return null;
  return (
    <Select.Item {...props} justify="between" align="center" gap="small">
      <Flex.Box x align="center" gap="small" className={CSS.BE("icd-entry", "label")}>
        {KIND_ICONS[entry.kind]}
        <Flex.Box y gap="tiny" className={CSS.BE("icd-entry", "text")}>
          <Text.Text weight={500} overflow="ellipsis">
            {entry.name}
          </Text.Text>
          <Text.Text level="small" color={8} overflow="ellipsis">
            {describe(entry)}
          </Text.Text>
        </Flex.Box>
      </Flex.Box>
      {status != null && <Status.Indicator variant={status.variant} />}
    </Select.Item>
  );
};

interface ContextMenuProps extends Menu.ContextMenuMenuProps {
  onRemove: (keys: string[]) => void;
}

const ContextMenu = ({ keys, onRemove }: ContextMenuProps): ReactElement => {
  const { mode } = Form.useContext();
  return (
    <PlatformContextMenu.Menu>
      {mode !== "preview" && keys.length > 0 && (
        <>
          <PlatformContextMenu.RemoveItem onClick={() => onRemove(keys)} />
          <Menu.Divider />
        </>
      )}
      <PlatformContextMenu.ReloadConsoleItem />
    </PlatformContextMenu.Menu>
  );
};

const isEntries = (key: string): boolean => key === "entries";

export interface EntriesProps {
  /** The keys of the selected entries. The editor shows the first. */
  selected: string[];
  onSelect: (keys: string[]) => void;
}

/**
 * Lists the entries of the ICD and adds them. Shift selects a range and Cmd toggles an
 * entry. The context menu removes the selected entries. A problem with the list as a
 * whole, such as a removed message that a task uses, shows below it.
 */
export const Entries = ({ selected, onSelect }: EntriesProps): ReactElement => {
  const entries = Form.useFieldList<string, icd.Entry>("entries");
  const { data, push, remove } = entries;
  const menuProps = Menu.useContextMenu();
  const handleAdd = (kind: icd.EntryType) => {
    const entry = createEntry(kind, entries.value());
    push(entry);
    onSelect([entry.key]);
  };
  const handleRemove = (keys: string[]) => onSelect(remove(keys).slice(0, 1));
  const status = useNestedStatus("entries", isEntries);
  return (
    <Flex.Box y empty grow className={CSS.BE("icd-editor", "entries")}>
      <Menu.ContextMenu
        {...menuProps}
        menu={(p) => <ContextMenu {...p} onRemove={handleRemove} />}
      >
        <Select.Frame<string, icd.Entry>
          multiple
          data={data}
          value={selected}
          onChange={onSelect}
          replaceOnSingle
          allowNone={false}
        >
          <List.Scroll
            full="y"
            className={menuProps.className}
            onContextMenu={menuProps.open}
          >
            <List.Items<string, icd.Entry>
              emptyContent={
                <Text.Text center level="small" color={9}>
                  No entries
                </Text.Text>
              }
            >
              {({ key, ...p }) => <Item key={key} {...p} />}
            </List.Items>
          </List.Scroll>
        </Select.Frame>
      </Menu.ContextMenu>
      {status != null && (
        <Input.HelpText
          variant={status.variant}
          className={CSS.BE("icd-editor", "entries-status")}
        >
          {status.message}
        </Input.HelpText>
      )}
      <PlatformButton.CreateListItem size="small" onClick={() => handleAdd("message")}>
        Add message
      </PlatformButton.CreateListItem>
      <PlatformButton.CreateListItem size="small" onClick={() => handleAdd("enum")}>
        Add enum
      </PlatformButton.CreateListItem>
    </Flex.Box>
  );
};
