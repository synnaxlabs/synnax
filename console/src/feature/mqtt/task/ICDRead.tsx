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
import { Access, ICD as PICD, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { DataType, deep, type payload, unique, uuid } from "@synnaxlabs/x";
import { type FC, useState } from "react";

import { use } from "@/feature/mqtt/device/queries";
import { Select as SelectDevice, useNoneSelected } from "@/feature/mqtt/device/Select";
import { type Device, MAKE, SCHEMAS } from "@/feature/mqtt/device/types";
import { useConnectModal } from "@/feature/mqtt/device/useConnectModal";
import { jsonTypeOf } from "@/feature/mqtt/task/createReadFields";
import { createDiscoverModal, type Kind } from "@/feature/mqtt/task/discover/Discover";
import { namePrefix } from "@/feature/mqtt/task/entries";
import { TopicLabel } from "@/feature/mqtt/task/EntryLabel";
import { QoSField } from "@/feature/mqtt/task/QoSField";
import {
  deployICDReadConfigZ,
  ICD_READ_SCHEMAS,
  ICD_READ_TYPE,
  type ICDReadEntry,
  type ICDReadSchemas,
  type ReadField,
} from "@/feature/mqtt/task/types";
import { Button as PlatformButton } from "@/platform/button";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { ICD } from "@/platform/icd";
import { Panel } from "@/platform/panel";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<ICDReadSchemas>>();

/** The form that edits the ICD holding the message format of a topic. */
type FormatForm = ReturnType<typeof PICD.useForm>["form"];

/** The open form of one ICD. */
interface Format {
  key: icd.Key;
  form: FormatForm;
}

/** The messages of each ICD, by ICD key. */
type Messages = Map<icd.Key, icd.Message[]>;

/** The channel data type that each field kind reads into. */
const DATA_TYPES: Record<payload.Kind, string> = {
  float: "float64",
  integer: "int64",
  unsigned: "uint64",
  boolean: "uint8",
  string: "string",
  timestamp: "timestamp",
  json: "json",
  uuid: "uuid",
  bytes: "bytes",
};

/** The field kind that a browsed sample of each JSON type starts as. */
const SAMPLE_KINDS = {
  number: "float",
  boolean: "boolean",
  string: "string",
} as const satisfies Record<payload.JSONType, payload.Kind>;

const TOPIC_INPUT_PROPS = { placeholder: "plant/line1/telemetry" } as const;

const isVariable = ({ type }: icd.JSONField): boolean =>
  new DataType(DATA_TYPES[type]).isVariable;

/**
 * The fields of a topic share one index, which a variable-length channel cannot have.
 * @returns the variable-length fields of a message that also holds fixed ones.
 */
const variableAmongFixed = (fields: icd.JSONField[]): icd.JSONField[] =>
  fields.every(isVariable) ? [] : fields.filter(isVariable);

const mixMessage = (fields: icd.JSONField[]): string =>
  `${fields.map(({ name }) => name).join(", ")}: a string, JSON, or bytes field cannot share a topic with number, boolean, or timestamp fields.`;

const jsonFields = (message?: icd.Message | null): icd.JSONField[] =>
  message?.payload.format === "json" ? message.payload.fields : [];

/** @returns the first message of the entry, from the entry's own ICD. */
const messageOf = (messages: Messages, entry: ICDReadEntry): icd.Message | undefined =>
  messages.get(entry.icd)?.find(({ key }) => key === entry.messages.at(0)?.message);

const retrieveMessages = async (client: Synnax, keys: icd.Key[]): Promise<Messages> => {
  const icds = await client.icds.retrieve(unique.unique(keys));
  return new Map(icds.map(({ key, messages }) => [key, messages]));
};

/** @returns the messages of the entries' ICDs, reading the open one from its form. */
const readMessages = async (
  client: Synnax,
  format: Format | null,
  entries: ICDReadEntry[],
): Promise<Messages> => {
  const keys = entries.map((e) => e.icd).filter((key) => key !== format?.key);
  const messages = await retrieveMessages(client, keys);
  if (format != null) messages.set(format.key, format.form.value().messages);
  return messages;
};

/** A topic, and the fields that a pick adds to its message format. */
interface Draft {
  type: "plain";
  topic: string;
  fields: icd.JSONField[];
}

/** Names a field by its pointer: "/motor/speed" is "motor_speed". */
const fieldName = (pointer: string): string =>
  pointer === "" ? "value" : pointer.slice(1).replaceAll("/", "_");

const toJSONField = (leaf: ReadField): icd.JSONField =>
  icd.jsonFieldZ.parse({
    name: fieldName(leaf.pointer),
    pointer: leaf.pointer,
    type: SAMPLE_KINDS[jsonTypeOf(leaf)],
  });

const DISCOVER_KIND: Kind<Draft> = {
  title: "Add data",
  hint: "Select the values to read. Each one becomes a field of the message format.",
  noun: "field",
  create: ({ topics }) =>
    [...topics].map(([topic, { leaves, pointers }]) => ({
      type: "plain",
      topic,
      fields: leaves.filter((l) => pointers.has(l.pointer)).map(toJSONField),
    })),
  createTopic: (topic) => ({ type: "plain", topic, fields: [] }),
};

const useDiscoverModal = createDiscoverModal(DISCOVER_KIND);

/** Inserts entries into the lists of one ICD. */
type Inserter = Pick<FormatForm, "insert">;

const fieldsPath = (key: icd.MessageKey) =>
  deep.variant(PICD.formPath.messages[key].payload, "format", "json").fields;

/** Appends each field whose pointer the message lacks. */
const appendFields = (
  to: Inserter,
  message: icd.Message,
  fields: icd.JSONField[],
): void => {
  const held = jsonFields(message);
  const pointers = new Set(held.map(({ pointer }) => pointer));
  let after = held.at(-1)?.key;
  for (const field of fields) {
    if (pointers.has(field.pointer)) continue;
    to.insert(fieldsPath(message.key), field, after);
    after = field.key;
  }
};

const createMessage = (topic: string, fields: icd.JSONField[]): icd.Message => ({
  key: uuid.create(),
  name: topic,
  payload: { format: "json", fields },
});

const entryOf = (
  icdKey: icd.Key,
  topic: string,
  message: icd.MessageKey,
): ICDReadEntry =>
  mqtt.icdReadEntryZ.parse({ icd: icdKey, topic, messages: [{ message }] });

interface AddTopicsResult {
  /** The entries of the topics that the task does not read yet. */
  added: ICDReadEntry[];
  /** The new message of each held entry whose message is gone, by entry key. */
  repointed: { key: string; message: icd.MessageKey }[];
}

/** Creates an ICD with a message for each draft, and the entries that read them. */
const createICD = async (
  client: Synnax,
  name: string,
  drafts: Draft[],
): Promise<AddTopicsResult> => {
  const messages = drafts.map(({ topic, fields }) => createMessage(topic, fields));
  const { key } = await client.icds.create({ name, messages });
  const added = drafts.map(({ topic }, i) => entryOf(key, topic, messages[i].key));
  return { added, repointed: [] };
};

interface AddTopicsParams {
  client: Synnax;
  /** The form of the ICD that takes the messages of new topics. */
  format: Format;
  entries: ICDReadEntry[];
  drafts: Draft[];
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
  const result: AddTopicsResult = { added: [], repointed: [] };
  format.form.batch(() => {
    for (const { topic, fields } of drafts) {
      const entry = entries.find((e) => e.topic === topic);
      const key = entry?.icd ?? format.key;
      const message = entry == null ? undefined : messageOf(messages, entry);
      if (message != null) {
        appendFields(inserterOf(key), message, fields);
        continue;
      }
      const prior = messages.get(key) ?? [];
      const created = createMessage(topic, fields);
      inserterOf(key).insert(PICD.formPath.messages, created, prior.at(-1)?.key);
      messages.set(key, [...prior, created]);
      if (entry == null) result.added.push(entryOf(key, topic, created.key));
      else result.repointed.push({ key: entry.key, message: created.key });
    }
  });
  for (const [key, pending] of actions) await client.icds.dispatch(key, pending);
  return result;
};

interface TimestampFieldsProps {
  path: deep.Path<ICDReadEntry>;
  /** The fields of the topic's message format. */
  fields: icd.JSONField[];
}

type TimingMode = "arrival" | "payload";

/** Picks where each sample of the topic gets its time. */
const TimestampFields: FC<TimestampFieldsProps> = ({ path, fields }) => {
  const { set } = PForm.useContext();
  const indexPath = path.messages[0].index;
  const held = PForm.useFieldValue(indexPath, { optional: true });
  const timestamps = fields.filter(({ type }) => type === "timestamp");
  const index = timestamps.find(({ key }) => key === held)?.key;
  const setIndex = (next: string | undefined) => set(indexPath, next);
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
            value={index}
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

interface TopicDetailsProps {
  path: deep.Path<ICDReadEntry>;
  format: FormatForm;
}

const TopicDetails: FC<TopicDetailsProps> = ({ path, format }) => {
  const openTab = Panel.useOpenTab();
  const entry = PForm.useFieldValue(path);
  const messageKey = entry.messages.at(0)?.message ?? "";
  const message = PForm.useFieldValue(PICD.formPath.messages[messageKey], {
    optional: true,
    ctx: format,
  });
  const mixed = variableAmongFixed(jsonFields(message));
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
          <TimestampFields path={path} fields={jsonFields(message)} />
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
              {mixed.length > 0 && (
                <Input.HelpText variant="error">{mixMessage(mixed)}</Input.HelpText>
              )}
            </PForm.Form>
          ) : (
            <Text.Text color={9}>
              The message format of this topic is not JSON. Open it to change its
              format.
            </Text.Text>
          )}
        </PForm.Section>
      </PForm.Sections>
    </Flex.Box>
  );
};

const TopicItem = (props: List.ItemRenderProps<string>) => {
  const path = root.config.entries[props.itemKey];
  const topic = PForm.useFieldValue(path.topic, { optional: true });
  const disabled = PForm.useFieldValue(path.disabled, { optional: true });
  if (topic == null) return null;
  return (
    <Select.Item
      {...props}
      justify="between"
      align="center"
      className={CSS.cls(disabled === true && CSS.M("off"))}
    >
      <TopicLabel topic={topic} />
      <Task.EnabledCheckbox path={path.disabled} />
    </Select.Item>
  );
};

const EntryTitle = ({ path }: { path: deep.Path<ICDReadEntry> }) => (
  <TopicLabel topic={PForm.useFieldValue(path.topic)} />
);

interface TopicsProps {
  device: Device;
  /** The form of the selected topic's ICD. Null while the task reads no topics. */
  format: Format | null;
  /** The key of the selected entry. Empty while the task reads no topics. */
  selected: string[];
  onSelect: (keys: string[]) => void;
}

/** The topics of the task beside the details of the selected one. */
const Topics: FC<TopicsProps> = ({ device, format, selected, onSelect }) => {
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
      const messages = await readMessages(client, format, entries);
      const held = entries.map((e): Draft => ({
        type: "plain",
        topic: e.topic,
        fields: jsonFields(messageOf(messages, e)),
      }));
      const drafts = await discover({ device, entries: held });
      if (drafts == null || drafts.length === 0) return;
      const { added, repointed } =
        format == null
          ? await createICD(client, ctx.get(root.name).value, drafts)
          : await addTopics({ client, format, entries, drafts });
      push(added);
      // The index and bindings name fields of the deleted message.
      repointed.forEach(({ key, message }) =>
        ctx.set(
          root.config.entries[key].messages[0],
          mqtt.icdReadMessageZ.parse({ message }),
        ),
      );
      const first = entries.find((e) => e.topic === drafts[0].topic) ?? added[0];
      onSelect([first.key]);
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

  const path = current == null ? null : root.config.entries[current];
  return (
    <Task.Views.Panes
      listTitle="Topics"
      list={
        <Flex.Box y empty grow>
          <Menu.ContextMenu {...menuProps} menu={contextMenu}>
            <Select.Frame<string, ICDReadEntry>
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
                      No topics
                    </Text.Text>
                  }
                >
                  {({ key, ...p }) => <TopicItem key={key} {...p} />}
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
      title={path == null ? undefined : <EntryTitle path={path} />}
    >
      {path == null || format == null ? (
        <Flex.Box y grow align="center" justify="center">
          <Text.Text status="disabled">Add data to read a topic</Text.Text>
        </Flex.Box>
      ) : (
        <TopicDetails key={current} path={path} format={format.form} />
      )}
    </Task.Views.Panes>
  );
};

interface WithFormatProps extends Omit<TopicsProps, "format"> {
  icdKey: icd.Key;
}

/** Opens the form of the selected topic's ICD, which each edit updates at once. */
const WithFormat: FC<WithFormatProps> = ({ icdKey, ...rest }) => {
  const isPreview = Task.useIsPreview();
  const canEdit = Access.useUpdateGranted(icd.resourceID(icdKey));
  const { form, flush } = PICD.useForm({
    query: { key: icdKey },
    mode: isPreview || !canEdit ? "preview" : "normal",
  });
  PICD.useCheck({ key: icdKey, form });
  Task.useFlushBeforeDeploy(flush);
  return <Topics {...rest} format={{ key: icdKey, form }} />;
};

const Content = ({ device }: PlatformDevice.TaskFormContentProps<Device>) => {
  const entries = PForm.useFieldValue(root.config.entries);
  const [selected, setSelected] = useState<string[]>([]);
  const key = selected.find((k) => entries.some((e) => e.key === k));
  const current = entries.find((e) => e.key === key) ?? entries.at(0);
  return current == null ? (
    <Topics device={device} format={null} selected={[]} onSelect={setSelected} />
  ) : (
    <WithFormat
      key={current.icd}
      device={device}
      icdKey={current.icd}
      selected={[current.key]}
      onSelect={setSelected}
    />
  );
};

const Properties = () => <SelectDevice />;

const Form = PlatformDevice.wrapTaskForm({
  use,
  useConfigure: useConnectModal,
  useNoneSelected,
  Content,
});

const getInitialValues: Task.GetInitialValues<ICDReadSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = ICD_READ_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: "MQTT ICD read task", type: ICD_READ_TYPE, config: cfg };
};

/**
 * @returns the fields of the entry's message format.
 * @throws {ValidationError} if the task cannot read the format.
 */
const formatFields = (messages: Messages, entry: ICDReadEntry): icd.JSONField[] => {
  const message = messageOf(messages, entry);
  if (message?.payload.format !== "json")
    throw new ValidationError(
      `The message format of ${entry.topic} is missing or not JSON`,
    );
  const { fields } = message.payload;
  if (fields.length === 0)
    throw new ValidationError(`The message format of ${entry.topic} has no fields`);
  const mixed = variableAmongFixed(fields);
  if (mixed.length > 0)
    throw new ValidationError(`In ${entry.topic}, ${mixMessage(mixed)}`);
  return fields;
};

/** Gives each field of each enabled topic's message format a channel, and binds it. */
const onConfigure: Task.OnConfigure<ICDReadSchemas["config"]> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({ key: config.device, schemas: SCHEMAS });
  const enabled = config.entries.filter(({ disabled }) => !disabled);
  const messages = await retrieveMessages(
    client,
    enabled.map((e) => e.icd),
  );
  // Every format is checked before any channel is created.
  const formats = enabled.map((entry) => ({
    entry,
    fields: formatFields(messages, entry),
  }));
  const before = deep.copy(dev.properties);
  try {
    for (const { entry, fields } of formats) {
      const [read] = entry.messages;
      const index = fields.find(
        ({ key, type }) => key === read.index && type === "timestamp",
      )?.key;
      const stored = dev.properties.read[entry.topic]?.channels ?? {};
      const bound = new Map(
        (await client.channels.retrieve(read.bindings.map((b) => b.channel))).map(
          (ch) => [ch.key, ch],
        ),
      );
      const prefix = namePrefix(dev.name, entry.topic);
      const channels = fields.map((f) => {
        const dataType = DATA_TYPES[f.type];
        const binding = read.bindings.find(({ field }) => field === f.key);
        const ch = binding == null ? undefined : bound.get(binding.channel);
        // An edit of the field's type or index role since the last deploy needs a new
        // channel.
        const stale =
          ch != null &&
          (!ch.dataType.equals(dataType) || ch.isIndex !== (f.key === index));
        if (stale && stored[f.pointer] === ch.key) delete stored[f.pointer];
        return {
          key: f.key,
          pointer: f.pointer,
          name: `${prefix}_${channel.escapeInvalidName(f.name)}`,
          dataType,
          channel: ch == null || stale ? 0 : ch.key,
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
      entry.messages[0] = {
        message: read.message,
        index,
        bindings: channels.map(({ key, channel }) => ({ field: key, channel })),
      };
    }
  } finally {
    await PlatformDevice.setProperties(client, dev, before, SCHEMAS);
  }
  return [config, dev.rack];
};

export const ICDRead = Task.wrapForm({
  Properties,
  Form,
  schemas: ICD_READ_SCHEMAS,
  deployConfigZ: deployICDReadConfigZ,
  type: "mqtt_icd_read",
  onConfigure,
});

const useCreateICDRead = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const ICDReadSelectable: Selector.Selectable = {
  type: ICD_READ_TYPE,
  title: "MQTT ICD read task",
  label: "Read with message format",
  icon: <Icon.Logo.MQTT />,
  useOnSelect: useCreateICDRead,
};
