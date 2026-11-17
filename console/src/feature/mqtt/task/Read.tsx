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
  channel,
  DisconnectedError,
  icd,
  mqtt,
  type Synnax,
  ValidationError,
} from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree } from "@synnaxlabs/lyra/tree";
import { ICD as PICD, Synnax as PSynnax } from "@synnaxlabs/pluto";
import {
  DataType,
  deep,
  type payload,
  primitive,
  type record,
  unique,
  uuid,
} from "@synnaxlabs/x";
import {
  type Dispatch,
  type FC,
  type SetStateAction,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";

import { use } from "@/feature/mqtt/device/queries";
import { Select as SelectDevice, useNoneSelected } from "@/feature/mqtt/device/Select";
import { type Device, MAKE, SCHEMAS } from "@/feature/mqtt/device/types";
import { useConnectModal } from "@/feature/mqtt/device/useConnectModal";
import { createDiscoverModal, type Kind } from "@/feature/mqtt/task/discover/Discover";
import { createSparkplugEntry, namePrefix } from "@/feature/mqtt/task/entries";
import { EntryLabel } from "@/feature/mqtt/task/EntryLabel";
import { type Leaf } from "@/feature/mqtt/task/leaves";
import { QoSField } from "@/feature/mqtt/task/QoSField";
import {
  sparkplugChannelName,
  sparkplugPropertiesKey,
} from "@/feature/mqtt/task/sparkplug";
import { SparkplugTagFields } from "@/feature/mqtt/task/SparkplugTagFields";
import {
  deployReadConfigZ,
  type PlainReadEntry,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadBinding,
  type ReadEntry,
  type ReadSchemas,
  type SparkplugReadEntry,
} from "@/feature/mqtt/task/types";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { ICD } from "@/platform/icd";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<ReadSchemas>>();

const DEFAULT_NAME = "MQTT read task";

const plainPath = (key: string) =>
  deep.variant(root.config.entries[key], "type", "plain");

/** The open form of one ICD. */
interface Format {
  key: icd.Key;
  form: ICD.FormatForm;
}

/** The channels a device stores for the topics it reads, by topic. */
type Stored = Device["properties"]["read"];

/** The field kind that a browsed sample of each JSON type starts as. */
const SAMPLE_KINDS = {
  number: "float",
  boolean: "boolean",
  string: "string",
} as const satisfies Record<payload.JSONType, payload.Kind>;

const TOPIC_INPUT_PROPS = { placeholder: "plant/line1/telemetry" } as const;

/** @returns a message that the entry reads, from the entry's own ICD. */
const messageOf = (
  messages: ICD.Messages,
  entry: PlainReadEntry,
  read: number,
): icd.Message | undefined =>
  messages.get(entry.icd)?.find(({ key }) => key === entry.messages.at(read)?.message);

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
  /** The keys of the fields that the task reads. */
  bound: string[];
}

/** What the discovery dialog adds: topics to read and Sparkplug B tags. */
type Addition = Draft | SparkplugReadEntry;

const isDraft = (addition: Addition): addition is Draft => addition.type === "plain";

/** Names a field by its pointer: "/motor/speed" is "motor_speed". */
const fieldName = (pointer: string): string =>
  pointer === "" ? "value" : pointer.slice(1).replaceAll("/", "_");

const toJSONField = (leaf: Leaf): icd.JSONField =>
  icd.jsonFieldZ.parse({
    name: fieldName(leaf.pointer),
    pointer: leaf.pointer,
    type: SAMPLE_KINDS[leaf.type],
  });

const createDraft = (topic: string, leaves: Leaf[] = []): Draft => {
  const fields = leaves.map(toJSONField);
  // A disabled leaf would mix a variable-length channel among fixed ones.
  const bound = fields.filter((_, i) => !leaves[i].disabled).map(({ key }) => key);
  return { type: "plain", topic, fields, bound };
};

const DISCOVER_KIND: Kind<Addition> = {
  title: "Add data",
  hint: "Select the values to read. Each one becomes a field of the message format.",
  noun: "field",
  tagNoun: "tag",
  create: ({ topics, tags }) => [
    ...[...topics].map(([topic, { leaves, pointers }]) =>
      createDraft(
        topic,
        leaves.filter((l) => pointers.has(l.pointer)),
      ),
    ),
    ...[...tags.values()].map((t) => createSparkplugEntry(t)),
  ],
  createTopic: (topic) => createDraft(topic),
  createTag: () => createSparkplugEntry(),
};

const useDiscoverModal = createDiscoverModal(DISCOVER_KIND);

/** @returns the channels that the bindings and tags of entries write. */
const writtenChannels = (entries: ReadEntry[]): Set<channel.Key> => {
  const written = new Set(
    entries.flatMap((e) =>
      isPlain(e)
        ? e.messages.flatMap(({ bindings }) => bindings.map((b) => b.channel))
        : [e.channel],
    ),
  );
  written.delete(0);
  return written;
};

/** @returns the key of the device properties that store the channels of a message. */
const storedKey = (topic: string, message: icd.MessageKey): string =>
  `${topic}#${message}`;

/**
 * @returns the channels that the device stores for read r of the entry. An older
 * Console stored the channels of a topic under the bare topic. They belong to the
 * message that binds one of them, or else to the first message.
 */
const storedOf = (
  stored: Stored,
  { topic, messages }: Pick<PlainReadEntry, "topic" | "messages">,
  r: number,
): Stored[string] | undefined => {
  const own = stored[storedKey(topic, messages[r].message)];
  if (own != null) return own;
  const legacy = stored[topic];
  if (legacy == null) return undefined;
  const channels = new Set([legacy.index, ...Object.values(legacy.channels)]);
  channels.delete(0);
  const owner = messages.findIndex(({ bindings }) =>
    bindings.some(({ channel }) => channels.has(channel)),
  );
  return Math.max(owner, 0) === r ? legacy : undefined;
};

/**
 * @returns the channel that the device stores for each pointer under any message of the
 * topic. The channels of the message itself come first.
 */
const topicChannels = (
  stored: Stored,
  topic: string,
  message: icd.MessageKey,
): Record<string, channel.Key> => {
  const channels: Record<string, channel.Key> = {};
  for (const [key, props] of Object.entries(stored))
    if (key === topic || key.startsWith(`${topic}#`))
      Object.assign(channels, props.channels);
  return { ...channels, ...stored[storedKey(topic, message)]?.channels };
};

/**
 * The prefix of the names of the channels that deploying creates for a message. A
 * message after the first adds its name to the prefix of the topic.
 */
const messagePrefix = (
  deviceName: string,
  topic: string,
  read: number,
  name: string,
): string => {
  const prefix = namePrefix(deviceName, topic);
  return read === 0 ? prefix : `${prefix}_${channel.escapeInvalidName(name)}`;
};

/**
 * @returns the bindings of fields, each to the channel stored for its pointer. A field
 * whose stored channel is in written gets no channel, as the Core refuses a channel
 * that two bindings write. Adds the channels it binds to written.
 */
const bindingsOf = (
  stored: Record<string, channel.Key>,
  fields: icd.JSONField[],
  written: Set<channel.Key>,
): ReadBinding[] =>
  fields.map(({ key: field, pointer }) => {
    const ch = stored[pointer] ?? 0;
    if (written.has(ch)) return mqtt.readBindingZ.parse({ field });
    if (ch !== 0) written.add(ch);
    return mqtt.readBindingZ.parse({ field, channel: ch });
  });

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
  const appended = fields.filter(({ pointer }) => !pointers.has(pointer));
  let after = held.at(-1)?.key;
  for (const field of appended) {
    to.insert(ICD.fieldsPath(message.key), field, after);
    after = field.key;
  }
  return appended;
};

const createMessage = (topic: string, fields: icd.JSONField[]): icd.Message => ({
  key: uuid.create(),
  name: topic === "" ? "Untitled" : topic,
  payload: { format: "json", fields },
});

const entryOf = (
  icdKey: icd.Key,
  topic: string,
  read: Pick<mqtt.ReadMessage, "message" | "bindings">,
): PlainReadEntry =>
  mqtt.plainReadEntryZ.parse({ type: "plain", icd: icdKey, topic, messages: [read] });

interface AddTopicsResult {
  /** The entries of the topics that the task does not read yet. */
  added: PlainReadEntry[];
  /**
   * The bindings that each held entry gains, by entry key. An entry whose message is
   * gone reads the new one.
   */
  held: { key: string; message?: icd.MessageKey; bindings: ReadBinding[] }[];
}

/** @returns the bindings of the fields of a draft that the message reads. */
type Bind = (
  draft: Draft,
  fields: icd.JSONField[],
  message: icd.MessageKey,
) => ReadBinding[];

/** @returns the name of the ICD of a task named name. */
const icdName = (name: string): string => (name.trim() === "" ? DEFAULT_NAME : name);

/** Creates an ICD with a message for each draft, and the entries that read them. */
const createICD = async (
  client: Synnax,
  name: string,
  drafts: Draft[],
  bind: Bind,
): Promise<AddTopicsResult> => {
  const messages = drafts.map(({ topic, fields }) => createMessage(topic, fields));
  const { key } = await client.icds.create({ name, messages });
  const added = drafts.map((d, i) =>
    entryOf(key, d.topic, {
      message: messages[i].key,
      bindings: bind(d, d.fields, messages[i].key),
    }),
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
    const pending = actions.get(key) ?? [];
    actions.set(key, pending);
    return {
      insert: (path, value, after) =>
        pending.push(icd.insert({ path: String(path).split("."), value, after })),
    };
  };
  const result: AddTopicsResult = { added: [], held: [] };
  format.form.batch(() => {
    for (const draft of drafts) {
      const entry = entries.find((e) => e.topic === draft.topic);
      const key = entry?.icd ?? format.key;
      const message = entry == null ? undefined : messageOf(messages, entry, 0);
      if (entry != null && message != null) {
        const appended = appendFields(inserterOf(key), message, draft.fields);
        result.held.push({
          key: entry.key,
          bindings: bind(draft, appended, message.key),
        });
        continue;
      }
      const prior = messages.get(key) ?? [];
      const created = createMessage(draft.topic, draft.fields);
      inserterOf(key).insert(PICD.formPath.messages, created, prior.at(-1)?.key);
      messages.set(key, [...prior, created]);
      const bindings = bind(draft, draft.fields, created.key);
      if (entry == null)
        result.added.push(
          entryOf(key, draft.topic, { message: created.key, bindings }),
        );
      else result.held.push({ key: entry.key, message: created.key, bindings });
    }
  });
  for (const [key, pending] of actions) await client.icds.dispatch(key, pending);
  return result;
};

/** @returns the prefix of the channel names of a message, current with its name. */
const usePrefix = (
  deviceName: string,
  path: deep.Path<PlainReadEntry>,
  read: number,
): string => {
  const formats = ICD.useFormats();
  const topic = PForm.useFieldValue(path.topic);
  const icdKey = PForm.useFieldValue(path.icd);
  const message = PForm.useFieldValue(path.messages[read].message);
  const name = PForm.useFieldValue(PICD.formPath.messages[message].name, {
    optional: true,
    ctx: formats.get(icdKey),
  });
  return messagePrefix(deviceName, topic, read, name ?? "");
};

/** @returns the messages of each open ICD, by ICD key. */
const messagesOf = (formats: ICD.Formats): ICD.Messages =>
  new Map([...formats].map(([key, form]) => [key, form.value().messages]));

/**
 * Starts or stops reading a field of a message. Starting reuses the channel that the
 * device stores for the field's pointer, unless another field reads it.
 */
const setRead = (
  ctx: PForm.ContextValue,
  entry: string,
  read: number,
  field: icd.JSONField,
  on: boolean,
  stored: Stored,
): void => {
  const path = plainPath(entry).messages[read];
  const bindings = ctx.get(path.bindings).value;
  const bound = bindings.some(({ field: key }) => key === field.key);
  if (bound === on || (on && !ICD.isReadable(field))) return;
  const channels = storedOf(stored, ctx.get(plainPath(entry)).value, read)?.channels;
  const written = writtenChannels(ctx.get(root.config.entries).value);
  ctx.set(
    path.bindings,
    on
      ? [...bindings, ...bindingsOf(channels ?? {}, [field], written)]
      : bindings.filter(({ field: k }) => k !== field.key),
  );
};

interface EntryNode {
  kind: "entry";
  entry: string;
}

/** A message of a topic that reads several, in the tree under its topic. */
interface MessageNode {
  kind: "message";
  /** The key of the topic's entry. */
  entry: string;
  /** The index of the message in the topic's reads. */
  read: number;
}

/** A field of a message, in the tree under its message, or under its topic. */
interface FieldNode {
  kind: "field";
  /** The key of the topic's entry. */
  entry: string;
  /** The index of the message in the topic's reads. */
  read: number;
  field: icd.JSONField;
}

type Node = EntryNode | MessageNode | FieldNode;

interface Index {
  nodes: Tree.Node[];
  byKey: Map<string, Node>;
}

const messageNodeKey = (entry: string, message: icd.MessageKey): string =>
  `${entry}/${message}`;

const fieldNodeKey = (entry: string, message: icd.MessageKey, field: string): string =>
  `${entry}/${message}/${field}`;

/** Each entry key with the message and shown fields of each read, or null for a tag. */
type Structure = [string, [icd.MessageKey, icd.JSONField[]][] | null][];

/**
 * The entries of the task as tree nodes. A topic holds the fields of its message, or a
 * node for each message when it reads several. A tag stands alone. The index field
 * stays hidden, as the timestamp section edits it.
 */
const useIndex = (formats: ICD.Formats): Index => {
  const { bind, get } = PForm.useContext();
  const subscribe = (onChange: () => void) => {
    const unbinds = [
      bind(onChange),
      ...[...formats.values()].map((f) => f.bind(onChange)),
    ];
    return () => unbinds.forEach((unbind) => unbind());
  };
  const getStructure = () =>
    JSON.stringify(
      get(root.config.entries).value.map((e): Structure[number] => [
        e.key,
        isPlain(e)
          ? e.messages.map(({ message, index }) => [
              message,
              ICD.fieldsOf(formats, e.icd, message).filter(({ key }) => key !== index),
            ])
          : null,
      ]),
    );
  const structure = useSyncExternalStore(subscribe, getStructure, getStructure);
  const byKey = new Map<string, Node>();
  const nodes = (JSON.parse(structure) as Structure).map(
    ([entry, reads]): Tree.Node => {
      byKey.set(entry, { kind: "entry", entry });
      if (reads == null) return { key: entry };
      const fieldNodes = (
        read: number,
        message: icd.MessageKey,
        fields: icd.JSONField[],
      ): Tree.Node[] =>
        fields.map((field) => {
          const key = fieldNodeKey(entry, message, field.key);
          byKey.set(key, { kind: "field", entry, read, field });
          return { key };
        });
      if (reads.length === 1)
        return { key: entry, children: fieldNodes(0, ...reads[0]) };
      const children = reads.map(([message, fields], read): Tree.Node => {
        const key = messageNodeKey(entry, message);
        byKey.set(key, { kind: "message", entry, read });
        return { key, children: fieldNodes(read, message, fields) };
      });
      return { key: entry, children };
    },
  );
  return { nodes, byKey };
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

interface MessageItemProps extends Tree.ItemRenderProps<string> {
  node: MessageNode;
}

const MessageItem = ({ node: { entry, read }, ...props }: MessageItemProps) => (
  <Task.Views.TreeGroupItem
    {...props}
    label={<ICD.MessageLabel path={plainPath(entry)} read={read} />}
  />
);

interface FieldItemProps extends Tree.ItemRenderProps<string> {
  node: FieldNode;
  stored: Stored;
}

const FieldItem = ({
  node: { entry, read, field },
  stored,
  ...props
}: FieldItemProps) => {
  const ctx = PForm.useContext();
  const path = plainPath(entry).messages[read];
  const bindings = PForm.useFieldValue(path.bindings);
  const entryOff = PForm.useFieldValue(plainPath(entry).disabled);
  const i = bindings.findIndex(({ field: key }) => key === field.key);
  return (
    <Task.Views.TreeFieldItem
      {...props}
      label={field.name}
      emptyLabel="Unnamed field"
      channel={i < 0 ? null : path.bindings[i].channel}
      off={i < 0}
      skipped={i < 0 || entryOff}
    >
      <ICD.ReadCheckbox
        bindings={path.bindings}
        field={field}
        onChange={(on) => setRead(ctx, entry, read, field, on, stored)}
      />
    </Task.Views.TreeFieldItem>
  );
};

/** @returns the props that the panels of a message of a topic share. */
const useMessageProps = (
  device: Device,
  path: deep.Path<PlainReadEntry>,
  read: number,
): Omit<ICD.MessageSectionsProps, "format"> => {
  const topic = PForm.useFieldValue(path.topic);
  const messages = PForm.useFieldValue(path.messages);
  const prefix = usePrefix(device.name, path, read);
  return {
    path,
    read,
    prefix,
    indexChannel:
      storedOf(device.properties.read, { topic, messages }, read)?.index ?? 0,
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
  device: Device;
}

const MessageSections = ({ device, path, read, format }: MessagePanelProps) => {
  const props = useMessageProps(device, path, read);
  return <ICD.MessageSections {...props} format={format} />;
};

interface MessageDetailsProps extends MessagePanelProps {
  onAddField: () => void;
}

const MessageDetails = ({
  device,
  path,
  read,
  format,
  onAddField,
}: MessageDetailsProps) => {
  const props = useMessageProps(device, path, read);
  return (
    <ICD.MessageDetails
      {...props}
      format={format}
      matchTitle="Documents"
      onAddField={onAddField}
    />
  );
};

interface FieldDetailsProps extends MessagePanelProps {
  field: icd.JSONField;
}

const FieldDetails = ({ device, ...rest }: FieldDetailsProps) => {
  const prefix = usePrefix(device.name, rest.path, rest.read);
  return <ICD.FieldDetails {...rest} prefix={prefix} />;
};

interface TopicDetailsProps {
  path: deep.Path<PlainReadEntry>;
  format: ICD.FormatForm;
  device: Device;
  /** Adds a field to the message of a topic that reads one. */
  onAddField: () => void;
  /** Adds a message to the topic, for documents that its messages do not cover. */
  onAddMessage: () => void;
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
}) => {
  const isPreview = Task.useIsPreview();
  const reads = PForm.useFieldValue(path.messages).length;
  return (
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      <PForm.Sections>
        <PForm.Section
          title="Subscription"
          actions={
            !isPreview && (
              <>
                {reads === 1 && <ICD.AddField onClick={onAddField} />}
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
            inputProps={TOPIC_INPUT_PROPS}
          />
          <QoSField path={path.qos} />
          <PForm.CheckboxField
            path={path.retainedIgnored}
            label="Ignore retained"
            padHelpText={false}
          />
        </PForm.Section>
        {reads === 1 && (
          <MessageSections path={path} read={0} format={format} device={device} />
        )}
      </PForm.Sections>
    </Flex.Box>
  );
};

// A Sparkplug B DateTime tag reads into a timestamp channel.
const SPARKPLUG_HIDDEN_DATA_TYPES = [DataType.UUID, DataType.JSON, DataType.BYTES];

interface TagDetailsProps {
  path: deep.Path<SparkplugReadEntry>;
  deviceName: string;
}

const TagDetails: FC<TagDetailsProps> = ({ path, deviceName }) => {
  const entry = PForm.useFieldValue(path);
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
            defaultName={sparkplugChannelName(deviceName, entry)}
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

/**
 * @returns the expanded state of a task's tree as it opens: every topic and tag, and
 * every message of a topic that reads several.
 */
const initialExpanded = (entries: ReadEntry[]): string[] =>
  entries.flatMap((e) =>
    isPlain(e) && e.messages.length > 1
      ? [e.key, ...e.messages.map(({ message }) => messageNodeKey(e.key, message))]
      : [e.key],
  );

interface EntriesProps {
  device: Device;
  /** The key of each selected tree item. */
  selected: string[];
  onSelect: Dispatch<SetStateAction<string[]>>;
  expanded: string[];
  onExpand: Dispatch<SetStateAction<string[]>>;
}

/** The entries of the task as a tree, beside the details of the selected item. */
const Entries: FC<EntriesProps> = ({
  device,
  selected,
  onSelect,
  expanded,
  onExpand,
}) => {
  const client = PSynnax.use();
  const ctx = PForm.useContext();
  const handleError = Status.useErrorHandler();
  const discover = useDiscoverModal();
  const formats = ICD.useFormats();
  const { push, remove } = PForm.useFieldList(root.config.entries);
  const entries = PForm.useFieldValue(root.config.entries);
  const stored = device.properties.read;
  const { nodes, byKey } = useIndex(formats);
  const first = entries.find(isPlain);
  const format = first == null ? undefined : formats.get(first.icd);
  const tree = Tree.use({
    nodes,
    selected,
    onSelectedChange: onSelect,
    expanded,
    onExpandedChange: onExpand,
    toggleOn: "caret",
  });

  // Adds run one at a time, so a second click cannot make a second ICD.
  const adding = useRef(false);
  const add = async () => {
    if (client == null) throw new DisconnectedError();
    const plain = entries.filter(isPlain);
    const messages = messagesOf(formats);
    const held = entries.map((e): Addition =>
      isPlain(e)
        ? { ...createDraft(e.topic), fields: ICD.jsonFields(messageOf(messages, e, 0)) }
        : e,
    );
    const additions = await discover({ device, entries: held });
    if (additions == null || additions.length === 0) return;
    const drafts = additions.filter(isDraft);
    const tags = additions.filter((a): a is SparkplugReadEntry => !isDraft(a));
    const written = writtenChannels(entries);
    const bind: Bind = ({ topic, bound }, fields, message) =>
      bindingsOf(
        topicChannels(stored, topic, message),
        fields.filter(({ key }) => bound.includes(key)),
        written,
      );
    const { added, held: grown } =
      drafts.length === 0
        ? { added: [], held: [] }
        : first == null || format == null
          ? await createICD(client, icdName(ctx.get(root.name).value), drafts, bind)
          : await addTopics({
              client,
              format: { key: first.icd, form: format },
              entries: plain,
              drafts,
              bind,
            });
    push([...added, ...tags]);
    grown.forEach(({ key, message, bindings }) => {
      const read = plainPath(key).messages[0];
      // The index and bindings of a deleted message name fields it no longer has.
      if (message != null)
        ctx.set(read, mqtt.readMessageZ.parse({ message, bindings }));
      else ctx.set(read.bindings, [...ctx.get(read.bindings).value, ...bindings]);
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
  const handleAdd = () =>
    handleError(async () => {
      if (adding.current) return;
      adding.current = true;
      await add().finally(() => {
        adding.current = false;
      });
    }, "Failed to add data");

  /**
   * Adds a message with no fields to the topic and to its ICD. The match tests the
   * pointer of the topic's other matches, so the messages tell documents apart.
   */
  const handleAddMessage = (entryKey: string) => {
    const path = plainPath(entryKey);
    const entry = ctx.get(path).value;
    const form = ICD.formatOf(formats, entry.icd);
    const held = form.value().messages;
    const rule = entry.messages
      .map(({ message }) => held.find(({ key }) => key === message)?.payload)
      .find((p) => p?.format === "json" && p.match != null);
    const match = rule?.format === "json" ? rule.match : undefined;
    const message: icd.Message = {
      key: uuid.create(),
      name: nextMessageName(entry, held),
      payload: {
        format: "json",
        match: ICD.createMatch(match?.type ?? "string", match?.pointer ?? ""),
        fields: [],
      },
    };
    form.insert(PICD.formPath.messages, message, held.at(-1)?.key);
    ctx.set(path.messages, [
      ...entry.messages,
      mqtt.readMessageZ.parse({ message: message.key }),
    ]);
    const shown = [...entry.messages.map((m) => m.message), message.key];
    onExpand((prev) =>
      unique.unique([
        ...prev,
        entryKey,
        ...shown.map((m) => messageNodeKey(entryKey, m)),
      ]),
    );
    onSelect([messageNodeKey(entryKey, message.key)]);
  };

  /** Adds a field to a message of a topic and to its ICD, reads it, and selects it. */
  const handleAddField = (entryKey: string, read: number) => {
    const entry = ctx.get(plainPath(entryKey)).value;
    const { message } = entry.messages[read];
    const fields = ICD.fieldsOf(formats, entry.icd, message);
    const field = ICD.createJSONField(fields.map(({ name }) => name));
    ICD.formatOf(formats, entry.icd).insert(
      ICD.fieldsPath(message),
      field,
      fields.at(-1)?.key,
    );
    setRead(ctx, entryKey, read, field, true, stored);
    const parent = entry.messages.length > 1 ? [messageNodeKey(entryKey, message)] : [];
    onExpand((prev) => unique.unique([...prev, entryKey, ...parent]));
    onSelect([fieldNodeKey(entryKey, message, field.key)]);
  };

  /** Removes each chosen field from its message in the ICD. */
  const handleRemoveFields = (chosen: FieldNode[]) => {
    for (const { entry, read, field } of chosen) {
      const { icd: icdKey, messages } = ctx.get(plainPath(entry)).value;
      ICD.formatOf(formats, icdKey).remove(
        ICD.fieldsPath(messages[read].message)[field.key],
      );
    }
    onSelect([]);
  };

  const handleSelect: Tree.TreeProps<string, record.Keyed<string>>["onSelect"] = (
    keys,
    { clicked },
  ) => {
    onSelect(keys);
    const kind = clicked == null ? undefined : byKey.get(clicked)?.kind;
    if (clicked != null && kind !== "field") tree.expand(clicked);
  };

  /** Sets whether the task reads each chosen field, and enables each chosen entry. */
  const setEnabled = (nodes: (EntryNode | FieldNode)[], enabled: boolean) =>
    ctx.batch(() => {
      for (const node of nodes)
        if (node.kind === "entry")
          ctx.set(root.config.entries[node.entry].disabled, !enabled);
        else setRead(ctx, node.entry, node.read, node.field, enabled, stored);
    });
  const isEnabled = (node: EntryNode | FieldNode): boolean => {
    if (node.kind === "entry")
      return !ctx.get(root.config.entries[node.entry].disabled).value;
    return ctx
      .get(plainPath(node.entry).messages[node.read].bindings)
      .value.some(({ field }) => field === node.field.key);
  };
  /** Removes each chosen entry, and each chosen message from its topic. */
  const handleRemove = (chosen: (EntryNode | MessageNode)[]) => {
    const removed = new Set(
      chosen.flatMap((n) => (n.kind === "entry" ? [n.entry] : [])),
    );
    const dropped = new Map<string, Set<number>>();
    for (const n of chosen)
      if (n.kind === "message" && !removed.has(n.entry))
        dropped.set(n.entry, (dropped.get(n.entry) ?? new Set()).add(n.read));
    ctx.batch(() => {
      dropped.forEach((reads, entry) => {
        const path = plainPath(entry).messages;
        ctx.set(
          path,
          ctx.get(path).value.filter((_, r) => !reads.has(r)),
        );
      });
      if (removed.size > 0) remove([...removed]);
    });
    onSelect([]);
  };
  /**
   * @returns whether the chosen nodes can go: entries, and messages that leave each of
   * their topics at least one.
   */
  const isRemovable = (chosen: Node[]): chosen is (EntryNode | MessageNode)[] => {
    if (chosen.some(({ kind }) => kind === "field")) return false;
    const counts = new Map<string, number>();
    for (const n of chosen)
      if (n.kind === "message") counts.set(n.entry, (counts.get(n.entry) ?? 0) + 1);
    return [...counts].every(
      ([entry, count]) =>
        chosen.some((n) => n.kind === "entry" && n.entry === entry) ||
        count < ctx.get(plainPath(entry).messages).value.length,
    );
  };
  const contextMenu = ({ keys }: Menu.ContextMenuMenuProps) => {
    const chosen = keys.flatMap((k) => byKey.get(k) ?? []);
    const switchable = chosen.filter(
      (n): n is EntryNode | FieldNode => n.kind !== "message",
    );
    const [only] = chosen;
    const topic =
      chosen.length === 1 &&
      only.kind === "entry" &&
      ctx.get(root.config.entries[only.entry].type).value === "plain"
        ? only.entry
        : undefined;
    let addField: (() => void) | undefined;
    if (chosen.length === 1 && only.kind === "message")
      addField = () => handleAddField(only.entry, only.read);
    else if (topic != null && ctx.get(plainPath(topic).messages).value.length === 1)
      addField = () => handleAddField(topic, 0);
    const fields = chosen.filter((n): n is FieldNode => n.kind === "field");
    let onRemove: (() => void) | undefined;
    if (fields.length > 0 && fields.length === chosen.length)
      onRemove = () => handleRemoveFields(fields);
    else if (isRemovable(chosen)) onRemove = () => handleRemove(chosen);
    return (
      <Task.Views.ContextMenu
        keys={keys}
        onRemove={onRemove}
        onEnable={
          switchable.some((n) => !isEnabled(n))
            ? () => setEnabled(switchable, true)
            : undefined
        }
        onDisable={
          switchable.some(isEnabled) ? () => setEnabled(switchable, false) : undefined
        }
        onAddField={addField}
        extraMenuItems={
          topic != null && (
            <Menu.Item itemKey="addMessage" onClick={() => handleAddMessage(topic)}>
              <Icon.Add />
              Add message
            </Menu.Item>
          )
        }
      />
    );
  };

  const renderItem = ({ key, ...p }: Tree.ItemRenderProps<string>) => {
    const node = byKey.get(p.itemKey);
    if (node?.kind === "field")
      return <FieldItem key={key} {...p} node={node} stored={stored} />;
    if (node?.kind === "message") return <MessageItem key={key} {...p} node={node} />;
    return <EntryItem key={key} {...p} />;
  };

  const current = selected.length > 0 ? byKey.get(selected[0]) : undefined;
  const entry =
    current == null ? undefined : entries.find(({ key }) => key === current.entry);
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
        onAddField={() => handleAddField(entry.key, current.read)}
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
          deviceName={device.name}
        />
      );
    else if (entryFormat != null)
      details = (
        <TopicDetails
          key={entry.key}
          path={plainPath(entry.key)}
          format={entryFormat}
          device={device}
          onAddField={() => handleAddField(entry.key, 0)}
          onAddMessage={() => handleAddMessage(entry.key)}
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

const Content = ({ device }: PlatformDevice.TaskFormContentProps<Device>) => {
  const entries = PForm.useFieldValue(root.config.entries);
  const [selected, setSelected] = useState<string[]>([]);
  const [expanded, setExpanded] = useState<string[]>(() => initialExpanded(entries));
  const plain = entries.filter(isPlain);
  const resolve = (entry: ReadEntry) => {
    if (entry.type !== "sparkplug") return {};
    const stored = device.properties.read[sparkplugPropertiesKey(entry)];
    return { channel: stored?.channels[""] ?? 0 };
  };
  const body = (
    <>
      <Entries
        device={device}
        selected={selected}
        onSelect={setSelected}
        expanded={expanded}
        onExpand={setExpanded}
      />
      {plain.flatMap(({ key, messages }) =>
        messages.map(({ message }, read) => (
          <ICD.SuitRead key={`${key}/${message}`} path={plainPath(key)} read={read} />
        )),
      )}
    </>
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

const Properties = () => <SelectDevice />;

const Form = PlatformDevice.wrapTaskForm({
  use,
  useConfigure: useConnectModal,
  useNoneSelected,
  Content,
});

const getInitialValues: Task.GetInitialValues<ReadSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = READ_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: DEFAULT_NAME, type: READ_TYPE, config: cfg };
};

/** A message that a topic reads, with the fields that the task reads of it. */
interface BoundMessage {
  entry: PlainReadEntry;
  /** The index of the message in the topic's reads. */
  read: number;
  message: icd.Message;
  fields: icd.JSONField[];
}

/**
 * @returns a message that the entry reads, with the fields that the task reads of it.
 * @throws {ValidationError} if the task cannot read the message.
 */
const boundMessage = (
  messages: ICD.Messages,
  entry: PlainReadEntry,
  read: number,
): BoundMessage => {
  const message = messageOf(messages, entry, read);
  const where =
    message != null && entry.messages.length > 1
      ? `${message.name} of ${entry.topic}`
      : entry.topic;
  if (message?.payload.format !== "json")
    throw new ValidationError(`The message format of ${where} is missing or not JSON`);
  const readMessage = entry.messages[read];
  const fields = ICD.readFields(message.payload.fields, readMessage);
  const index = fields.find(({ key }) => key === readMessage.index);
  if (index != null && !ICD.isIndexable(index))
    throw new ValidationError(
      `In ${where}, the index field ${index.name} must be a timestamp field with no default`,
    );
  if (fields.length === 0)
    throw new ValidationError(`The task reads no fields of ${where}`);
  const mixed = ICD.variableAmongFixed(fields);
  if (mixed.length > 0)
    throw new ValidationError(`In ${where}, ${ICD.mixMessage(mixed)}`);
  return { entry, read, message, fields };
};

/** Gives a tag a channel: the one it has, the one the device stores, or a new one. */
const configureTag = async (
  client: Synnax,
  dev: Device,
  entry: SparkplugReadEntry,
): Promise<void> => {
  const name = primitive.isNonZero(entry.name)
    ? entry.name
    : sparkplugChannelName(dev.name, entry);
  // A tag reads whole into one channel, as the field at the empty pointer.
  const field = {
    key: entry.key,
    pointer: "",
    channel: entry.channel,
    name,
    dataType: entry.dataType,
  };
  await Task.configureReadChannels({
    client,
    read: dev.properties.read,
    propertiesKey: sparkplugPropertiesKey(entry),
    fields: [field],
    namePrefix: name,
    indexKey: "",
  });
  entry.channel = field.channel;
};

/**
 * Gives each field that a message reads a channel: the one it has, the one the device
 * stores for its pointer unless written holds it, or a new one of the binding's data
 * type. Adds the channels it binds to written.
 */
const configureMessage = async (
  client: Synnax,
  dev: Device,
  { entry, read: r, message, fields }: BoundMessage,
  written: Set<channel.Key>,
): Promise<void> => {
  const read = entry.messages[r];
  const index = fields.find(({ key }) => key === read.index)?.key;
  const propertiesKey = storedKey(entry.topic, read.message);
  const props = storedOf(dev.properties.read, entry, r);
  if (props != null && dev.properties.read[propertiesKey] == null) {
    dev.properties.read[propertiesKey] = props;
    delete dev.properties.read[entry.topic];
  }
  const stored = props?.channels ?? {};
  const bindingOf = (f: icd.JSONField): ReadBinding =>
    read.bindings.find(({ field }) => field === f.key) ??
    mqtt.readBindingZ.parse({ field: f.key });
  // A field with no channel takes the one the device stores for its pointer, unless
  // another binding writes it.
  for (const f of fields)
    if (bindingOf(f).channel === 0 && written.has(stored[f.pointer]))
      delete stored[f.pointer];
  const reused = fields.map((f) => bindingOf(f).channel || (stored[f.pointer] ?? 0));
  const live = new Map(
    (await client.channels.retrieve(reused)).map((ch) => [ch.key, ch]),
  );
  const prefix = messagePrefix(dev.name, entry.topic, r, message.name);
  const channels = fields.map((f, i) => {
    const binding = bindingOf(f);
    const isIndex = f.key === index;
    const suited = binding.dataType !== "" && ICD.converts(f.type, binding.dataType);
    const dataType = suited ? binding.dataType : "";
    const ch = live.get(reused[i]);
    // An edit of the field's kind or index role since the last deploy needs a new
    // channel.
    const stale =
      ch != null &&
      (!ICD.converts(f.type, ch.dataType.toString()) || ch.isIndex !== isIndex);
    if (stale && stored[f.pointer] === ch.key) delete stored[f.pointer];
    return {
      key: f.key,
      pointer: f.pointer,
      name: `${prefix}_${channel.escapeInvalidName(f.name)}`,
      dataType: isIndex
        ? DataType.TIMESTAMP.toString()
        : ICD.bindingDataType({ dataType }, f.type),
      channel: stale ? 0 : binding.channel,
      binding: { ...binding, dataType },
    };
  });
  await Task.configureReadChannels({
    client,
    read: dev.properties.read,
    propertiesKey,
    fields: channels,
    namePrefix: prefix,
    indexKey: index ?? "",
  });
  channels.forEach(({ channel }) => written.add(channel));
  entry.messages[r] = {
    message: read.message,
    index,
    bindings: channels.map(({ channel, binding }) => ({ ...binding, channel })),
  };
};

/** Gives each field and tag that an enabled entry reads a channel, and binds it. */
const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (client, config) => {
  const dev = await client.devices.retrieve({ key: config.device, schemas: SCHEMAS });
  const enabled = config.entries.filter(({ disabled }) => !disabled);
  const topics = enabled.filter(isPlain);
  const messages = await ICD.retrieveMessages(
    client,
    topics.map((e) => e.icd),
  );
  // Every format is checked before any channel is created.
  const bound = topics.flatMap((entry) =>
    entry.messages.map((_, read) => boundMessage(messages, entry, read)),
  );
  const before = deep.copy(dev.properties);
  try {
    const written = writtenChannels(config.entries);
    for (const message of bound) await configureMessage(client, dev, message, written);
    for (const entry of enabled)
      if (entry.type === "sparkplug") await configureTag(client, dev, entry);
  } finally {
    await PlatformDevice.setProperties(client, dev, before, SCHEMAS);
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

export const useCreateRead = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const ReadSelectable: Selector.Selectable = {
  type: READ_TYPE,
  title: DEFAULT_NAME,
  label: "Read",
  icon: <Icon.Logo.MQTT />,
  useOnSelect: useCreateRead,
};
