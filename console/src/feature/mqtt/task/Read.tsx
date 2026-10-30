// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/mqtt/task/Form.css";

import { channel, mqtt } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { DataType, deep, primitive } from "@synnaxlabs/x";
import { type FC } from "react";

import { use } from "@/feature/mqtt/device/queries";
import { Select as SelectDevice } from "@/feature/mqtt/device/Select";
import { type Device, SCHEMAS } from "@/feature/mqtt/device/types";
import { useConnectModal } from "@/feature/mqtt/device/useConnectModal";
import { createDiscoverModal, type Kind } from "@/feature/mqtt/task/discover/Discover";
import {
  createEntry,
  createSparkplugEntry,
  ENTRIES_PATH,
  namePrefix,
} from "@/feature/mqtt/task/entries";
import { EntryLabel } from "@/feature/mqtt/task/EntryLabel";
import { QoSField } from "@/feature/mqtt/task/QoSField";
import {
  sparkplugChannelName,
  sparkplugPropertiesKey,
} from "@/feature/mqtt/task/sparkplug";
import { SparkplugTagFields } from "@/feature/mqtt/task/SparkplugTagFields";
import {
  deployReadConfigZ,
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

const POINTER_INPUT_PROPS = { placeholder: "/temperature" } as const;

type TimingMode = "arrival" | "payload";

const TimestampFields: FC<{ path: string }> = ({ path }) => {
  const index = PForm.useFieldValue<string>(`${path}.index`);
  const { get, set } = PForm.useContext();
  const isPayloadTiming = index !== "";

  const handleChange = (mode: TimingMode) => {
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
  };

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
          <Task.Payload.TimeFormatField
            path={`${path}.fields.${index}.timeFormat`}
            label="Format"
          />
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

// A Sparkplug B DateTime tag reads into a timestamp channel.
const SPARKPLUG_HIDDEN_DATA_TYPES = [DataType.UUID, DataType.JSON, DataType.BYTES];

interface PaneProps {
  path: string;
  deviceName: string;
}

const SparkplugEntryPane: FC<PaneProps> = ({ path, deviceName }) => {
  const entry = PForm.useFieldValue<SparkplugReadEntry>(path);
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
        <Task.Payload.DataTypeField
          path={`${path}.dataType`}
          channel={entry.channel}
          hideDataTypes={SPARKPLUG_HIDDEN_DATA_TYPES}
        />
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
          <Task.Payload.DataTypeField
            path={`${path}.dataType`}
            channel={fieldChannel}
          />
          <Task.ChannelNameField
            channel={fieldChannel}
            namePath={`${path}.name`}
            defaultName={
              namePrefix(deviceName, topic) + channel.escapeInvalidName(pointer)
            }
          />
        </PForm.Section>
        <Task.Payload.EnumMappingSection path={`${path}.enumValues`} />
      </PForm.Sections>
    </Flex.Box>
  );
};

const createField = (): ReadField => mqtt.readFieldZ.parse({});

const DISCOVER_KIND: Kind<ReadEntry> = {
  title: "Add data",
  hint: "Select the values to record. Each one becomes a channel.",
  noun: "channel",
  // A topic that the task reads comes back as its entry, with the new fields added.
  create: ({ topics, tags }, entries) => [
    ...[...topics].map(([topic, { leaves, pointers }]): ReadEntry => {
      const fields = leaves
        .filter((l) => pointers.has(l.pointer))
        .map((l) => ({ ...l, disabled: false }));
      const existing = entries.find(
        (e): e is PlainReadEntry => e.type === "plain" && e.topic === topic,
      );
      return existing == null
        ? { ...createEntry(), topic, fields }
        : { ...existing, fields: [...existing.fields, ...fields] };
    }),
    ...[...tags.values()].map((t) => createSparkplugEntry(t)),
  ],
  createTopic: (topic) => ({ ...createEntry(), topic }),
  createTag: () => createSparkplugEntry(),
};

const useDiscoverModal = createDiscoverModal(DISCOVER_KIND);

const entryLabel = Component.renderProp(({ path }: Task.Views.GroupProps) => (
  <EntryLabel path={path} />
));

const Content = ({ device }: PlatformDevice.TaskFormContentProps<Device>) => {
  const entries = PForm.useFieldValue<ReadEntry[]>(ENTRIES_PATH);
  const discover = useDiscoverModal();
  const createEntries = async () => (await discover({ device, entries })) ?? [];
  const entryDetails = ({ path }: Task.Views.GroupProps) => (
    <EntryPane path={path} deviceName={device.name} />
  );
  const fieldDetails = (props: Task.Views.FieldProps) => (
    <FieldPane {...props} deviceName={device.name} />
  );

  const resolve = (entry: ReadEntry) => {
    if (entry.type !== "sparkplug") return {};
    const props = device.properties.read[sparkplugPropertiesKey(entry)];
    return { channel: props?.channels[""] ?? 0 };
  };

  return (
    <Flex.Box x grow empty>
      <Task.Views.GroupTree<ReadEntry, ReadField>
        path={ENTRIES_PATH}
        listTitle="Entries"
        emptyMessage="No entries"
        placeholder="Select an entry or field to configure"
        addLabel="Add data"
        createGroups={createEntries}
        createField={createField}
        emptyPointerLabel="Whole payload"
        groupsSwitchable
        groupLabel={entryLabel}
        groupDetails={entryDetails}
        fieldDetails={fieldDetails}
      />
      <Task.BindChannels<ReadEntry> path={ENTRIES_PATH} resolve={resolve} />
      {entries.map(
        ({ key, type }) =>
          type === "plain" && (
            <Task.BindReadChannels
              key={key}
              path={`${ENTRIES_PATH}.${key}`}
              propertiesKeyField="topic"
              read={device.properties.read}
            />
          ),
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

const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (client, config) => {
  const dev = await client.devices.retrieve({ key: config.device, schemas: SCHEMAS });
  const saved = deep.copy(dev.properties);
  try {
    for (const entry of config.entries) {
      if (entry.disabled) continue;
      if (entry.type === "sparkplug") {
        const name = primitive.isNonZero(entry.name)
          ? entry.name
          : sparkplugChannelName(dev.name, entry);
        // A tag reads whole into one channel, as the field at the empty pointer.
        const field = {
          key: entry.key,
          pointer: "",
          channel: entry.channel,
          name,
          dataType: entry.dataType,
        };
        await Task.configureReadChannels({
          client,
          read: dev.properties.read,
          propertiesKey: sparkplugPropertiesKey(entry),
          fields: [field],
          namePrefix: name,
          indexKey: "",
        });
        entry.channel = field.channel;
        continue;
      }
      await Task.configureReadChannels({
        client,
        read: dev.properties.read,
        propertiesKey: entry.topic,
        fields: entry.fields,
        namePrefix: namePrefix(dev.name, entry.topic),
        indexKey: entry.index,
      });
    }
  } finally {
    if (!deep.equal(saved, dev.properties)) await client.devices.create(dev, SCHEMAS);
  }
  return [config, dev.rack];
};

export const Read = Task.wrapForm({
  Properties,
  Form,
  schemas: READ_SCHEMAS,
  deployConfigZ: deployReadConfigZ,
  type: "mqtt_read",
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
