// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/mqtt/task/Form.css";

import { channel, mqtt, type Synnax as Client } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Telem } from "@synnaxlabs/pluto";
import { DataType, primitive } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";

import { Browser } from "@/feature/mqtt/device/Browser";
import { use } from "@/feature/mqtt/device/queries";
import { Select as SelectDevice } from "@/feature/mqtt/device/Select";
import { type SparkplugHaulTag } from "@/feature/mqtt/device/SparkplugBrowser";
import { type Device, SCHEMAS } from "@/feature/mqtt/device/types";
import { useConnectModal } from "@/feature/mqtt/device/useConnectModal";
import { createReadFields } from "@/feature/mqtt/task/createReadFields";
import { EntryLabel } from "@/feature/mqtt/task/EntryLabel";
import { useEntryDrop } from "@/feature/mqtt/task/EntryList";
import { QoSField } from "@/feature/mqtt/task/QoSField";
import {
  fromSparkplugDataType,
  sparkplugChannelName,
  sparkplugPropertiesKey,
  toSparkplugDataType,
} from "@/feature/mqtt/task/sparkplug";
import { SparkplugTagFields } from "@/feature/mqtt/task/SparkplugTagFields";
import { TimeFormatField } from "@/feature/mqtt/task/TimeFormatField";
import {
  type BrowsedTopic,
  deployReadConfigZ,
  HIDDEN_DATA_TYPES,
  type PlainReadEntry,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadEntry,
  type ReadField,
  type ReadSchemas,
  type SparkplugReadEntry,
} from "@/feature/mqtt/task/types";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { Form as PlatformForm } from "@/platform/form";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const Properties = () => (
  <>
    <SelectDevice />
    <Flex.Box x grow>
      <Task.Fields.DataSaving />
      <Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

const ENTRIES_PATH = "config.entries";

/** The prefix configure gives the channels of a topic's fields. */
const namePrefix = (devName: string, topic: string): string =>
  `${channel.escapeInvalidName(devName)}_${channel.escapeInvalidName(topic)}`;

const POINTER_INPUT_PROPS = { placeholder: "/temperature" } as const;

const renderTelemSelectDataType = Component.renderProp(
  (p: Telem.SelectDataTypeProps) => (
    <Telem.SelectDataType {...p} hideDataTypes={HIDDEN_DATA_TYPES} location="bottom" />
  ),
);

interface BinderProps {
  device: Device;
  entryKey: string;
}

/**
 * Binds a plain entry's fields to the channels the device stores for its topic.
 * Mounted for every entry, as the details pane renders only the selected one.
 */
const FieldBinder = ({ device, entryKey }: BinderProps) => {
  const path = `${ENTRIES_PATH}.${entryKey}`;
  const topic = PForm.useFieldValue<string>(`${path}.topic`);
  const indexKey = PForm.useFieldValue<string>(`${path}.index`);
  const resolve = useCallback(
    (field: ReadField) => {
      const props = device.properties.read[topic];
      if (props == null) return { channel: 0 };
      return {
        channel:
          field.key === indexKey ? props.index : (props.channels[field.pointer] ?? 0),
      };
    },
    [device, topic, indexKey],
  );
  return <Task.BindChannels<ReadField> path={`${path}.fields`} resolve={resolve} />;
};

type TimingMode = "arrival" | "payload";

const TimestampFields: FC<{ path: string }> = ({ path }) => {
  const index = PForm.useFieldValue<string>(`${path}.index`);
  const { get, set } = PForm.useContext();
  const isPayloadTiming = index !== "";

  const handleChange = useCallback(
    (mode: TimingMode) => {
      const { fields, index } = get<PlainReadEntry>(path).value;
      if (mode === "payload" && index === "") {
        const indexField: ReadField = {
          ...mqtt.readFieldZ.parse({}),
          dataType: DataType.TIMESTAMP.toString(),
          timeFormat: "unix_sec",
        };
        set(`${path}.fields`, [...fields, indexField]);
        set(`${path}.index`, indexField.key);
      } else if (mode === "arrival" && index !== "") {
        set(
          `${path}.fields`,
          fields.filter((f) => f.key !== index),
        );
        set(`${path}.index`, "");
      }
    },
    [path, get, set],
  );

  return (
    <>
      <Input.Item label="Source" padHelpText={false}>
        <Select.Buttons<TimingMode>
          value={isPayloadTiming ? "payload" : "arrival"}
          onChange={handleChange}
        >
          <Select.Item<TimingMode> itemKey="arrival">Arrival time</Select.Item>
          <Select.Item<TimingMode> itemKey="payload">Payload value</Select.Item>
        </Select.Buttons>
      </Input.Item>
      {isPayloadTiming && (
        <>
          <PForm.TextField
            path={`${path}.fields.${index}.pointer`}
            label="Pointer"
            padHelpText={false}
            inputProps={TIMESTAMP_POINTER_INPUT_PROPS}
          />
          <TimeFormatField path={`${path}.fields.${index}.timeFormat`} label="Format" />
        </>
      )}
    </>
  );
};

const TIMESTAMP_POINTER_INPUT_PROPS = { placeholder: "/timestamp" } as const;

const TOPIC_INPUT_PROPS = { placeholder: "plant/line1/temperature" } as const;

const PlainEntryPane: FC<{ path: string }> = ({ path }) => (
  <PForm.Sections>
    <PForm.Section title="Subscription">
      <PForm.TextField
        path={`${path}.topic`}
        label="Topic"
        padHelpText={false}
        inputProps={TOPIC_INPUT_PROPS}
      />
      <QoSField path={`${path}.qos`} />
      <PForm.SwitchField
        path={`${path}.retainedIgnored`}
        label="Ignore retained"
        padHelpText={false}
      />
    </PForm.Section>
    <PForm.Section title="Timestamp">
      <TimestampFields path={path} />
    </PForm.Section>
  </PForm.Sections>
);

const SPARKPLUG_HIDDEN_DATA_TYPES = [DataType.UUID, DataType.JSON, DataType.BYTES];

const renderSparkplugSelectDataType = Component.renderProp(
  (p: Telem.SelectDataTypeProps) => (
    <Telem.SelectDataType
      {...p}
      hideDataTypes={SPARKPLUG_HIDDEN_DATA_TYPES}
      location="bottom"
    />
  ),
);

interface PaneProps {
  path: string;
  deviceName: string;
}

const SparkplugEntryPane: FC<PaneProps> = ({ path, deviceName }) => {
  const entry = PForm.useFieldValue<SparkplugReadEntry>(path);
  const bound = entry.channel !== 0;
  return (
    <PForm.Sections>
      <PForm.Section title="Tag">
        <SparkplugTagFields path={path} />
      </PForm.Section>
      <PForm.Section title="Channel">
        <Task.ChannelNameField
          channel={entry.channel}
          namePath={`${path}.name`}
          defaultName={sparkplugChannelName(deviceName, entry)}
        />
        <PForm.Field<string>
          path={`${path}.dataType`}
          label="Data type"
          padHelpText={false}
          helpText={bound ? "Set on the channel" : undefined}
        >
          {(p) => renderSparkplugSelectDataType({ ...p, disabled: bound })}
        </PForm.Field>
      </PForm.Section>
    </PForm.Sections>
  );
};

const EntryPane: FC<PaneProps> = ({ path, deviceName }) => {
  const type = PForm.useFieldValue<ReadEntry["type"]>(`${path}.type`);
  return (
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      {type === "plain" ? (
        <PlainEntryPane path={path} />
      ) : (
        <SparkplugEntryPane path={path} deviceName={deviceName} />
      )}
    </Flex.Box>
  );
};

interface FieldPaneProps extends Task.Views.FieldProps, Pick<PaneProps, "deviceName"> {}

const FieldPane: FC<FieldPaneProps> = ({ groupKey, path, deviceName }) => {
  const { channel: fieldChannel, pointer } = PForm.useFieldValue<ReadField>(path);
  const topic = PForm.useFieldValue<string>(`${ENTRIES_PATH}.${groupKey}.topic`);
  const bound = fieldChannel !== 0;
  return (
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      <PForm.Sections>
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
            {(p) => renderTelemSelectDataType({ ...p, disabled: bound })}
          </PForm.Field>
          <Task.ChannelNameField
            channel={fieldChannel}
            namePath={`${path}.name`}
            defaultName={
              namePrefix(deviceName, topic) + channel.escapeInvalidName(pointer)
            }
          />
        </PForm.Section>
        <PForm.Section title="Enum mapping">
          <PlatformForm.KeyValueEditor
            path={`${path}.enumValues`}
            keyField="label"
            keyPlaceholder="String (e.g. ON)"
            valueType="number"
          />
        </PForm.Section>
      </PForm.Sections>
    </Flex.Box>
  );
};

const createEntry = (topic?: BrowsedTopic): PlainReadEntry => ({
  ...mqtt.plainReadEntryZ.parse({ type: "plain" }),
  ...(topic != null && {
    topic: topic.topic,
    fields: createReadFields(topic.payload),
  }),
});

const createSparkplugEntry = (tag?: SparkplugHaulTag): SparkplugReadEntry => {
  const entry = mqtt.sparkplugReadEntryZ.parse({ type: "sparkplug" });
  if (tag == null) return entry;
  const { dataType, ...tagID } = tag;
  return {
    ...entry,
    ...tagID,
    dataType: fromSparkplugDataType(toSparkplugDataType(dataType)),
  };
};

const ENTRY_KINDS: Task.Views.GroupKind<ReadEntry>[] = [
  { name: "topic", create: createEntry },
  { name: "Sparkplug B tag", create: createSparkplugEntry },
];

const createField = (): ReadField => mqtt.readFieldZ.parse({});

const entryLabel = Component.renderProp(({ path }: Task.Views.GroupProps) => (
  <EntryLabel path={path} />
));

const Content = ({ device }: PlatformDevice.TaskFormContentProps<Device>) => {
  const entries = PForm.useFieldValue<ReadEntry[]>(ENTRIES_PATH);
  const isPreview = Task.useIsPreview();
  const drop = useEntryDrop<ReadEntry>({
    path: ENTRIES_PATH,
    create: createEntry,
    createSparkplug: createSparkplugEntry,
  });
  const entryDetails = useCallback(
    ({ path }: Task.Views.GroupProps) => (
      <EntryPane path={path} deviceName={device.name} />
    ),
    [device.name],
  );
  const fieldDetails = useCallback(
    ({ groupKey, fieldKey, path }: Task.Views.FieldProps) => (
      <FieldPane
        groupKey={groupKey}
        fieldKey={fieldKey}
        path={path}
        deviceName={device.name}
      />
    ),
    [device.name],
  );

  const resolve = useCallback(
    (entry: ReadEntry) => {
      if (entry.type !== "sparkplug") return {};
      const props = device.properties.read[sparkplugPropertiesKey(entry)];
      return { channel: props?.channels[""] ?? 0, index: props?.index ?? 0 };
    },
    [device],
  );

  return (
    <Flex.Box x grow empty>
      {!isPreview && <Browser device={device} />}
      <Task.Views.GroupTree<ReadEntry, ReadField>
        path={ENTRIES_PATH}
        listTitle="Entries"
        emptyMessage="No entries"
        placeholder="Select an entry or field to configure"
        groupKinds={ENTRY_KINDS}
        createField={createField}
        emptyPointerLabel="Whole payload"
        groupLabel={entryLabel}
        groupDetails={entryDetails}
        fieldDetails={fieldDetails}
        drop={drop}
      />
      <Task.BindChannels<ReadEntry> path={ENTRIES_PATH} resolve={resolve} />
      {entries.map(
        ({ key, type }) =>
          type === "plain" && <FieldBinder key={key} device={device} entryKey={key} />,
      )}
    </Flex.Box>
  );
};

const Form = PlatformDevice.wrapTaskForm({
  use,
  useConfigure: useConnectModal,
  Content,
});

const getInitialValues: Task.GetInitialValues<ReadSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = READ_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: "MQTT read task", type: READ_TYPE, config: cfg };
};

/** @returns true when it stored new channels in the properties of the device. */
const configureSparkplugEntry = async (
  client: Client,
  dev: Device,
  entry: SparkplugReadEntry,
): Promise<boolean> => {
  const propertiesKey = sparkplugPropertiesKey(entry);
  const storedKey = dev.properties.read[propertiesKey]?.channels[""];
  for (const key of [entry.channel, storedKey]) {
    const ch = await Task.retrieveChannel(client, key);
    if (ch == null) continue;
    entry.channel = ch.key;
    entry.index = ch.index;
    return false;
  }
  const name = primitive.isNonZero(entry.name)
    ? entry.name
    : sparkplugChannelName(dev.name, entry);
  const ch = await Task.createChannel(client, name, entry.dataType);
  entry.channel = ch.key;
  entry.index = ch.index;
  dev.properties.read[propertiesKey] = { index: ch.index, channels: { "": ch.key } };
  return true;
};

const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (client, config) => {
  const dev = await client.devices.retrieve({ key: config.device, schemas: SCHEMAS });
  let modified = false;
  try {
    for (const entry of config.entries) {
      if (entry.disabled) continue;
      if (entry.type === "sparkplug") {
        if (await configureSparkplugEntry(client, dev, entry)) modified = true;
        continue;
      }
      dev.properties.read[entry.topic] ??= { index: 0, channels: {} };
      const changed = await Task.configureReadChannels({
        client,
        props: dev.properties.read[entry.topic],
        fields: entry.fields,
        namePrefix: namePrefix(dev.name, entry.topic),
        indexKey: entry.index,
      });
      modified ||= changed;
    }
  } finally {
    if (modified) await client.devices.create(dev, SCHEMAS);
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

export const useCreateRead = Task.createUseCreate({
  getInitialValues,
});

export const ReadSelectable = Selector.createSelectable({
  type: READ_TYPE,
  title: "MQTT read task",
  icon: <Icon.Logo.MQTT />,
  useOnSelect: useCreateRead,
});
