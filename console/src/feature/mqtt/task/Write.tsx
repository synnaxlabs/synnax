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
import {
  Button,
  Channel as PChannel,
  Component,
  Divider,
  Flex,
  Form as PForm,
  Header,
  Icon,
  List,
  Menu,
  Select,
  Telem,
  Text,
} from "@synnaxlabs/pluto";
import { DataType, id, json, primitive } from "@synnaxlabs/x";
import { type FC, useCallback, useMemo, useState } from "react";

import { Browser } from "@/feature/mqtt/device/Browser";
import { use } from "@/feature/mqtt/device/queries";
import { Select as SelectDevice } from "@/feature/mqtt/device/Select";
import { type SparkplugHaulTag } from "@/feature/mqtt/device/SparkplugBrowser";
import { type Device, SCHEMAS } from "@/feature/mqtt/device/types";
import { useConnectModal } from "@/feature/mqtt/device/useConnectModal";
import { ContextMenu } from "@/feature/mqtt/task/ContextMenu";
import { QoSField } from "@/feature/mqtt/task/QoSField";
import {
  fromSparkplugDataType,
  sparkplugChannelName,
  sparkplugPropertiesKey,
  toSparkplugDataType,
} from "@/feature/mqtt/task/sparkplug";
import { SparkplugTagFields } from "@/feature/mqtt/task/SparkplugTagFields";
import { SparkplugTypeField } from "@/feature/mqtt/task/SparkplugTypeField";
import { TimeFormatField } from "@/feature/mqtt/task/TimeFormatField";
import { TopicList } from "@/feature/mqtt/task/TopicList";
import { SparkplugListItem, TopicListItem } from "@/feature/mqtt/task/TopicListItem";
import {
  type BrowsedTopic,
  deployWriteConfigZ,
  type GeneratorType,
  HIDDEN_DATA_TYPES,
  type PlainWriteTarget,
  type SparkplugWriteTarget,
  type TimeFormat,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteField,
  type WriteSchemas,
  type WriteTarget,
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
      <Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

const TARGETS_PATH = "config.targets";

const JSON_TYPE_DATA: Select.StaticEntry<mqtt.JSONType>[] = [
  { key: "number", name: "Number" },
  { key: "string", name: "String" },
  { key: "boolean", name: "Boolean" },
];

const GENERATOR_DATA: Select.StaticEntry<GeneratorType | TimeFormat>[] = [
  { key: "uuid", name: "UUID" },
  { key: "iso8601", name: "Timestamp (ISO 8601)" },
  { key: "unix_sec", name: "Timestamp (s)" },
  { key: "unix_ms", name: "Timestamp (ms)" },
  { key: "unix_us", name: "Timestamp (µs)" },
  { key: "unix_ns", name: "Timestamp (ns)" },
];

const getTargetChannelNameID = (targetKey: string) => `write-target-ch-${targetKey}`;

const TargetListItem = (props: List.ItemProps<string>) => {
  const { itemKey } = props;
  const path = `${TARGETS_PATH}.${itemKey}`;
  const isPlain = PForm.useFieldValue<WriteTarget["type"]>(`${path}.type`) === "plain";
  // A plain target holds its channel in a field; a Sparkplug target is the field.
  const channelKeyPath = isPlain ? `${path}.channel.channel` : `${path}.channel`;
  const namePath = isPlain ? `${path}.channel.name` : `${path}.name`;
  const channel = PForm.useFieldValue<number>(channelKeyPath);
  const extra = useMemo(
    () => (
      <Task.ChannelName
        channel={channel}
        namePath={namePath}
        id={getTargetChannelNameID(itemKey)}
        level="small"
        color={9}
      />
    ),
    [channel, namePath, itemKey],
  );
  const Item = isPlain ? TopicListItem : SparkplugListItem;
  return <Item {...props} path={TARGETS_PATH} extra={extra} />;
};

const targetListItem = Component.renderProp(TargetListItem);

const EnumValuesEditor: FC<{ channelPath: string }> = ({ channelPath }) => (
  <PlatformForm.KeyValueEditor
    path={`${channelPath}.enumValues`}
    label="Enum mappings"
    keyField="label"
    keyPlaceholder="String (e.g. ON)"
    valueType="number"
    valueFirst
  />
);

const ChannelFieldSection: FC<{ targetPath: string }> = ({ targetPath }) => {
  const channelPath = `${targetPath}.channel`;
  const channelKey = PForm.useFieldValue<number>(`${channelPath}.channel`);
  const jsonType = PForm.useFieldValue<string>(`${channelPath}.jsonType`);
  const channelQuery = useMemo(
    () => (primitive.isNonZero(channelKey) ? { key: channelKey } : null),
    [channelKey],
  );
  const { data: dataType } = PChannel.useResultDataType(channelQuery);
  const { set } = PForm.useContext();

  // The driver rejects enum values on any JSON type but string.
  const handleJSONTypeChange = useCallback(
    (value: string) => {
      if (value !== "string") set(`${channelPath}.enumValues`, []);
    },
    [set, channelPath],
  );

  return (
    <>
      <Header.Header>
        <Header.Title weight={500} color={9}>
          Channel
        </Header.Title>
      </Header.Header>
      <Flex.Box className={CSS.B("channel-field-section")}>
        <Flex.Box x align="end" gap="large">
          <PForm.TextField
            path={`${channelPath}.pointer`}
            label="JSON pointer"
            grow
            inputProps={JSON_POINTER_INPUT_PROPS}
          />
          <PForm.Field<string>
            path={`${channelPath}.jsonType`}
            label="JSON type"
            className={CSS.B("json-type-select")}
            onChange={handleJSONTypeChange}
          >
            {renderSelectJSONType}
          </PForm.Field>
        </Flex.Box>
        {channelKey === 0 && (
          <PForm.Field<string>
            path={`${channelPath}.dataType`}
            label="Synnax data type"
            showHelpText={false}
            className={CSS.B("data-type-select")}
          >
            {renderSelectDataType}
          </PForm.Field>
        )}
        {dataType != null && DataType.TIMESTAMP.equals(dataType) && (
          <TimeFormatField path={`${channelPath}.timeFormat`} label="Time format" />
        )}
        {jsonType === "string" && <EnumValuesEditor channelPath={channelPath} />}
      </Flex.Box>
    </>
  );
};

const renderSelectJSONType = Component.renderProp(
  (
    p: Omit<
      Select.StaticProps<string, Select.StaticEntry<mqtt.JSONType>>,
      "data" | "resourceName"
    >,
  ) => (
    <Select.Static<string, Select.StaticEntry<mqtt.JSONType>>
      {...p}
      data={JSON_TYPE_DATA}
      resourceName="JSON type"
    />
  ),
);

const JSON_POINTER_INPUT_PROPS = { placeholder: "/value" } as const;

const renderSelectDataType = Component.renderProp((p: Telem.SelectDataTypeProps) => (
  <Telem.SelectDataType
    {...p}
    hideDataTypes={HIDDEN_DATA_TYPES}
    hideVariableDensity
    location="bottom"
  />
));

const generatorDisplayKey = (
  generator: GeneratorType | null | undefined,
  timeFormat: TimeFormat | null | undefined,
): GeneratorType | TimeFormat => {
  if (generator === "timestamp") return timeFormat ?? "iso8601";
  return "uuid";
};

const FieldListItem = (props: List.ItemProps<string> & { targetKey: string }) => {
  const { itemKey, targetKey } = props;
  const path = `${TARGETS_PATH}.${targetKey}.fields.${itemKey}`;
  const fieldType = PForm.useFieldValue<string>(`${path}.type`);
  const jsonType = PForm.useFieldValue<mqtt.JSONType | undefined>(`${path}.jsonType`, {
    optional: true,
  });
  const generator = PForm.useFieldValue<GeneratorType | undefined>(
    `${path}.generator`,
    { optional: true },
  );
  const timeFormat = PForm.useFieldValue<TimeFormat | undefined>(`${path}.timeFormat`, {
    optional: true,
  });
  const { set } = PForm.useContext();

  const handleJSONTypeChange = useCallback(
    (value: mqtt.JSONType) => {
      set(`${path}.jsonType`, value);
      set(`${path}.value`, json.ZERO_PRIMITIVES[value]);
    },
    [set, path],
  );

  const handleGeneratorChange = useCallback(
    (key: GeneratorType | TimeFormat) => {
      if (key === "uuid") {
        set(`${path}.generator`, "uuid");
        set(`${path}.timeFormat`, undefined);
      } else {
        set(`${path}.generator`, "timestamp");
        set(`${path}.timeFormat`, key);
      }
    },
    [set, path],
  );

  return (
    <Select.ListItem {...props} justify="between" align="center" x>
      <PForm.TextField
        path={`${path}.pointer`}
        showLabel={false}
        showHelpText={false}
        inputProps={FIELD_POINTER_INPUT_PROPS}
        grow
      />
      {fieldType === "static" && (
        <Select.Static<mqtt.JSONType, Select.StaticEntry<mqtt.JSONType>>
          value={jsonType ?? "string"}
          onChange={handleJSONTypeChange}
          data={JSON_TYPE_DATA}
          resourceName="type"
          className={CSS.B("field-data-type")}
        />
      )}
      {fieldType === "static" && jsonType === "string" && (
        <PForm.TextField
          path={`${path}.value`}
          showLabel={false}
          showHelpText={false}
          inputProps={STRING_INPUT_PROPS}
          className={CSS.B("static-field-value")}
        />
      )}
      {fieldType === "static" && jsonType === "number" && (
        <PForm.NumericField
          path={`${path}.value`}
          showLabel={false}
          showHelpText={false}
          className={CSS.B("static-field-value")}
        />
      )}
      {fieldType === "static" && jsonType === "boolean" && (
        <PForm.SwitchField
          path={`${path}.value`}
          showLabel={false}
          showHelpText={false}
        />
      )}
      {fieldType === "generated" && (
        <Select.Static<
          GeneratorType | TimeFormat,
          Select.StaticEntry<GeneratorType | TimeFormat>
        >
          value={generatorDisplayKey(generator, timeFormat)}
          onChange={handleGeneratorChange}
          data={GENERATOR_DATA}
          resourceName="generator"
          variant="floating"
        />
      )}
      <Text.Text level="small" color={9}>
        {fieldType}
      </Text.Text>
    </Select.ListItem>
  );
};

const FIELD_POINTER_INPUT_PROPS = { placeholder: "/field" } as const;

const STRING_INPUT_PROPS = { placeholder: "value" } as const;

const AdditionalFields: FC<{ targetKey: string }> = ({ targetKey }) => {
  const path = `${TARGETS_PATH}.${targetKey}.fields`;
  const { data, push, remove } = PForm.useFieldList<string, WriteField>(path);
  const [selected, setSelected] = useState<string[]>([]);
  const isPreview = Task.useIsPreview();

  const handleAddStatic = useCallback(() => {
    const field: WriteField = {
      key: id.create(),
      pointer: "",
      jsonType: "string",
      type: "static",
      value: "",
    };
    push(field);
    setSelected([field.key]);
  }, [push]);

  const handleAddGenerated = useCallback(() => {
    const field: WriteField = {
      key: id.create(),
      pointer: "",
      type: "generated",
      generator: "uuid",
    };
    push(field);
    setSelected([field.key]);
  }, [push]);

  const handleRemove = useCallback(
    (keys: string[]) => {
      remove(keys);
      setSelected([]);
    },
    [remove],
  );

  const listItem = useCallback(
    ({ key, ...p }: List.ItemProps<string>) => (
      <FieldListItem {...p} key={key} targetKey={targetKey} />
    ),
    [targetKey],
  );

  const menuProps = Menu.useContextMenu();
  const menuRenderProp = useCallback(
    (p: Menu.ContextMenuMenuProps) => (
      <ContextMenu keys={p.keys} onRemove={handleRemove} />
    ),
    [handleRemove],
  );

  return (
    <Flex.Box y grow empty className={CSS.B("additional-fields")}>
      <Header.Header>
        <Header.Title weight={500} color={9}>
          Additional fields
        </Header.Title>
        {!isPreview && (
          <Header.Actions>
            <Button.Button
              onClick={handleAddStatic}
              variant="filled"
              tooltip="Add static field"
              size="small"
            >
              <Icon.Add />
            </Button.Button>
            <Button.Button
              onClick={handleAddGenerated}
              variant="filled"
              tooltip="Add generated field"
              size="small"
            >
              <Icon.Time />
            </Button.Button>
          </Header.Actions>
        )}
      </Header.Header>
      <Menu.ContextMenu {...menuProps} menu={menuRenderProp}>
        <Select.Frame<string, WriteField>
          multiple
          data={data}
          value={selected}
          onChange={setSelected}
          replaceOnSingle
          allowNone
        >
          <List.Items<string, WriteField>
            full="y"
            className={CSS.cls(menuProps.className, CSS.B("field-list-items"))}
            onContextMenu={menuProps.open}
            emptyContent={EMPTY_CONTENT}
          >
            {listItem}
          </List.Items>
        </Select.Frame>
      </Menu.ContextMenu>
    </Flex.Box>
  );
};

const EMPTY_CONTENT = <Empty.Action message="No additional fields" />;

const SparkplugTargetDetails: FC<{ path: string }> = ({ path }) => {
  const channel = PForm.useFieldValue<number>(`${path}.channel`);
  return (
    <Flex.Box y grow empty className={CSS.B("topic-details")}>
      <Flex.Box gap="small" empty className={CSS.B("topic-details-form")}>
        <SparkplugTagFields path={path} />
        <Flex.Box x align="center" justify="between">
          <Task.ChannelName channel={channel} namePath={`${path}.name`} />
          <SparkplugTypeField
            path={`${path}.sparkplugType`}
            label="Sparkplug B type"
            showHelpText={false}
            className={CSS.B("data-type-select")}
          />
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
  );
};

const TargetDetails: FC<{ targetKey: string }> = ({ targetKey }) => {
  const path = `${TARGETS_PATH}.${targetKey}`;
  const type = PForm.useFieldValue<WriteTarget["type"]>(`${path}.type`);
  if (type === "sparkplug") return <SparkplugTargetDetails path={path} />;
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
          <PForm.SwitchField path={`${path}.retained`} label="Retain message" />
        </Flex.Box>
      </Flex.Box>
      <Divider.Divider x />
      <ChannelFieldSection targetPath={path} />
      <Divider.Divider x />
      <AdditionalFields key={targetKey} targetKey={targetKey} />
    </Flex.Box>
  );
};

const TOPIC_INPUT_PROPS = { placeholder: "plant/line1/valve/set" } as const;

const createTarget = (topic?: BrowsedTopic): PlainWriteTarget => ({
  ...mqtt.plainWriteTargetZ.parse({ type: "plain" }),
  ...(topic != null && { topic: topic.topic }),
});

const createSparkplugTarget = (tag?: SparkplugHaulTag): SparkplugWriteTarget => {
  const target = mqtt.sparkplugWriteTargetZ.parse({ type: "sparkplug" });
  if (tag == null) return target;
  const { dataType, ...tagID } = tag;
  return { ...target, ...tagID, sparkplugType: toSparkplugDataType(dataType) };
};

const duplicateTarget = (target: WriteTarget): WriteTarget => {
  if (target.type === "sparkplug")
    return { ...target, key: id.create(), channel: 0, name: "" };
  return {
    ...target,
    key: id.create(),
    channel: { ...target.channel, channel: 0 },
    fields: target.fields.map((f) => ({ ...f, key: id.create() })),
  };
};

const renameChannel = (key: string) => Text.edit(getTargetChannelNameID(key));

const Content = ({ device }: PlatformDevice.TaskFormContentProps<Device>) => {
  const [selected, setSelected] = useState<string[]>([]);
  const isPreview = Task.useIsPreview();
  return (
    <Flex.Box x grow empty>
      {!isPreview && <Browser device={device} />}
      <TopicList<WriteTarget>
        path={TARGETS_PATH}
        title="Targets"
        noun="target"
        selected={selected}
        onSelect={setSelected}
        create={createTarget}
        createSparkplug={createSparkplugTarget}
        duplicate={duplicateTarget}
        onRename={renameChannel}
      >
        {targetListItem}
      </TopicList>
      <Divider.Divider y />
      <Flex.Box y grow empty className={CSS.B("topic-details-pane")}>
        <Task.Views.DetailsHeader
          path={selected.length > 0 ? `${TARGETS_PATH}.${selected[0]}` : ""}
          disabled={selected.length === 0}
        />
        {selected.length > 0 ? (
          <TargetDetails targetKey={selected[0]} />
        ) : (
          <Flex.Box y grow align="center" justify="center">
            <Text.Text status="disabled">Select a target to configure</Text.Text>
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

const getInitialValues: Task.GetInitialValues<WriteSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = WRITE_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: "MQTT write task", type: WRITE_TYPE, config: cfg };
};

const onConfigure: Task.OnConfigure<WriteSchemas["config"]> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({ key: config.device, schemas: SCHEMAS });
  const safeDevName = channel.escapeInvalidName(dev.name);
  let modified = false;
  try {
    for (const target of config.targets) {
      if (target.disabled) continue;
      let changed: boolean;
      if (target.type === "plain") {
        const { channel: field, topic } = target;
        [field.channel, changed] = await Task.configureCommandChannel(
          client,
          dev.properties.write,
          {
            propertiesKey: topic,
            channel: field.channel,
            name: primitive.isNonZero(field.name)
              ? field.name
              : `${safeDevName}_${channel.escapeInvalidName(topic)}_cmd`,
            dataType: field.dataType,
          },
        );
      } else
        [target.channel, changed] = await Task.configureCommandChannel(
          client,
          dev.properties.write,
          {
            propertiesKey: sparkplugPropertiesKey(target),
            channel: target.channel,
            name: primitive.isNonZero(target.name)
              ? target.name
              : `${sparkplugChannelName(dev.name, target)}_cmd`,
            dataType: fromSparkplugDataType(target.sparkplugType),
          },
        );
      modified ||= changed;
    }
  } finally {
    if (modified) await client.devices.create(dev, SCHEMAS);
  }
  return [config, dev.rack];
};

export const Write = Task.wrapForm({
  Properties,
  Form,
  schemas: WRITE_SCHEMAS,
  deployConfigZ: deployWriteConfigZ,
  type: WRITE_TYPE,
  getInitialValues,
  onConfigure,
});

export const useCreateWrite = Task.createUseCreate({
  getInitialValues,
});

export const WriteSelectable = Selector.createSelectable({
  type: WRITE_TYPE,
  title: "MQTT write task",
  icon: <Icon.Logo.MQTT />,
  useOnSelect: useCreateWrite,
});
