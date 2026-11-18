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
import { Input } from "@synnaxlabs/lyra/input";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree } from "@synnaxlabs/lyra/tree";
import { Access, Channel, ICD as PICD, Synnax as PSynnax } from "@synnaxlabs/pluto";
import {
  DataType,
  deep,
  errors,
  type payload,
  primitive,
  type record,
  unique,
  uuid,
} from "@synnaxlabs/x";
import {
  createContext,
  type Dispatch,
  type FC,
  type PropsWithChildren,
  type ReactElement,
  type SetStateAction,
  useContext,
  useEffect,
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
import { Panel } from "@/platform/panel";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<ReadSchemas>>();

const DEFAULT_NAME = "MQTT read task";

const plainPath = (key: string) =>
  deep.variant(root.config.entries[key], "type", "plain");

/** The form that edits the ICD holding the message format of a topic. */
type FormatForm = ReturnType<typeof PICD.useForm>["form"];

/** The open form of one ICD. */
interface Format {
  key: icd.Key;
  form: FormatForm;
}

/** The messages of each ICD, by ICD key. */
type Messages = Map<icd.Key, icd.Message[]>;

/** The channels a device stores for the topics it reads, by topic. */
type Stored = Device["properties"]["read"];

/** The field kind that a browsed sample of each JSON type starts as. */
const SAMPLE_KINDS = {
  number: "float",
  boolean: "boolean",
  string: "string",
} as const satisfies Record<payload.JSONType, payload.Kind>;

const TOPIC_INPUT_PROPS = { placeholder: "plant/line1/telemetry" } as const;

const isVariable = ({ type }: icd.JSONField): boolean =>
  new DataType(ICD.DEFAULT_DATA_TYPES[type]).isVariable;

/**
 * The fields of a topic share one index, which a variable-length channel cannot have.
 * @returns the variable-length fields among fields that also hold fixed ones.
 */
const variableAmongFixed = (fields: icd.JSONField[]): icd.JSONField[] =>
  fields.every(isVariable) ? [] : fields.filter(isVariable);

const mixMessage = (fields: icd.JSONField[]): string =>
  `${fields.map(({ name }) => name).join(", ")}: a string, JSON, or bytes field cannot share a message with number, boolean, or timestamp fields.`;

const jsonFields = (message?: icd.Message | null): icd.JSONField[] =>
  message?.payload.format === "json" ? message.payload.fields : [];

/** @returns the fields that a read gives channels: the bound ones and the index. */
const readFields = (
  fields: icd.JSONField[],
  { index, bindings }: Pick<mqtt.ReadMessage, "index" | "bindings">,
): icd.JSONField[] =>
  fields.filter(
    ({ key }) => key === index || bindings.some(({ field }) => field === key),
  );

/** @returns a message that the entry reads, from the entry's own ICD. */
const messageOf = (
  messages: Messages,
  entry: PlainReadEntry,
  read: number,
): icd.Message | undefined =>
  messages.get(entry.icd)?.find(({ key }) => key === entry.messages.at(read)?.message);

const retrieveMessages = async (client: Synnax, keys: icd.Key[]): Promise<Messages> => {
  const icds = await client.icds.retrieve(unique.unique(keys));
  return new Map(icds.map(({ key, messages }) => [key, messages]));
};

/** @returns the messages of the entries' ICDs, reading the open one from its form. */
const readMessages = async (
  client: Synnax,
  format: Format | null,
  entries: PlainReadEntry[],
): Promise<Messages> => {
  const keys = entries.map((e) => e.icd).filter((key) => key !== format?.key);
  const messages = await retrieveMessages(client, keys);
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
type Inserter = Pick<FormatForm, "insert">;

const fieldsPath = (key: icd.MessageKey) =>
  deep.variant(PICD.formPath.messages[key].payload, "format", "json").fields;

/**
 * Appends each field whose pointer the message lacks.
 * @returns the fields it appends.
 */
const appendFields = (
  to: Inserter,
  message: icd.Message,
  fields: icd.JSONField[],
): icd.JSONField[] => {
  const held = jsonFields(message);
  const pointers = new Set(held.map(({ pointer }) => pointer));
  const appended = fields.filter(({ pointer }) => !pointers.has(pointer));
  let after = held.at(-1)?.key;
  for (const field of appended) {
    to.insert(fieldsPath(message.key), field, after);
    after = field.key;
  }
  return appended;
};

/**
 * @returns a message for the fields of a topic, named after the topic. A name that a
 * message in taken holds gets a number, such as `plant/oven 2`.
 */
const createMessage = (
  topic: string,
  fields: icd.JSONField[],
  taken: icd.Message[],
): icd.Message => {
  const names = new Set(taken.map(({ name }) => name));
  const base = topic === "" ? "Untitled" : topic;
  let name = base;
  for (let n = 2; names.has(name); n++) name = `${base} ${n}`;
  return { key: uuid.create(), name, payload: { format: "json", fields } };
};

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

/**
 * @returns the ICD that takes the messages of new topics: the one that the most entries
 * read, or the first of those. Undefined when no entry reads one.
 */
const homeICD = (entries: PlainReadEntry[]): icd.Key | undefined => {
  const counts = new Map<icd.Key, number>();
  for (const { icd: key } of entries) counts.set(key, (counts.get(key) ?? 0) + 1);
  let home: icd.Key | undefined;
  let most = 0;
  for (const [key, count] of counts)
    if (count > most) {
      home = key;
      most = count;
    }
  return home;
};

/** Creates an ICD with a message for each draft, and the entries that read them. */
const createICD = async (
  client: Synnax,
  name: string,
  drafts: Draft[],
  bind: Bind,
): Promise<AddTopicsResult> => {
  const messages: icd.Message[] = [];
  for (const { topic, fields } of drafts)
    messages.push(createMessage(topic, fields, messages));
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
      const created = createMessage(draft.topic, draft.fields, prior);
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

/** @returns whether the Core can read the field: a bytes field needs a pointer. */
const isReadable = ({ type, pointer }: icd.JSONField): boolean =>
  type !== "bytes" || pointer !== "";

/**
 * @returns read, changed to suit the fields of its message: no binding of a field that
 * is gone or that the Core cannot read, no data type that its field cannot convert to,
 * and no index that is not a timestamp field.
 */
const suit = (read: mqtt.ReadMessage, fields: icd.JSONField[]): mqtt.ReadMessage => {
  const byKey = new Map(fields.map((f) => [f.key, f]));
  const indexField = read.index == null ? undefined : byKey.get(read.index);
  const index = indexField?.type === "timestamp" ? read.index : undefined;
  const bindings = read.bindings.flatMap((b) => {
    const field = byKey.get(b.field);
    // The binding of a former index field writes to an index channel.
    if (
      field == null ||
      !isReadable(field) ||
      (b.field === read.index && index == null)
    )
      return [];
    if (b.dataType === "" || ICD.converts(field.type, b.dataType)) return [b];
    return [{ ...b, dataType: "" }];
  });
  return { message: read.message, ...(index == null ? {} : { index }), bindings };
};

/**
 * @returns read, indexed on the field of next, or on nothing. The index field reads
 * into the index channel that the device stores for the topic.
 */
const reindex = (
  read: mqtt.ReadMessage,
  next: string | undefined,
  indexChannel: channel.Key,
): mqtt.ReadMessage => {
  const bindings = read.bindings.filter(
    ({ field }) => field !== read.index && field !== next,
  );
  if (next == null) return { message: read.message, bindings };
  const binding = mqtt.readBindingZ.parse({ field: next, channel: indexChannel });
  return { message: read.message, index: next, bindings: [...bindings, binding] };
};

/** The open forms of the ICDs that the task's topics read, by ICD key. */
type Formats = ReadonlyMap<icd.Key, FormatForm>;

const FormatsContext = createContext<Formats>(new Map());
FormatsContext.displayName = "MQTT.FormatsContext";

/** @returns the open form of an ICD that a topic reads. */
const formatOf = (formats: Formats, key: icd.Key): FormatForm => {
  const format = formats.get(key);
  if (format == null) throw new Error(`The form of ICD ${key} is not open`);
  return format;
};

/** @returns the prefix of the channel names of a message, current with its name. */
const usePrefix = (
  deviceName: string,
  path: deep.Path<PlainReadEntry>,
  read: number,
): string => {
  const formats = useContext(FormatsContext);
  const topic = PForm.useFieldValue(path.topic);
  const icdKey = PForm.useFieldValue(path.icd);
  const message = PForm.useFieldValue(path.messages[read].message);
  const name = PForm.useFieldValue(PICD.formPath.messages[message].name, {
    optional: true,
    ctx: formats.get(icdKey),
  });
  return messagePrefix(deviceName, topic, read, name ?? "");
};

interface TimestampFieldsProps {
  path: deep.Path<PlainReadEntry>;
  /** The index of the message in the topic's reads. */
  read: number;
  /** The fields of the message format. */
  fields: icd.JSONField[];
  device: Device;
}

type TimingMode = "arrival" | "payload";

/** Picks where each sample of a message gets its time. */
const TimestampFields: FC<TimestampFieldsProps> = ({ path, read, fields, device }) => {
  const { get, set } = PForm.useContext();
  const readPath = path.messages[read];
  const topic = PForm.useFieldValue(path.topic);
  const messages = PForm.useFieldValue(path.messages);
  const held = PForm.useFieldValue(readPath.index, { optional: true });
  const bindings = PForm.useFieldValue(readPath.bindings);
  const entryDisabled = PForm.useFieldValue(path.disabled);
  const prefix = usePrefix(device.name, path, read);
  const indexChannel =
    storedOf(device.properties.read, { topic, messages }, read)?.index ?? 0;
  const indexBinding = bindings.find(({ field }) => field === held);
  const timestamps = fields.filter(({ type }) => type === "timestamp");
  const index = timestamps.find(({ key }) => key === held);
  const setIndex = (next: string | undefined) =>
    set(readPath, reindex(get(readPath).value, next, indexChannel));
  const handleMode = (mode: TimingMode) =>
    setIndex(mode === "payload" ? timestamps.at(0)?.key : undefined);
  return (
    <>
      <Input.Item
        label="Source"
        helpText={
          timestamps.length === 0
            ? "Give a field a timestamp type to read the time from the payload."
            : undefined
        }
      >
        <Select.Buttons<TimingMode>
          value={index == null ? "arrival" : "payload"}
          onChange={handleMode}
        >
          <Select.Item<TimingMode> itemKey="arrival">Arrival time</Select.Item>
          <Select.Item<TimingMode> itemKey="payload" disabled={timestamps.length === 0}>
            Payload value
          </Select.Item>
        </Select.Buttons>
      </Input.Item>
      {index != null && (
        <Input.Item label="Field" padHelpText={false}>
          <Select.Simple<string>
            value={index.key}
            onChange={setIndex}
            resourceName="field"
            allowNone={false}
          >
            {timestamps.map(({ key, name }) => (
              <Select.Item<string> key={key} itemKey={key}>
                {name}
              </Select.Item>
            ))}
          </Select.Simple>
        </Input.Item>
      )}
      {indexBinding != null && (
        <Task.ChannelNameField
          channel={indexBinding.channel}
          defaultName={`${prefix}_time`}
          skipped={entryDisabled}
        />
      )}
    </>
  );
};

/** @returns the data types that a channel of a field of the kind cannot take. */
const hiddenDataTypes = (kind: payload.Kind): DataType[] =>
  DataType.ALL.filter((dt) => !ICD.converts(kind, dt.toString()));

const UNREADABLE_TOOLTIP =
  "A bytes field needs a JSON pointer. Give it one in the message format.";

interface UnbindProps {
  path: deep.Target<channel.Key>;
  kind: payload.Kind;
  channel: channel.Key;
}

/**
 * Unbinds a channel that the field's kind can no longer convert to, so deploying
 * creates a new one.
 */
const Unbind = ({ path, kind, channel: key }: UnbindProps) => {
  const { set } = PForm.useContext();
  const dataType = Channel.useResultDataType({ key }).data?.toString();
  useEffect(() => {
    if (dataType != null && !ICD.converts(kind, dataType)) set(path, 0);
  }, [dataType, kind]);
  return null;
};

interface WithFormatProps extends PropsWithChildren {
  icdKey: icd.Key;
}

/** Opens the form of an ICD that a topic reads. Each edit updates the ICD at once. */
const WithFormat = ({ icdKey, children }: WithFormatProps): ReactElement => {
  const isPreview = Task.useIsPreview();
  const canEdit = Access.useUpdateGranted(icd.resourceID(icdKey));
  const { form, flush } = PICD.useForm({
    query: { key: icdKey },
    mode: isPreview || !canEdit ? "preview" : "normal",
  });
  PICD.useCheck({ key: icdKey, form });
  Task.useFlushBeforeDeploy(flush);
  const formats = new Map(useContext(FormatsContext)).set(icdKey, form);
  return <FormatsContext value={formats}>{children}</FormatsContext>;
};

/** @returns the messages of each open ICD, by ICD key. */
const messagesOf = (formats: Formats): Messages =>
  new Map([...formats].map(([key, form]) => [key, form.value().messages]));

/** @returns the fields of a message of an ICD. */
const fieldsOf = (
  formats: Formats,
  icdKey: icd.Key,
  message: icd.MessageKey,
): icd.JSONField[] =>
  jsonFields(
    formats
      .get(icdKey)
      ?.value()
      .messages.find(({ key }) => key === message),
  );

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
  if (bound === on || (on && !isReadable(field))) return;
  const channels = storedOf(stored, ctx.get(plainPath(entry)).value, read)?.channels;
  const written = writtenChannels(ctx.get(root.config.entries).value);
  ctx.set(
    path.bindings,
    on
      ? [...bindings, ...bindingsOf(channels ?? {}, [field], written)]
      : bindings.filter(({ field: k }) => k !== field.key),
  );
};

interface ReadCheckboxProps {
  /** The key of the topic's entry. */
  entry: string;
  /** The index of the message in the topic's reads. */
  read: number;
  field: icd.JSONField;
  stored: Stored;
}

/** Reads the field into a channel, or stops reading it. */
const ReadCheckbox: FC<ReadCheckboxProps> = ({ entry, read, field, stored }) => {
  const ctx = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const bindings = PForm.useFieldValue(plainPath(entry).messages[read].bindings);
  const bound = bindings.some(({ field: key }) => key === field.key);
  const readable = isReadable(field);
  let tooltip: string | undefined = bound ? "Disable" : "Enable";
  if (isPreview) tooltip = undefined;
  else if (!readable) tooltip = UNREADABLE_TOOLTIP;
  return (
    <Input.Checkbox
      value={bound}
      onChange={(on) => setRead(ctx, entry, read, field, on, stored)}
      checkedIcon={<Status.Indicator variant="success" />}
      uncheckedIcon={<Status.Indicator variant="disabled" />}
      size="small"
      disabled={isPreview || !readable}
      aria-label={`Read ${field.name}`}
      tooltip={tooltip}
    />
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
const useIndex = (formats: Formats): Index => {
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
              fieldsOf(formats, e.icd, message).filter(({ key }) => key !== index),
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

interface MessageLabelProps {
  path: deep.Path<PlainReadEntry>;
  /** The index of the message in the topic's reads. */
  read: number;
}

/** Names a message that a topic reads by its name in the ICD. */
const MessageLabel = ({ path, read }: MessageLabelProps) => {
  const formats = useContext(FormatsContext);
  const icdKey = PForm.useFieldValue(path.icd);
  const message = PForm.useFieldValue(path.messages[read].message);
  const name = PForm.useFieldValue(PICD.formPath.messages[message].name, {
    optional: true,
    ctx: formats.get(icdKey),
  });
  return (
    <Text.Text level="small" weight={500} color={name == null ? 8 : 10} overflow="fade">
      <Icon.JSON />
      {name ?? "Message not found"}
    </Text.Text>
  );
};

interface MessageItemProps extends Tree.ItemRenderProps<string> {
  node: MessageNode;
}

const MessageItem = ({ node: { entry, read }, ...props }: MessageItemProps) => (
  <Task.Views.TreeGroupItem
    {...props}
    label={<MessageLabel path={plainPath(entry)} read={read} />}
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
      <ReadCheckbox entry={entry} read={read} field={field} stored={stored} />
    </Task.Views.TreeFieldItem>
  );
};

interface OpenInEditorProps {
  icdKey: icd.Key;
}

const OpenInEditor = ({ icdKey }: OpenInEditorProps): ReactElement => {
  const openTab = Panel.useOpenTab();
  return (
    <Button.Button
      variant="text"
      size="small"
      tooltip="Open in the ICD editor"
      onClick={() => openTab({ variant: "resource", resource: icd.resourceID(icdKey) })}
    >
      <Icon.Library />
      Open in editor
    </Button.Button>
  );
};

interface AddFieldProps {
  onClick: () => void;
}

const AddField = ({ onClick }: AddFieldProps): ReactElement | null => {
  const isPreview = Task.useIsPreview();
  if (isPreview) return null;
  return (
    <Button.Button variant="text" size="small" onClick={onClick}>
      <Icon.Add />
      Add field
    </Button.Button>
  );
};

interface FieldDetailsProps {
  path: deep.Path<PlainReadEntry>;
  /** The index of the message in the topic's reads. */
  read: number;
  field: icd.JSONField;
  format: FormatForm;
  device: Device;
}

/**
 * The format of a field, which each edit saves to the ICD at once, then the channel
 * that the task reads it into.
 */
const FieldDetails: FC<FieldDetailsProps> = ({ path, read, field, format, device }) => {
  const readPath = path.messages[read];
  const bindings = PForm.useFieldValue(readPath.bindings);
  const message = PForm.useFieldValue(readPath.message);
  const icdKey = PForm.useFieldValue(path.icd);
  const entryDisabled = PForm.useFieldValue(path.disabled);
  const prefix = usePrefix(device.name, path, read);
  const i = bindings.findIndex(({ field: key }) => key === field.key);
  return (
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      <PForm.Sections>
        <PForm.Form<typeof PICD.formSchema> {...format}>
          <ICD.JSONField
            path={fieldsPath(message)[field.key]}
            actions={<OpenInEditor icdKey={icdKey} />}
          />
        </PForm.Form>
        <PForm.Section title="Channel">
          {i < 0 ? (
            <Text.Text status="disabled">
              The task does not read this field. Click its dot to read it.
            </Text.Text>
          ) : (
            <>
              <Task.ChannelNameField
                channel={bindings[i].channel}
                defaultName={`${prefix}_${channel.escapeInvalidName(field.name)}`}
                skipped={entryDisabled}
              />
              <Task.Payload.DataTypeField
                path={readPath.bindings[i].dataType}
                channel={bindings[i].channel}
                defaultDataType={ICD.DEFAULT_DATA_TYPES[field.type]}
                hideDataTypes={hiddenDataTypes(field.type)}
              />
            </>
          )}
        </PForm.Section>
      </PForm.Sections>
    </Flex.Box>
  );
};

interface MessageSectionsProps {
  path: deep.Path<PlainReadEntry>;
  /** The index of the message in the topic's reads. */
  read: number;
  format: FormatForm;
  device: Device;
}

/** The timestamp of a message that a topic reads, and the problems of its format. */
const MessageSections: FC<MessageSectionsProps> = ({ path, read, format, device }) => {
  const icdKey = PForm.useFieldValue(path.icd);
  const readMessage = PForm.useFieldValue(path.messages[read]);
  const message = PForm.useFieldValue(PICD.formPath.messages[readMessage.message], {
    optional: true,
    ctx: format,
  });
  const fields = jsonFields(message);
  const mixed = variableAmongFixed(readFields(fields, readMessage));
  return (
    <>
      {message == null && (
        <PForm.Section title="Message">
          <Status.Summary
            variant="warning"
            message="Message not found"
            description={
              read === 0
                ? "The ICD has no message for this topic. Add data to make one."
                : "The ICD no longer has this message. Remove it from the topic."
            }
            center={false}
          />
        </PForm.Section>
      )}
      {message != null && message.payload.format !== "json" && (
        <PForm.Section title="Message" actions={<OpenInEditor icdKey={icdKey} />}>
          <Text.Text color={9}>
            The format of this message is not JSON. Open it to change its format.
          </Text.Text>
        </PForm.Section>
      )}
      <PForm.Section title="Timestamp">
        <TimestampFields path={path} read={read} fields={fields} device={device} />
        {mixed.length > 0 && (
          <Input.HelpText variant="error">{mixMessage(mixed)}</Input.HelpText>
        )}
      </PForm.Section>
      <PForm.Form<typeof PICD.formSchema> {...format}>
        <ICD.FieldsStatus path={fieldsPath(readMessage.message)} />
      </PForm.Form>
    </>
  );
};

const choiceKey = (icdKey: icd.Key, message: icd.MessageKey): string =>
  `${icdKey}/${message}`;

interface Choice {
  icd: icd.ICD;
  message: icd.Message;
}

interface MessageFormatFieldProps {
  path: deep.Path<PlainReadEntry>;
  onChange: (icdKey: icd.Key, message: icd.Message) => void;
}

/** Picks the message that a topic reads from the JSON messages of every ICD. */
const MessageFormatField: FC<MessageFormatFieldProps> = ({ path, onChange }) => {
  const isPreview = Task.useIsPreview();
  const icdKey = PForm.useFieldValue(path.icd);
  const { message } = PForm.useFieldValue(path.messages[0]);
  const { data, getItem, more } = PICD.useList({ pageSize: 1000 });
  useEffect(() => more(), []);
  const choices = new Map<string, Choice>(
    getItem(data).flatMap((parent) =>
      parent.messages
        .filter(({ payload }) => payload.format === "json")
        .map((m) => [choiceKey(parent.key, m.key), { icd: parent, message: m }]),
    ),
  );
  const handleChange = (key: string) => {
    const choice = choices.get(key);
    if (choice == null) throw new Error(`No message format has the key ${key}`);
    onChange(choice.icd.key, choice.message);
  };
  return (
    <Input.Item label="Message format" padHelpText={false}>
      <Select.Simple<string>
        resourceName="message format"
        value={choiceKey(icdKey, message)}
        onChange={handleChange}
        variant="floating"
        allowNone={false}
        preview={isPreview}
      >
        {[...choices].map(([key, choice]) => (
          <Select.Item key={key} itemKey={key}>
            <Text.Text>{choice.message.name}</Text.Text>
            <Text.Text color={9}>{choice.icd.name}</Text.Text>
          </Select.Item>
        ))}
      </Select.Simple>
    </Input.Item>
  );
};

interface TopicDetailsProps {
  path: deep.Path<PlainReadEntry>;
  format: FormatForm;
  device: Device;
  /** Adds a field to the message of a topic that reads one. */
  onAddField: () => void;
  /** Adds a message to the topic, for documents that its messages do not cover. */
  onAddMessage: () => void;
  /** Makes a topic that reads one message read a message of any ICD. */
  onFormatChange: (icdKey: icd.Key, message: icd.Message) => void;
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
}) => {
  const isPreview = Task.useIsPreview();
  const reads = PForm.useFieldValue(path.messages);
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
                {encoding === "json" && <AddField onClick={onAddField} />}
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
          {reads === 1 && <MessageFormatField path={path} onChange={onFormatChange} />}
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

interface MessageDetailsProps {
  path: deep.Path<PlainReadEntry>;
  /** The index of the message in the topic's reads. */
  read: number;
  format: FormatForm;
  device: Device;
  onAddField: () => void;
}

/** The name and match of a message of a topic that reads several. */
const MessageDetails: FC<MessageDetailsProps> = ({
  path,
  read,
  format,
  device,
  onAddField,
}) => {
  const icdKey = PForm.useFieldValue(path.icd);
  const key = PForm.useFieldValue(path.messages[read].message);
  const encoding = PForm.useFieldValue(PICD.formPath.messages[key].payload.format, {
    optional: true,
    ctx: format,
  });
  return (
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      <PForm.Sections>
        {encoding === "json" && (
          <PForm.Form<typeof PICD.formSchema> {...format}>
            <PForm.Section
              title="Message"
              actions={
                <>
                  <AddField onClick={onAddField} />
                  <OpenInEditor icdKey={icdKey} />
                </>
              }
            >
              <PForm.TextField
                path={PICD.formPath.messages[key].name}
                label="Name"
                padHelpText={false}
              />
            </PForm.Section>
            <PForm.Section title="Documents">
              <ICD.Match
                path={deep.variant(
                  PICD.formPath.messages[key].payload,
                  "format",
                  "json",
                )}
              />
            </PForm.Section>
          </PForm.Form>
        )}
        <MessageSections path={path} read={read} format={format} device={device} />
      </PForm.Sections>
    </Flex.Box>
  );
};

interface SuitReadProps {
  path: deep.Path<PlainReadEntry>;
  /** The index of the message in the topic's reads. */
  read: number;
}

/**
 * Keeps the read of a message suited to its format, whose edits can leave a read that
 * the Core refuses to save. Unbinds each channel that its field can no longer convert
 * to.
 */
const SuitRead: FC<SuitReadProps> = ({ path, read }) => {
  const { set } = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const formats = useContext(FormatsContext);
  const icdKey = PForm.useFieldValue(path.icd);
  const readPath = path.messages[read];
  const readMessage = PForm.useFieldValue(readPath);
  const message = PForm.useFieldValue(PICD.formPath.messages[readMessage.message], {
    optional: true,
    ctx: formats.get(icdKey),
  });
  const fields = jsonFields(message);
  useEffect(() => {
    if (isPreview || message?.payload.format !== "json") return;
    const suited = suit(readMessage, fields);
    if (!deep.equal(suited, readMessage)) set(readPath, suited);
  }, [isPreview, readMessage, message]);
  if (isPreview) return null;
  return readMessage.bindings.map((b, i) => {
    const field = fields.find(({ key }) => key === b.field);
    if (field == null || b.channel === 0) return null;
    return (
      <Unbind
        key={b.field}
        path={readPath.bindings[i].channel}
        kind={field.type}
        channel={b.channel}
      />
    );
  });
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
  const formats = useContext(FormatsContext);
  const { push, remove } = PForm.useFieldList(root.config.entries);
  const entries = PForm.useFieldValue(root.config.entries);
  const stored = device.properties.read;
  const { nodes, byKey } = useIndex(formats);
  const home = homeICD(entries.filter(isPlain));
  const format = home == null ? undefined : formats.get(home);
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
        ? { ...createDraft(e.topic), fields: jsonFields(messageOf(messages, e, 0)) }
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
      try {
        await add();
      } catch (e) {
        adding.current = false;
        throw errors.fromUnknown(e);
      }
      adding.current = false;
    }, "Failed to add data");

  /**
   * Adds a message with no fields to the topic and to its ICD. The match tests the
   * pointer of the topic's other matches, so the messages tell documents apart.
   */
  const handleAddMessage = (entryKey: string) => {
    const path = plainPath(entryKey);
    const entry = ctx.get(path).value;
    const form = formatOf(formats, entry.icd);
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

  /**
   * Makes a topic read a message of any ICD, and each field of it that can share one
   * index. The message that the topic read leaves its ICD when it has no fields and no
   * other topic reads it, such as the one that adding a topic by name makes.
   */
  const handleFormatChange = (
    entryKey: string,
    icdKey: icd.Key,
    message: icd.Message,
  ) => {
    const path = plainPath(entryKey);
    const entry = ctx.get(path).value;
    const [prior] = entry.messages;
    if (entry.icd === icdKey && prior.message === message.key) return;
    const others = entries.filter(({ key }) => key !== entryKey);
    const form = formatOf(formats, entry.icd);
    const left = form.value().messages.find(({ key }) => key === prior.message);
    const shared = others.some(
      (e) =>
        isPlain(e) &&
        e.icd === entry.icd &&
        e.messages.some((m) => m.message === prior.message),
    );
    if (left != null && left.payload.fields.length === 0 && !shared)
      form.remove(PICD.formPath.messages[prior.message]);
    const readable = jsonFields(message).filter(isReadable);
    const mixed = new Set(variableAmongFixed(readable));
    const bindings = bindingsOf(
      stored,
      storedKey(entry.topic, 0, message.key),
      readable.filter((f) => !mixed.has(f)),
      writtenChannels(others),
    );
    ctx.batch(() => {
      ctx.set(path.icd, icdKey);
      ctx.set(path.messages, [
        mqtt.readMessageZ.parse({ message: message.key, bindings }),
      ]);
    });
  };

  /** Whether the ICD holds the message of a read of a topic, as JSON. */
  const holdsJSON = (entryKey: string, read: number): boolean => {
    const { icd: icdKey, messages } = ctx.get(plainPath(entryKey)).value;
    const message = messages[read].message;
    return (
      formats
        .get(icdKey)
        ?.value()
        .messages.find(({ key }) => key === message)?.payload.format === "json"
    );
  };

  /** Adds a field to a message of a topic and to its ICD, reads it, and selects it. */
  const handleAddField = (entryKey: string, read: number) => {
    const entry = ctx.get(plainPath(entryKey)).value;
    const { message } = entry.messages[read];
    const fields = fieldsOf(formats, entry.icd, message);
    const field = ICD.createJSONField(fields.map(({ name }) => name));
    formatOf(formats, entry.icd).insert(fieldsPath(message), field, fields.at(-1)?.key);
    setRead(ctx, entryKey, read, field, true, stored);
    const parent = entry.messages.length > 1 ? [messageNodeKey(entryKey, message)] : [];
    onExpand((prev) => unique.unique([...prev, entryKey, ...parent]));
    onSelect([fieldNodeKey(entryKey, message, field.key)]);
  };

  /** Removes each chosen field from its message in the ICD. */
  const handleRemoveFields = (chosen: FieldNode[]) => {
    for (const { entry, read, field } of chosen) {
      const { icd: icdKey, messages } = ctx.get(plainPath(entry)).value;
      formatOf(formats, icdKey).remove(fieldsPath(messages[read].message)[field.key]);
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
    if (
      chosen.length === 1 &&
      only.kind === "message" &&
      holdsJSON(only.entry, only.read)
    )
      addField = () => handleAddField(only.entry, only.read);
    else if (
      topic != null &&
      ctx.get(plainPath(topic).messages).value.length === 1 &&
      holdsJSON(topic, 0)
    )
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
    title = <MessageLabel path={plainPath(entry.key)} read={current.read} />;
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
          onFormatChange={(icdKey, message) =>
            handleFormatChange(entry.key, icdKey, message)
          }
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
          <SuitRead key={`${key}/${message}`} path={plainPath(key)} read={read} />
        )),
      )}
    </>
  );
  return (
    <>
      {unique.unique(plain.map((e) => e.icd)).reduceRight(
        (child, key) => (
          <WithFormat key={key} icdKey={key}>
            {child}
          </WithFormat>
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
  messages: Messages,
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
  const fields = readFields(message.payload.fields, readMessage);
  const index = fields.find(({ key }) => key === readMessage.index);
  if (index != null && index.type !== "timestamp")
    throw new ValidationError(
      `In ${where}, the index field ${index.name} must be a timestamp field`,
    );
  if (fields.length === 0)
    throw new ValidationError(`The task reads no fields of ${where}`);
  const mixed = variableAmongFixed(fields);
  if (mixed.length > 0) throw new ValidationError(`In ${where}, ${mixMessage(mixed)}`);
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
  const messages = await retrieveMessages(
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
