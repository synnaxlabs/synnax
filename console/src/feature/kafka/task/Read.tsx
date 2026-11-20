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
  type icd,
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
  type StartOffset,
} from "@/feature/kafka/task/types";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { ICD } from "@/platform/icd";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<ReadSchemas>>();

const ANY_KEY = "Any key";

const recordPath = (key: string) => root.config.records[key];

const SELECT_TRIGGER_PROPS = { className: CSS.BE("kafka-select", "trigger") };

const SELECT_DIALOG_PROPS = { className: CSS.BE("kafka-select", "dialog") };

const TEXT_FIELD_CLASS = CSS.BE("kafka-properties", "text");

const TOPIC_INPUT_PROPS = { placeholder: "sensor-readings" } as const;

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
        path={root.config.topic}
        label="Topic"
        inputProps={TOPIC_INPUT_PROPS}
        grow
      />
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
const writtenChannels = (records: ReadRecord[]): Set<channel.Key> => {
  const written = new Set(
    records
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

const recordOf = (icdKey: icd.Key, message: icd.MessageKey): ReadRecord =>
  kafka.readRecordZ.parse({ icd: icdKey, messages: [{ message }] });

/** @returns the name of the ICD of a task named name. */
const icdName = (name: string): string =>
  name.trim() === "" ? DEFAULT_READ_NAME : name;

/** @returns the prefix of the channel names of a message, current with its name. */
const usePrefix = (
  deviceName: string,
  path: deep.Path<ReadRecord>,
  read: number,
): string => {
  const formats = ICD.useFormats();
  const topic = PForm.useFieldValue(root.config.topic);
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

const RecordItem = (props: Tree.ItemRenderProps<string>) => {
  const path = recordPath(props.itemKey);
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
  path: deep.Path<ReadRecord>,
  read: number,
): Omit<ICD.MessageSectionsProps, "format"> => {
  const { index, bindings } = PForm.useFieldValue(path.messages[read]);
  const prefix = usePrefix(device.name, path, read);
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
  path: deep.Path<ReadRecord>;
  /** The index of the message in the record's reads. */
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
      matchTitle="Values"
      anyLabel="Any value"
      onAddField={onAddField}
    />
  );
};

interface FieldDetailsProps extends MessagePanelProps {
  field: ICD.TreeField;
}

const FieldDetails = ({ device, ...rest }: FieldDetailsProps) => {
  const prefix = usePrefix(device.name, rest.path, rest.read);
  return <ICD.FieldDetails {...rest} prefix={prefix} />;
};

interface RecordDetailsProps {
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
          <MessageSections path={path} read={0} format={format} device={device} />
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

/** The records of the task as a tree, beside the details of the selected item. */
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
  const { push } = PForm.useFieldList(root.config.records);
  const records = PForm.useFieldValue(root.config.records);
  const stored = device.properties.read;
  const { tree, handleSelect, contextMenu, renderItem, current, addField, addMessage } =
    ICD.Tree.use({
      sources: root.config.records,
      sourceOf: (r) => r,
      pathOf: recordPath,
      SourceItem: RecordItem,
      // Starting reuses the channel that the device stores for the field's pointer,
      // unless an enabled record reads it.
      bindingOf: (key, read, { key: field, pointer }) => {
        const { recordKey, messages } = ctx.get(recordPath(key)).value;
        const topic = ctx.get(root.config.topic).value;
        const ch =
          stored[storedKey(topic, recordKey, messages[read].message)]?.channels[
            pointer
          ] ?? 0;
        const written = writtenChannels(ctx.get(root.config.records).value);
        return kafka.readBindingZ.parse({ field, channel: written.has(ch) ? 0 : ch });
      },
      messageName: (key, held) => {
        const rec = ctx.get(recordPath(key)).value;
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
   * Puts messages in the ICD of the first record, or in a new ICD for a task with no
   * record, then adds the records that read them.
   */
  const addRecords = async (
    messages: icd.Message[],
    recordsOf: (icdKey: icd.Key) => ReadRecord[],
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
    push(added);
    onSelect([added[0].key]);
    tree.expand(added[0].key);
  };

  /** @returns the messages of the ICD of the first record. */
  const heldMessages = (): icd.Message[] => {
    const first = records.at(0);
    return first == null ? [] : ICD.formatOf(formats, first.icd).value().messages;
  };

  /** Adds a record that reads any key. */
  const addRecord = async () => {
    const message = createMessage(uniqueName(ANY_KEY, heldMessages()));
    await addRecords([message], (icdKey) => [recordOf(icdKey, message.key)]);
  };

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
          const path = recordPath(record).messages[read].bindings;
          ctx.set(path, [
            ...ctx.get(path).value,
            ...keys.map((field) => kafka.readBindingZ.parse({ field })),
          ]);
        }
      }
    });

  /** Adds the fields that the user picks from records of the topic, or one record. */
  const add = async () => {
    const topic = ctx.get(root.config.topic).value;
    const reads = readsOf(records, messagesOf);
    const result = await addData({ device, topic, reads, label });
    if (result == null) return;
    if (result.variant === "record") return await addRecord();
    const { shapes, picks } = result;
    extend(createExtensions(shapes, picks, records, messagesOf));
    const additions = createAdditions(shapes, picks, records, heldMessages(), label);
    if (additions.length === 0) return;
    await addRecords(
      additions.map(({ message }) => message),
      (icdKey) => createRecords(icdKey, additions),
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
  const format = rec == null ? undefined : formats.get(rec.icd);
  let details = null;
  let title = null;
  let detailsPath: deep.Target<unknown> | null = null;
  if (rec != null && format != null && current?.kind === "field") {
    const read = recordPath(rec.key).messages[current.read];
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
        path={recordPath(rec.key)}
        read={current.read}
        field={current.field}
        format={format}
        device={device}
      />
    );
  } else if (rec != null && format != null && current?.kind === "message") {
    detailsPath = recordPath(rec.key).messages[current.read];
    title = <ICD.MessageLabel path={recordPath(rec.key)} read={current.read} />;
    details = (
      <MessageDetails
        key={selected[0]}
        path={recordPath(rec.key)}
        read={current.read}
        format={format}
        device={device}
        onAddField={() => addField(rec.key, current.read)}
      />
    );
  } else if (rec != null) {
    detailsPath = recordPath(rec.key);
    title = <RecordLabel path={recordPath(rec.key)} />;
    if (format != null)
      details = (
        <RecordDetails
          key={rec.key}
          path={recordPath(rec.key)}
          format={format}
          device={device}
          onAddField={() => addField(rec.key, 0)}
          onAddMessage={() => addMessage(rec.key)}
        />
      );
  }
  return (
    <Task.Views.Panes
      listTitle="Records"
      list={
        <Task.Views.TreeList
          {...tree}
          onSelect={handleSelect}
          menu={contextMenu}
          emptyMessage="No records"
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
            Add data to read the records of the topic
          </Text.Text>
        </Flex.Box>
      )}
    </Task.Views.Panes>
  );
};

/**
 * Plans the channels of each message, in deploy order with one namer. A message plans
 * with the channels that earlier ones keep added to written, so it leaves a stored
 * channel that another binding writes. Reads from the Core alone.
 */
const planRead = async (
  client: Synnax,
  dev: Pick<Device, "name" | "properties">,
  topic: string,
  bound: ICD.BoundMessage<ReadRecord>[],
  written: Set<channel.Key>,
): Promise<ICD.MessagePlan[]> => {
  const namer = Task.createNamer(client);
  const plans: ICD.MessagePlan[] = [];
  for (const b of bound) {
    const { recordKey, messages } = b.source;
    const plan = await ICD.planMessage(
      client,
      namer,
      b,
      messagePrefix(dev.name, topic, recordKey, b.read, b.message.name),
      dev.properties.read[storedKey(topic, recordKey, messages[b.read].message)],
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
  const topic = PForm.useFieldValue(root.config.topic);
  const records = PForm.useFieldValue(root.config.records);
  const messages = ICD.useMessages();
  const bound = records
    .filter(({ disabled }) => !disabled)
    .flatMap((source) =>
      source.messages.flatMap((read, r) => {
        const message = ICD.messageOf(messages, source, r);
        if (message == null) return [];
        const fields = ICD.readFields(ICD.jsonFields(message), read);
        return [{ source, read: r, message, fields }];
      }),
    );
  const sources = bound.map(({ source: { recordKey, messages }, read, message }) => ({
    recordKey,
    read: messages[read],
    message,
    prefix: messagePrefix(device.name, topic, recordKey, read, message.name),
    props: device.properties.read[storedKey(topic, recordKey, messages[read].message)],
  }));
  const plans = Task.useChannelPlans(
    sources,
    async (client) =>
      await planRead(client, device, topic, bound, writtenChannels(records)),
  );
  const reads = bound.map(({ source, read }, i): ICD.PlannedRead => ({
    path: recordPath(source.key),
    read,
    plan: plans[i],
  }));
  return <ICD.Plans plans={reads}>{children}</ICD.Plans>;
};

const Content = ({ device }: PlatformDevice.TaskFormContentProps<Device>) => {
  const records = PForm.useFieldValue(root.config.records);
  const [selected, setSelected] = useState<string[]>([]);
  const [expanded, setExpanded] = useState<string[]>(() =>
    ICD.Tree.initialExpanded(records),
  );
  const body = (
    <Plans device={device}>
      <Records
        device={device}
        selected={selected}
        onSelect={setSelected}
        expanded={expanded}
        onExpand={setExpanded}
      />
      {records.flatMap(({ key, messages }) =>
        messages.map(({ message }, read) => (
          <ICD.SuitRead key={`${key}/${message}`} path={recordPath(key)} read={read} />
        )),
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
  const enabled = config.records.filter(({ disabled }) => !disabled);
  const messages = await ICD.retrieveMessages(
    client,
    enabled.map((r) => r.icd),
  );
  // Every format is checked before any channel is created.
  const bound = enabled.flatMap((rec) =>
    rec.messages.map((_, read) =>
      ICD.boundMessage(messages, rec, read, `record ${recordLabel(rec)}`),
    ),
  );
  // A deploy that fails partway keeps the channels it made for the next one.
  const before = deep.copy(dev.properties);
  try {
    const plans = await planRead(
      client,
      dev,
      config.topic,
      bound,
      writtenChannels(config.records),
    );
    for (const [i, b] of bound.entries()) {
      const { recordKey, messages } = b.source;
      await ICD.applyMessage(client, b, plans[i], {
        read: dev.properties.read,
        key: storedKey(config.topic, recordKey, messages[b.read].message),
      });
    }
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
