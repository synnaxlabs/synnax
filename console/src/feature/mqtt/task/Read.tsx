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
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Channel, ICD as PICD, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { DataType, deep, type payload, primitive, unique, uuid } from "@synnaxlabs/x";
import { type FC, useEffect, useState } from "react";

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
  bindingDataType,
  converts,
  DEFAULT_DATA_TYPES,
  deployReadConfigZ,
  type PlainReadEntry,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadBinding,
  type ReadEntry,
  type ReadSchemas,
  type SparkplugReadEntry,
} from "@/feature/mqtt/task/types";
import { Button as PlatformButton } from "@/platform/button";
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
  new DataType(DEFAULT_DATA_TYPES[type]).isVariable;

/**
 * The fields of a topic share one index, which a variable-length channel cannot have.
 * @returns the variable-length fields among fields that also hold fixed ones.
 */
const variableAmongFixed = (fields: icd.JSONField[]): icd.JSONField[] =>
  fields.every(isVariable) ? [] : fields.filter(isVariable);

const mixMessage = (fields: icd.JSONField[]): string =>
  `${fields.map(({ name }) => name).join(", ")}: a string, JSON, or bytes field cannot share a topic with number, boolean, or timestamp fields.`;

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

/** @returns the first message of the entry, from the entry's own ICD. */
const messageOf = (
  messages: Messages,
  entry: PlainReadEntry,
): icd.Message | undefined =>
  messages.get(entry.icd)?.find(({ key }) => key === entry.messages.at(0)?.message);

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

/**
 * @returns the bindings of fields, each to the channel the device stores for it. A
 * field whose stored channel is in written gets no channel, as the Core refuses a
 * channel that two bindings write. Adds the channels it binds to written.
 */
const bindingsOf = (
  stored: Stored,
  topic: string,
  fields: icd.JSONField[],
  written: Set<channel.Key>,
): ReadBinding[] =>
  fields.map(({ key, pointer }) => {
    const ch = stored[topic]?.channels[pointer] ?? 0;
    if (written.has(ch)) return mqtt.readBindingZ.parse({ field: key });
    if (ch !== 0) written.add(ch);
    return mqtt.readBindingZ.parse({ field: key, channel: ch });
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

/** @returns the bindings of the fields of a draft that the task reads. */
type Bind = (draft: Draft, fields: icd.JSONField[]) => ReadBinding[];

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
    entryOf(key, d.topic, { message: messages[i].key, bindings: bind(d, d.fields) }),
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
      const message = entry == null ? undefined : messageOf(messages, entry);
      if (entry != null && message != null) {
        const appended = appendFields(inserterOf(key), message, draft.fields);
        result.held.push({ key: entry.key, bindings: bind(draft, appended) });
        continue;
      }
      const prior = messages.get(key) ?? [];
      const created = createMessage(draft.topic, draft.fields);
      inserterOf(key).insert(PICD.formPath.messages, created, prior.at(-1)?.key);
      messages.set(key, [...prior, created]);
      const bindings = bind(draft, draft.fields);
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

const isIndexable = (field?: icd.JSONField): boolean =>
  field?.type === "timestamp" && field.default == null;

/**
 * @returns read, changed to suit the fields of its message: no binding of a field that
 * is gone or that the Core cannot read, no data type that its field cannot convert to,
 * and no index that is not a timestamp field. An index field with a default stays, for
 * its error to show.
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
    if (b.dataType === "" || converts(field.type, b.dataType)) return [b];
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

interface TimestampFieldsProps {
  path: deep.Path<PlainReadEntry>;
  /** The fields of the topic's message format. */
  fields: icd.JSONField[];
  /** The index channel that the device stores for the topic, or 0. */
  indexChannel: channel.Key;
}

type TimingMode = "arrival" | "payload";

/** Picks where each sample of the topic gets its time. */
const TimestampFields: FC<TimestampFieldsProps> = ({ path, fields, indexChannel }) => {
  const { get, set } = PForm.useContext();
  const readPath = path.messages[0];
  const held = PForm.useFieldValue(readPath.index, { optional: true });
  const timestamps = fields.filter(({ type }) => type === "timestamp");
  const index = timestamps.find(({ key }) => key === held);
  const setIndex = (next: string | undefined) =>
    set(readPath, reindex(get(readPath).value, next, indexChannel));
  const handleMode = (mode: TimingMode) =>
    setIndex(mode === "payload" ? timestamps.at(0)?.key : undefined);
  const invalid = index != null && !isIndexable(index);
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
        <Input.Item
          label="Field"
          padHelpText={false}
          helpText={
            invalid
              ? `${index.name} has a default, so it cannot be the index. Remove its default in the message format.`
              : undefined
          }
          status={invalid ? "error" : undefined}
        >
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
    </>
  );
};

/** The data types that a channel of a field of each kind can take. */
const hiddenDataTypes = (kind: payload.Kind): DataType[] =>
  DataType.ALL.filter((dt) => !converts(kind, dt.toString()));

const READ_TOOLTIP = "Read the field into a channel.";
const INDEX_TOOLTIP = "The task reads the time of each sample from this field.";
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
    if (dataType != null && !converts(kind, dataType)) set(path, 0);
  }, [dataType, kind]);
  return null;
};

interface BindingProps {
  /** The path of the topic's entry. */
  path: deep.Path<PlainReadEntry>;
  field: icd.JSONField;
  /** The prefix of the names of the channels that deploying creates. */
  prefix: string;
  stored: Stored;
}

/** Reads a field into a channel, or not, and names the channel. */
const Binding: FC<BindingProps> = ({ path, field, prefix, stored }) => {
  const { get, set } = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const read = path.messages[0];
  const topic = PForm.useFieldValue(path.topic);
  const entryDisabled = PForm.useFieldValue(path.disabled);
  const index = PForm.useFieldValue(read.index, { optional: true });
  const bindings = PForm.useFieldValue(read.bindings);
  const i = bindings.findIndex(({ field: key }) => key === field.key);
  const binding = i < 0 ? undefined : bindings[i];
  const isIndex = field.key === index;
  const readable = isReadable(field);
  const handleRead = (on: boolean) =>
    set(
      read.bindings,
      on
        ? [
            ...bindings,
            ...bindingsOf(
              stored,
              topic,
              [field],
              writtenChannels(get(root.config.entries).value),
            ),
          ]
        : bindings.filter(({ field: key }) => key !== field.key),
    );
  let tooltip = READ_TOOLTIP;
  if (isIndex) tooltip = INDEX_TOOLTIP;
  else if (!readable) tooltip = UNREADABLE_TOOLTIP;
  return (
    <Flex.Box x className={CSS.B("mqtt-read-binding")}>
      <Input.Item label={field.name} padHelpText={false}>
        <Input.Checkbox
          value={binding != null}
          onChange={handleRead}
          disabled={isPreview || isIndex || !readable}
          aria-label={`Read ${field.name}`}
          tooltip={tooltip}
        />
      </Input.Item>
      {binding != null && !isIndex && (
        <Task.Payload.DataTypeField
          path={read.bindings[i].dataType}
          channel={binding.channel}
          defaultDataType={DEFAULT_DATA_TYPES[field.type]}
          hideDataTypes={hiddenDataTypes(field.type)}
        />
      )}
      {binding != null && (
        <Task.ChannelNameField
          channel={binding.channel}
          defaultName={
            isIndex
              ? `${prefix}_time`
              : `${prefix}_${channel.escapeInvalidName(field.name)}`
          }
          skipped={entryDisabled}
        />
      )}
      {binding != null && binding.channel !== 0 && !isPreview && (
        <Unbind
          path={read.bindings[i].channel}
          kind={field.type}
          channel={binding.channel}
        />
      )}
    </Flex.Box>
  );
};

interface TopicDetailsProps {
  path: deep.Path<PlainReadEntry>;
  format: FormatForm;
  device: Device;
}

const TopicDetails: FC<TopicDetailsProps> = ({ path, format, device }) => {
  const openTab = Panel.useOpenTab();
  const entry = PForm.useFieldValue(path);
  const messageKey = entry.messages.at(0)?.message ?? "";
  const message = PForm.useFieldValue(PICD.formPath.messages[messageKey], {
    optional: true,
    ctx: format,
  });
  const fields = jsonFields(message);
  const read = entry.messages.at(0);
  const mixed = read == null ? [] : variableAmongFixed(readFields(fields, read));
  const prefix = namePrefix(device.name, entry.topic);
  const { set } = PForm.useContext();
  const isPreview = Task.useIsPreview();
  // An edit of the message format can leave the read one that the Core refuses to save.
  useEffect(() => {
    if (isPreview || read == null || message?.payload.format !== "json") return;
    const suited = suit(read, fields);
    if (!deep.equal(suited, read)) set(path.messages[0], suited);
  }, [isPreview, read, message]);
  const openEditor = () =>
    openTab({ variant: "resource", resource: icd.resourceID(entry.icd) });
  return (
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      <PForm.Sections>
        <PForm.Section title="Subscription">
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
        <PForm.Section title="Timestamp">
          <TimestampFields
            path={path}
            fields={fields}
            indexChannel={device.properties.read[entry.topic]?.index ?? 0}
          />
        </PForm.Section>
        <PForm.Section
          title="Message format"
          actions={
            <Button.Button
              variant="text"
              size="small"
              tooltip="Open in the ICD editor"
              onClick={openEditor}
            >
              <Icon.Library />
              Open in editor
            </Button.Button>
          }
        >
          {message == null ? (
            <Status.Summary
              variant="warning"
              message="Message not found"
              description="The ICD has no message for this topic. Add data to make one."
              center={false}
            />
          ) : message.payload.format === "json" ? (
            <PForm.Form<typeof PICD.formSchema> {...format}>
              <ICD.Fields path={fieldsPath(messageKey)} encoding="json" />
              <ICD.FieldsStatus path={fieldsPath(messageKey)} />
            </PForm.Form>
          ) : (
            <Text.Text color={9}>
              The message format of this topic is not JSON. Open it to change its
              format.
            </Text.Text>
          )}
        </PForm.Section>
        {fields.length > 0 && (
          <PForm.Section title="Channels">
            {fields.map((field) => (
              <Binding
                key={field.key}
                path={path}
                field={field}
                prefix={prefix}
                stored={device.properties.read}
              />
            ))}
            {mixed.length > 0 && (
              <Input.HelpText variant="error">{mixMessage(mixed)}</Input.HelpText>
            )}
          </PForm.Section>
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

const EntryItem = (props: List.ItemRenderProps<string>) => {
  const path = root.config.entries[props.itemKey];
  const type = PForm.useFieldValue(path.type, { optional: true });
  const disabled = PForm.useFieldValue(path.disabled, { optional: true });
  if (type == null) return null;
  return (
    <Select.Item
      {...props}
      justify="between"
      align="center"
      className={CSS.cls(disabled === true && CSS.M("off"))}
    >
      <EntryLabel path={path} />
      <Task.EnabledCheckbox path={path.disabled} />
    </Select.Item>
  );
};

interface EntriesProps {
  device: Device;
  /** The form of the ICD of the selected topic, or of the first topic while a tag is
   * selected. Null while the task reads no topics. */
  format: Format | null;
  /** The key of the selected entry. Empty while the task has no entries. */
  selected: string[];
  onSelect: (keys: string[]) => void;
}

/** The entries of the task beside the details of the selected one. */
const Entries: FC<EntriesProps> = ({ device, format, selected, onSelect }) => {
  const client = PSynnax.use();
  const ctx = PForm.useContext();
  const handleError = Status.useErrorHandler();
  const isPreview = Task.useIsPreview();
  const discover = useDiscoverModal();
  const menuProps = Menu.useContextMenu();
  const { data, push, remove } = PForm.useFieldList(root.config.entries);
  const entries = PForm.useFieldValue(root.config.entries);
  const current = selected.at(0);

  const handleAdd = () =>
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      const plain = entries.filter(isPlain);
      const messages = await readMessages(client, format, plain);
      const held = entries.map((e): Addition =>
        isPlain(e)
          ? { ...createDraft(e.topic), fields: jsonFields(messageOf(messages, e)) }
          : e,
      );
      const additions = await discover({ device, entries: held });
      if (additions == null || additions.length === 0) return;
      const drafts = additions.filter(isDraft);
      const tags = additions.filter((a): a is SparkplugReadEntry => !isDraft(a));
      const written = writtenChannels(entries);
      const bind: Bind = ({ topic, bound }, fields) =>
        bindingsOf(
          device.properties.read,
          topic,
          fields.filter(({ key }) => bound.includes(key)),
          written,
        );
      const { added, held: grown } =
        drafts.length === 0
          ? { added: [], held: [] }
          : format == null
            ? await createICD(client, icdName(ctx.get(root.name).value), drafts, bind)
            : await addTopics({ client, format, entries: plain, drafts, bind });
      push([...added, ...tags]);
      grown.forEach(({ key, message, bindings }) => {
        const read = plainPath(key).messages[0];
        // The index and bindings of a deleted message name fields it no longer has.
        if (message != null)
          ctx.set(read, mqtt.readMessageZ.parse({ message, bindings }));
        else ctx.set(read.bindings, [...ctx.get(read.bindings).value, ...bindings]);
      });
      const first = additions[0];
      const firstKey = isDraft(first)
        ? (
            entries.find((e) => isPlain(e) && e.topic === first.topic) ??
            added.find((e) => e.topic === first.topic)
          )?.key
        : first.key;
      if (firstKey != null) onSelect([firstKey]);
    }, "Failed to add data");

  const setEnabled = (keys: string[], enabled: boolean) =>
    keys.forEach((k) => ctx.set(root.config.entries[k].disabled, !enabled));
  const contextMenu = ({ keys }: Menu.ContextMenuMenuProps) => {
    const chosen = entries.filter(({ key }) => keys.includes(key));
    return (
      <Task.Views.ContextMenu
        keys={keys}
        onRemove={remove}
        onEnable={
          chosen.some((e) => e.disabled) ? () => setEnabled(keys, true) : undefined
        }
        onDisable={
          chosen.some((e) => !e.disabled) ? () => setEnabled(keys, false) : undefined
        }
      />
    );
  };

  const entry = entries.find(({ key }) => key === current);
  const path = entry == null ? null : root.config.entries[entry.key];
  let details;
  if (entry == null) details = null;
  else if (entry.type === "sparkplug")
    details = (
      <TagDetails
        key={entry.key}
        path={deep.variant(root.config.entries[entry.key], "type", "sparkplug")}
        deviceName={device.name}
      />
    );
  else if (format != null)
    details = (
      <TopicDetails
        key={entry.key}
        path={plainPath(entry.key)}
        format={format.form}
        device={device}
      />
    );
  return (
    <Task.Views.Panes
      listTitle="Entries"
      list={
        <Flex.Box y empty grow>
          <Menu.ContextMenu {...menuProps} menu={contextMenu}>
            <Select.Frame<string, ReadEntry>
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
                <List.Items<string>
                  emptyContent={
                    <Text.Text center level="small" color={9}>
                      No entries
                    </Text.Text>
                  }
                >
                  {({ key, ...p }) => <EntryItem key={key} {...p} />}
                </List.Items>
              </List.Scroll>
            </Select.Frame>
          </Menu.ContextMenu>
          {!isPreview && (
            <PlatformButton.CreateListItem size="small" onClick={handleAdd}>
              Add data
            </PlatformButton.CreateListItem>
          )}
        </Flex.Box>
      }
      detailsPath={path}
      title={path == null ? undefined : <EntryLabel path={path} />}
    >
      {details ?? (
        <Flex.Box y grow align="center" justify="center">
          <Text.Text status="disabled">Add data to read a topic or a tag</Text.Text>
        </Flex.Box>
      )}
    </Task.Views.Panes>
  );
};

interface WithFormatProps extends Omit<EntriesProps, "format"> {
  icdKey: icd.Key;
}

/** Opens the form of a topic's ICD, which each edit updates at once. */
const WithFormat: FC<WithFormatProps> = ({ icdKey, ...rest }) => {
  const isPreview = Task.useIsPreview();
  const canEdit = Access.useUpdateGranted(icd.resourceID(icdKey));
  const { form, flush } = PICD.useForm({
    query: { key: icdKey },
    mode: isPreview || !canEdit ? "preview" : "normal",
  });
  PICD.useCheck({ key: icdKey, form });
  Task.useFlushBeforeDeploy(flush);
  return <Entries {...rest} format={{ key: icdKey, form }} />;
};

const Content = ({ device }: PlatformDevice.TaskFormContentProps<Device>) => {
  const entries = PForm.useFieldValue(root.config.entries);
  const [selected, setSelected] = useState<string[]>([]);
  const key = selected.find((k) => entries.some((e) => e.key === k));
  const current = entries.find((e) => e.key === key) ?? entries.at(0);
  // A selected tag keeps the ICD of the first topic open, for the topics it adds.
  const topic = current != null && isPlain(current) ? current : entries.find(isPlain);
  const props = {
    device,
    selected: current == null ? [] : [current.key],
    onSelect: setSelected,
  };
  const resolve = (entry: ReadEntry) => {
    if (entry.type !== "sparkplug") return {};
    const stored = device.properties.read[sparkplugPropertiesKey(entry)];
    return { channel: stored?.channels[""] ?? 0 };
  };
  return (
    <>
      {topic == null ? (
        <Entries {...props} format={null} />
      ) : (
        <WithFormat key={topic.icd} icdKey={topic.icd} {...props} />
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

/**
 * @returns the fields of the entry's message format that the task reads.
 * @throws {ValidationError} if the task cannot read the format.
 */
const boundFields = (messages: Messages, entry: PlainReadEntry): icd.JSONField[] => {
  const message = messageOf(messages, entry);
  if (message?.payload.format !== "json")
    throw new ValidationError(
      `The message format of ${entry.topic} is missing or not JSON`,
    );
  const [read] = entry.messages;
  const fields = readFields(message.payload.fields, read);
  const index = fields.find(({ key }) => key === read.index);
  if (index != null && !isIndexable(index))
    throw new ValidationError(
      `In ${entry.topic}, the index field ${index.name} must be a timestamp field with no default`,
    );
  if (fields.length === 0)
    throw new ValidationError(`The task reads no fields of ${entry.topic}`);
  const mixed = variableAmongFixed(fields);
  if (mixed.length > 0)
    throw new ValidationError(`In ${entry.topic}, ${mixMessage(mixed)}`);
  return fields;
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
 * Gives each field that a topic reads a channel: the one it has, the one the device
 * stores for its pointer unless written holds it, or a new one of the binding's data
 * type. Adds the channels it binds to written.
 */
const configureTopic = async (
  client: Synnax,
  dev: Device,
  entry: PlainReadEntry,
  fields: icd.JSONField[],
  written: Set<channel.Key>,
): Promise<void> => {
  const [read] = entry.messages;
  const index = fields.find(({ key }) => key === read.index)?.key;
  const stored = dev.properties.read[entry.topic]?.channels ?? {};
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
  const prefix = namePrefix(dev.name, entry.topic);
  const channels = fields.map((f, i) => {
    const binding = bindingOf(f);
    const isIndex = f.key === index;
    const suited = binding.dataType !== "" && converts(f.type, binding.dataType);
    const dataType = suited ? binding.dataType : "";
    const ch = live.get(reused[i]);
    // An edit of the field's kind or index role since the last deploy needs a new
    // channel.
    const stale =
      ch != null &&
      (!converts(f.type, ch.dataType.toString()) || ch.isIndex !== isIndex);
    if (stale && stored[f.pointer] === ch.key) delete stored[f.pointer];
    return {
      key: f.key,
      pointer: f.pointer,
      name: `${prefix}_${channel.escapeInvalidName(f.name)}`,
      dataType: isIndex
        ? DataType.TIMESTAMP.toString()
        : bindingDataType({ dataType }, f.type),
      channel: stale ? 0 : binding.channel,
      binding: { ...binding, dataType },
    };
  });
  await Task.configureReadChannels({
    client,
    read: dev.properties.read,
    propertiesKey: entry.topic,
    fields: channels,
    namePrefix: prefix,
    indexKey: index ?? "",
  });
  channels.forEach(({ channel }) => written.add(channel));
  entry.messages[0] = {
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
  const formats = topics.map((entry) => ({
    entry,
    fields: boundFields(messages, entry),
  }));
  const before = deep.copy(dev.properties);
  try {
    const written = writtenChannels(config.entries);
    for (const { entry, fields } of formats)
      await configureTopic(client, dev, entry, fields, written);
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
  type: READ_TYPE,
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
