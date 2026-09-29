// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type library } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { stopPropagation } from "@synnaxlabs/lyra/util";
import { uuid } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { useNestedStatus } from "@/feature/library/editor/useNestedStatus";
import { CSS } from "@/platform/css";

const KIND_NAMES: Record<library.EntryType, string> = {
  enum: "Enum",
  message: "Message",
};

const KIND_ICONS: Record<library.EntryType, Icon.ReactElement> = {
  enum: <Icon.Constant />,
  message: <Icon.Binary />,
};

const uniqueName = (kind: library.EntryType, entries: library.Entry[]): string => {
  const names = new Set(entries.map(({ name }) => name));
  let i = entries.length + 1;
  while (names.has(`${KIND_NAMES[kind]} ${i}`)) i++;
  return `${KIND_NAMES[kind]} ${i}`;
};

const createEntry = (
  kind: library.EntryType,
  entries: library.Entry[],
): library.Entry => {
  const base = { key: uuid.create(), name: uniqueName(kind, entries) };
  if (kind === "enum") return { ...base, kind, values: [] };
  return { ...base, kind, format: "binary", fields: [], delimiter: "," };
};

interface ItemProps extends List.ItemRenderProps<string> {
  onRemove: (key: string) => void;
}

const Item = ({ onRemove, ...rest }: ItemProps): ReactElement | null => {
  const { itemKey } = rest;
  const path = `entries.${itemKey}`;
  const entry = Form.useFieldValue<library.Entry>(path, { optional: true });
  const status = useNestedStatus(path);
  if (entry == null) return null;
  return (
    <Select.Item {...rest} justify="between" align="center">
      <Text.Text overflow="ellipsis" gap="small">
        {KIND_ICONS[entry.kind]}
        {entry.name}
      </Text.Text>
      <Flex.Box x align="center" gap="tiny">
        {status != null && <Status.Indicator variant={status.variant} />}
        <Button.Button
          variant="text"
          size="small"
          tooltip={`Remove ${entry.name}`}
          onClick={(e) => {
            stopPropagation(e);
            onRemove(itemKey);
          }}
        >
          <Icon.Close />
        </Button.Button>
      </Flex.Box>
    </Select.Item>
  );
};

export interface EntriesProps {
  /** The key of the entry the editor shows. */
  selected: string | null;
  onSelect: (key: string | null) => void;
}

/** Lists the entries of the library, and adds and removes them. */
export const Entries = ({ selected, onSelect }: EntriesProps): ReactElement => {
  const entries = Form.useFieldList<string, library.Entry>("entries");
  const { data, push, remove } = entries;
  const handleAdd = (kind: library.EntryType) => {
    const entry = createEntry(kind, entries.value());
    push(entry);
    onSelect(entry.key);
  };
  const handleRemove = (key: string) => {
    const remaining = remove(key);
    if (key === selected) onSelect(remaining[0] ?? null);
  };
  return (
    <Flex.Box y empty className={CSS.BE("library-editor", "entries")}>
      <Select.Frame<string, library.Entry>
        data={data}
        value={selected ?? undefined}
        onChange={onSelect}
        allowNone={false}
      >
        <List.Scroll full="y">
          <List.Items<string, library.Entry>
            emptyContent={
              <Text.Text center level="small" color={9}>
                No entries
              </Text.Text>
            }
          >
            {({ key, ...p }) => <Item key={key} {...p} onRemove={handleRemove} />}
          </List.Items>
        </List.Scroll>
      </Select.Frame>
      <Flex.Box x gap="small" className={CSS.BE("library-editor", "add")}>
        <Button.Button variant="outlined" onClick={() => handleAdd("enum")} grow>
          <Icon.Add />
          Add enum
        </Button.Button>
        <Button.Button variant="outlined" onClick={() => handleAdd("message")} grow>
          <Icon.Add />
          Add message
        </Button.Button>
      </Flex.Box>
    </Flex.Box>
  );
};
