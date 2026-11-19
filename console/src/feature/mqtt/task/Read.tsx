// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/mqtt/task/Form.css";

import { channel, DisconnectedError, icd, mqtt, type Synnax } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { type Tree } from "@synnaxlabs/lyra/tree";
import { ICD as PICD, Synnax as PSynnax } from "@synnaxlabs/pluto";
import {
  DataType,
  deep,
  errors,
  type payload,
  primitive,
  unique,
  uuid,
} from "@synnaxlabs/x";
import {
  type Dispatch,
  type FC,
  Fragment,
  type SetStateAction,
  useRef,
  useState,
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
}

/** What the discovery dialog adds: topics to read and Sparkplug B tags. */
type Addition = Draft | SparkplugReadEntry;

const isDraft = (addition: Addition): addition is Draft => addition.type === "plain";

const createDraft = (topic: string, leaves: Leaf[] = []): Draft => {
  const names = ICD.fieldNames(leaves.map(({ pointer }) => pointer));
  return {
    type: "plain",
    topic,
    fields: leaves.map((leaf, i) =>
      icd.jsonFieldZ.parse({
        name: names[i],
        pointer: leaf.pointer,
        type: SAMPLE_KINDS[leaf.type],
      }),
    ),
  };
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
): icd.ReadBinding[] =>
  fields.map(({ key: field, pointer }) => {
    const ch = stored[pointer] ?? 0;
    if (written.has(ch)) return icd.readBindingZ.parse({ field });
    if (ch !== 0) written.add(ch);
    return icd.readBindingZ.parse({ field, channel: ch });
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
  read: Pick<icd.ReadMessage, "message" | "bindings">,
): PlainReadEntry =>
  mqtt.plainReadEntryZ.parse({ type: "plain", icd: icdKey, topic, messages: [read] });

interface AddTopicsResult {
  /** The entries of the topics that the task does not read yet. */
  added: PlainReadEntry[];
  /**
   * The bindings that each held entry gains, by entry key. An entry whose message is
   * gone reads the new one.
   */
  held: { key: string; message?: icd.MessageKey; bindings: icd.ReadBinding[] }[];
}

/** @returns the bindings of the fields of a draft that the message reads. */
type Bind = (
  draft: Draft,
  fields: icd.JSONField[],
  message: icd.MessageKey,
) => icd.ReadBinding[];

/** @returns the name of the ICD of a task named name. */
const icdName = (name: string): string => (name.trim() === "" ? DEFAULT_NAME : name);

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
      const message = entry == null ? undefined : ICD.messageOf(messages, entry, 0);
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

/**
 * @returns what the device stores for a message read, without the stored channel of a
 * field with no channel when written holds it, as another binding writes it. Changes
 * neither stored nor bound.
 */
const storedProps = (
  stored: Stored,
  topic: string,
  { source: { messages }, read: r, fields }: ICD.BoundMessage<ICD.ReadSource>,
  written: ReadonlySet<channel.Key>,
): Task.ReadChannelProps | undefined => {
  const props = storedOf(stored, { topic, messages }, r);
  if (props == null) return undefined;
  const { bindings } = messages[r];
  const channels = { ...props.channels };
  for (const f of fields) {
    const ch = bindings.find(({ field }) => field === f.key)?.channel ?? 0;
    if (ch === 0 && written.has(channels[f.pointer])) delete channels[f.pointer];
  }
  return { ...props, channels };
};

/** @returns what the device stores for each message of a topic, as deploy finds it. */
const usePropsOf = (device: Device, path: deep.Path<PlainReadEntry>): ICD.PropsOf => {
  const entries = PForm.useFieldValue(root.config.entries);
  const topic = PForm.useFieldValue(path.topic);
  const written = writtenChannels(entries);
  return (bound) => storedProps(device.properties.read, topic, bound, written);
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

interface PlannerProps {
  device: Device;
  path: deep.Path<PlainReadEntry>;
  /** The index of the message in the topic's reads. */
  read: number;
}

/** Plans the channels of a message of a topic, for the tree and the panes. */
const Planner = ({ device, path, read }: PlannerProps) => {
  const prefix = usePrefix(device.name, path, read);
  const propsOf = usePropsOf(device, path);
  return <ICD.Planner path={path} read={read} prefix={prefix} propsOf={propsOf} />;
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
  const topic = PForm.useFieldValue(path.topic);
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
                {encoding === "json" && <ICD.AddField onClick={onAddField} />}
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
  device: Device;
}

const TagDetails: FC<TagDetailsProps> = ({ path, device }) => {
  const entry = PForm.useFieldValue(path);
  const field = tagField(device.name, entry);
  const props = device.properties.read[sparkplugPropertiesKey(entry)];
  const plan = Task.useChannelPlan(
    { field, props },
    async (client) =>
      await Task.planReadChannels({
        client,
        props,
        fields: [field],
        namePrefix: field.name,
        indexKey: "",
      }),
  );
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
  const { push } = PForm.useFieldList(root.config.entries);
  const entries = PForm.useFieldValue(root.config.entries);
  const stored = device.properties.read;
  const { tree, handleSelect, contextMenu, renderItem, current, addField, addMessage } =
    ICD.Tree.use({
      sources: root.config.entries,
      sourceOf: (e) => (isPlain(e) ? e : null),
      pathOf: plainPath,
      SourceItem: EntryItem,
      // Starting reuses the channel that the device stores for the field's pointer,
      // unless another field reads it.
      bindingOf: (entry, read, field) => {
        const channels = storedOf(
          stored,
          ctx.get(plainPath(entry)).value,
          read,
        )?.channels;
        const written = writtenChannels(ctx.get(root.config.entries).value);
        return bindingsOf(channels ?? {}, [field], written)[0];
      },
      messageName: (entry, held) =>
        nextMessageName(ctx.get(plainPath(entry)).value, held),
      selected,
      onSelect,
      expanded,
      onExpand,
    });
  const first = entries.find(isPlain);
  const format = first == null ? undefined : formats.get(first.icd);

  // Adds run one at a time, so a second click cannot make a second ICD.
  const adding = useRef(false);
  const add = async () => {
    if (client == null) throw new DisconnectedError();
    const plain = entries.filter(isPlain);
    const messages = messagesOf(formats);
    const held = entries.map((e): Addition =>
      isPlain(e)
        ? {
            ...createDraft(e.topic),
            fields: ICD.jsonFields(ICD.messageOf(messages, e, 0)),
          }
        : e,
    );
    const additions = await discover({ device, entries: held });
    if (additions == null || additions.length === 0) return;
    const drafts = additions.filter(isDraft);
    const tags = additions.filter((a): a is SparkplugReadEntry => !isDraft(a));
    const written = writtenChannels(entries);
    const bind: Bind = ({ topic }, fields, message) =>
      bindingsOf(topicChannels(stored, topic, message), fields, written);
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
      if (message != null) ctx.set(read, icd.readMessageZ.parse({ message, bindings }));
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
  const [expanded, setExpanded] = useState<string[]>(() =>
    ICD.Tree.initialExpanded(entries),
  );
  const plain = entries.filter(isPlain);
  const resolve = (entry: ReadEntry) => {
    if (entry.type !== "sparkplug") return {};
    const stored = device.properties.read[sparkplugPropertiesKey(entry)];
    return { channel: stored?.channels[""] ?? 0 };
  };
  const body = (
    <ICD.Plans>
      <Entries
        device={device}
        selected={selected}
        onSelect={setSelected}
        expanded={expanded}
        onExpand={setExpanded}
      />
      {plain.flatMap(({ key, messages }) =>
        messages.map(({ message }, read) => (
          <Fragment key={`${key}/${message}`}>
            <ICD.SuitRead path={plainPath(key)} read={read} />
            <Planner device={device} path={plainPath(key)} read={read} />
          </Fragment>
        )),
      )}
    </ICD.Plans>
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

/** Gives a tag a channel: the one it has, the one the device stores, or a new one. */
const configureTag = async (
  client: Synnax,
  dev: Device,
  entry: SparkplugReadEntry,
): Promise<void> => {
  const field = tagField(dev.name, entry);
  await Task.configureReadChannels({
    client,
    stored: { read: dev.properties.read, key: sparkplugPropertiesKey(entry) },
    fields: [field],
    namePrefix: field.name,
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
  bound: ICD.BoundMessage<PlainReadEntry>,
  written: Set<channel.Key>,
): Promise<void> => {
  const { source: entry, read: r, message } = bound;
  const propertiesKey = storedKey(entry.topic, entry.messages[r].message);
  const props = storedProps(dev.properties.read, entry.topic, bound, written);
  if (props != null) {
    if (dev.properties.read[propertiesKey] == null)
      delete dev.properties.read[entry.topic];
    dev.properties.read[propertiesKey] = props;
  }
  const channels = await ICD.configureMessage(
    client,
    bound,
    messagePrefix(dev.name, entry.topic, r, message.name),
    { read: dev.properties.read, key: propertiesKey },
  );
  channels.forEach((ch) => written.add(ch));
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
    entry.messages.map((_, read) =>
      ICD.boundMessage(messages, entry, read, entry.topic),
    ),
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
