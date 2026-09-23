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
import {
  Button,
  Component,
  Divider,
  Flex,
  Form as PForm,
  Header,
  Icon,
  Input,
  type List,
  Select,
  Telem,
  Text,
} from "@synnaxlabs/pluto";
import { DataType, id, primitive } from "@synnaxlabs/x";
import { type FC, useCallback, useState } from "react";

import { Browser } from "@/feature/mqtt/device/Browser";
import { use } from "@/feature/mqtt/device/queries";
import { Select as SelectDevice } from "@/feature/mqtt/device/Select";
import { type SparkplugHaulTag } from "@/feature/mqtt/device/SparkplugBrowser";
import { type Device, SCHEMAS } from "@/feature/mqtt/device/types";
import { useConnectModal } from "@/feature/mqtt/device/useConnectModal";
import { createReadFields } from "@/feature/mqtt/task/createReadFields";
import { QoSField } from "@/feature/mqtt/task/QoSField";
import {
  fromSparkplugDataType,
  sparkplugChannelName,
  sparkplugPropertiesKey,
  toSparkplugDataType,
} from "@/feature/mqtt/task/sparkplug";
import { SparkplugTagFields } from "@/feature/mqtt/task/SparkplugTagFields";
import { TimeFormatField } from "@/feature/mqtt/task/TimeFormatField";
import { TopicList } from "@/feature/mqtt/task/TopicList";
import { SparkplugListItem, TopicListItem } from "@/feature/mqtt/task/TopicListItem";
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
import { Empty } from "@/platform/empty";
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

const PlainEntryListItem = (props: List.ItemProps<string>) => {
  const fields = PForm.useFieldValue<ReadField[]>(
    `${ENTRIES_PATH}.${props.itemKey}.fields`,
  );
  return (
    <TopicListItem
      {...props}
      path={ENTRIES_PATH}
      extra={
        <Text.Text level="small" color={9}>
          {fields.length} field{fields.length === 1 ? "" : "s"}
        </Text.Text>
      }
    />
  );
};

const EntryListItem = (props: List.ItemProps<string>) => {
  const type = PForm.useFieldValue<ReadEntry["type"]>(
    `${ENTRIES_PATH}.${props.itemKey}.type`,
  );
  if (type === "plain") return <PlainEntryListItem {...props} />;
  return <SparkplugListItem {...props} path={ENTRIES_PATH} />;
};

const entryListItem = Component.renderProp(EntryListItem);

interface FieldListItemProps extends Task.ChannelListItemProps {
  entryKey: string;
}

const FieldListItem = ({ entryKey, ...props }: FieldListItemProps) => {
  const { itemKey } = props;
  const path = `${ENTRIES_PATH}.${entryKey}.fields.${itemKey}`;
  const fieldChannel = PForm.useFieldValue<number>(`${path}.channel`);
  const enumValues = PForm.useFieldValue<Record<string, number>[]>(
    `${path}.enumValues`,
    { defaultValue: [] },
  );
  const enumCount = enumValues.length;
  const enumCountText =
    enumCount === 0 ? "" : `${enumCount} enum${enumCount === 1 ? "" : "s"}`;
  return (
    <Select.ListItem {...props} justify="between" align="center" x>
      <PForm.TextField
        path={`${path}.pointer`}
        showLabel={false}
        showHelpText={false}
        inputProps={POINTER_INPUT_PROPS}
        grow
      />
      {fieldChannel === 0 && (
        <PForm.Field<string>
          path={`${path}.dataType`}
          showLabel={false}
          showHelpText={false}
          hideIfNull
        >
          {renderTelemSelectDataType}
        </PForm.Field>
      )}
      {enumCountText !== "" && (
        <Text.Text level="small" color={9}>
          {enumCountText}
        </Text.Text>
      )}
      <Flex.Box x align="center" grow justify="end">
        <Task.ChannelName
          channel={fieldChannel}
          namePath={`${path}.name`}
          id={Task.getChannelNameID(itemKey)}
        />
        <Task.EnableDisableButton path={`${path}.disabled`} />
      </Flex.Box>
    </Select.ListItem>
  );
};

const POINTER_INPUT_PROPS = { placeholder: "/temperature" } as const;

const renderTelemSelectDataType = Component.renderProp(
  (p: Telem.SelectDataTypeProps) => (
    <Telem.SelectDataType
      {...p}
      className={CSS.B("field-data-type")}
      hideDataTypes={HIDDEN_DATA_TYPES}
      location="bottom"
    />
  ),
);

interface FieldListProps {
  entryKey: string;
}

const FieldList = ({ entryKey }: FieldListProps) => {
  const entryPath = `${ENTRIES_PATH}.${entryKey}`;
  const path = `${entryPath}.fields`;
  const { data: allData, push, remove } = PForm.useFieldList<string, ReadField>(path);
  const [selected, setSelected] = useState<string[]>([]);
  const ctx = PForm.useContext();
  const isPreview = Task.useIsPreview();

  const index = PForm.useFieldValue<string>(`${entryPath}.index`);
  const data = allData.filter((key) => key !== index);

  const handleAdd = useCallback(() => {
    const { fields, index } = ctx.get<PlainReadEntry>(entryPath).value;
    const last = fields.findLast((f) => f.key !== index);
    const field: ReadField = {
      ...(last != null
        ? { ...last, ...Task.READ_CHANNEL_OVERRIDE }
        : mqtt.readFieldZ.parse({})),
      key: id.create(),
    };
    push(field);
    setSelected([field.key]);
  }, [push, ctx, entryPath]);

  const handleDuplicate = useCallback(
    (channels: ReadField[], keys: string[]) => {
      const duplicated = channels
        .filter(({ key }) => keys.includes(key))
        .map((ch) => ({
          ...ch,
          ...Task.READ_CHANNEL_OVERRIDE,
          key: id.create(),
        }));
      push(duplicated);
    },
    [push],
  );

  const listItem = useCallback(
    ({ key, ...p }: Task.ChannelListItemProps) => (
      <FieldListItem {...p} key={key} entryKey={entryKey} />
    ),
    [entryKey],
  );

  const selectedFieldKey = selected.length === 1 ? selected[0] : null;
  const selectedFieldPath =
    selectedFieldKey != null ? `${path}.${selectedFieldKey}.enumValues` : null;

  return (
    <>
      <Task.ChannelList<ReadField>
        data={data}
        remove={remove}
        onDuplicate={handleDuplicate}
        onSelect={setSelected}
        selected={selected}
        path={path}
        style={FIELD_LIST_STYLE}
        header={
          <Header.Header>
            <Header.Title weight={500} color={9}>
              Fields
            </Header.Title>
            {!isPreview && (
              <Header.Actions empty align="end">
                <Button.Button
                  onClick={handleAdd}
                  variant="filled"
                  tooltip="Add field"
                  size="small"
                >
                  <Icon.Add />
                </Button.Button>
              </Header.Actions>
            )}
          </Header.Header>
        }
        emptyContent={
          <Empty.Action
            message="No fields"
            action={isPreview ? undefined : "Add field"}
            onClick={handleAdd}
          />
        }
        listItem={listItem}
        contextMenuItems={Task.readChannelContextMenuItem}
      />
      {selectedFieldPath != null && (
        <Flex.Box y empty className={CSS.B("enum-mapping")}>
          <Divider.Divider x padded />
          <PlatformForm.KeyValueEditor
            path={selectedFieldPath}
            label="Enum mapping"
            keyField="label"
            keyPlaceholder="String (e.g. ON)"
            valueType="number"
          />
        </Flex.Box>
      )}
    </>
  );
};

const FIELD_LIST_STYLE = {
  paddingBottom: "1rem",
  maxWidth: "100%",
  overflow: "visible",
} as const;

type TimingMode = "arrival" | "payload";
const TIMING_MODE_KEYS: TimingMode[] = ["arrival", "payload"];

const TimingToggle: FC<{ path: string }> = ({ path }) => {
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
    <Flex.Box x align="end" wrap>
      <Input.Item label="Timestamp source" padHelpText>
        <Select.Buttons<TimingMode>
          value={isPayloadTiming ? "payload" : "arrival"}
          onChange={handleChange}
          keys={TIMING_MODE_KEYS}
        >
          <Select.Button<TimingMode> itemKey="arrival">Arrival time</Select.Button>
          <Select.Button<TimingMode> itemKey="payload">Payload</Select.Button>
        </Select.Buttons>
      </Input.Item>
      {isPayloadTiming && (
        <>
          <PForm.TextField
            path={`${path}.fields.${index}.pointer`}
            label="Timestamp pointer"
            inputProps={TIMESTAMP_POINTER_INPUT_PROPS}
            grow
          />
          <TimeFormatField path={`${path}.fields.${index}.timeFormat`} label="Format" />
        </>
      )}
    </Flex.Box>
  );
};

const TIMESTAMP_POINTER_INPUT_PROPS = { placeholder: "/timestamp" } as const;

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

const SparkplugEntryDetails: FC<{ path: string }> = ({ path }) => {
  const entryChannel = PForm.useFieldValue<number>(`${path}.channel`);
  return (
    <Flex.Box y grow empty className={CSS.B("topic-details")}>
      <Flex.Box gap="small" empty className={CSS.B("topic-details-form")}>
        <SparkplugTagFields path={path} />
        <Flex.Box x align="center" justify="between">
          <Task.ChannelName channel={entryChannel} namePath={`${path}.name`} />
          {entryChannel === 0 && (
            <PForm.Field<string>
              path={`${path}.dataType`}
              label="Data type"
              showHelpText={false}
              className={CSS.B("data-type-select")}
            >
              {renderSparkplugSelectDataType}
            </PForm.Field>
          )}
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
  );
};

const EntryDetails: FC<{ entryKey: string }> = ({ entryKey }) => {
  const path = `${ENTRIES_PATH}.${entryKey}`;
  const type = PForm.useFieldValue<ReadEntry["type"]>(`${path}.type`);
  if (type === "sparkplug") return <SparkplugEntryDetails path={path} />;
  return (
    <Flex.Box y grow empty className={CSS.B("topic-details")}>
      <Flex.Box gap="small" empty className={CSS.B("topic-details-form")}>
        <PForm.TextField
          path={`${path}.topic`}
          label="Topic"
          inputProps={TOPIC_INPUT_PROPS}
        />
        <Flex.Box x align="end" gap="large">
          <QoSField path={`${path}.qos`} />
          <PForm.SwitchField
            path={`${path}.retainedIgnored`}
            label="Ignore retained messages"
          />
        </Flex.Box>
        <TimingToggle path={path} />
      </Flex.Box>
      <Divider.Divider x />
      <FieldList key={entryKey} entryKey={entryKey} />
    </Flex.Box>
  );
};

const TOPIC_INPUT_PROPS = { placeholder: "plant/line1/temperature" } as const;

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

const duplicateEntry = (entry: ReadEntry): ReadEntry => {
  if (entry.type === "sparkplug")
    return { ...entry, ...Task.READ_CHANNEL_OVERRIDE, key: id.create(), index: 0 };
  const fields = entry.fields.map((f) => ({
    ...f,
    ...Task.READ_CHANNEL_OVERRIDE,
    key: id.create(),
  }));
  const indexPosition = entry.fields.findIndex((f) => f.key === entry.index);
  return {
    ...entry,
    key: id.create(),
    fields,
    index: fields[indexPosition]?.key ?? "",
  };
};

const Content = ({ device }: PlatformDevice.TaskFormContentProps<Device>) => {
  const [selected, setSelected] = useState<string[]>([]);
  const isPreview = Task.useIsPreview();
  return (
    <Flex.Box x grow empty>
      {!isPreview && <Browser device={device} />}
      <TopicList<ReadEntry>
        path={ENTRIES_PATH}
        title="Entries"
        noun="entry"
        selected={selected}
        onSelect={setSelected}
        create={createEntry}
        createSparkplug={createSparkplugEntry}
        duplicate={duplicateEntry}
      >
        {entryListItem}
      </TopicList>
      <Divider.Divider y />
      <Flex.Box y grow empty className={CSS.B("topic-details-pane")}>
        <Task.Views.DetailsHeader
          path={selected.length > 0 ? `${ENTRIES_PATH}.${selected[0]}` : ""}
          disabled={selected.length === 0}
        />
        {selected.length > 0 ? (
          <EntryDetails entryKey={selected[0]} />
        ) : (
          <Flex.Box y grow align="center" justify="center">
            <Text.Text status="disabled">Select an entry to configure</Text.Text>
          </Flex.Box>
        )}
      </Flex.Box>
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
  const safeDevName = channel.escapeInvalidName(dev.name);
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
        namePrefix: `${safeDevName}_${channel.escapeInvalidName(entry.topic)}`,
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
