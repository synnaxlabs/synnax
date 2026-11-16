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
import { DataType, deep, type payload, uuid } from "@synnaxlabs/x";
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

/** The form that edits the ICD holding the message format of each topic. */
type FormatForm = ReturnType<typeof PICD.useForm>["form"];

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

const messageOf = (
  messages: icd.Message[],
  entry: ICDReadEntry,
): icd.Message | undefined =>
  messages.find(({ key }) => key === entry.messages.at(0)?.message);

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

/** @returns the inserts that append each field whose pointer the message lacks. */
const appendFields = (message: icd.Message, fields: icd.JSONField[]): icd.Action[] => {
  const held = jsonFields(message);
  const pointers = new Set(held.map(({ pointer }) => pointer));
  const path = ["messages", message.key, "payload", "fields"];
  let after = held.at(-1)?.key;
  return fields
    .filter(({ pointer }) => !pointers.has(pointer))
    .map((value) => {
      const insert = icd.insert({ path, value, after });
      after = value.key;
      return insert;
    });
};

interface AddTopicsParams {
  client: Synnax;
  /** Names the ICD when the task has none. */
  taskName: string;
  entries: ICDReadEntry[];
  drafts: Draft[];
}

/**
 * Appends each draft's fields to the message format of its topic. A topic with no
 * format gets a new message, named after the topic, in the ICD of the task. A task with
 * no ICD gets one, named after the task.
 * @returns the entries of the topics that the task does not read yet.
 */
const addTopics = async ({
  client,
  taskName,
  entries,
  drafts,
}: AddTopicsParams): Promise<ICDReadEntry[]> => {
  const key = entries.at(0)?.icd;
  const messages = key == null ? [] : (await client.icds.retrieve(key)).messages;
  const created: icd.Message[] = [];
  const actions: icd.Action[] = [];
  const added: { topic: string; message: icd.MessageKey }[] = [];
  for (const { topic, fields } of drafts) {
    const entry = entries.find((e) => e.topic === topic);
    let message = entry == null ? undefined : messageOf(messages, entry);
    if (message != null) actions.push(...appendFields(message, fields));
    else {
      const after = (created.at(-1) ?? messages.at(-1))?.key;
      message = {
        key: uuid.create(),
        name: topic,
        payload: { format: "json", fields },
      };
      actions.push(icd.insert({ path: ["messages"], value: message, after }));
      created.push(message);
    }
    if (entry == null) added.push({ topic, message: message.key });
  }
  let icdKey = key;
  if (icdKey == null)
    ({ key: icdKey } = await client.icds.create({ name: taskName, messages: created }));
  else if (actions.length > 0) await client.icds.dispatch(icdKey, actions);
  return added.map(({ topic, message }) =>
    mqtt.icdReadEntryZ.parse({ icd: icdKey, topic, messages: [{ message }] }),
  );
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
  const messages = PForm.useFieldValue(path.messages);
  const [read] = messages;
  const timestamps = fields.filter(({ type }) => type === "timestamp");
  const index = timestamps.find(({ key }) => key === read?.index)?.key;
  const setIndex = (next: string | undefined) =>
    set(path.messages, [{ ...read, index: next }]);
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
  const messagePath = PICD.formPath.messages[messageKey];
  const fieldsPath = deep.variant(messagePath.payload, "format", "json").fields;
  const message = PForm.useFieldValue(messagePath, { optional: true, ctx: format });
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
          {message == null ? null : message.payload.format === "json" ? (
            <PForm.Form<typeof PICD.formSchema> {...format}>
              <ICD.Fields path={fieldsPath} encoding="json" />
              <ICD.FieldsStatus path={fieldsPath} />
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
  /** Null until the task has an ICD. */
  format: FormatForm | null;
}

/** The topics of the task beside the details of the selected one. */
const Topics: FC<TopicsProps> = ({ device, format }) => {
  const client = PSynnax.use();
  const ctx = PForm.useContext();
  const handleError = Status.useErrorHandler();
  const isPreview = Task.useIsPreview();
  const discover = useDiscoverModal();
  const menuProps = Menu.useContextMenu();
  const { data, push, remove } = PForm.useFieldList(root.config.entries);
  const entries = PForm.useFieldValue(root.config.entries);
  const [selected, setSelected] = useState<string[]>([]);
  const current = selected.find((key) => data.includes(key)) ?? data.at(0);

  const handleAdd = () =>
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      const messages = format?.get(PICD.formPath.messages).value ?? [];
      const held = entries.map((e): Draft => ({
        type: "plain",
        topic: e.topic,
        fields: jsonFields(messageOf(messages, e)),
      }));
      const drafts = await discover({ device, entries: held });
      if (drafts == null || drafts.length === 0) return;
      const taskName = ctx.get(root.name).value;
      const added = await addTopics({ client, taskName, entries, drafts });
      push(added);
      const first = entries.find((e) => e.topic === drafts[0].topic) ?? added[0];
      setSelected([first.key]);
    }, "Failed to add data");

  const handleRemove = (keys: string[]) => {
    remove(keys);
    setSelected(selected.filter((k) => !keys.includes(k)));
  };
  const setEnabled = (keys: string[], enabled: boolean) =>
    keys.forEach((k) => ctx.set(root.config.entries[k].disabled, !enabled));
  const contextMenu = ({ keys }: Menu.ContextMenuMenuProps) => {
    const chosen = entries.filter(({ key }) => keys.includes(key));
    return (
      <Task.Views.ContextMenu
        keys={keys}
        onRemove={handleRemove}
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
              value={current == null ? [] : [current]}
              onChange={setSelected}
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
        <TopicDetails key={current} path={path} format={format} />
      )}
    </Task.Views.Panes>
  );
};

interface WithFormatProps {
  device: Device;
  icdKey: icd.Key;
}

/** Opens the form of the task's ICD, which each edit updates at once. */
const WithFormat: FC<WithFormatProps> = ({ device, icdKey }) => {
  const isPreview = Task.useIsPreview();
  const canEdit = Access.useUpdateGranted(icd.resourceID(icdKey));
  const { form, flush } = PICD.useForm({
    query: { key: icdKey },
    mode: isPreview || !canEdit ? "preview" : "normal",
  });
  PICD.useCheck({ key: icdKey, form });
  Task.useFlushBeforeDeploy(flush);
  return <Topics device={device} format={form} />;
};

const Content = ({ device }: PlatformDevice.TaskFormContentProps<Device>) => {
  const icdKey = PForm.useFieldValue(root.config.entries).at(0)?.icd;
  return icdKey == null ? (
    <Topics device={device} format={null} />
  ) : (
    <WithFormat key={icdKey} device={device} icdKey={icdKey} />
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
const formatFields = async (
  client: Synnax,
  entry: ICDReadEntry,
): Promise<icd.JSONField[]> => {
  const { messages } = await client.icds.retrieve(entry.icd);
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
  // Every format is checked before any channel is created.
  const formats = await Promise.all(
    config.entries
      .filter(({ disabled }) => !disabled)
      .map(async (entry) => ({ entry, fields: await formatFields(client, entry) })),
  );
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
      entry.messages = [
        {
          message: read.message,
          index,
          bindings: channels.map(({ key, channel }) => ({ field: key, channel })),
        },
      ];
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
