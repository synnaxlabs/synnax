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
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Channel as PChannel, Telem } from "@synnaxlabs/pluto";
import { DataType, id, json, primitive } from "@synnaxlabs/x";
import { type FC, useCallback, useMemo, useState } from "react";

import { Browser } from "@/feature/mqtt/device/Browser";
import { use } from "@/feature/mqtt/device/queries";
import { Select as SelectDevice } from "@/feature/mqtt/device/Select";
import { type SparkplugHaulTag } from "@/feature/mqtt/device/SparkplugBrowser";
import { type Device, SCHEMAS } from "@/feature/mqtt/device/types";
import { useConnectModal } from "@/feature/mqtt/device/useConnectModal";
import { EntryLabel } from "@/feature/mqtt/task/EntryLabel";
import { AddEntryButtons, useEntryDrop } from "@/feature/mqtt/task/EntryList";
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

const JSON_TYPE_ITEMS = (
  <>
    <Select.Item itemKey="number">Number</Select.Item>
    <Select.Item itemKey="string">String</Select.Item>
    <Select.Item itemKey="boolean">Boolean</Select.Item>
  </>
);

const getTargetChannelNameID = (targetKey: string) => `write-target-ch-${targetKey}`;

/** The name configure gives the command channel of a plain target. */
const commandChannelName = (devName: string, topic: string): string =>
  `${channel.escapeInvalidName(devName)}_${channel.escapeInvalidName(topic)}_cmd`;

/** A plain target holds its channel in a field; a Sparkplug B target is the field. */
const channelPaths = (path: string, type: WriteTarget["type"]) =>
  type === "plain"
    ? { key: `${path}.channel.channel`, name: `${path}.channel.name` }
    : { key: `${path}.channel`, name: `${path}.name` };

const TargetItem = (props: List.ItemProps<string>) => {
  const { itemKey } = props;
  const path = `${TARGETS_PATH}.${itemKey}`;
  const type = PForm.useFieldValue<WriteTarget["type"]>(`${path}.type`);
  const disabled = PForm.useFieldValue<boolean>(`${path}.disabled`);
  const paths = channelPaths(path, type);
  const channelKey = PForm.useFieldValue<number>(paths.key);
  return (
    <Select.Item
      {...props}
      y
      align="start"
      gap="small"
      className={CSS.cls(CSS.B("target-item"), disabled && CSS.M("off"))}
    >
      <Flex.Box x align="center" gap="small" full="x">
        <EntryLabel path={path} />
        <Task.EnabledCheckbox path={`${path}.disabled`} />
      </Flex.Box>
      <Task.ChannelName
        channel={channelKey}
        namePath={paths.name}
        id={getTargetChannelNameID(itemKey)}
        level="small"
        weight={450}
        color={9}
        overflow="ellipsis"
      />
    </Select.Item>
  );
};

const targetItem = Component.renderProp(TargetItem);

const EMPTY_TARGETS = <Empty.Action message="No targets" />;

interface DataTypeFieldProps {
  path: string;
  bound: boolean;
}

const DataTypeField = ({ path, bound }: DataTypeFieldProps) => (
  <PForm.Field<string>
    path={path}
    label="Data type"
    padHelpText={false}
    helpText={bound ? "Set on the channel" : undefined}
  >
    {(p) => renderSelectDataType({ ...p, disabled: bound })}
  </PForm.Field>
);

interface ChannelSectionProps {
  targetPath: string;
  deviceName: string;
}

const ChannelSection: FC<ChannelSectionProps> = ({ targetPath, deviceName }) => {
  const channelPath = `${targetPath}.channel`;
  const channelKey = PForm.useFieldValue<number>(`${channelPath}.channel`);
  const jsonType = PForm.useFieldValue<string>(`${channelPath}.jsonType`);
  const topic = PForm.useFieldValue<string>(`${targetPath}.topic`);
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
      <PForm.Section title="Channel">
        <Task.ChannelNameField
          channel={channelKey}
          namePath={`${channelPath}.name`}
          defaultName={commandChannelName(deviceName, topic)}
        />
        <DataTypeField path={`${channelPath}.dataType`} bound={channelKey !== 0} />
        <PForm.TextField
          path={`${channelPath}.pointer`}
          label="Pointer"
          padHelpText={false}
          inputProps={JSON_POINTER_INPUT_PROPS}
        />
        <PForm.Field<string>
          path={`${channelPath}.jsonType`}
          label="JSON type"
          padHelpText={false}
          onChange={handleJSONTypeChange}
        >
          {renderSelectJSONType}
        </PForm.Field>
        {dataType != null && DataType.TIMESTAMP.equals(dataType) && (
          <TimeFormatField path={`${channelPath}.timeFormat`} label="Time format" />
        )}
      </PForm.Section>
      {jsonType === "string" && (
        <PForm.Section title="Enum mapping">
          <PlatformForm.KeyValueEditor
            path={`${channelPath}.enumValues`}
            keyField="label"
            keyPlaceholder="String (e.g. ON)"
            valueType="number"
            valueFirst
          />
        </PForm.Section>
      )}
    </>
  );
};

const renderSelectJSONType = Component.renderProp(
  (p: Omit<Select.SingleSimpleProps<string>, "children" | "resourceName">) => (
    <Select.Simple<string> {...p} resourceName="JSON type">
      {JSON_TYPE_ITEMS}
    </Select.Simple>
  ),
);

const JSON_POINTER_INPUT_PROPS = { placeholder: "Empty sends the bare value" } as const;

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
    <Select.Item {...props} justify="between" align="center" x>
      <PForm.TextField
        path={`${path}.pointer`}
        showLabel={false}
        showHelpText={false}
        inputProps={FIELD_POINTER_INPUT_PROPS}
      />
      {fieldType === "static" && (
        <Select.Simple<mqtt.JSONType>
          value={jsonType ?? "string"}
          onChange={handleJSONTypeChange}
          resourceName="type"
        >
          {JSON_TYPE_ITEMS}
        </Select.Simple>
      )}
      {fieldType === "static" && jsonType === "string" && (
        <PForm.TextField
          path={`${path}.value`}
          showLabel={false}
          showHelpText={false}
          inputProps={STRING_INPUT_PROPS}
        />
      )}
      {fieldType === "static" && jsonType === "number" && (
        <PForm.NumericField
          path={`${path}.value`}
          showLabel={false}
          showHelpText={false}
          inputProps={NUMBER_INPUT_PROPS}
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
        <Select.Simple<GeneratorType | TimeFormat>
          value={generatorDisplayKey(generator, timeFormat)}
          onChange={handleGeneratorChange}
          resourceName="generator"
          variant="floating"
        >
          <Select.Item itemKey="uuid">UUID</Select.Item>
          <Select.Item itemKey="iso8601">Timestamp (ISO 8601)</Select.Item>
          <Select.Item itemKey="unix_sec">Timestamp (s)</Select.Item>
          <Select.Item itemKey="unix_ms">Timestamp (ms)</Select.Item>
          <Select.Item itemKey="unix_us">Timestamp (µs)</Select.Item>
          <Select.Item itemKey="unix_ns">Timestamp (ns)</Select.Item>
        </Select.Simple>
      )}
      <Text.Text level="small" color={9}>
        {fieldType}
      </Text.Text>
    </Select.Item>
  );
};

// The section layout dissolves each field's own box, so sizing lives on the inputs.
const FIELD_POINTER_INPUT_PROPS = { placeholder: "/field", grow: true } as const;

const STRING_INPUT_PROPS = {
  placeholder: "value",
  className: CSS.B("static-field-value"),
} as const;

const NUMBER_INPUT_PROPS = { className: CSS.B("static-field-value") } as const;

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
      <Task.Views.ContextMenu keys={p.keys} onRemove={handleRemove} />
    ),
    [handleRemove],
  );

  const actions = !isPreview && (
    <>
      <Button.Button
        onClick={handleAddStatic}
        variant="text"
        size="small"
        tooltip="Add static field"
      >
        <Icon.Add />
        Static
      </Button.Button>
      <Button.Button
        onClick={handleAddGenerated}
        variant="text"
        size="small"
        tooltip="Add generated field"
      >
        <Icon.Time />
        Generated
      </Button.Button>
    </>
  );

  return (
    <PForm.Section title="Additional fields" actions={actions}>
      <Menu.ContextMenu {...menuProps} menu={menuRenderProp}>
        <Select.Frame<string, WriteField>
          multiple
          data={data}
          value={selected}
          onChange={setSelected}
          replaceOnSingle
          allowNone={false}
          autoSelectOnNone
        >
          <List.Scroll className={menuProps.className} onContextMenu={menuProps.open}>
            <List.Items<string, WriteField> emptyContent={EMPTY_FIELDS}>
              {listItem}
            </List.Items>
          </List.Scroll>
        </Select.Frame>
      </Menu.ContextMenu>
    </PForm.Section>
  );
};

const EMPTY_FIELDS = <Empty.Action message="No additional fields" />;

interface PaneProps {
  path: string;
  deviceName: string;
}

const SparkplugTargetPane: FC<PaneProps> = ({ path, deviceName }) => {
  const target = PForm.useFieldValue<SparkplugWriteTarget>(path);
  return (
    <PForm.Sections>
      <PForm.Section title="Tag">
        <SparkplugTagFields path={path} />
      </PForm.Section>
      <PForm.Section title="Channel">
        <Task.ChannelNameField
          channel={target.channel}
          namePath={`${path}.name`}
          defaultName={`${sparkplugChannelName(deviceName, target)}_cmd`}
        />
        <SparkplugTypeField
          path={`${path}.sparkplugType`}
          label="Sparkplug B type"
          padHelpText={false}
        />
      </PForm.Section>
    </PForm.Sections>
  );
};

const PlainTargetPane: FC<PaneProps & { targetKey: string }> = ({
  path,
  deviceName,
  targetKey,
}) => (
  <PForm.Sections>
    <PForm.Section title="Publish">
      <PForm.TextField
        path={`${path}.topic`}
        label="Topic"
        padHelpText={false}
        inputProps={TOPIC_INPUT_PROPS}
      />
      <QoSField path={`${path}.qos`} />
      <PForm.SwitchField path={`${path}.retained`} label="Retain" padHelpText={false} />
    </PForm.Section>
    <ChannelSection targetPath={path} deviceName={deviceName} />
    <AdditionalFields key={targetKey} targetKey={targetKey} />
  </PForm.Sections>
);

const TargetPane: FC<{ targetKey: string; deviceName: string }> = ({
  targetKey,
  deviceName,
}) => {
  const path = `${TARGETS_PATH}.${targetKey}`;
  const type = PForm.useFieldValue<WriteTarget["type"]>(`${path}.type`);
  return (
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      {type === "plain" ? (
        <PlainTargetPane path={path} deviceName={deviceName} targetKey={targetKey} />
      ) : (
        <SparkplugTargetPane path={path} deviceName={deviceName} />
      )}
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
  const { data, push, remove } = PForm.useFieldList<string, WriteTarget>(TARGETS_PATH);
  const ctx = PForm.useContext();
  const isPreview = Task.useIsPreview();

  const handleAddTargets = useCallback(
    (added: WriteTarget[]) => {
      push(added);
      setSelected([added[0].key]);
    },
    [push],
  );
  const handleAddTarget = useCallback(
    () => handleAddTargets([createTarget()]),
    [handleAddTargets],
  );
  const handleAddSparkplugTarget = useCallback(
    () => handleAddTargets([createSparkplugTarget()]),
    [handleAddTargets],
  );
  const dropProps = useEntryDrop<WriteTarget>({
    path: TARGETS_PATH,
    create: createTarget,
    createSparkplug: createSparkplugTarget,
    onAdd: handleAddTargets,
  });

  const handleRemove = useCallback(
    (keys: string[]) => {
      remove(keys);
      setSelected([]);
    },
    [remove],
  );

  const handleDuplicate = useCallback(
    (keys: string[]) => {
      const duplicated = ctx
        .get<WriteTarget[]>(TARGETS_PATH)
        .value.filter(({ key }) => keys.includes(key))
        .map(duplicateTarget);
      if (duplicated.length > 0) handleAddTargets(duplicated);
    },
    [ctx, handleAddTargets],
  );

  const menuProps = Menu.useContextMenu();
  const menuRenderProp = useCallback(
    ({ keys }: Menu.ContextMenuMenuProps) => {
      const disabled = keys.map(
        (key) => ctx.get<boolean>(`${TARGETS_PATH}.${key}.disabled`).value,
      );
      const setDisabled = (value: boolean) =>
        keys.forEach((key) => ctx.set(`${TARGETS_PATH}.${key}.disabled`, value));
      return (
        <Task.Views.ContextMenu
          keys={keys}
          onRemove={handleRemove}
          onDuplicate={handleDuplicate}
          onRename={renameChannel}
          onEnable={disabled.includes(true) ? () => setDisabled(false) : undefined}
          onDisable={disabled.includes(false) ? () => setDisabled(true) : undefined}
        />
      );
    },
    [ctx, handleRemove, handleDuplicate],
  );

  const resolve = useCallback(
    (target: WriteTarget): Partial<WriteTarget> => {
      const { write } = device.properties;
      if (target.type === "sparkplug")
        return { channel: write[sparkplugPropertiesKey(target)] ?? 0 };
      return { channel: { ...target.channel, channel: write[target.topic] ?? 0 } };
    },
    [device],
  );

  const current = selected.length > 0 ? selected[0] : null;

  return (
    <Flex.Box x grow empty>
      {!isPreview && <Browser device={device} />}
      <Task.Views.Panes
        listTitle="Targets"
        list={
          <>
            <Menu.ContextMenu {...menuProps} menu={menuRenderProp}>
              <Select.Frame<string, WriteTarget>
                multiple
                data={data}
                value={selected}
                onChange={setSelected}
                replaceOnSingle
                allowNone={false}
                autoSelectOnNone
              >
                <List.Scroll
                  full="y"
                  className={menuProps.className}
                  onContextMenu={menuProps.open}
                  {...dropProps}
                >
                  <List.Items<string, WriteTarget> emptyContent={EMPTY_TARGETS}>
                    {targetItem}
                  </List.Items>
                </List.Scroll>
              </Select.Frame>
            </Menu.ContextMenu>
            <AddEntryButtons
              onAdd={handleAddTarget}
              onAddSparkplug={handleAddSparkplugTarget}
            />
          </>
        }
        detailsPath={current != null ? `${TARGETS_PATH}.${current}` : null}
        title={current != null && <EntryLabel path={`${TARGETS_PATH}.${current}`} />}
      >
        {current != null ? (
          <TargetPane targetKey={current} deviceName={device.name} />
        ) : (
          <Flex.Box y grow align="center" justify="center">
            <Text.Text status="disabled">Select a target to configure</Text.Text>
          </Flex.Box>
        )}
      </Task.Views.Panes>
      <Task.BindChannels<WriteTarget> path={TARGETS_PATH} resolve={resolve} />
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
              : commandChannelName(dev.name, topic),
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
