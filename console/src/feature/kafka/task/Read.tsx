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
import { type FC } from "react";

import { Device } from "@/feature/kafka/device";
import { TimeFormatField } from "@/feature/kafka/task/TimeFormatField";
import {
  deployReadConfigZ,
  isString,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadField,
  type ReadRecord,
  type ReadSchemas,
  type StartOffset,
} from "@/feature/kafka/task/types";
import { CSS } from "@/platform/css";
import { Form as PlatformForm } from "@/platform/form";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

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
    <Device.Select />
    <Flex.Box x grow wrap>
      <PForm.TextField
        className={TEXT_FIELD_CLASS}
        path="config.topic"
        label="Topic"
        inputProps={TOPIC_INPUT_PROPS}
        grow
      />
      <PForm.TextField
        className={TEXT_FIELD_CLASS}
        path="config.group"
        label="Consumer group"
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
const defaultChannelName = (devName: string, recordKey: string, pointer: string) =>
  channel.escapeInvalidName(devName) +
  (recordKey === "" ? "" : `_${channel.escapeInvalidName(recordKey)}`) +
  channel.escapeInvalidName(pointer);

/** The name configure gives a record's index channel when it carries none. */
const defaultIndexName = (devName: string, recordKey: string) =>
  defaultChannelName(devName, recordKey, "_time");

const HIDDEN_DATA_TYPES = [DataType.BYTES];

const renderSelectDataType = Component.renderProp((p: Telem.SelectDataTypeProps) => (
  <Telem.SelectDataType {...p} hideDataTypes={HIDDEN_DATA_TYPES} location="bottom" />
));

interface ChannelNameFieldProps {
  path: string;
  defaultName: string;
}

/**
 * Names a field's channel. Until configure creates the channel, the name lives on the
 * field; after, an edit renames the channel itself.
 */
const ChannelNameField = ({ path, defaultName }: ChannelNameFieldProps) => {
  const key = PForm.useFieldValue<channel.Key>(`${path}.channel`);
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
  const handleChange = (next: string) => {
    if (next.length > 0 && next !== name) update({ key, name: next });
  };
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

const RecordLabel = ({ path }: Task.Views.GroupProps) => {
  const recordKey = PForm.useFieldValue<string>(`${path}.recordKey`);
  return (
    <Text.Text
      level="p"
      weight={500}
      color={recordKey === "" ? 9 : 10}
      overflow="ellipsis"
    >
      {recordKey === "" ? "Any key" : recordKey}
    </Text.Text>
  );
};

type TimingMode = "arrival" | "value";

/** Picks how a record's samples are stamped: on arrival, or by a value it carries. */
const TimestampFields = ({ path }: { path: string }) => {
  const { set } = PForm.useContext();
  const { index, fields, recordKey } = PForm.useFieldValue<ReadRecord>(path);
  const indexField = fields.find((f) => f.key === index);
  const dev = Device.useFromConfig();
  const handleChange = (mode: TimingMode) => {
    if (mode === "value" && indexField == null) {
      const field: ReadField = {
        ...kafka.readFieldZ.parse({}),
        key: id.create(),
        pointer: "/timestamp",
        dataType: DataType.TIMESTAMP.toString(),
        timeFormat: "unix_sec",
      };
      set(`${path}.fields`, [...fields, field]);
      set(`${path}.index`, field.key);
    } else if (mode === "arrival" && indexField != null) {
      set(
        `${path}.fields`,
        fields.filter((f) => f.key !== index),
      );
      set(`${path}.index`, "");
    }
  };
  const fieldPath = `${path}.fields.${indexField?.key}`;
  return (
    <>
      <Input.Item label="Source" padHelpText={false}>
        <Select.Buttons<TimingMode>
          value={indexField == null ? "arrival" : "value"}
          onChange={handleChange}
        >
          <Select.Item<TimingMode> itemKey="arrival">Arrival time</Select.Item>
          <Select.Item<TimingMode> itemKey="value">Record value</Select.Item>
        </Select.Buttons>
      </Input.Item>
      {indexField != null && (
        <>
          <PForm.TextField
            path={`${fieldPath}.pointer`}
            label="Pointer"
            padHelpText={false}
            inputProps={TIMESTAMP_POINTER_INPUT_PROPS}
          />
          <TimeFormatField path={`${fieldPath}.timeFormat`} label="Time format" />
          <ChannelNameField
            path={fieldPath}
            defaultName={dev == null ? "" : defaultIndexName(dev.name, recordKey)}
          />
        </>
      )}
    </>
  );
};

const TIMESTAMP_POINTER_INPUT_PROPS = { placeholder: "/timestamp" } as const;

const RecordDetails = ({ path }: Task.Views.GroupProps) => (
  <PForm.Sections className={CSS.B("kafka-details")}>
    <PForm.Section title="Record">
      <PForm.TextField
        path={`${path}.recordKey`}
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

const FieldDetails = ({ groupKey, path }: Task.Views.FieldProps) => {
  const { channel: key, dataType, pointer } = PForm.useFieldValue<ReadField>(path);
  const recordKey = PForm.useFieldValue<string>(`config.records.${groupKey}.recordKey`);
  const dev = Device.useFromConfig();
  const bound = key !== 0;
  const isTimestamp = DataType.TIMESTAMP.equals(dataType);
  // Labels map to numbers, so only a numeric field takes an enum mapping.
  const enumerable = !isTimestamp && !isString(dataType);
  return (
    <PForm.Sections className={CSS.B("kafka-details")}>
      <PForm.Section title="Field">
        <PForm.TextField
          path={`${path}.pointer`}
          label="Pointer"
          padHelpText={false}
          inputProps={POINTER_INPUT_PROPS}
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
        <ChannelNameField
          path={path}
          defaultName={
            dev == null ? "" : defaultChannelName(dev.name, recordKey, pointer)
          }
        />
      </PForm.Section>
      {enumerable && (
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

const recordLabel = Component.renderProp(RecordLabel);

const recordDetails = Component.renderProp(RecordDetails);

const fieldDetails = Component.renderProp(FieldDetails);

const createRecord = (): Omit<ReadRecord, "key"> => kafka.readRecordZ.parse({});

const createField = (): Omit<ReadField, "key"> => kafka.readFieldZ.parse({});

const Form: FC = () => (
  <Task.Views.GroupTree<ReadRecord, ReadField>
    path="config.records"
    listTitle="Records"
    groupName="record"
    placeholder="Select a record or field to configure"
    createGroup={createRecord}
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

/**
 * Creates the channels the enabled fields lack. Each record writes against one index:
 * its index field's channel when it has one, else an index stamped on arrival.
 */
const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (client, config) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: Device.SCHEMAS,
  });
  for (const rec of config.records) {
    const fields = rec.fields.filter((f) => !f.disabled);
    const existing = new Map<string, channel.Channel>();
    for (const field of fields) {
      if (field.channel === 0) continue;
      const ch = await retrieveChannel(client, field.channel);
      if (ch == null) field.channel = 0;
      else existing.set(field.key, ch);
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
});
