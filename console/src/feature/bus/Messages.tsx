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
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Library } from "@synnaxlabs/pluto";
import {
  type ReactElement,
  type ReactNode,
  useCallback,
  useMemo,
  useState,
} from "react";

import { type Accepts, type Message, messagesOf } from "@/feature/bus/types";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { Task } from "@/platform/task";

const MESSAGES_PATH = "config.messages";

/**
 * @returns the message entries of the library the form's config names, keyed by entry
 * key, and the error that kept them from loading.
 */
const useLibraryMessages = (): {
  messages: Map<library.EntryKey, library.MessageEntry>;
  error?: string;
} => {
  const key = Form.useFieldValue<library.Key>("config.library", { optional: true });
  const { data, variant, status } = Library.useResult(key == null ? null : { key });
  const messages = useMemo(() => messagesOf(data), [data]);
  return { messages, error: variant === "error" ? status.message : undefined };
};

/** @returns a short label for how a message is matched on its transport. */
const describeMatch = ({ payload }: library.MessageEntry): string => {
  if (payload.format === "text") return payload.prefix;
  const id = payload.identifier;
  if (id == null) return "";
  switch (id.type) {
    case "can":
      return `0x${id.id.toString(16).toUpperCase()}${id.extended ? " ext" : ""}`;
    case "field":
      return `= ${id.value}`;
  }
};

interface MessageItemProps extends List.ItemRenderProps<library.EntryKey> {
  entries: Map<library.EntryKey, library.MessageEntry>;
  onRemove: (key: library.EntryKey) => void;
}

const MessageItem = ({ entries, onRemove, ...props }: MessageItemProps) => {
  const { itemKey, index } = props;
  const isPreview = Task.useIsPreview();
  const entry = entries.get(itemKey);
  return (
    <Select.Item {...props} justify="between" align="center" x>
      <Flex.Box y gap="tiny" className={CSS.BE("bus-message", "label")}>
        <Text.Text weight={500} overflow="ellipsis">
          {entry?.name ?? "Unknown message"}
        </Text.Text>
        <Text.Text level="small" color={8}>
          {entry == null ? "" : describeMatch(entry)}
        </Text.Text>
      </Flex.Box>
      <Flex.Box x align="center" gap="tiny">
        <Task.EnabledCheckbox path={`${MESSAGES_PATH}.${index}.disabled`} />
        {!isPreview && (
          <Button.Button
            variant="text"
            size="small"
            aria-label="Remove message"
            tooltip="Remove"
            onClick={(e) => {
              e.stopPropagation();
              onRemove(itemKey);
            }}
          >
            <Icon.Close />
          </Button.Button>
        )}
      </Flex.Box>
    </Select.Item>
  );
};

interface AddMessageProps {
  entries: library.MessageEntry[];
  onAdd: (entry: library.MessageEntry) => void;
}

const ADD_TRIGGER_PROPS = { placeholder: "Add message", icon: <Icon.Add /> } as const;

/** Picks a library message to add. Carries the status of the messages list. */
const AddMessage = ({ entries, onAdd }: AddMessageProps) => {
  const byKey = useMemo(() => new Map(entries.map((e) => [e.key, e])), [entries]);
  return (
    <Form.Field<Message[]>
      path={MESSAGES_PATH}
      showLabel={false}
      padHelpText={false}
      className={CSS.BE("bus-messages", "add")}
    >
      {({ preview }) =>
        preview === true ? null : (
          <Select.Simple<library.EntryKey>
            resourceName="message"
            triggerProps={ADD_TRIGGER_PROPS}
            onChange={(key: library.EntryKey) => {
              const entry = byKey.get(key);
              if (entry != null) onAdd(entry);
            }}
            emptyContent={<Empty.Action message="No messages to add" />}
          >
            {entries.map((e) => (
              <Select.Item<library.EntryKey> key={e.key} itemKey={e.key}>
                {e.name}
              </Select.Item>
            ))}
          </Select.Simple>
        )
      }
    </Form.Field>
  );
};

/** Props the details of a selected message render with. */
export interface DetailsProps {
  /** The form path of the message in the task's config. */
  path: string;
  /** The library entry of the message. */
  entry: library.MessageEntry;
}

export interface MessagesProps {
  /** Whether the task's integration can carry a library message. */
  accepts: Accepts;
  /** Creates the config entry for a library message the user adds. */
  create: (entry: library.MessageEntry) => Message;
  /** Renders the fields of the selected message. */
  details: Component.RenderProp<DetailsProps>;
}

const EMPTY_CONTENT = <Empty.Action message="No messages" />;

/**
 * Lists the library messages a task binds beside the details of the selected one. The
 * picker offers the library's messages the integration can carry.
 */
export const Messages = ({ accepts, create, details }: MessagesProps): ReactElement => {
  const { messages: entries, error } = useLibraryMessages();
  const ctx = Form.useContext();
  const messages = Form.useFieldValue<Message[]>(MESSAGES_PATH);
  const keys = useMemo(() => messages.map((m) => m.message), [messages]);
  const [selected, setSelected] = useState<library.EntryKey | undefined>(keys[0]);
  const addable = useMemo(
    () => [...entries.values()].filter((e) => accepts(e) && !keys.includes(e.key)),
    [entries, keys, accepts],
  );
  const handleAdd = useCallback(
    (entry: library.MessageEntry) => {
      ctx.set(MESSAGES_PATH, [
        ...ctx.get<Message[]>(MESSAGES_PATH).value,
        create(entry),
      ]);
      setSelected(entry.key);
    },
    [ctx, create],
  );
  const handleRemove = useCallback(
    (key: library.EntryKey) => {
      const next = ctx
        .get<Message[]>(MESSAGES_PATH)
        .value.filter((m) => m.message !== key);
      ctx.set(MESSAGES_PATH, next);
      if (selected === key) setSelected(next[0]?.message);
    },
    [ctx, selected],
  );
  const renderItem = useMemo(
    () =>
      Component.renderProp((p: List.ItemRenderProps<library.EntryKey>) => (
        <MessageItem {...p} entries={entries} onRemove={handleRemove} />
      )),
    [entries, handleRemove],
  );
  const index = selected == null ? -1 : keys.indexOf(selected);
  const path = index === -1 ? null : `${MESSAGES_PATH}.${index}`;
  const entry = selected == null ? undefined : entries.get(selected);
  let content: ReactNode;
  if (error != null) content = <Placeholder>{error}</Placeholder>;
  else if (path == null)
    content = <Placeholder>Select a message to configure</Placeholder>;
  else if (entry == null) content = <Placeholder>Loading message</Placeholder>;
  else content = details({ path, entry });
  return (
    <Task.Views.Panes
      listTitle="Messages"
      list={
        <Flex.Box y empty grow className={CSS.B("bus-messages")}>
          <Select.Frame<library.EntryKey, undefined>
            data={keys}
            value={selected}
            onChange={setSelected}
            allowNone={false}
            autoSelectOnNone
          >
            <List.Scroll full="y">
              <List.Items<library.EntryKey> emptyContent={EMPTY_CONTENT}>
                {renderItem}
              </List.Items>
            </List.Scroll>
          </Select.Frame>
          <AddMessage entries={addable} onAdd={handleAdd} />
        </Flex.Box>
      }
      detailsPath={path}
      title={entry?.name}
    >
      {content}
    </Task.Views.Panes>
  );
};

const Placeholder = ({ children }: { children: ReactNode }) => (
  <Flex.Box y grow align="center" justify="center">
    <Text.Text status="disabled">{children}</Text.Text>
  </Flex.Box>
);
