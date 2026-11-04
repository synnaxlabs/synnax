// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/kafka/task/Task.css";

import { type channel, kafka } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Channel as PChannel } from "@synnaxlabs/pluto";
import { DataType, deep, id, primitive } from "@synnaxlabs/x";
import { type FC } from "react";

import { Select as SelectDevice } from "@/feature/kafka/device/Select";
import { MAKE, SCHEMAS } from "@/feature/kafka/device/types";
import {
  deployWriteConfigZ,
  isString,
  type RecordKey,
  type TimeFormat,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteChannel,
  type WriteSchemas,
} from "@/feature/kafka/task/types";
import { CSS } from "@/platform/css";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<WriteSchemas>>();

const SELECT_TRIGGER_PROPS = { className: CSS.BE("kafka-select", "trigger") };

const TEXT_FIELD_CLASS = CSS.BE("kafka-properties", "text");

const SELECT_DIALOG_PROPS = { className: CSS.BE("kafka-select", "dialog") };

const renderRecordKeySelect = Component.renderProp(
  (p: Omit<Select.SingleSimpleProps<RecordKey>, "children" | "resourceName">) => (
    <Select.Simple<RecordKey>
      {...p}
      resourceName="record key"
      triggerProps={SELECT_TRIGGER_PROPS}
      dialogProps={SELECT_DIALOG_PROPS}
    >
      <Select.Item itemKey="channel_name">Channel name</Select.Item>
      <Select.Item itemKey="none">None</Select.Item>
    </Select.Simple>
  ),
);

const renderTimeFormatSelect = Component.renderProp(
  (p: Omit<Select.SingleSimpleProps<TimeFormat>, "children" | "resourceName">) => (
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
  ),
);

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

const useChannelName = (key: channel.Key): string | undefined => {
  const query = primitive.isNonZero(key) ? { key } : null;
  return PChannel.useResultName(query).data;
};

const ChannelListItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = root.config.channels[itemKey];
  const { channel: key, jsonType, disabled } = PForm.useFieldValue(path);
  const name = useChannelName(key);
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
          color={name == null ? 8 : 10}
          overflow="fade"
          className={CSS.BE("kafka-item", "name")}
        >
          {name ?? "Select a channel"}
        </Text.Text>
        <Text.Text level="small" color={9}>
          {jsonType}
        </Text.Text>
      </Flex.Box>
      <Task.EnabledCheckbox path={path.disabled} />
    </Select.Item>
  );
};

const listItem = Component.renderProp(ChannelListItem);

const renderSelectChannel = Component.renderProp(
  (p: Pick<PChannel.SelectSingleProps, "value" | "onChange">) => (
    <PChannel.SelectSingle {...p} />
  ),
);

const JSON_TYPE_ITEMS = (
  <>
    <Select.Item itemKey="number">Number</Select.Item>
    <Select.Item itemKey="string">String</Select.Item>
    <Select.Item itemKey="boolean">Boolean</Select.Item>
  </>
);

const renderSelectJSONType = Component.renderProp(
  (p: Omit<Select.SingleSimpleProps<string>, "children" | "resourceName">) => (
    <Select.Simple<string> {...p} resourceName="JSON type">
      {JSON_TYPE_ITEMS}
    </Select.Simple>
  ),
);

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
  const name = useChannelName(key);
  return (
    <Text.Text level="p" weight={500} color={name == null ? 8 : 10}>
      {name ?? "New channel"}
    </Text.Text>
  );
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

const getInitialValues: Task.GetInitialValues<WriteSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = WRITE_SCHEMAS.config.parse(
    config ?? {
      record: { channelPointer: "/channel", timestampPointer: "/timestamp" },
    },
  );
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: "Kafka write task", type: WRITE_TYPE, config: cfg };
};

/**
 * A cleared pointer is omitted from the record instead of placed at "". A string,
 * JSON, or UUID channel goes out as a string, and a timestamp channel by its time
 * format, whatever JSON type the form holds. Disabled rows are left alone, so a
 * deleted channel on one cannot block the deploy.
 */
const onConfigure: Task.OnConfigure<WriteSchemas["config"]> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: SCHEMAS,
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
  return [config, dev.rack];
};

export const Write = Task.wrapForm({
  Properties,
  Form,
  schemas: WRITE_SCHEMAS,
  deployConfigZ: deployWriteConfigZ,
  type: WRITE_TYPE,
  onConfigure,
});

export const useCreateWrite = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const WriteSelectable: Selector.Selectable = {
  type: WRITE_TYPE,
  title: "Kafka write task",
  label: "Write",
  icon: <Icon.Logo.Kafka />,
  useOnSelect: useCreateWrite,
};
