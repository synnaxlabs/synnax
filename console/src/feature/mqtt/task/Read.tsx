// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/mqtt/task/Form.css";

import {
  type channel,
  DisconnectedError,
  icd,
  mqtt,
  type Synnax,
} from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { useExclusiveCallback } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import type { Tree } from "@synnaxlabs/lyra/tree";
import { ICD as PICD, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { DataType, deep, map, primitive, unique } from "@synnaxlabs/x";
import {
  createContext,
  type Dispatch,
  type FC,
  type PropsWithChildren,
  type ReactElement,
  type RefObject,
  type SetStateAction,
  useContext,
  useRef,
  useState,
} from "react";

import { Device } from "@/feature/mqtt/device";
import { DEFAULT_READ_NAME } from "@/feature/mqtt/task/create";
import type { Kind } from "@/feature/mqtt/task/discover/Discover";
import { createDiscoverModal } from "@/feature/mqtt/task/discover/modal";
import {
  groupOf,
  type ShapePick,
  type TopicPick,
} from "@/feature/mqtt/task/discover/Topics";
import { createSparkplugEntry, namePrefix } from "@/feature/mqtt/task/entries";
import { EntryLabel } from "@/feature/mqtt/task/EntryLabel";
import type { Leaf } from "@/feature/mqtt/task/leaves";
import { QoSField } from "@/feature/mqtt/task/QoSField";
import {
  sparkplugChannelName,
  sparkplugPropertiesKey,
} from "@/feature/mqtt/task/sparkplug";
import { SparkplugTagFields } from "@/feature/mqtt/task/SparkplugTagFields";
import {
  deployReadConfigZ,
  type HeldEntry,
  type HeldMessage,
  type PlainReadEntry,
  READ_SCHEMAS,
  type ReadEntry,
  type ReadSchemas,
  type SparkplugReadEntry,
} from "@/feature/mqtt/task/types";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { ICD } from "@/platform/icd";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<ReadSchemas>>();

const plainPath = (key: string) =>
  deep.variant(root.config.entries[key], "type", "plain");

/** The open form of one ICD. */
interface Format {
  key: icd.Key;
  form: ICD.FormatForm;
}

const TOPIC_INPUT_PROPS = { placeholder: "plant/line1/telemetry" } as const;
// A topic added by name has no topic yet, so its details open on the topic.
const EMPTY_TOPIC_INPUT_PROPS = { ...TOPIC_INPUT_PROPS, autoFocus: true } as const;

/** @returns the messages of the entries' ICDs, reading the open one from its form. */
const readMessages = async (
  client: Synnax,
  format: Format | null,
  entries: PlainReadEntry[],
): Promise<ICD.Messages> => {
  const keys = entries.map((e) => e.icd).filter((key) => key !== format?.key);
  const messages = await ICD.retrieveMessages(client, keys);
  if (format != null) messages.set(format.key, format.form.value().messages);
  return messages;
};

const isPlain = (entry: ReadEntry): entry is PlainReadEntry => entry.type === "plain";

/** A topic, and the fields that a pick adds to its message format. */
interface Draft {
  type: "plain";
  topic: string;
  fields: icd.JSONField[];
  /** The key of the field in fields that indexes a new message. */
  index?: icd.FieldKey;
  /**
   * The pointer of the field that each document of a new message holds. Absent when the
   * message reads every document of the topic.
   */
  present?: string;
  /**
   * The position, in the reads of the entry of a held topic, of the message that the
   * draft extends. Absent for a new message.
   */
  read?: number;
}

/** What the discovery dialog adds: topics to read and Sparkplug B tags. */
type Addition = Draft | SparkplugReadEntry;

const isDraft = (addition: Addition): addition is Draft => addition.type === "plain";

/** @param index - The pointer of the leaf that indexes the message. */
const createDraft = (
  topic: string,
  leaves: Leaf[] = [],
  index?: string,
  present?: string,
): Draft => {
  const fields = ICD.createFields(leaves);
  return {
    type: "plain",
    topic,
    fields,
    index: fields.find(({ pointer }) => pointer === index)?.key,
    present,
  };
};

/** @returns the leaves of a pick, with its index. */
const pickedLeaves = ({ shape, pointers, index }: ShapePick): Leaf[] =>
  shape.leaves.filter((l) => pointers.has(l.pointer) || l.pointer === index);

/**
 * @returns the drafts of the picks of a topic. The picks of the shapes whose documents
 * the Core gives one held message make one draft that extends it. Each other picked
 * shape gets a new message.
 */
const draftsOf = (topic: string, pick: TopicPick): Draft[] => {
  const { added, extended } = groupOf(pick);
  const drafts = added.map((p) =>
    createDraft(topic, pickedLeaves(p), p.index, p.shape.present),
  );
  for (const [read, picks] of extended) {
    const leaves = unique.by(picks.flatMap(pickedLeaves), ({ pointer }) => pointer);
    const index = picks.find((p) => p.index != null)?.index;
    drafts.push({ ...createDraft(topic, leaves, index), read });
  }
  return drafts;
};

const DISCOVER_KIND: Kind<Addition> = {
  title: "Add data",
  hint: "Select the values to read. Each one becomes a field of the message format.",
  tagHint: "Select the tags to read. Each one becomes a channel.",
  noun: "field",
  tagNoun: "tag",
  indexed: true,
  create: ({ topics, tags }) => [
    ...[...topics].flatMap(([topic, pick]) => draftsOf(topic, pick)),
    ...[...tags.values()].map((t) => createSparkplugEntry(t)),
  ],
  // A topic that the task reads keeps its first message.
  createTopic: (topic) => ({ ...createDraft(topic), read: 0 }),
  createTag: () => createSparkplugEntry(),
};

const useDiscoverModal = createDiscoverModal(DISCOVER_KIND);

/**
 * @returns the channels that the bindings and tags of the enabled entries write. The
 * Core skips disabled entries.
 */
const writtenChannels = (entries: ReadEntry[]): Set<channel.Key> => {
  const written = new Set(
    entries
      .filter(({ disabled }) => !disabled)
      .flatMap((e) =>
        isPlain(e)
          ? e.messages.flatMap(({ bindings }) => bindings.map((b) => b.channel))
          : [e.channel],
      ),
  );
  written.delete(0);
  return written;
};

/**
 * @returns where the device stores the channels of the messages of entry. An older
 * Console stored the channels of a topic under the bare topic.
 */
const storedSource = ({
  topic,
  messages,
}: Pick<PlainReadEntry, "topic" | "messages">): ICD.Stored.Source => ({
  key: topic,
  legacy: topic,
  messages,
});

/**
 * @returns stored with the channels that an older Console stored under the topic of
 * entries split among the messages that read the topic.
 */
const splitStored = (
  stored: Record<string, Task.ReadChannelProps>,
  entries: ReadEntry[],
): Record<string, Task.ReadChannelProps> =>
  ICD.Stored.split(
    stored,
    entries
      .filter(isPlain)
      .map((entry) => ({ ...storedSource(entry), disabled: entry.disabled })),
  );

/** Inserts entries into the lists of one ICD. */
type Inserter = Pick<ICD.FormatForm, "insert">;

/**
 * Appends each field whose pointer the message lacks.
 * @returns the fields it appends.
 */
const appendFields = (
  to: Inserter,
  message: icd.Message,
  fields: icd.JSONField[],
): icd.JSONField[] => {
  const held = ICD.jsonFields(message);
  const pointers = new Set(held.map(({ pointer }) => pointer));
  const added = fields.filter(({ pointer }) => !pointers.has(pointer));
  const names = ICD.fieldNames(
    added.map(({ pointer }) => pointer),
    held.map(({ name }) => name),
  );
  const appended = added.map((field, i) => ({ ...field, name: names[i] }));
  let after = held.at(-1)?.key;
  for (const field of appended) {
    to.insert(ICD.fieldsPath(message.key), field, after);
    after = field.key;
  }
  return appended;
};

/** @returns a message for the fields of a draft, named after its topic and kind. */
const createMessage = (
  { topic, fields, present }: Draft,
  taken: icd.Message[],
): icd.Message => {
  const message = ICD.createMessage(ICD.kindName(topic, present), fields, taken);
  if (present == null) return message;
  const match: icd.JSONMatch = { type: "present", pointer: present };
  return { ...message, payload: { format: "json", fields, match } };
};

type Read = Pick<icd.ReadMessage, "message" | "index" | "bindings">;

/** @returns an entry for each topic of reads, which reads its messages in order. */
const entriesOf = (
  icdKey: icd.Key,
  reads: [topic: string, Read][],
): PlainReadEntry[] => {
  const byTopic = new Map<string, Read[]>();
  for (const [topic, read] of reads)
    map.getOrInsert(byTopic, topic, () => []).push(read);
  return [...byTopic].map(([topic, messages]) =>
    mqtt.plainReadEntryZ.parse({ type: "plain", icd: icdKey, topic, messages }),
  );
};

interface AddTopicsResult {
  /** The entries of the topics that the task does not read yet. */
  added: PlainReadEntry[];
  /**
   * The bindings that each held entry gains, by entry key. A read with no position is a
   * new message that the entry reads after its others. An entry whose message is gone
   * reads the new one. An entry whose message had no fields gains the index.
   */
  held: {
    key: string;
    read?: number;
    message?: icd.MessageKey;
    index?: icd.FieldKey;
    bindings: icd.ReadBinding[];
  }[];
}

/** @returns the bindings of the fields of a draft that the message reads. */
type Bind = (
  draft: Draft,
  fields: icd.JSONField[],
  message: icd.MessageKey,
) => icd.ReadBinding[];

/** @returns the name of the ICD of a task named name. */
const icdName = (name: string): string =>
  name.trim() === "" ? DEFAULT_READ_NAME : name;

/** Creates an ICD with a message for each draft, and the entries that read them. */
const createICD = async (
  client: Synnax,
  name: string,
  drafts: Draft[],
  bind: Bind,
): Promise<AddTopicsResult> => {
  const messages: icd.Message[] = [];
  for (const draft of drafts) messages.push(createMessage(draft, messages));
  const { key } = await client.icds.create({ name, messages });
  const added = entriesOf(
    key,
    drafts.map((d, i) => [
      d.topic,
      {
        message: messages[i].key,
        index: d.index,
        bindings: bind(d, d.fields, messages[i].key),
      },
    ]),
  );
  return { added, held: [] };
};

interface AddTopicsParams {
  client: Synnax;
  /** The form of the ICD that takes the messages of new topics. */
  format: Format;
  entries: PlainReadEntry[];
  drafts: Draft[];
  bind: Bind;
}

/**
 * Appends each draft's fields to the message format of its topic, in the ICD of its
 * entry. A topic with no format gets a new message, named after the topic, in the ICD
 * of its entry, or in the ICD of the form. Edits to the ICD of the form go through the
 * form.
 */
const addTopics = async ({
  client,
  format,
  entries,
  drafts,
  bind,
}: AddTopicsParams): Promise<AddTopicsResult> => {
  const messages = await readMessages(client, format, entries);
  const actions = new Map<icd.Key, icd.Action[]>();
  const inserterOf = (key: icd.Key): Inserter => {
    if (key === format.key) return format.form;
    const pending = map.getOrInsert(actions, key, () => []);
    return {
      insert: (path, value, after) =>
        pending.push(icd.insert({ path: String(path).split("."), value, after })),
    };
  };
  const result: AddTopicsResult = { added: [], held: [] };
  const reads: [topic: string, Read][] = [];
  format.form.batch(() => {
    for (const draft of drafts) {
      const entry = entries.find((e) => e.topic === draft.topic);
      const key = entry?.icd ?? format.key;
      const { read } = draft;
      const message =
        entry == null || read == null
          ? undefined
          : ICD.messageOf(messages, entry, read);
      if (entry != null && message != null) {
        const appended = appendFields(inserterOf(key), message, draft.fields);
        result.held.push({
          key: entry.key,
          read,
          index: draft.index,
          bindings: bind(draft, appended, message.key),
        });
        continue;
      }
      const prior = messages.get(key) ?? [];
      const created = createMessage(draft, prior);
      inserterOf(key).insert(PICD.formPath.messages, created, prior.at(-1)?.key);
      messages.set(key, [...prior, created]);
      const bindings = bind(draft, draft.fields, created.key);
      if (entry == null)
        reads.push([
          draft.topic,
          { message: created.key, index: draft.index, bindings },
        ]);
      else
        result.held.push({
          key: entry.key,
          read,
          message: created.key,
          index: draft.index,
          bindings,
        });
    }
  });
  result.added = entriesOf(format.key, reads);
  for (const [key, pending] of actions) await client.icds.dispatch(key, pending);
  return result;
};

const EntryItem = (props: Tree.ItemRenderProps<string>) => {
  const path = root.config.entries[props.itemKey];
  const type = PForm.useFieldValue(path.type, { optional: true });
  const disabled = PForm.useFieldValue(path.disabled, { optional: true }) === true;
  if (type == null) return null;
  return (
    <Task.Views.TreeGroupItem
      {...props}
      label={<EntryLabel path={path} />}
      channel={
        type === "sparkplug"
          ? deep.variant(path, "type", "sparkplug").channel
          : undefined
      }
      off={disabled}
    >
      <Task.EnabledCheckbox path={path.disabled} />
    </Task.Views.TreeGroupItem>
  );
};

/** @returns the props that the panels of a message of a topic share. */
const useMessageProps = (
  device: Device.Device,
  path: deep.Path<PlainReadEntry>,
  read: number,
): Omit<ICD.MessageSectionsProps, "format"> => {
  const topic = PForm.useFieldValue(path.topic);
  const messages = PForm.useFieldValue(path.messages);
  const entries = PForm.useFieldValue(root.config.entries);
  const prefix = namePrefix(device.name, topic);
  return {
    path,
    read,
    prefix,
    indexChannel:
      ICD.Stored.of(
        splitStored(device.properties.read, entries),
        storedSource({ topic, messages }),
        read,
      )?.index ?? 0,
    sourceLabel: "Arrival time",
    missingDescription:
      read === 0
        ? "The ICD has no message for this topic. Add data to make one."
        : "The ICD no longer has this message. Remove it from the topic.",
  };
};

interface MessagePanelProps {
  path: deep.Path<PlainReadEntry>;
  /** The index of the message in the topic's reads. */
  read: number;
  format: ICD.FormatForm;
  device: Device.Device;
}

const MessageSections = ({ device, path, read, format }: MessagePanelProps) => {
  const props = useMessageProps(device, path, read);
  return <ICD.MessageSections {...props} format={format} />;
};

interface MessageDetailsProps extends MessagePanelProps {
  onAddField: () => void;
  bindDisabled: boolean;
}

const MessageDetails = ({
  device,
  path,
  read,
  format,
  onAddField,
  bindDisabled,
}: MessageDetailsProps) => {
  const props = useMessageProps(device, path, read);
  return (
    <ICD.MessageDetails
      {...props}
      format={format}
      matchTitle="Documents"
      anyLabel="Any document"
      onAddField={onAddField}
      addFieldDisabled={bindDisabled}
    />
  );
};

interface FieldDetailsProps extends MessagePanelProps {
  field: ICD.TreeField;
}

const FieldDetails = ({ device, ...rest }: FieldDetailsProps) => {
  const topic = PForm.useFieldValue(rest.path.topic);
  const prefix = namePrefix(device.name, topic);
  return <ICD.FieldDetails {...rest} prefix={prefix} />;
};

interface TopicDetailsProps {
  path: deep.Path<PlainReadEntry>;
  format: ICD.FormatForm;
  device: Device.Device;
  /** Adds a field to the message of a topic that reads one. */
  onAddField: () => void;
  /** Adds a message to the topic, for documents that its messages do not cover. */
  onAddMessage: () => void;
  /** Makes a topic that reads one message read a message of any ICD. */
  onFormatChange: (icdKey: icd.Key, message: icd.Message) => void;
  bindDisabled: boolean;
}

/**
 * The subscription of a topic. A topic that reads one message also shows its
 * timestamp, which each message of a topic that reads several shows on its own.
 */
const TopicDetails: FC<TopicDetailsProps> = ({
  path,
  format,
  device,
  onAddField,
  onAddMessage,
  onFormatChange,
  bindDisabled,
}) => {
  const isPreview = Task.useIsPreview();
  const topic = PForm.useFieldValue(path.topic);
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
          title="Subscription"
          actions={
            !isPreview && (
              <>
                {encoding === "json" && (
                  <ICD.AddField onClick={onAddField} disabled={bindDisabled} />
                )}
                <Button.Button variant="text" size="small" onClick={onAddMessage}>
                  <Icon.Add />
                  Add message
                </Button.Button>
              </>
            )
          }
        >
          <PForm.TextField
            path={path.topic}
            label="Topic"
            padHelpText={false}
            inputProps={topic === "" ? EMPTY_TOPIC_INPUT_PROPS : TOPIC_INPUT_PROPS}
          />
          {reads.length === 1 && (
            <ICD.MessageFormatField
              icdKey={icdKey}
              message={reads[0].message}
              onChange={onFormatChange}
              disabled={bindDisabled}
              preview={isPreview}
            />
          )}
          <QoSField path={path.qos} />
          <PForm.CheckboxField
            path={path.retainedIgnored}
            label="Ignore retained"
            padHelpText={false}
          />
        </PForm.Section>
        {reads.length === 1 && (
          <MessageSections path={path} read={0} format={format} device={device} />
        )}
      </PForm.Sections>
    </Flex.Box>
  );
};

// A Sparkplug B DateTime tag reads into a timestamp channel.
const SPARKPLUG_HIDDEN_DATA_TYPES = [DataType.UUID, DataType.JSON, DataType.BYTES];

/** @returns the field of a tag, which reads whole into one channel at pointer "". */
const tagField = (
  deviceName: string,
  entry: SparkplugReadEntry,
): Task.ReadChannelField => ({
  key: entry.key,
  pointer: "",
  channel: entry.channel,
  name: primitive.isNonZero(entry.name)
    ? entry.name
    : sparkplugChannelName(deviceName, entry),
  dataType: entry.dataType,
});

interface TagDetailsProps {
  path: deep.Path<SparkplugReadEntry>;
  device: Device.Device;
}

const TagDetails: FC<TagDetailsProps> = ({ path, device }) => {
  const entry = PForm.useFieldValue(path);
  const plan = useContext(TagPlansContext).get(entry.key);
  const planned = plan == null ? undefined : Task.newChannelName(plan, entry.key);
  return (
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      <PForm.Sections>
        <PForm.Section title="Tag">
          <SparkplugTagFields path={path} />
        </PForm.Section>
        <PForm.Section title="Channel">
          <Task.ChannelNameField
            channel={entry.channel}
            namePath={path.name}
            defaultName={planned ?? sparkplugChannelName(device.name, entry)}
            plannedName={planned}
            skipped={entry.disabled}
          />
          <Task.Payload.DataTypeField
            path={path.dataType}
            channel={entry.channel}
            hideDataTypes={SPARKPLUG_HIDDEN_DATA_TYPES}
          />
        </PForm.Section>
      </PForm.Sections>
    </Flex.Box>
  );
};

/**
 * @returns a name for the next message of a topic, numbered after its count, that no
 * message of the ICD has.
 */
const nextMessageName = (entry: PlainReadEntry, taken: icd.Message[]): string => {
  const names = new Set(taken.map(({ name }) => name));
  const base = entry.topic === "" ? "Untitled" : entry.topic;
  let n = entry.messages.length + 1;
  while (names.has(`${base} ${n}`)) n++;
  return `${base} ${n}`;
};

interface EntriesProps {
  device: Device.Device;
  /** The key of each selected tree item. */
  selected: string[];
  onSelect: Dispatch<SetStateAction<string[]>>;
  expanded: string[];
  onExpand: Dispatch<SetStateAction<string[]>>;
  /** The key of each message that the task made. */
  made: RefObject<Set<string>>;
}

/** The entries of the task as a tree, beside the details of the selected item. */
const Entries: FC<EntriesProps> = ({
  device,
  selected,
  onSelect,
  expanded,
  onExpand,
  made,
}) => {
  const client = PSynnax.use();
  const ctx = PForm.useContext();
  const handleError = Status.useErrorHandler();
  const discover = useDiscoverModal();
  const formats = ICD.useFormats();
  const { push } = PForm.useFieldList(root.config.entries);
  const entries = PForm.useFieldValue(root.config.entries);
  const stored = device.properties.read;
  const { dataTypes, bindDisabled } = ICD.Stored.useDataTypes(stored);
  const {
    tree,
    handleSelect,
    contextMenu,
    renderItem,
    current,
    addField,
    addMessage,
    setMessage,
  } = ICD.Tree.use({
    sources: root.config.entries,
    sourceOf: (e) => (isPlain(e) ? e : null),
    pathOf: plainPath,
    SourceItem: EntryItem,
    // Starting reuses the channel that the device stores for the field's pointer,
    // unless another field reads it. A topic that reads one message also reuses the
    // channels of its earlier messages, so a new message format keeps its channels.
    bindDisabled,
    bindingOf: (key, read, field) => {
      const entry = ctx.get(plainPath(key)).value;
      const all = ctx.get(root.config.entries).value;
      const held = splitStored(stored, all);
      const channels =
        entry.messages.length === 1
          ? ICD.Stored.channels(held, storedSource(entry), entry.messages[0].message)
          : (ICD.Stored.of(held, storedSource(entry), read)?.channels ?? {});
      const written = writtenChannels(all);
      return ICD.Stored.bindingsOf(channels, [field], written, dataTypes)[0];
    },
    messageName: (entry, held) =>
      nextMessageName(ctx.get(plainPath(entry)).value, held),
    made,
    selected,
    onSelect,
    expanded,
    onExpand,
  });
  const home = ICD.mostRead(entries.filter(isPlain));
  const format = home == null ? undefined : formats.get(home);

  const add = async () => {
    if (client == null) throw new DisconnectedError();
    const plain = entries.filter(isPlain);
    const messages = ICD.messagesOf(formats);
    const held = entries.map((e): HeldEntry => {
      if (!isPlain(e)) return e;
      return {
        type: "plain",
        topic: e.topic,
        messages: e.messages.map((_, r): HeldMessage => {
          const message = ICD.messageOf(messages, e, r);
          const match =
            message?.payload.format === "json" ? message.payload.match : undefined;
          return { match, fields: ICD.jsonFields(message) };
        }),
      };
    });
    const additions = await discover({ device, entries: held });
    if (additions == null || additions.length === 0) return;
    const drafts = additions.filter(isDraft);
    const tags = additions.filter((a): a is SparkplugReadEntry => !isDraft(a));
    const written = writtenChannels(entries);
    const split = splitStored(stored, entries);
    const bind: Bind = ({ topic }, fields, message) => {
      const entry = plain.find((e) => e.topic === topic) ?? { topic, messages: [] };
      return ICD.Stored.bindingsOf(
        ICD.Stored.channels(split, storedSource(entry), message),
        fields,
        written,
        dataTypes,
      );
    };
    const { added, held: grown } =
      drafts.length === 0
        ? { added: [], held: [] }
        : home == null || format == null
          ? await createICD(client, icdName(ctx.get(root.name).value), drafts, bind)
          : await addTopics({
              client,
              format: { key: home, form: format },
              entries: plain,
              drafts,
              bind,
            });
    push([...added, ...tags]);
    for (const { messages } of added)
      for (const { message } of messages) made.current.add(message);
    for (const { message } of grown) if (message != null) made.current.add(message);
    grown.forEach(({ key, read: position, message, index, bindings }) => {
      const reads = plainPath(key).messages;
      if (position == null) {
        const added = icd.readMessageZ.parse({ message, index, bindings });
        return ctx.set(reads, [...ctx.get(reads).value, added]);
      }
      const read = reads[position];
      // The index and bindings of a deleted message name fields it no longer has.
      if (message != null)
        ctx.set(read, icd.readMessageZ.parse({ message, index, bindings }));
      else {
        const { value } = ctx.get(read);
        ctx.set(read, {
          ...value,
          index: index ?? value.index,
          bindings: [...value.bindings, ...bindings],
        });
      }
    });
    const firstAddition = additions[0];
    const firstKey = isDraft(firstAddition)
      ? (
          entries.find((e) => isPlain(e) && e.topic === firstAddition.topic) ??
          added.find((e) => e.topic === firstAddition.topic)
        )?.key
      : firstAddition.key;
    if (firstKey == null) return;
    onSelect([firstKey]);
    tree.expand(firstKey);
  };
  // Adds run one at a time, so a second click cannot make a second ICD.
  const addExclusive = useExclusiveCallback(add);
  const handleAdd = () => handleError(addExclusive, "Failed to add data");

  const entry =
    current == null ? undefined : entries.find(({ key }) => key === current.source);
  const entryFormat =
    entry != null && isPlain(entry) ? formats.get(entry.icd) : undefined;
  let details = null;
  let title = null;
  let detailsPath: deep.Target<unknown> | null = null;
  if (
    entry != null &&
    isPlain(entry) &&
    current?.kind === "field" &&
    entryFormat != null
  ) {
    const read = plainPath(entry.key).messages[current.read];
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
        path={plainPath(entry.key)}
        read={current.read}
        field={current.field}
        format={entryFormat}
        device={device}
      />
    );
  } else if (entry != null && current?.kind === "message" && entryFormat != null) {
    detailsPath = plainPath(entry.key).messages[current.read];
    title = <ICD.MessageLabel path={plainPath(entry.key)} read={current.read} />;
    details = (
      <MessageDetails
        key={selected[0]}
        path={plainPath(entry.key)}
        read={current.read}
        format={entryFormat}
        device={device}
        onAddField={() => addField(entry.key, current.read)}
        bindDisabled={bindDisabled}
      />
    );
  } else if (entry != null) {
    detailsPath = root.config.entries[entry.key];
    title = <EntryLabel path={root.config.entries[entry.key]} />;
    if (entry.type === "sparkplug")
      details = (
        <TagDetails
          key={entry.key}
          path={deep.variant(root.config.entries[entry.key], "type", "sparkplug")}
          device={device}
        />
      );
    else if (entryFormat != null)
      details = (
        <TopicDetails
          key={entry.key}
          path={plainPath(entry.key)}
          format={entryFormat}
          device={device}
          onAddField={() => addField(entry.key, 0)}
          onAddMessage={() => addMessage(entry.key)}
          onFormatChange={(icdKey, message) => setMessage(entry.key, icdKey, message)}
          bindDisabled={bindDisabled}
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
          addLabel="Add data"
          onAdd={handleAdd}
          addDisabled={bindDisabled}
        >
          {renderItem}
        </Task.Views.TreeList>
      }
      detailsPath={detailsPath}
      title={title ?? undefined}
    >
      {details ?? (
        <Flex.Box y grow align="center" justify="center">
          <Text.Text status="disabled">Add data to read a topic or a tag</Text.Text>
        </Flex.Box>
      )}
    </Task.Views.Panes>
  );
};

/** The plan of each enabled tag, by entry key. */
const TagPlansContext = createContext<
  ReadonlyMap<string, Task.ReadChannelPlan | undefined>
>(new Map());
TagPlansContext.displayName = "MQTT.TagPlansContext";

/** The plan of a message or a tag, in the plan of the whole task. */
type SourcePlan =
  | { type: "message"; plan: ICD.MessagePlan }
  | { type: "tag"; plan: Task.ReadChannelPlan };

interface PlansProps extends PropsWithChildren {
  device: Device.Device;
}

/**
 * Plans the channels of every message and tag that the task reads, together and as
 * deploying plans them, for the tree and the panes.
 */
const Plans = ({ device, children }: PlansProps): ReactElement => {
  const entries = PForm.useFieldValue(root.config.entries);
  const messages = ICD.useMessages();
  const enabled = entries.filter(({ disabled }) => !disabled);
  const bound = enabled.filter(isPlain).flatMap((source) =>
    source.messages.flatMap((read, r) => {
      const message = ICD.messageOf(messages, source, r);
      if (message == null) return [];
      const fields = ICD.readFields(ICD.jsonFields(message), read);
      return [{ source, read: r, message, fields }];
    }),
  );
  const tags = enabled.filter((e) => e.type === "sparkplug");
  const stored = splitStored(device.properties.read, entries);
  const sources = [
    ...bound.map((b) => ({
      key: b.source.key,
      position: b.read,
      topic: b.source.topic,
      read: b.source.messages[b.read],
      message: b.message,
      prefix: namePrefix(device.name, b.source.topic),
      props: ICD.Stored.of(stored, storedSource(b.source), b.read),
    })),
    ...tags.map((entry) => ({
      entry,
      name: device.name,
      props: device.properties.read[sparkplugPropertiesKey(entry)],
    })),
  ];
  const plans = Task.useChannelPlans<SourcePlan>(sources, async (client, signal) => {
    const plan = await planRead(client, device, bound, tags, entries, signal);
    return [
      ...plan.messages.map((p): SourcePlan => ({ type: "message", plan: p })),
      ...plan.tags.map((p): SourcePlan => ({ type: "tag", plan: p })),
    ];
  });
  const reads = bound.map(({ source, read }, i): ICD.PlannedRead => {
    const p = plans[i];
    return {
      path: plainPath(source.key),
      read,
      plan: p?.type === "message" ? p.plan : undefined,
    };
  });
  const tagPlans = new Map(
    tags.map((entry, i) => {
      const p = plans[bound.length + i];
      return [entry.key, p?.type === "tag" ? p.plan : undefined];
    }),
  );
  return (
    <ICD.Plans plans={reads}>
      <TagPlansContext value={tagPlans}>{children}</TagPlansContext>
    </ICD.Plans>
  );
};

const Content = ({ device }: PlatformDevice.TaskFormContentProps<Device.Device>) => {
  const entries = PForm.useFieldValue(root.config.entries);
  const [selected, setSelected] = useState<string[]>([]);
  const [expanded, setExpanded] = useState<string[]>(() =>
    ICD.Tree.initialExpanded(entries),
  );
  const made = useRef(new Set<string>());
  const plain = entries.filter(isPlain);
  const resolve = (entry: ReadEntry) => {
    if (entry.type !== "sparkplug") return {};
    const stored = device.properties.read[sparkplugPropertiesKey(entry)];
    return { channel: stored?.channels[""] ?? 0 };
  };
  const body = (
    <Plans device={device}>
      <Entries
        device={device}
        selected={selected}
        onSelect={setSelected}
        expanded={expanded}
        onExpand={setExpanded}
        made={made}
      />
      {plain.flatMap(({ key, messages }) =>
        messages.map(({ message }, read) => (
          <ICD.SuitRead key={`${key}/${message}`} path={plainPath(key)} read={read} />
        )),
      )}
    </Plans>
  );
  return (
    <>
      {unique.unique(plain.map((e) => e.icd)).reduceRight(
        (child, key) => (
          <ICD.WithFormat key={key} icdKey={key}>
            {child}
          </ICD.WithFormat>
        ),
        body,
      )}
      <Task.BindChannels path={root.config.entries} resolve={resolve} />
    </>
  );
};

const Properties = () => <Device.Select />;

const Form = PlatformDevice.wrapTaskForm({
  use: Device.use,
  useConfigure: Device.useConnectModal,
  useNoneSelected: Device.useNoneSelected,
  Content,
});

/** The channels that deploying gives the messages and tags that a task reads. */
interface ReadPlan {
  /** The plan of each message, in the order that planRead takes them. */
  messages: ICD.MessagePlan[];
  /** The plan of each tag, in the order that planRead takes them. */
  tags: Task.ReadChannelPlan[];
}

/**
 * Plans the channels of each message and then each tag, in deploy order with one
 * namer. A message plans with the channels that entries bind and that earlier messages
 * keep, so it leaves a stored channel that another binding writes. Reads from the Core
 * alone, and changes no argument. Throws between sources once signal aborts.
 */
const planRead = async (
  client: Synnax,
  dev: Pick<Device.Device, "name" | "properties">,
  bound: ICD.BoundMessage<PlainReadEntry>[],
  tags: SparkplugReadEntry[],
  entries: ReadEntry[],
  signal?: AbortSignal,
): Promise<ReadPlan> => {
  const namer = Task.createNamer(client);
  const kept = writtenChannels(entries);
  const stored = splitStored(dev.properties.read, entries);
  const messages: ICD.MessagePlan[] = [];
  for (const b of bound) {
    signal?.throwIfAborted();
    const { source: entry, read: r } = b;
    const plan = await ICD.planMessage(
      client,
      namer,
      b,
      namePrefix(dev.name, entry.topic),
      ICD.Stored.of(stored, storedSource(entry), r),
      kept,
      bound,
    );
    ICD.keptChannels(plan).forEach((ch) => kept.add(ch));
    messages.push(plan);
  }
  const plans: Task.ReadChannelPlan[] = [];
  for (const entry of tags) {
    signal?.throwIfAborted();
    const field = tagField(dev.name, entry);
    plans.push(
      await Task.planReadChannels({
        client,
        namer,
        props: dev.properties.read[sparkplugPropertiesKey(entry)],
        fields: [field],
        namePrefix: field.name,
        indexKey: "",
      }),
    );
  }
  return { messages, tags: plans };
};

/** Gives a tag its channel, as plan plans it. */
const applyTag = async (
  client: Synnax,
  dev: Device.Device,
  entry: SparkplugReadEntry,
  plan: Task.ReadChannelPlan,
): Promise<void> => {
  const field = tagField(dev.name, entry);
  await Task.applyReadChannels(client, plan, [field], {
    read: dev.properties.read,
    key: sparkplugPropertiesKey(entry),
  });
  entry.channel = field.channel;
};

/**
 * Gives each field that a message reads its channel, as plan plans it, and stores the
 * channels under the message.
 */
const applyMessage = async (
  client: Synnax,
  dev: Device.Device,
  bound: ICD.BoundMessage<PlainReadEntry>,
  plan: ICD.MessagePlan,
): Promise<void> => {
  const { source: entry, read: r } = bound;
  await ICD.applyMessage(client, bound, plan, {
    read: dev.properties.read,
    key: ICD.Stored.key(entry.topic, entry.messages[r].message),
  });
};

/** Gives each field and tag that an enabled entry reads a channel, and binds it. */
const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (client, config) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: Device.SCHEMAS,
  });
  const enabled = config.entries.filter(({ disabled }) => !disabled);
  const topics = enabled.filter(isPlain);
  const messages = await ICD.retrieveMessages(
    client,
    topics.map((e) => e.icd),
  );
  // Every format is checked before any channel is created.
  const bound = topics.flatMap((entry) =>
    entry.messages.map((_, read) =>
      ICD.boundMessage(messages, entry, read, entry.topic),
    ),
  );
  const before = deep.copy(dev.properties);
  try {
    // Each entry gets its share of the channels stored under its topic, even when it is
    // disabled or a create below fails.
    dev.properties.read = splitStored(dev.properties.read, config.entries);
    const tags = enabled.filter((e) => e.type === "sparkplug");
    const plan = await planRead(client, dev, bound, tags, config.entries);
    for (const [i, message] of bound.entries())
      await applyMessage(client, dev, message, plan.messages[i]);
    for (const [i, entry] of tags.entries())
      await applyTag(client, dev, entry, plan.tags[i]);
  } finally {
    await PlatformDevice.setProperties(client, dev, before, Device.SCHEMAS);
  }
  return [config, dev.rack];
};

export const Read = Task.wrapForm({
  Properties,
  Form,
  schemas: READ_SCHEMAS,
  deployConfigZ: deployReadConfigZ,
  type: "mqtt_read",
  onConfigure,
});
