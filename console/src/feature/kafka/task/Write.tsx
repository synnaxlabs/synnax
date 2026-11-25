// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/kafka/task/Task.css";

import { channel, DisconnectedError, type icd, kafka } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { useExclusiveCallback } from "@synnaxlabs/lyra/hooks";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import type { Tree } from "@synnaxlabs/lyra/tree";
import { Channel as PChannel, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { DataType, deep, id, primitive, unique } from "@synnaxlabs/x";
import {
  type Dispatch,
  type FC,
  type ReactNode,
  type SetStateAction,
  useState,
} from "react";

import { Device } from "@/feature/kafka/device";
import { DEFAULT_WRITE_NAME } from "@/feature/kafka/task/create";
import { RecordLabel, TopicLabel } from "@/feature/kafka/task/Labels";
import {
  deployICDWriteConfigZ,
  deployWriteConfigZ,
  isString,
  type RecordKey,
  type TimeFormat,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteChannel,
  type WriteSchemas,
  type WriteTarget,
} from "@/feature/kafka/task/types";
import { FLAGS } from "@/flags";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { ICD } from "@/platform/icd";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<WriteSchemas>>();

const SELECT_TRIGGER_PROPS = { className: CSS.BE("kafka-select", "trigger") };

const TEXT_FIELD_CLASS = CSS.BE("kafka-properties", "text");

const SELECT_DIALOG_PROPS = { className: CSS.BE("kafka-select", "dialog") };

const SelectRecordKey = (
  p: Omit<Select.SingleSimpleProps<RecordKey>, "children" | "resourceName">,
) => (
  <Select.Simple<RecordKey>
    {...p}
    resourceName="record key"
    triggerProps={SELECT_TRIGGER_PROPS}
    dialogProps={SELECT_DIALOG_PROPS}
  >
    <Select.Item itemKey="channel_name">Channel name</Select.Item>
    <Select.Item itemKey="none">None</Select.Item>
  </Select.Simple>
);

const renderRecordKeySelect = Component.renderProp(SelectRecordKey);

const SelectTimeFormat = (
  p: Omit<Select.SingleSimpleProps<TimeFormat>, "children" | "resourceName">,
) => (
  <Select.Simple<TimeFormat>
    {...p}
    resourceName="time format"
    triggerProps={SELECT_TRIGGER_PROPS}
    dialogProps={SELECT_DIALOG_PROPS}
  >
    <Select.Item itemKey="iso8601">ISO 8601</Select.Item>
    <Select.Item itemKey="unix_sec">Unix (s)</Select.Item>
    <Select.Item itemKey="unix_ms">Unix (ms)</Select.Item>
    <Select.Item itemKey="unix_us">Unix (µs)</Select.Item>
    <Select.Item itemKey="unix_ns">Unix (ns)</Select.Item>
  </Select.Simple>
);

const renderTimeFormatSelect = Component.renderProp(SelectTimeFormat);

const Properties = () => (
  <>
    <Device.Select />
    <Flex.Box x grow wrap>
      <PForm.TextField
        className={TEXT_FIELD_CLASS}
        path={root.config.topic}
        label="Topic"
        inputProps={TOPIC_INPUT_PROPS}
        grow
      />
      <PForm.Field path={root.config.recordKey} label="Record key">
        {renderRecordKeySelect}
      </PForm.Field>
    </Flex.Box>
    <Flex.Box x grow wrap>
      <PForm.TextField
        className={TEXT_FIELD_CLASS}
        path={root.config.record.valuePointer}
        label="Value pointer"
        inputProps={VALUE_POINTER_INPUT_PROPS}
        grow
      />
      <PForm.TextField
        className={TEXT_FIELD_CLASS}
        path={root.config.record.channelPointer}
        label="Channel pointer"
        inputProps={CHANNEL_POINTER_INPUT_PROPS}
        grow
      />
      <PForm.TextField
        className={TEXT_FIELD_CLASS}
        path={root.config.record.timestampPointer}
        label="Timestamp pointer"
        inputProps={TIMESTAMP_POINTER_INPUT_PROPS}
        grow
      />
      <PForm.Field path={root.config.record.timeFormat} label="Time format">
        {renderTimeFormatSelect}
      </PForm.Field>
    </Flex.Box>
  </>
);

const TOPIC_INPUT_PROPS = { placeholder: "synnax-samples" } as const;

const VALUE_POINTER_INPUT_PROPS = { placeholder: "/value" } as const;

const CHANNEL_POINTER_INPUT_PROPS = { placeholder: "Omitted" } as const;

const TIMESTAMP_POINTER_INPUT_PROPS = { placeholder: "Omitted" } as const;

const ChannelListItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = root.config.channels[itemKey];
  const { channel: key, jsonType, disabled } = PForm.useFieldValue(path);
  return (
    <Select.Item
      {...props}
      justify="between"
      align="center"
      x
      className={CSS.cls(CSS.B("kafka-item"), disabled && CSS.M("off"))}
    >
      <Flex.Box x align="center" gap="small" className={CSS.BE("kafka-item", "label")}>
        <Task.ChannelName
          channel={key}
          picked
          level="p"
          weight={500}
          color={10}
          className={CSS.BE("kafka-item", "name")}
        />
        <Text.Text level="small" color={9}>
          {jsonType}
        </Text.Text>
      </Flex.Box>
      <Task.EnabledCheckbox path={path.disabled} />
    </Select.Item>
  );
};

const listItem = Component.renderProp(ChannelListItem);

const SelectChannel = (p: Pick<PChannel.SelectSingleProps, "value" | "onChange">) => (
  <PChannel.SelectSingle {...p} />
);

const renderSelectChannel = Component.renderProp(SelectChannel);

const JSON_TYPE_ITEMS = (
  <>
    <Select.Item itemKey="number">Number</Select.Item>
    <Select.Item itemKey="string">String</Select.Item>
    <Select.Item itemKey="boolean">Boolean</Select.Item>
  </>
);

const SelectJSONType = (
  p: Omit<Select.SingleSimpleProps<string>, "children" | "resourceName">,
) => (
  <Select.Simple<string> {...p} resourceName="JSON type">
    {JSON_TYPE_ITEMS}
  </Select.Simple>
);

const renderSelectJSONType = Component.renderProp(SelectJSONType);

const ChannelDetails = ({ path }: Task.Views.DetailsProps<WriteChannel>) => {
  const { channel: key, jsonType } = PForm.useFieldValue(path);
  const query = primitive.isNonZero(key) ? { key } : null;
  const { data: dataType } = PChannel.useResultDataType(query);
  const isTimestamp = dataType != null && DataType.TIMESTAMP.equals(dataType);
  // A string, JSON, or UUID channel always goes out as a string, and the time format
  // alone encodes a timestamp channel, so neither has a JSON type.
  const typed = dataType == null || (!isString(dataType) && !isTimestamp);
  return (
    <PForm.Sections className={CSS.B("kafka-details")}>
      <PForm.Section title="Channel">
        <PForm.Field path={path.channel} label="Channel" required>
          {renderSelectChannel}
        </PForm.Field>
        {typed && (
          <PForm.Field path={path.jsonType} label="JSON type" padHelpText={false}>
            {renderSelectJSONType}
          </PForm.Field>
        )}
        {isTimestamp && (
          <Task.Payload.TimeFormatField path={path.timeFormat} label="Time format" />
        )}
      </PForm.Section>
      {jsonType === "string" && typed && (
        <Task.Payload.EnumMappingSection path={path.enumValues} valueFirst />
      )}
    </PForm.Sections>
  );
};

const details = Component.renderProp(ChannelDetails);

const ChannelTitle = ({ path }: Task.Views.DetailsProps<WriteChannel>) => {
  const key = PForm.useFieldValue(path.channel);
  return <Task.ChannelName channel={key} picked level="p" weight={500} color={10} />;
};

const detailsTitle = Component.renderProp(ChannelTitle);

const createChannel = (channels: WriteChannel[], keyToCopy?: string): WriteChannel => {
  const source = channels.find((c) => c.key === keyToCopy);
  return {
    ...(source != null ? { ...source, channel: 0 } : kafka.writeChannelZ.parse({})),
    key: id.create(),
  };
};

/** The channels are chosen in the form, so the device binds none. */
const resolveNothing = () => null;

const Form: FC = () => (
  <Task.Views.ListAndDetails<WriteChannel>
    path={root.config.channels}
    listItem={listItem}
    details={details}
    detailsTitle={detailsTitle}
    createChannel={createChannel}
    resolve={resolveNothing}
  />
);

/** The prefix of the names of the command channels that deploying gives a target. */
const commandPrefix = (
  deviceName: string,
  { topic, recordKey }: Pick<WriteTarget, "topic" | "recordKey">,
): string => {
  const prefix = `${channel.escapeInvalidName(deviceName)}_${channel.escapeInvalidName(topic)}`;
  return recordKey === ""
    ? prefix
    : `${prefix}_${channel.escapeInvalidName(recordKey)}`;
};

/** The key of the command channel of a target's field in the device's write record. */
const writeKey = (
  { topic, recordKey }: Pick<WriteTarget, "topic" | "recordKey">,
  field: Pick<icd.JSONField, "key">,
): string => `${topic}#${recordKey}#${field.key}`;

/** Keys and names the command channels of a target, as deploying does. */
const namingOf = (deviceName: string, target: WriteTarget): ICD.Write.ChannelNaming => {
  const prefix = commandPrefix(deviceName, target);
  return {
    key: (field) => writeKey(target, field),
    name: (field) => ICD.Write.commandName(prefix, field),
  };
};

/**
 * A cleared pointer is omitted from the record instead of placed at "". A string,
 * JSON, or UUID channel goes out as a string, and a timestamp channel by its time
 * format, whatever JSON type the form holds. Disabled rows are left alone, so a
 * deleted channel on one cannot block the deploy. Each channel binding of an enabled
 * target gets a command channel, which the device keeps for the next deploy.
 */
const onConfigure: Task.OnConfigure<WriteSchemas["config"]> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: Device.SCHEMAS,
  });
  const chs = await client.channels.retrieve(
    config.channels
      .filter((c) => !c.disabled)
      .map((c) => c.channel)
      .filter((k) => k !== 0),
  );
  const stringOnly = new Set(chs.filter((c) => isString(c.dataType)).map((c) => c.key));
  const timestamps = new Set(
    chs.filter((c) => DataType.TIMESTAMP.equals(c.dataType)).map((c) => c.key),
  );
  config.channels.forEach((c) => {
    if (stringOnly.has(c.channel)) c.jsonType = "string";
    if (timestamps.has(c.channel)) {
      c.jsonType = "number";
      c.enumValues = [];
    }
  });
  if (config.record.channelPointer === "") delete config.record.channelPointer;
  if (config.record.timestampPointer === "") delete config.record.timestampPointer;
  const before = deep.copy(dev.properties);
  try {
    for (const target of config.targets) {
      if (target.disabled) continue;
      await ICD.Write.configureChannels(
        client,
        dev.properties.write,
        target,
        namingOf(dev.name, target),
      );
    }
  } finally {
    await PlatformDevice.setProperties(client, dev, before, Device.SCHEMAS);
  }
  return [config, dev.rack];
};

const ICDProperties = () => <Device.Select />;

/** Stands in for the record key of a target whose records have no key. */
const NO_KEY = "No key";

const RECORD_KEY_INPUT_PROPS = { placeholder: NO_KEY } as const;

/** A topic in the tree. It only groups the targets that produce to it. */
const TopicItem = ({ name, ...props }: ICD.Write.GroupItemProps) => (
  <Task.Views.TreeGroupItem {...props} label={<TopicLabel name={name} />} />
);

const TOPICS: ICD.Write.Groups<WriteTarget> = {
  of: ({ topic }) => topic,
  Item: TopicItem,
};

/** A target in the tree, named by its record key. */
const TargetItem = (props: Tree.ItemRenderProps<string>) => {
  const path = root.config.targets[props.itemKey];
  const disabled = PForm.useFieldValue(path.disabled, { optional: true });
  const recordKey = PForm.useFieldValue(path.recordKey, { optional: true });
  if (disabled == null || recordKey == null) return null;
  return (
    <Task.Views.TreeGroupItem
      {...props}
      label={<RecordLabel recordKey={recordKey} emptyLabel={NO_KEY} />}
      off={disabled}
    >
      <Task.EnabledCheckbox path={path.disabled} />
    </Task.Views.TreeGroupItem>
  );
};

interface TargetDetailsProps {
  path: deep.Path<WriteTarget>;
  onAddField: () => void;
}

/** Where a target produces, and the format of its message. */
const TargetDetails = ({ path, onAddField }: TargetDetailsProps) => {
  const { set } = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const target = PForm.useFieldValue(path);
  const message = ICD.useMessage(target.icd, target.message.message);
  const handleFormatChange = (icdKey: icd.Key, picked: icd.Message) =>
    set(path, { ...target, icd: icdKey, message: ICD.Write.create(picked) });
  return (
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      <PForm.Sections>
        <PForm.Section title="Produce">
          <PForm.TextField
            path={path.topic}
            label="Topic"
            padHelpText={false}
            inputProps={TOPIC_INPUT_PROPS}
          />
          <PForm.TextField
            path={path.recordKey}
            label="Record key"
            padHelpText={false}
            inputProps={RECORD_KEY_INPUT_PROPS}
          />
        </PForm.Section>
        <PForm.Section
          title="Message"
          actions={
            <ICD.AddField
              onClick={onAddField}
              disabled={message?.payload.format !== "json"}
            />
          }
        >
          <ICD.MessageFormatField
            icdKey={target.icd}
            message={target.message.message}
            onChange={handleFormatChange}
            preview={isPreview}
          />
        </PForm.Section>
        <ICD.MessageStatus
          message={message}
          missingDescription={ICD.Write.MISSING_DESCRIPTION}
        />
      </PForm.Sections>
    </Flex.Box>
  );
};

/** @returns the name of the ICD of a task named name. */
const icdName = (name: string): string =>
  name.trim() === "" ? DEFAULT_WRITE_NAME : name;

const duplicate = (target: WriteTarget): WriteTarget => ({
  ...target,
  key: id.create(),
  message: ICD.Write.unbind(target.message),
});

const messageOf = ({
  key,
  icd: icdKey,
  message,
}: WriteTarget): ICD.Write.TargetMessage => ({
  icd: icdKey,
  message: message.message,
  bindings: root.config.targets[key].message.bindings,
});

interface TargetsProps {
  device: Device.Device;
  selected: string[];
  onSelect: Dispatch<SetStateAction<string[]>>;
  expanded: string[];
  onExpand: Dispatch<SetStateAction<string[]>>;
}

/** The targets of the task as a tree of topics, beside the selected item's details. */
const Targets = ({ device, selected, onSelect, expanded, onExpand }: TargetsProps) => {
  const client = PSynnax.use();
  const ctx = PForm.useContext();
  const handleError = Status.useErrorHandler();
  const formats = ICD.useFormats();
  const targets = PForm.useFieldListUtils(root.config.targets);
  const all = PForm.useFieldValue(root.config.targets);
  const {
    tree,
    handleSelect,
    contextMenu,
    renderItem,
    current,
    currentGroup,
    addField,
    added,
  } = ICD.Write.useTree<WriteTarget>({
    path: root.config.targets,
    messageOf,
    groups: TOPICS,
    TargetItem,
    duplicate,
    selected,
    onSelect,
    expanded,
    onExpand,
  });
  const commandChannelsOf = ICD.Write.useCommandChannels();
  const specs = all
    .filter(({ disabled }) => !disabled)
    .flatMap((t) => commandChannelsOf(t, namingOf(device.name, t)));
  const names = Task.useCommandChannelNames(device.properties.write, specs);

  const add = async () => {
    if (client == null) throw new DisconnectedError();
    const {
      icd: icdKey,
      messages: [message],
    } = await ICD.Write.addMessages({
      client,
      formats,
      targets: targets.value(),
      name: icdName(ctx.get(root.name).value),
      messages: [{ name: "", fields: [] }],
    });
    const target = kafka.writeTargetZ.parse({
      icd: icdKey,
      message: ICD.Write.create(message),
    });
    targets.push([target]);
    onSelect([target.key]);
    onExpand((prev) =>
      unique.unique([...prev, ...ICD.Write.expandedOf([target], TOPICS)]),
    );
  };
  // Adds run one at a time, so a second click cannot make a second ICD.
  const addExclusive = useExclusiveCallback(add);
  const handleAdd = () => handleError(addExclusive, "Failed to add a target");

  const target =
    current == null ? undefined : all.find(({ key }) => key === current.target);
  const path = target == null ? null : root.config.targets[target.key];
  let title: ReactNode = null;
  let details: ReactNode = null;
  let detailsPath: deep.Target<unknown> | null = null;
  if (currentGroup != null) {
    title = <TopicLabel name={currentGroup} />;
    details = (
      <Flex.Box y grow align="center" justify="center">
        <Text.Text status="disabled">Select a record key of the topic</Text.Text>
      </Flex.Box>
    );
  } else if (path != null && target != null && current?.kind === "field") {
    const i = target.message.bindings.findIndex(
      ({ field }) => field === current.field.key,
    );
    detailsPath = i < 0 ? path.message : path.message.bindings[i];
    const naming = namingOf(device.name, target);
    title = (
      <Text.Text level="p" weight={500} overflow="fade">
        {current.field.name}
      </Text.Text>
    );
    details = (
      <ICD.Write.FieldDetails
        key={selected[0]}
        path={path.message.bindings}
        icdKey={target.icd}
        message={target.message.message}
        field={current.field.key}
        channelName={names.get(naming.key(current.field)) ?? naming.name(current.field)}
        skipped={target.disabled}
        autoFocus={current.field.key === added}
      />
    );
  } else if (path != null && target != null) {
    detailsPath = path;
    title = <RecordLabel recordKey={target.recordKey} emptyLabel={NO_KEY} />;
    details = (
      <TargetDetails
        key={target.key}
        path={path}
        onAddField={() => addField(target.key)}
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
          emptyMessage="No targets"
          addLabel="Add target"
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
            Add a target to send records to a topic
          </Text.Text>
        </Flex.Box>
      )}
      {all.map(({ key, icd: icdKey }) => (
        <ICD.Write.Suit
          key={key}
          icdKey={icdKey}
          path={root.config.targets[key].message}
        />
      ))}
    </Task.Views.Panes>
  );
};

const ICDContent = ({ device }: PlatformDevice.TaskFormContentProps<Device.Device>) => {
  const targets = PForm.useFieldValue(root.config.targets);
  const [selected, setSelected] = useState<string[]>([]);
  const [expanded, setExpanded] = useState<string[]>(() =>
    ICD.Write.expandedOf(targets, TOPICS),
  );
  return (
    <ICD.WithFormats icdKeys={targets.map(({ icd }) => icd)}>
      <Targets
        device={device}
        selected={selected}
        onSelect={setSelected}
        expanded={expanded}
        onExpand={setExpanded}
      />
    </ICD.WithFormats>
  );
};

const ICDForm = PlatformDevice.wrapTaskForm({
  use: Device.use,
  useConfigure: Device.useConnectModal,
  useNoneSelected: Device.useNoneSelected,
  Content: ICDContent,
});

export const Write = Task.wrapForm({
  Properties: FLAGS.icds ? ICDProperties : Properties,
  Form: FLAGS.icds ? ICDForm : Form,
  schemas: WRITE_SCHEMAS,
  deployConfigZ: FLAGS.icds ? deployICDWriteConfigZ : deployWriteConfigZ,
  type: WRITE_TYPE,
  onConfigure,
});
