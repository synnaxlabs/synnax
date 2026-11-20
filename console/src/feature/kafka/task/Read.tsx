// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/kafka/task/Task.css";

import {
  channel,
  DisconnectedError,
  icd,
  kafka,
  type Synnax,
} from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import type { Tree } from "@synnaxlabs/lyra/tree";
import { ICD as PICD, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { deep, errors, unique, uuid } from "@synnaxlabs/x";
import {
  type Dispatch,
  type FC,
  type PropsWithChildren,
  type ReactElement,
  type SetStateAction,
  useRef,
  useState,
} from "react";

import { use } from "@/feature/kafka/device/queries";
import { Select as SelectDevice, useNoneSelected } from "@/feature/kafka/device/Select";
import { type Device, SCHEMAS } from "@/feature/kafka/device/types";
import { useConnectModal } from "@/feature/kafka/device/useConnectModal";
import { useAddDataModal } from "@/feature/kafka/task/AddData";
import {
  type Addition,
  createAdditions,
  createExtensions,
  createRecords,
  type Extension,
  type MessagesOf,
  readsOf,
  uniqueName,
} from "@/feature/kafka/task/additions";
import { DEFAULT_READ_NAME } from "@/feature/kafka/task/create";
import {
  deployReadConfigZ,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadRecord,
  type ReadSchemas,
  type ReadTopic,
  type StartOffset,
} from "@/feature/kafka/task/types";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { ICD } from "@/platform/icd";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<ReadSchemas>>();

const ANY_KEY = "Any key";

const topicPath = (topic: string) => root.config.topics[topic];

const recordPath = (topic: string, record: string) => topicPath(topic).records[record];

/** @returns the key of the topic that holds a record, or "" for a removed record. */
const topicOf = (topics: ReadTopic[], record: string): string =>
  topics.find(({ records }) => records.some(({ key }) => key === record))?.key ?? "";

const SELECT_TRIGGER_PROPS = { className: CSS.BE("kafka-select", "trigger") };

const SELECT_DIALOG_PROPS = { className: CSS.BE("kafka-select", "dialog") };

const TEXT_FIELD_CLASS = CSS.BE("kafka-properties", "text");

const TOPIC_INPUT_PROPS = { placeholder: "sensor-readings" } as const;

const TOPIC_KIND_COLOR = "var(--pluto-primary-z)";

const GROUP_INPUT_PROPS = { placeholder: "Derived from the task" } as const;

const RECORD_KEY_INPUT_PROPS = { placeholder: ANY_KEY } as const;

const SelectStartOffset = (
  p: Omit<Select.SingleSimpleProps<StartOffset>, "children" | "resourceName">,
) => (
  <Select.Simple<StartOffset>
    {...p}
    resourceName="start offset"
    triggerProps={SELECT_TRIGGER_PROPS}
    dialogProps={SELECT_DIALOG_PROPS}
  >
    <Select.Item itemKey="latest">Latest</Select.Item>
    <Select.Item itemKey="earliest">Earliest</Select.Item>
    <Select.Item itemKey="none">Committed only</Select.Item>
  </Select.Simple>
);

const renderStartOffsetSelect = Component.renderProp(SelectStartOffset);

const Properties = () => (
  <>
    <SelectDevice />
    <Flex.Box x grow wrap>
      <PForm.TextField
        className={TEXT_FIELD_CLASS}
        path={root.config.group}
        label="Consumer group"
        inputProps={GROUP_INPUT_PROPS}
        grow
      />
      <PForm.Field path={root.config.startOffset} label="Start from">
        {renderStartOffsetSelect}
      </PForm.Field>
    </Flex.Box>
  </>
);

/** @returns the record key, or a stand-in for a record that reads any key. */
const recordLabel = ({ recordKey }: Pick<ReadRecord, "recordKey">): string =>
  recordKey === "" ? ANY_KEY : recordKey;

/**
 * The prefix of the names of the channels that deploying creates for a message. A
 * message after the first adds its name to the prefix of the record.
 */
const messagePrefix = (
  deviceName: string,
  topic: string,
  recordKey: string,
  read: number,
  name: string,
): string => {
  let prefix = `${channel.escapeInvalidName(deviceName)}_${channel.escapeInvalidName(topic)}`;
  if (recordKey !== "") prefix += `_${channel.escapeInvalidName(recordKey)}`;
  return read === 0 ? prefix : `${prefix}_${channel.escapeInvalidName(name)}`;
};

/** @returns the key of the device properties that store the channels of a message. */
const storedKey = (topic: string, recordKey: string, message: icd.MessageKey): string =>
  `${topic}#${recordKey}#${message}`;

/** @returns the channels that enabled records bind. The Core skips disabled ones. */
const writtenChannels = (topics: ReadTopic[]): Set<channel.Key> => {
  const written = new Set(
    topics
      .flatMap(({ records }) => records)
      .filter(({ disabled }) => !disabled)
      .flatMap(({ messages }) =>
        messages.flatMap(({ bindings }) => bindings.map((b) => b.channel)),
      ),
  );
  written.delete(0);
  return written;
};

const createMessage = (name: string, match?: icd.JSONMatch): icd.Message => ({
  key: uuid.create(),
  name,
  payload: { format: "json", fields: [], ...(match == null ? {} : { match }) },
});

/** @returns the name of the ICD of a task named name. */
const icdName = (name: string): string =>
  name.trim() === "" ? DEFAULT_READ_NAME : name;

/** @returns the prefix of the channel names of a message, current with its name. */
const usePrefix = (
  deviceName: string,
  topicKey: string,
  path: deep.Path<ReadRecord>,
  read: number,
): string => {
  const formats = ICD.useFormats();
  const topic = PForm.useFieldValue(topicPath(topicKey).name);
  const recordKey = PForm.useFieldValue(path.recordKey);
  const icdKey = PForm.useFieldValue(path.icd);
  const message = PForm.useFieldValue(path.messages[read].message);
  const name = PForm.useFieldValue(PICD.formPath.messages[message].name, {
    optional: true,
    ctx: formats.get(icdKey),
  });
  return messagePrefix(deviceName, topic, recordKey, read, name ?? "");
};

/** Names a record by its record key. */
const RecordLabel = ({ path }: { path: deep.Path<ReadRecord> }) => {
  const recordKey = PForm.useFieldValue(path.recordKey);
  return (
    <Flex.Box x align="center" className={CSS.B("kafka-record-label")}>
      <Task.Views.ItemLabel
        kind="JSON"
        kindColor="var(--pluto-secondary-z)"
        color={recordKey === "" ? 8 : 10}
        variant="code"
      >
        {recordLabel({ recordKey })}
      </Task.Views.ItemLabel>
    </Flex.Box>
  );
};

/** Names a topic. */
const TopicLabel = ({ path }: { path: deep.Path<ReadTopic> }) => {
  const name = PForm.useFieldValue(path.name);
  return (
    <Task.Views.ItemLabel
      kind="Topic"
      kindColor={TOPIC_KIND_COLOR}
      color={name === "" ? 8 : 10}
      variant="code"
    >
      {name === "" ? "New topic" : name}
    </Task.Views.ItemLabel>
  );
};

const TopicItem = (props: Tree.ItemRenderProps<string>) => {
  const path = topicPath(props.itemKey);
  const name = PForm.useFieldValue(path.name, { optional: true });
  if (name == null) return null;
  return <Task.Views.TreeGroupItem {...props} label={<TopicLabel path={path} />} />;
};

const RecordItem = (props: Tree.ItemRenderProps<string>) => {
  const ctx = PForm.useContext();
  const topic = topicOf(ctx.get(root.config.topics).value, props.itemKey);
  const path = recordPath(topic, props.itemKey);
  const disabled = PForm.useFieldValue(path.disabled, { optional: true });
  if (disabled == null) return null;
  return (
    <Task.Views.TreeGroupItem
      {...props}
      label={<RecordLabel path={path} />}
      off={disabled}
    >
      <Task.EnabledCheckbox path={path.disabled} />
    </Task.Views.TreeGroupItem>
  );
};

/** @returns the props that the panels of a message of a record share. */
const useMessageProps = (
  device: Device,
  topic: string,
  path: deep.Path<ReadRecord>,
  read: number,
): Omit<ICD.MessageSectionsProps, "format"> => {
  const { index, bindings } = PForm.useFieldValue(path.messages[read]);
  const prefix = usePrefix(device.name, topic, path, read);
  return {
    path,
    read,
    prefix,
    // A new index field keeps the channel of the old one.
    indexChannel: bindings.find(({ field }) => field === index)?.channel ?? 0,
    sourceLabel: "Kafka timestamp",
    missingDescription:
      "The ICD no longer has this message. Add a message to the record, then remove this one.",
  };
};

interface MessagePanelProps {
  /** The key of the topic of the record. */
  topic: string;
  path: deep.Path<ReadRecord>;
  /** The index of the message in the record's reads. */
  read: number;
  format: ICD.FormatForm;
  device: Device;
}

const MessageSections = ({ device, topic, path, read, format }: MessagePanelProps) => {
  const props = useMessageProps(device, topic, path, read);
  return <ICD.MessageSections {...props} format={format} />;
};

interface MessageDetailsProps extends MessagePanelProps {
  onAddField: () => void;
}

const MessageDetails = ({
  device,
  topic,
  path,
  read,
  format,
  onAddField,
}: MessageDetailsProps) => {
  const props = useMessageProps(device, topic, path, read);
  return (
    <ICD.MessageDetails
      {...props}
      format={format}
      matchTitle="Values"
      anyLabel="Any value"
      onAddField={onAddField}
    />
  );
};

interface FieldDetailsProps extends MessagePanelProps {
  field: ICD.TreeField;
}

const FieldDetails = ({ device, topic, ...rest }: FieldDetailsProps) => {
  const prefix = usePrefix(device.name, topic, rest.path, rest.read);
  return <ICD.FieldDetails {...rest} prefix={prefix} />;
};

/** The name of a topic. */
const TopicDetails = ({ path }: { path: deep.Path<ReadTopic> }) => (
  <Flex.Box y grow empty className={CSS.BE("details", "form")}>
    <PForm.Sections>
      <PForm.Section title="Topic">
        <PForm.TextField
          path={path.name}
          label="Name"
          padHelpText={false}
          inputProps={TOPIC_INPUT_PROPS}
        />
      </PForm.Section>
    </PForm.Sections>
  </Flex.Box>
);

interface RecordDetailsProps {
  /** The key of the topic of the record. */
  topic: string;
  path: deep.Path<ReadRecord>;
  format: ICD.FormatForm;
  device: Device;
  /** Adds a field to the message of a record that reads one. */
  onAddField: () => void;
  /** Adds a message to the record, for values that its messages do not cover. */
  onAddMessage: () => void;
}

/**
 * The record key of a record. A record that reads one message also shows its timestamp,
 * which each message of a record that reads several shows on its own.
 */
const RecordDetails: FC<RecordDetailsProps> = ({
  topic,
  path,
  format,
  device,
  onAddField,
  onAddMessage,
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
          title="Record"
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
            path={path.recordKey}
            label="Record key"
            padHelpText={false}
            inputProps={RECORD_KEY_INPUT_PROPS}
          />
        </PForm.Section>
        {reads.length === 1 && (
          <MessageSections
            topic={topic}
            path={path}
            read={0}
            format={format}
            device={device}
          />
        )}
      </PForm.Sections>
    </Flex.Box>
  );
};

interface RecordsProps {
  device: Device;
  /** The key of each selected tree item. */
  selected: string[];
  onSelect: Dispatch<SetStateAction<string[]>>;
  expanded: string[];
  onExpand: Dispatch<SetStateAction<string[]>>;
}

/** The topics of the task as a tree, beside the details of the selected item. */
const Records: FC<RecordsProps> = ({
  device,
  selected,
  onSelect,
  expanded,
  onExpand,
}) => {
  const client = PSynnax.use();
  const ctx = PForm.useContext();
  const handleError = Status.useErrorHandler();
  const formats = ICD.useFormats();
  const topics = PForm.useFieldValue(root.config.topics);
  const records = topics.flatMap((t) => t.records);
  const stored = device.properties.read;
  const pathOf = (record: string) =>
    recordPath(topicOf(ctx.get(root.config.topics).value, record), record);
  const {
    tree,
    handleSelect,
    contextMenu,
    renderItem,
    current,
    currentGroup,
    addField,
    addMessage,
  } = ICD.Tree.use({
    groups: {
      path: root.config.topics,
      sourcesOf: (topic) => topicPath(topic).records,
      Item: TopicItem,
    },
    sourceOf: (r) => r,
    pathOf,
    SourceItem: RecordItem,
    // Starting reuses the channel that the device stores for the field's pointer,
    // unless an enabled record reads it.
    bindingOf: (key, read, { key: field, pointer }) => {
      const all = ctx.get(root.config.topics).value;
      const topic = ctx.get(topicPath(topicOf(all, key))).value;
      const { recordKey, messages } = ctx.get(pathOf(key)).value;
      const ch =
        stored[storedKey(topic.name, recordKey, messages[read].message)]?.channels[
          pointer
        ] ?? 0;
      const written = writtenChannels(all);
      return icd.readBindingZ.parse({ field, channel: written.has(ch) ? 0 : ch });
    },
    messageName: (key, held) => {
      const rec = ctx.get(pathOf(key)).value;
      return uniqueName(`${recordLabel(rec)} ${rec.messages.length + 1}`, held);
    },
    selected,
    onSelect,
    expanded,
    onExpand,
  });

  const addData = useAddDataModal();
  const label = (recordKey: string) => recordLabel({ recordKey });

  /**
   * Adds records to the topic named name, or adds that topic with the records in one
   * change when the task has none.
   * @returns the key of the topic.
   */
  const pushRecords = (name: string, added: ReadRecord[]): string => {
    const held = ctx.get(root.config.topics).value.find((t) => t.name === name);
    if (held != null) {
      PForm.fieldListUtils(ctx, topicPath(held.key).records).push(added);
      return held.key;
    }
    const topic = kafka.readTopicZ.parse({ name, records: added });
    PForm.fieldListUtils(ctx, root.config.topics).push(topic);
    return topic.key;
  };

  /**
   * Puts messages in the ICD of the first record, or in a new ICD for a task with no
   * record, then adds the records that read them to their topics.
   * @param recordsOf - Returns the records to add to each topic, by topic name.
   */
  const addRecords = async (
    messages: icd.Message[],
    recordsOf: (icdKey: icd.Key) => Map<string, ReadRecord[]>,
  ) => {
    if (client == null) throw new DisconnectedError();
    const first = records.at(0);
    let icdKey: icd.Key;
    if (first == null) {
      const name = icdName(ctx.get(root.name).value);
      icdKey = (await client.icds.create({ name, messages })).key;
    } else {
      icdKey = first.icd;
      const format = ICD.formatOf(formats, icdKey);
      let after = format.value().messages.at(-1)?.key;
      format.batch(() => {
        for (const message of messages) {
          format.insert(PICD.formPath.messages, message, after);
          after = message.key;
        }
      });
    }
    const added = recordsOf(icdKey);
    const keys = [...added].map(([topic, recs]) => pushRecords(topic, recs));
    const [{ key: shown }] = [...added.values()].flat();
    onSelect([shown]);
    onExpand((prev) => unique.unique([...prev, ...keys, shown]));
  };

  /** @returns the messages of the ICD of the first record. */
  const heldMessages = (): icd.Message[] => {
    const first = records.at(0);
    return first == null ? [] : ICD.formatOf(formats, first.icd).value().messages;
  };

  /** @returns the name of the topic of the selected item, or of the only topic. */
  const selectedTopic = (): string => {
    let key = currentGroup;
    if (key == null && current != null) key = topicOf(topics, current.source);
    if (key == null && topics.length === 1) key = topics[0].key;
    return topics.find((t) => t.key === key)?.name ?? "";
  };

  /** @returns the records of the topic named name. */
  const topicRecords = (name: string): ReadRecord[] =>
    topics.find((t) => t.name === name)?.records ?? [];

  const messagesOf: MessagesOf = (icdKey) =>
    ICD.formatOf(formats, icdKey).value().messages;

  /** Puts new fields in messages that records read, and reads them on those records. */
  const extend = (extensions: Extension[]) =>
    ctx.batch(() => {
      for (const { icd: icdKey, message, fields, bindings } of extensions) {
        const format = ICD.formatOf(formats, icdKey);
        let after = ICD.fieldsOf(formats, icdKey, message).at(-1)?.key;
        format.batch(() => {
          for (const field of fields) {
            format.insert(ICD.fieldsPath(message), field, after);
            after = field.key;
          }
        });
        for (const { record, read, fields: keys } of bindings) {
          const path = pathOf(record).messages[read].bindings;
          ctx.set(path, [
            ...ctx.get(path).value,
            ...keys.map((field) => icd.readBindingZ.parse({ field })),
          ]);
        }
      }
    });

  /** Adds the fields that the user picks from topics, and any empty record. */
  const add = async () => {
    const result = await addData({
      device,
      topic: selectedTopic(),
      topics: topics.map(({ name }) => name),
      reads: (name) => readsOf(topicRecords(name), messagesOf),
      label,
    });
    if (result == null) return;
    const additions = new Map<string, Addition[]>();
    let held = heldMessages();
    for (const { topic, shapes, picks } of result.topics) {
      const records = topicRecords(topic);
      extend(createExtensions(shapes, picks, records, messagesOf));
      const reads = readsOf(records, messagesOf);
      const added = createAdditions(shapes, picks, reads, held, label);
      if (added.length === 0) continue;
      additions.set(topic, added);
      held = [...held, ...added.map(({ message }) => message)];
    }
    if (result.record != null) {
      const empty = {
        message: createMessage(uniqueName(ANY_KEY, held)),
        recordKeys: [""],
      };
      additions.set(result.record, [...(additions.get(result.record) ?? []), empty]);
    }
    if (additions.size === 0) return;
    await addRecords(
      [...additions.values()].flat().map(({ message }) => message),
      (icdKey) =>
        new Map(
          [...additions].map(([topic, added]) => [topic, createRecords(icdKey, added)]),
        ),
    );
  };

  // Adds run one at a time, so a second click cannot make a second ICD.
  const adding = useRef(false);
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

  const rec =
    current == null ? undefined : records.find(({ key }) => key === current.source);
  const topic = rec == null ? "" : topicOf(topics, rec.key);
  const format = rec == null ? undefined : formats.get(rec.icd);
  let details = null;
  let title = null;
  let detailsPath: deep.Target<unknown> | null = null;
  if (currentGroup != null) {
    detailsPath = topicPath(currentGroup);
    title = <TopicLabel path={topicPath(currentGroup)} />;
    details = <TopicDetails key={currentGroup} path={topicPath(currentGroup)} />;
  } else if (rec != null && format != null && current?.kind === "field") {
    const read = recordPath(topic, rec.key).messages[current.read];
    const i = rec.messages[current.read].bindings.findIndex(
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
        topic={topic}
        path={recordPath(topic, rec.key)}
        read={current.read}
        field={current.field}
        format={format}
        device={device}
      />
    );
  } else if (rec != null && format != null && current?.kind === "message") {
    detailsPath = recordPath(topic, rec.key).messages[current.read];
    title = <ICD.MessageLabel path={recordPath(topic, rec.key)} read={current.read} />;
    details = (
      <MessageDetails
        key={selected[0]}
        topic={topic}
        path={recordPath(topic, rec.key)}
        read={current.read}
        format={format}
        device={device}
        onAddField={() => addField(rec.key, current.read)}
      />
    );
  } else if (rec != null) {
    detailsPath = recordPath(topic, rec.key);
    title = <RecordLabel path={recordPath(topic, rec.key)} />;
    if (format != null)
      details = (
        <RecordDetails
          key={rec.key}
          topic={topic}
          path={recordPath(topic, rec.key)}
          format={format}
          device={device}
          onAddField={() => addField(rec.key, 0)}
          onAddMessage={() => addMessage(rec.key)}
        />
      );
  }
  return (
    <Task.Views.Panes
      listTitle="Topics"
      list={
        <Task.Views.TreeList
          {...tree}
          onSelect={handleSelect}
          menu={contextMenu}
          emptyMessage="No topics"
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
          <Text.Text status="disabled">
            Add data to read the records of a topic
          </Text.Text>
        </Flex.Box>
      )}
    </Task.Views.Panes>
  );
};

/** A message that a record of a topic reads. */
interface BoundMessage extends ICD.BoundMessage<ReadRecord> {
  topic: ReadTopic;
}

/** @returns the key of the device properties that store the channels of a message. */
const storedKeyOf = ({ topic, source, read }: BoundMessage): string =>
  storedKey(topic.name, source.recordKey, source.messages[read].message);

/** @returns the prefix of the names of the channels that a message creates. */
const prefixOf = (deviceName: string, { topic, source, read, message }: BoundMessage) =>
  messagePrefix(deviceName, topic.name, source.recordKey, read, message.name);

/**
 * Plans the channels of each message, in deploy order with one namer. A message plans
 * with the channels that earlier ones keep added to written, so it leaves a stored
 * channel that another binding writes. Reads from the Core alone. Throws between
 * messages once signal aborts.
 */
const planRead = async (
  client: Synnax,
  dev: Pick<Device, "name" | "properties">,
  bound: BoundMessage[],
  written: Set<channel.Key>,
  signal?: AbortSignal,
): Promise<ICD.MessagePlan[]> => {
  const namer = Task.createNamer(client);
  const plans: ICD.MessagePlan[] = [];
  for (const b of bound) {
    signal?.throwIfAborted();
    const plan = await ICD.planMessage(
      client,
      namer,
      b,
      prefixOf(dev.name, b),
      dev.properties.read[storedKeyOf(b)],
      written,
    );
    ICD.keptChannels(plan).forEach((ch) => written.add(ch));
    plans.push(plan);
  }
  return plans;
};

interface PlansProps extends PropsWithChildren {
  device: Device;
}

/**
 * Plans the channels of every message that the task reads, together and as deploying
 * plans them, for the tree and the panes.
 */
const Plans = ({ device, children }: PlansProps): ReactElement => {
  const topics = PForm.useFieldValue(root.config.topics);
  const messages = ICD.useMessages();
  const bound = topics.flatMap((topic) =>
    topic.records
      .filter(({ disabled }) => !disabled)
      .flatMap((source) =>
        source.messages.flatMap((read, r): BoundMessage[] => {
          const message = ICD.messageOf(messages, source, r);
          if (message == null) return [];
          const fields = ICD.readFields(ICD.jsonFields(message), read);
          return [{ topic, source, read: r, message, fields }];
        }),
      ),
  );
  const sources = bound.map((b) => ({
    key: b.source.key,
    position: b.read,
    read: b.source.messages[b.read],
    message: b.message,
    prefix: prefixOf(device.name, b),
    props: device.properties.read[storedKeyOf(b)],
  }));
  const plans = Task.useChannelPlans(
    sources,
    async (client, signal) =>
      await planRead(client, device, bound, writtenChannels(topics), signal),
  );
  const reads = bound.map(({ topic, source, read }, i): ICD.PlannedRead => ({
    path: recordPath(topic.key, source.key),
    read,
    plan: plans[i],
  }));
  return <ICD.Plans plans={reads}>{children}</ICD.Plans>;
};

const Content = ({ device }: PlatformDevice.TaskFormContentProps<Device>) => {
  const topics = PForm.useFieldValue(root.config.topics);
  const records = topics.flatMap((t) => t.records);
  // The tree selects its first item, a topic, when nothing is selected.
  const [selected, setSelected] = useState<string[]>(() =>
    records.length === 0 ? [] : [records[0].key],
  );
  const [expanded, setExpanded] = useState<string[]>(() => [
    ...topics.map(({ key }) => key),
    ...ICD.Tree.initialExpanded(records),
  ]);
  const body = (
    <Plans device={device}>
      <Records
        device={device}
        selected={selected}
        onSelect={setSelected}
        expanded={expanded}
        onExpand={setExpanded}
      />
      {topics.flatMap((topic) =>
        topic.records.flatMap(({ key, messages }) =>
          messages.map(({ message }, read) => (
            <ICD.SuitRead
              key={`${key}/${message}`}
              path={recordPath(topic.key, key)}
              read={read}
            />
          )),
        ),
      )}
    </Plans>
  );
  return unique.unique(records.map((r) => r.icd)).reduceRight(
    (child, key) => (
      <ICD.WithFormat key={key} icdKey={key}>
        {child}
      </ICD.WithFormat>
    ),
    body,
  );
};

const Form = PlatformDevice.wrapTaskForm({
  use,
  useConfigure: useConnectModal,
  useNoneSelected,
  Content,
});

/** Gives each field that an enabled record reads a channel, and binds it. */
const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (client, config) => {
  const dev = await client.devices.retrieve({ key: config.device, schemas: SCHEMAS });
  const enabled = config.topics.flatMap((topic) =>
    topic.records.filter(({ disabled }) => !disabled).map((rec) => ({ topic, rec })),
  );
  const messages = await ICD.retrieveMessages(
    client,
    enabled.map(({ rec }) => rec.icd),
  );
  // Every format is checked before any channel is created.
  const bound = enabled.flatMap(({ topic, rec }) =>
    rec.messages.map((_, read): BoundMessage => ({
      ...ICD.boundMessage(
        messages,
        rec,
        read,
        `record ${recordLabel(rec)} in topic ${topic.name}`,
      ),
      topic,
    })),
  );
  // A deploy that fails partway keeps the channels it made for the next one.
  const before = deep.copy(dev.properties);
  try {
    const plans = await planRead(client, dev, bound, writtenChannels(config.topics));
    for (const [i, b] of bound.entries())
      await ICD.applyMessage(client, b, plans[i], {
        read: dev.properties.read,
        key: storedKeyOf(b),
      });
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
