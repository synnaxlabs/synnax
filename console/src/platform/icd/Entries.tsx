// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type device, DisconnectedError, icd } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { useExclusiveCallback } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import type { Tree } from "@synnaxlabs/lyra/tree";
import { ICD as PICD, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { deep, payload, uuid } from "@synnaxlabs/x";
import {
  type Dispatch,
  type ReactElement,
  type RefObject,
  type SetStateAction,
  useRef,
  useState,
} from "react";

import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { MessageLabel, SuitRead } from "@/platform/icd/Bindings";
import {
  AddField,
  FieldDetails,
  MessageDetails,
  MessageMark,
  MessageSections,
  type MessageSectionsProps,
} from "@/platform/icd/Details";
import {
  type FormatForm,
  formatOf,
  useFormats,
  WithFormats,
} from "@/platform/icd/Formats";
import { MessageFormatField } from "@/platform/icd/MessageFormatField";
import { createPayload, FORMAT_KINDS } from "@/platform/icd/payload";
import { Plans } from "@/platform/icd/Plans";
import { PollFields } from "@/platform/icd/Poll";
import { isTaskFormat, mostRead } from "@/platform/icd/read";
import { initialExpanded } from "@/platform/icd/tree/nodes";
import { use as useTree } from "@/platform/icd/tree/use";
import { Task } from "@/platform/task";

/** The entries of every task form that reads ICD messages from a port or a socket. */
const ENTRIES = deep.path<{ config: { entries: icd.ReadEntry[] } }>().config.entries;

const entryPath = (key: string): deep.Path<icd.ReadEntry> => ENTRIES[key];

const MISSING_DESCRIPTION =
  "The ICD no longer has this message. Add a message to the entry, then remove this one.";

/** @returns a name like "Message 2" that no message in held has. */
const messageName = (held: icd.Message[]): string => {
  const names = new Set(held.map(({ name }) => name));
  let n = held.length + 1;
  while (names.has(`Message ${n}`)) n++;
  return `Message ${n}`;
};

interface EntryLabelProps {
  icdKey: icd.Key;
  /** The first message of the entry. */
  message?: icd.MessageKey;
}

/** Names an entry by the format of its first message and the name of its ICD. */
const EntryLabel = ({ icdKey, message }: EntryLabelProps): ReactElement => {
  const ctx = useFormats().get(icdKey);
  const name = PForm.useFieldValue(PICD.formPath.name, { optional: true, ctx });
  const format = PForm.useFieldValue(
    PICD.formPath.messages[message ?? ""].payload.format,
    { optional: true, ctx },
  );
  return (
    <Task.Views.ItemLabel
      kind={format == null ? "" : FORMAT_KINDS[format]}
      kindColor="var(--pluto-secondary-z)"
    >
      {name ?? ""}
    </Task.Views.ItemLabel>
  );
};

const EntryItem = (props: Tree.ItemRenderProps<string>) => {
  const path = entryPath(props.itemKey);
  const disabled = PForm.useFieldValue(path.disabled, { optional: true });
  const icdKey = PForm.useFieldValue(path.icd, { optional: true });
  const messages = PForm.useFieldValue(path.messages, { optional: true });
  if (disabled == null || icdKey == null || messages == null) return null;
  return (
    <Task.Views.TreeGroupItem
      {...props}
      label={<EntryLabel icdKey={icdKey} message={messages[0]?.message} />}
      off={disabled}
    >
      <MessageMark
        icdKey={icdKey}
        messages={messages.map(({ message }) => message)}
        missingDescription={() => MISSING_DESCRIPTION}
      />
      <Task.EnabledCheckbox path={path.disabled} />
    </Task.Views.TreeGroupItem>
  );
};

type MessageProps = Omit<MessageSectionsProps, "format">;

/** @returns the props that the panels of a message of an entry share. */
const messageProps = (
  entry: icd.ReadEntry,
  read: number,
  prefix: string,
): MessageProps => {
  const { index, bindings } = entry.messages[read];
  return {
    path: entryPath(entry.key),
    read,
    prefix,
    // A new index field keeps the channel of the old one.
    indexChannel: bindings.find(({ field }) => field === index)?.channel ?? 0,
    sourceLabel: "Arrival time",
    missingDescription: MISSING_DESCRIPTION,
  };
};

interface EntryDetailsProps {
  path: deep.Path<icd.ReadEntry>;
  /** The open form of the ICD of the entry. */
  format?: FormatForm;
  /** The props of the message of an entry that reads one. */
  message?: MessageProps;
  /** Adds a field to the message of an entry that reads one. */
  onAddField: () => void;
  /** Adds a message to the entry, for frames that its messages do not cover. */
  onAddMessage: () => void;
  /** Reads a message of an ICD instead of the one message that the entry reads. */
  onFormatChange: (icdKey: icd.Key, message: icd.Message) => void;
  /** Makes the entry read a new message of the format with no fields. */
  onFormatCreate: (format: payload.Format) => void;
}

/**
 * The poll of an entry. An entry that reads one message also shows its format and
 * timestamp, which each message of an entry that reads several shows on its own.
 */
const EntryDetails = ({
  path,
  format,
  message,
  onAddField,
  onAddMessage,
  onFormatChange,
  onFormatCreate,
}: EntryDetailsProps): ReactElement => {
  const isPreview = Task.useIsPreview();
  const reads = PForm.useFieldValue(path.messages);
  const icdKey = PForm.useFieldValue(path.icd);
  const encoding = PForm.useFieldValue(
    PICD.formPath.messages[reads.length === 1 ? reads[0].message : ""].payload.format,
    { optional: true, ctx: format },
  );
  return (
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      <PForm.Sections>
        <PForm.Section
          title="Entry"
          actions={
            !isPreview && (
              <>
                {encoding != null && isTaskFormat(encoding) && (
                  <AddField onClick={onAddField} />
                )}
                <Button.Button variant="text" size="small" onClick={onAddMessage}>
                  <Icon.Add />
                  Add message
                </Button.Button>
              </>
            )
          }
        >
          {reads.length === 1 && (
            <MessageFormatField
              icdKey={icdKey}
              message={reads[0].message}
              onChange={onFormatChange}
              formats={payload.FORMATS}
              onCreate={onFormatCreate}
              preview={isPreview}
            />
          )}
          <PollFields path={path} />
        </PForm.Section>
        {format != null && message != null && (
          <MessageSections {...message} format={format} />
        )}
      </PForm.Sections>
    </Flex.Box>
  );
};

interface PanesProps extends EntriesProps {
  /** The key of each selected tree item. */
  selected: string[];
  onSelect: Dispatch<SetStateAction<string[]>>;
  expanded: string[];
  onExpand: Dispatch<SetStateAction<string[]>>;
  /** The key of each message that the task made. */
  made: RefObject<Set<string>>;
}

export interface EntriesProps {
  device: Pick<device.Device, "name">;
}

/** The entries as a tree, beside the details of the selected item. */
const Panes = ({
  device,
  selected,
  onSelect,
  expanded,
  onExpand,
  made,
}: PanesProps): ReactElement => {
  const client = PSynnax.use();
  const ctx = PForm.useContext();
  const handleError = Status.useErrorHandler();
  const formats = useFormats();
  const entries = PForm.useFieldValue(ENTRIES);
  const prefix = channel.escapeInvalidName(device.name);
  const {
    tree,
    handleSelect,
    contextMenu,
    renderItem,
    current,
    addField,
    addMessage,
    setMessage,
    setNewMessage,
  } = useTree({
    sources: ENTRIES,
    sourceOf: (entry) => entry,
    pathOf: entryPath,
    SourceItem: EntryItem,
    missingDescription: () => MISSING_DESCRIPTION,
    bindingOf: (_, __, { key }) => icd.readBindingZ.parse({ field: key }),
    messageName: (_, held) => messageName(held),
    made,
    selected,
    onSelect,
    expanded,
    onExpand,
  });

  /**
   * Adds an entry that reads a new binary message with no fields, in the ICD that the
   * most entries read, or in a new ICD named after the device when no entry reads one.
   */
  const add = async () => {
    if (client == null) throw new DisconnectedError();
    const home = mostRead(ctx.get(ENTRIES).value);
    const form = home == null ? null : formatOf(formats, home);
    const held = form?.value().messages ?? [];
    const message: icd.Message = {
      key: uuid.create(),
      name: messageName(held),
      payload: createPayload("binary"),
    };
    let icdKey = home;
    if (form == null)
      icdKey = (await client.icds.create({ name: device.name, messages: [message] }))
        .key;
    else form.insert(PICD.formPath.messages, message, held.at(-1)?.key);
    made.current.add(message.key);
    const entry = icd.readEntryZ.parse({
      icd: icdKey,
      messages: [icd.readMessageZ.parse({ message: message.key })],
    });
    PForm.fieldListUtils(ctx, ENTRIES).push(entry);
    onSelect([entry.key]);
    tree.expand(entry.key);
  };
  // Adds run one at a time, so a second click cannot make a second ICD.
  const addExclusive = useExclusiveCallback(add);
  const handleAdd = () => handleError(addExclusive, "Failed to add entry");
  const isPreview = Task.useIsPreview();

  const entry =
    current == null ? undefined : entries.find(({ key }) => key === current.source);
  const format = entry == null ? undefined : formats.get(entry.icd);
  let details = null;
  let title = null;
  let detailsPath: deep.Target<unknown> | null = null;
  if (entry != null && format != null && current?.kind === "field") {
    const read = entryPath(entry.key).messages[current.read];
    const i = entry.messages[current.read].bindings.findIndex(
      ({ field }) => field === current.field.key,
    );
    detailsPath = i < 0 ? read : read.bindings[i];
    title = (
      <Text.Text level="p" weight={500} overflow="fade">
        {current.field.name}
      </Text.Text>
    );
    details = (
      <FieldDetails
        key={selected[0]}
        path={entryPath(entry.key)}
        read={current.read}
        field={current.field}
        format={format}
        prefix={prefix}
      />
    );
  } else if (entry != null && format != null && current?.kind === "message") {
    detailsPath = entryPath(entry.key).messages[current.read];
    title = <MessageLabel path={entryPath(entry.key)} read={current.read} />;
    details = (
      <MessageDetails
        key={selected[0]}
        {...messageProps(entry, current.read, prefix)}
        format={format}
        matchTitle="Values"
        anyLabel="Any value"
        onAddField={() => addField(entry.key, current.read)}
      />
    );
  } else if (entry != null) {
    detailsPath = entryPath(entry.key);
    title = <EntryLabel icdKey={entry.icd} message={entry.messages[0]?.message} />;
    details = (
      <EntryDetails
        key={entry.key}
        path={entryPath(entry.key)}
        format={format}
        message={
          entry.messages.length === 1 ? messageProps(entry, 0, prefix) : undefined
        }
        onAddField={() => addField(entry.key, 0)}
        onAddMessage={() => addMessage(entry.key)}
        onFormatChange={(icdKey, message) => setMessage(entry.key, icdKey, message)}
        onFormatCreate={(f) => setNewMessage(entry.key, f)}
      />
    );
  }
  return (
    <Task.Views.Panes
      listTitle="Entries"
      list={
        <Task.Views.TreeList
          {...tree}
          onSelect={handleSelect}
          menu={contextMenu}
          emptyMessage="No entries"
          addLabel="Add entry"
          onAdd={handleAdd}
        >
          {renderItem}
        </Task.Views.TreeList>
      }
      detailsPath={detailsPath}
      title={title ?? undefined}
    >
      {details ?? (
        <Empty.Action
          grow
          message="No entry selected"
          action={isPreview ? undefined : "Add entry"}
          onClick={handleAdd}
        />
      )}
    </Task.Views.Panes>
  );
};

/**
 * The entries of a task that reads ICD messages from a port or a socket, at
 * `config.entries` of its form. Each entry reads the messages of one device, and can
 * poll it with a query. The channels of an entry take the name of the device as their
 * prefix.
 */
export const Entries = ({ device }: EntriesProps): ReactElement => {
  const entries = PForm.useFieldValue(ENTRIES);
  // A new ICD mounts the children of WithFormats again, so their state lives here.
  const [selected, setSelected] = useState<string[]>([]);
  const [expanded, setExpanded] = useState(() => initialExpanded(entries));
  const made = useRef(new Set<string>());
  return (
    <WithFormats icdKeys={entries.map((e) => e.icd)}>
      <Plans plans={[]}>
        <Panes
          device={device}
          selected={selected}
          onSelect={setSelected}
          expanded={expanded}
          onExpand={setExpanded}
          made={made}
        />
        {entries.flatMap(({ key, messages }) =>
          messages.map(({ message }, read) => (
            <SuitRead key={`${key}/${message}`} path={entryPath(key)} read={read} />
          )),
        )}
      </Plans>
    </WithFormats>
  );
};
