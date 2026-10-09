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
  kafka,
  NotFoundError,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Channel as PChannel, Telem } from "@synnaxlabs/pluto";
import { DataType, errors, id, primitive } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";

import { useFromConfig } from "@/feature/kafka/device/queries";
import { Select as SelectDevice } from "@/feature/kafka/device/Select";
import * as Device from "@/feature/kafka/device/types";
import { TimeFormatField } from "@/feature/kafka/task/TimeFormatField";
import {
  deployReadConfigZ,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadField,
  type ReadSchemas,
  type StartOffset,
} from "@/feature/kafka/task/types";
import { FLAGS } from "@/flags";
import { CSS } from "@/platform/css";
import { Form as PlatformForm } from "@/platform/form";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const renderStartOffsetSelect = Component.renderProp(
  (p: Omit<Select.SingleSimpleProps<StartOffset>, "children" | "resourceName">) => (
    <Select.Simple<StartOffset> {...p} resourceName="start offset">
      <Select.Item itemKey="latest">Latest</Select.Item>
      <Select.Item itemKey="earliest">Earliest</Select.Item>
      <Select.Item itemKey="none">Committed only</Select.Item>
    </Select.Simple>
  ),
);

const Properties = () => (
  <>
    <SelectDevice />
    <Flex.Box x grow>
      <PForm.TextField
        path="config.topic"
        label="Topic"
        inputProps={TOPIC_INPUT_PROPS}
        grow
      />
      <PForm.TextField
        path="config.group"
        label="Consumer group"
        optional
        inputProps={GROUP_INPUT_PROPS}
        grow
      />
      <PForm.Field<StartOffset> path="config.startOffset" label="Start from">
        {renderStartOffsetSelect}
      </PForm.Field>
      <Task.Fields.DataSaving />
      <Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

const TOPIC_INPUT_PROPS = { placeholder: "sensor-readings" } as const;

const GROUP_INPUT_PROPS = { placeholder: "Derived from the task" } as const;

/** The name configure gives a field's channel when the field carries none. */
const defaultChannelName = (devName: string, pointer: string): string =>
  channel.escapeInvalidName(devName) + channel.escapeInvalidName(pointer);

const useDefaultChannelName = (pointer: string): string => {
  const dev = useFromConfig();
  return dev == null ? "" : defaultChannelName(dev.name, pointer);
};

const FieldListItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = `config.fields.${itemKey}`;
  const {
    pointer,
    recordKey,
    channel: key,
    disabled,
  } = PForm.useFieldValue<ReadField>(path);
  const defaultName = useDefaultChannelName(pointer);
  return (
    <Select.Item
      {...props}
      justify="between"
      align="center"
      x
      className={CSS.cls(CSS.B("kafka-item"), disabled && CSS.M("off"))}
    >
      <Flex.Box x align="center" gap="small" className={CSS.BE("kafka-item", "label")}>
        <Text.Text
          level="p"
          weight={500}
          color={pointer === "" ? 8 : 10}
          overflow="ellipsis"
        >
          {pointer === "" ? "New field" : pointer}
        </Text.Text>
        {recordKey !== "" && (
          <Text.Text level="small" color={9}>
            key={recordKey}
          </Text.Text>
        )}
      </Flex.Box>
      <Flex.Box x align="center" gap="small">
        <Task.ChannelName
          channel={key}
          namePath={`${path}.name`}
          defaultName={defaultName}
          id={Task.getChannelNameID(itemKey)}
          level="small"
          weight={450}
          color={9}
          overflow="ellipsis"
        />
        <Task.EnabledCheckbox path={`${path}.disabled`} />
      </Flex.Box>
    </Select.Item>
  );
};

const listItem = Component.renderProp(FieldListItem);

const HIDDEN_DATA_TYPES = [DataType.UUID, DataType.BYTES];

const renderSelectDataType = Component.renderProp((p: Telem.SelectDataTypeProps) => (
  <Telem.SelectDataType
    {...p}
    hideDataTypes={HIDDEN_DATA_TYPES}
    hideVariableDensity
    location="bottom"
  />
));

/**
 * Names a field's channel. Until configure creates the channel, the name lives on the
 * field; after, an edit renames the channel itself.
 */
const ChannelNameField = ({ path }: { path: string }) => {
  const { channel: key, pointer } = PForm.useFieldValue<ReadField>(path);
  const defaultName = useDefaultChannelName(pointer);
  if (key === 0)
    return (
      <PForm.TextField
        path={`${path}.name`}
        label="Channel"
        padHelpText={false}
        inputProps={{ placeholder: defaultName === "" ? "Channel name" : defaultName }}
      />
    );
  return <ExistingChannelNameField channel={key} />;
};

const ExistingChannelNameField = ({ channel: key }: { channel: channel.Key }) => {
  const { data: name = "" } = PChannel.useResultName({ key });
  const { update } = PChannel.useRename();
  const canRename = Access.useUpdateGranted(channel.TYPE_ONTOLOGY_ID);
  const isPreview = Task.useIsPreview();
  const handleChange = useCallback(
    (next: string) => {
      if (next.length > 0 && next !== name) update({ key, name: next });
    },
    [key, name, update],
  );
  return (
    <Input.Item label="Channel" padHelpText={false}>
      <Input.Text
        value={name}
        onChange={handleChange}
        onlyChangeOnBlur
        disabled={isPreview || !canRename}
      />
    </Input.Item>
  );
};

const FieldDetails = ({ path }: Task.Views.DetailsProps) => {
  const { channel: key, dataType } = PForm.useFieldValue<ReadField>(path);
  const bound = key !== 0;
  const isTimestamp = DataType.TIMESTAMP.equals(dataType);
  return (
    <PForm.Sections className={CSS.B("kafka-details")}>
      <PForm.Section title="Field">
        <PForm.TextField
          path={`${path}.pointer`}
          label="Pointer"
          padHelpText={false}
          inputProps={POINTER_INPUT_PROPS}
        />
        <PForm.TextField
          path={`${path}.recordKey`}
          label="Record key"
          optional
          padHelpText={false}
          inputProps={RECORD_KEY_INPUT_PROPS}
        />
        <PForm.Field<string>
          path={`${path}.dataType`}
          label="Data type"
          padHelpText={false}
          helpText={bound ? "Set on the channel" : undefined}
        >
          {(p) => renderSelectDataType({ ...p, disabled: bound })}
        </PForm.Field>
        {isTimestamp && (
          <TimeFormatField path={`${path}.timeFormat`} label="Time format" />
        )}
        <ChannelNameField path={path} />
      </PForm.Section>
      {!isTimestamp && (
        <PForm.Section title="Enum mapping">
          <PlatformForm.KeyValueEditor
            path={`${path}.enumValues`}
            keyField="label"
            keyPlaceholder="String (e.g. ON)"
            valueType="number"
          />
        </PForm.Section>
      )}
    </PForm.Sections>
  );
};

const POINTER_INPUT_PROPS = { placeholder: "/temperature" } as const;

const RECORD_KEY_INPUT_PROPS = { placeholder: "Every record" } as const;

const details = Component.renderProp(FieldDetails);

const FieldTitle = ({ path }: Task.Views.DetailsProps) => {
  const pointer = PForm.useFieldValue<string>(`${path}.pointer`);
  return (
    <Text.Text level="p" weight={500} color={pointer === "" ? 8 : 10}>
      {pointer === "" ? "New field" : pointer}
    </Text.Text>
  );
};

const detailsTitle = Component.renderProp(FieldTitle);

const createField = (fields: ReadField[], keyToCopy?: string): ReadField => {
  const source = fields.find((f) => f.key === keyToCopy) ?? fields[fields.length - 1];
  return {
    ...(source != null
      ? { ...source, ...Task.READ_CHANNEL_OVERRIDE }
      : kafka.readFieldZ.parse({})),
    key: id.create(),
  };
};

/** The device binds no channels: the field keeps the channel configure creates. */
const resolveNothing = () => null;

const Form: FC = () => (
  <Task.Views.ListAndDetails<ReadField>
    path="config.fields"
    listItem={listItem}
    details={details}
    detailsTitle={detailsTitle}
    createChannel={createField}
    contextMenuItems={Task.readChannelContextMenuItem}
    resolve={resolveNothing}
  />
);

const getInitialValues: Task.GetInitialValues<ReadSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = READ_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: "Kafka read task", type: READ_TYPE, config: cfg };
};

const retrieveChannel = async (
  client: Client,
  key: channel.Key,
): Promise<channel.Channel | null> => {
  try {
    return await client.channels.retrieve(key);
  } catch (e) {
    if (NotFoundError.matches(e)) return null;
    throw errors.fromUnknown(e);
  }
};

const isTimestampField = (f: ReadField): boolean =>
  DataType.TIMESTAMP.equals(f.dataType);

/**
 * Creates the channels the enabled fields lack. Fields sharing a record key form one
 * group on one index: the group's first timestamp field is that index, else an index
 * is created for it and stamped with the receive time.
 */
const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (client, config) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: Device.SCHEMAS,
  });
  const safeDevName = channel.escapeInvalidName(dev.name);
  const groups = new Map<string, ReadField[]>();
  for (const field of config.fields) {
    if (field.disabled) continue;
    const group = groups.get(field.recordKey) ?? [];
    group.push(field);
    groups.set(field.recordKey, group);
  }
  for (const [recordKey, fields] of groups) {
    const existing = new Map<string, channel.Channel>();
    for (const field of fields) {
      if (field.channel === 0) continue;
      const ch = await retrieveChannel(client, field.channel);
      if (ch == null) field.channel = 0;
      else existing.set(field.key, ch);
    }
    const stamp = fields.find(isTimestampField);
    let index = 0;
    for (const ch of existing.values()) {
      index = ch.isIndex ? ch.key : ch.index;
      if (index !== 0) break;
    }
    if (index === 0) {
      const safeKey =
        recordKey === "" ? "" : `_${channel.escapeInvalidName(recordKey)}`;
      const name = primitive.isNonZero(stamp?.name)
        ? stamp.name
        : `${safeDevName}${safeKey}_time`;
      index = (
        await client.channels.create({ name, dataType: "timestamp", isIndex: true })
      ).key;
    }
    for (const field of fields) {
      if (existing.has(field.key)) continue;
      if (field === stamp) {
        field.channel = index;
        continue;
      }
      const name = primitive.isNonZero(field.name)
        ? field.name
        : defaultChannelName(dev.name, field.pointer);
      const ch = await client.channels.create({
        name,
        dataType: field.dataType,
        index,
      });
      field.channel = ch.key;
    }
  }
  return [config, dev.rack];
};

export const Read = Task.wrapForm({
  Properties,
  Form,
  schemas: READ_SCHEMAS,
  deployConfigZ: deployReadConfigZ,
  type: READ_TYPE,
  getInitialValues,
  onConfigure,
});

export const useCreateRead = Task.createUseCreate({ getInitialValues });

export const ReadSelectable = Selector.createSelectable({
  type: READ_TYPE,
  title: "Kafka read task",
  icon: <Icon.Logo.Kafka />,
  useOnSelect: useCreateRead,
  useVisible: () => FLAGS.kafka,
});
