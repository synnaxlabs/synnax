// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/kafka/task/Task.css";

import { channel, kafka } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { deep, id, primitive } from "@synnaxlabs/x";
import { type FC } from "react";

import { useFromConfig } from "@/feature/kafka/device/queries";
import { Select as SelectDevice } from "@/feature/kafka/device/Select";
import { MAKE, SCHEMAS } from "@/feature/kafka/device/types";
import {
  deployReadConfigZ,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadField,
  type ReadRecord,
  type ReadSchemas,
  type StartOffset,
  withDataType,
} from "@/feature/kafka/task/types";
import { CSS } from "@/platform/css";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<ReadSchemas>>();

const SELECT_TRIGGER_PROPS = { className: CSS.BE("kafka-select", "trigger") };

const TEXT_FIELD_CLASS = CSS.BE("kafka-properties", "text");

const SELECT_DIALOG_PROPS = { className: CSS.BE("kafka-select", "dialog") };

const renderStartOffsetSelect = Component.renderProp(
  (p: Omit<Select.SingleSimpleProps<StartOffset>, "children" | "resourceName">) => (
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

const TOPIC_INPUT_PROPS = { placeholder: "sensor-readings" } as const;

const GROUP_INPUT_PROPS = { placeholder: "Derived from the task" } as const;

/** The name configure gives a field's channel when the field carries none. */
const defaultChannelName = (devName: string, recordKey: string, pointer: string) =>
  channel.escapeInvalidName(devName) +
  (recordKey === "" ? "" : `_${channel.escapeInvalidName(recordKey)}`) +
  channel.escapeInvalidName(pointer);

/** The name configure gives a record's index channel when it carries none. */
const defaultIndexName = (devName: string, recordKey: string) =>
  defaultChannelName(devName, recordKey, "_time");

const RecordLabel = ({ path }: Task.Views.GroupProps<ReadRecord>) => {
  const recordKey = PForm.useFieldValue(path.recordKey);
  return (
    <Text.Text level="p" weight={500} color={recordKey === "" ? 9 : 10} overflow="fade">
      {recordKey === "" ? "Any key" : recordKey}
    </Text.Text>
  );
};

type TimingMode = "kafka" | "value";

/** Picks how a record's samples are stamped: by Kafka, or by a value it carries. */
const TimestampFields = ({ path }: { path: deep.Path<ReadRecord> }) => {
  const { set } = PForm.useContext();
  const { index, fields, recordKey } = PForm.useFieldValue(path);
  const indexField = fields.find((f) => f.key === index);
  const dev = useFromConfig();
  const handleChange = (mode: TimingMode) => {
    if (mode === "value" && indexField == null) {
      const field = kafka.timestampReadFieldZ.parse({
        type: "timestamp",
        key: id.create(),
        pointer: "/timestamp",
        timeFormat: "unix_sec",
      });
      set(path.fields, [...fields, field]);
      set(path.index, field.key);
    } else if (mode === "kafka" && indexField != null) {
      set(
        path.fields,
        fields.filter((f) => f.key !== index),
      );
      set(path.index, "");
    }
  };
  return (
    <>
      <Input.Item label="Source" padHelpText={false}>
        <Select.Buttons<TimingMode>
          value={indexField == null ? "kafka" : "value"}
          onChange={handleChange}
        >
          <Select.Item<TimingMode> itemKey="kafka">Kafka timestamp</Select.Item>
          <Select.Item<TimingMode> itemKey="value">Record value</Select.Item>
        </Select.Buttons>
      </Input.Item>
      {indexField != null && (
        <>
          <PForm.TextField
            path={path.fields[indexField.key].pointer}
            label="Pointer"
            padHelpText={false}
            inputProps={TIMESTAMP_POINTER_INPUT_PROPS}
          />
          <Task.Payload.TimeFormatField
            path={
              deep.variant(path.fields[indexField.key], "type", "timestamp").timeFormat
            }
            label="Time format"
          />
          <Task.ChannelNameField
            channel={indexField.channel}
            namePath={path.fields[indexField.key].name}
            defaultName={dev == null ? "" : defaultIndexName(dev.name, recordKey)}
          />
        </>
      )}
    </>
  );
};

const TIMESTAMP_POINTER_INPUT_PROPS = { placeholder: "/timestamp" } as const;

const RecordDetails = ({ path }: Task.Views.GroupProps<ReadRecord>) => (
  <PForm.Sections className={CSS.B("kafka-details")}>
    <PForm.Section title="Record">
      <PForm.TextField
        path={path.recordKey}
        label="Record key"
        padHelpText={false}
        inputProps={RECORD_KEY_INPUT_PROPS}
      />
    </PForm.Section>
    <PForm.Section title="Timestamp">
      <TimestampFields path={path} />
    </PForm.Section>
  </PForm.Sections>
);

const RECORD_KEY_INPUT_PROPS = { placeholder: "Any key" } as const;

const FieldDetails = ({ groupKey, path }: Task.Views.FieldProps<ReadField>) => {
  const field = PForm.useFieldValue(path);
  const { set } = PForm.useContext();
  const recordKey = PForm.useFieldValue(root.config.records[groupKey].recordKey);
  const dev = useFromConfig();
  const { channel: key, pointer, disabled } = field;
  const handleDataTypeChange = (dataType: string) =>
    set(path, withDataType(field, dataType));
  return (
    <PForm.Sections className={CSS.B("kafka-details")}>
      <PForm.Section title="Field">
        <PForm.TextField
          path={path.pointer}
          label="Pointer"
          padHelpText={false}
          inputProps={POINTER_INPUT_PROPS}
        />
        <Task.Payload.ReadDataTypeField
          value={Task.Payload.dataTypeOf(field)}
          onChange={handleDataTypeChange}
          channel={key}
        />
        {field.type === "timestamp" && (
          <Task.Payload.TimeFormatField
            path={deep.variant(path, "type", "timestamp").timeFormat}
            label="Time format"
          />
        )}
        <Task.ChannelNameField
          channel={key}
          namePath={path.name}
          defaultName={
            dev == null ? "" : defaultChannelName(dev.name, recordKey, pointer)
          }
          skipped={disabled}
        />
      </PForm.Section>
      <EnumMapping field={field} path={path} />
    </PForm.Sections>
  );
};

interface EnumMappingProps {
  field: ReadField;
  path: deep.Path<ReadField>;
}

/** Maps labels to channel values for a field whose kind takes them. */
const EnumMapping = ({ field, path }: EnumMappingProps) => {
  switch (field.type) {
    case "float":
      return (
        <Task.Payload.EnumMappingSection
          path={deep.variant(path, "type", "float").enumValues}
        />
      );
    case "integer":
    case "unsigned":
    case "string":
      return (
        <Task.Payload.EnumMappingSection
          path={deep.variant(path, "type", field.type).enumValues}
          valueType="string"
          defaultValue={field.type === "string" ? "" : "0"}
        />
      );
    default:
      return null;
  }
};

const POINTER_INPUT_PROPS = { placeholder: "/temperature" } as const;

const recordLabel = Component.renderProp(RecordLabel);

const recordDetails = Component.renderProp(RecordDetails);

const fieldDetails = Component.renderProp(FieldDetails);

const createRecords = async (): Promise<ReadRecord[]> => [kafka.readRecordZ.parse({})];

const createField = (): Omit<ReadField, "key"> =>
  kafka.floatReadFieldZ.parse({ type: "float" });

const Form: FC = () => (
  <Task.Views.GroupTree<ReadRecord, ReadField>
    path={root.config.records}
    listTitle="Records"
    emptyMessage="No records"
    placeholder="Select a record or field to configure"
    addLabel="Add record"
    createGroups={createRecords}
    createField={createField}
    groupLabel={recordLabel}
    groupDetails={recordDetails}
    fieldDetails={fieldDetails}
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

/**
 * Creates the channels the enabled fields lack. Each record writes against one index:
 * its index field's channel when it has one, else an index stamped by Kafka.
 */
const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (client, config) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: SCHEMAS,
  });
  for (const rec of config.records) {
    const fields = rec.fields.filter((f) => !f.disabled);
    const existing = new Map<string, channel.Channel>();
    for (const field of fields) {
      if (field.channel === 0) continue;
      existing.set(field.key, await client.channels.retrieve(field.channel));
    }
    if (fields.length === 0) continue;
    const stamp = fields.find((f) => f.key === rec.index);
    let index = 0;
    for (const ch of existing.values()) {
      index = ch.isIndex ? ch.key : ch.index;
      if (index !== 0) break;
    }
    if (index === 0) {
      const name = primitive.isNonZero(stamp?.name)
        ? stamp.name
        : defaultIndexName(dev.name, rec.recordKey);
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
        : defaultChannelName(dev.name, rec.recordKey, field.pointer);
      const ch = await client.channels.create({
        name,
        dataType: Task.Payload.dataTypeOf(field),
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
  onConfigure,
});

export const useCreateRead = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const ReadSelectable: Selector.Selectable = {
  type: READ_TYPE,
  title: "Kafka read task",
  label: "Read",
  icon: <Icon.Logo.Kafka />,
  useOnSelect: useCreateRead,
};
