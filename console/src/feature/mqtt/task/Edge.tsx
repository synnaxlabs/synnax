// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/mqtt/task/Form.css";

import { channel, DisconnectedError, mqtt } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Channel as PChannel, Synnax } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";

import { use } from "@/feature/mqtt/device/queries";
import { Select as SelectDevice, useNoneSelected } from "@/feature/mqtt/device/Select";
import { MAKE, SCHEMAS } from "@/feature/mqtt/device/types";
import { useConnectModal } from "@/feature/mqtt/device/useConnectModal";
import {
  fromSparkplugDataType,
  sparkplugPropertiesKey,
} from "@/feature/mqtt/task/sparkplug";
import { SparkplugNodeFields } from "@/feature/mqtt/task/SparkplugTagFields";
import { SparkplugTypeField } from "@/feature/mqtt/task/SparkplugTypeField";
import {
  deployEdgeConfigZ,
  EDGE_SCHEMAS,
  EDGE_TYPE,
  type EdgeSchemas,
  type EdgeTag,
} from "@/feature/mqtt/task/types";
import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { Empty } from "@/platform/empty";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<EdgeSchemas>>();

const Properties = () => (
  <>
    <SelectDevice />
    <SparkplugNodeFields path={root.config} />
    <Flex.Box x grow>
      <PForm.NumericField
        path={root.config.authority}
        label="Authority"
        helpText="The control authority of the writes that commands make"
        inputProps={AUTHORITY_INPUT_PROPS}
      />
    </Flex.Box>
  </>
);

const AUTHORITY_INPUT_PROPS = { bounds: { lower: 0, upper: 255 } } as const;

const renderSelectChannel = Component.renderProp((p: PChannel.SelectSingleProps) => (
  <PChannel.SelectSingle {...p} location="bottom" />
));

const TagListItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = root.config.tags[itemKey];
  const tag = PForm.useFieldValue(path);
  const { commandChannel, commandChannelName } = tag;
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();

  const handleChannelChange = (key: channel.Key, { get, set }: PForm.ContextValue) => {
    if (get(path.name).value !== "") return;
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      const ch = await client.channels.retrieve(key);
      set(path.name, ch.name);
    }, "Failed to name the tag");
  };

  return (
    <Select.Item {...props} justify="between" align="center" x>
      <PForm.Field
        path={path.channel}
        showLabel={false}
        padHelpText={false}
        onChange={handleChannelChange}
      >
        {renderSelectChannel}
      </PForm.Field>
      <PForm.TextField
        path={path.name}
        showLabel={false}
        padHelpText={false}
        inputProps={NAME_INPUT_PROPS}
        grow
      />
      <SparkplugTypeField
        path={path.tagType}
        showLabel={false}
        showHelpText={false}
        className={CSS.B("field-data-type")}
      />
      <Flex.Box
        x
        align="center"
        grow
        justify="end"
        className={CSS.BE("mqtt-tag", "command")}
      >
        {commandChannel === 0 && commandChannelName == null ? (
          <Text.Text level="small" color={9}>
            No command channel
          </Text.Text>
        ) : (
          <Task.ChannelName
            channel={commandChannel}
            namePath={path.commandChannelName}
            defaultName={deriveCommandChannelName(tag)}
            skipped={tag.disabled}
            id={Task.getChannelNameID(itemKey, "cmd")}
          />
        )}
        <Task.EnabledCheckbox path={path.disabled} />
      </Flex.Box>
    </Select.Item>
  );
};

const NAME_INPUT_PROPS = { placeholder: "oven/temperature" } as const;

const listItem = Component.renderProp(TagListItem);

const isWritable = ({ commandChannel, commandChannelName }: EdgeTag): boolean =>
  commandChannel !== 0 || commandChannelName != null;

const deriveCommandChannelName = ({ name }: EdgeTag): string =>
  channel.escapeInvalidName(`${name}_cmd`);

const ContextMenuItem = ({ channels, keys }: Task.ContextMenuItemProps<EdgeTag>) => {
  const handleError = Status.useErrorHandler();
  const { set } = PForm.useContext();
  const canCreate = Access.useCreateGranted(channel.TYPE_RESOURCE_ID);
  const canRename = Access.useUpdateGranted(channel.TYPE_RESOURCE_ID);
  const withoutCommands = channels.filter((tag) => !isWritable(tag));
  const withCommands = channels.filter(isWritable);

  const handleAdd = () =>
    handleError(async () => {
      if (withoutCommands.some(({ name }) => name === ""))
        throw new Error("Name the tag before adding a command channel");
      withoutCommands.forEach((tag) =>
        set(
          root.config.tags[tag.key].commandChannelName,
          deriveCommandChannelName(tag),
        ),
      );
    }, "Failed to add command channel");

  const handleRemove = () =>
    withCommands.forEach(({ key }) => {
      const tag = root.config.tags[key];
      set(tag.commandChannel, 0);
      set(tag.commandChannelName, undefined);
    });

  const handleRename = () => Text.edit(Task.getChannelNameID(keys[0], "cmd"));

  return (
    <>
      {canCreate && withoutCommands.length > 0 && (
        <Menu.Item itemKey="addCommandChannel" onClick={handleAdd}>
          <Icon.Add />
          Add command channel
        </Menu.Item>
      )}
      {withCommands.length > 0 && (
        <Menu.Item itemKey="removeCommandChannel" onClick={handleRemove}>
          <Icon.Close />
          Remove command channel
        </Menu.Item>
      )}
      {canRename && keys.length === 1 && withCommands.length === 1 && (
        <ContextMenu.RenameItem onClick={handleRename} />
      )}
      <Menu.Divider />
    </>
  );
};

const contextMenuItems = Component.renderProp(ContextMenuItem);

const createTag = (tags: EdgeTag[]): EdgeTag => ({
  ...mqtt.edgeTagZ.parse({}),
  ...(tags.length > 0 && { tagType: tags[tags.length - 1].tagType }),
});

const HEADER = <Task.Views.Header title="Tags" />;

const EMPTY_CONTENT = <Empty.Action message="No tags in task" />;

const Content = () => (
  <Task.Views.List<EdgeTag>
    path={root.config.tags}
    header={HEADER}
    emptyContent={EMPTY_CONTENT}
    addLabel="Add tag"
    createChannel={createTag}
    listItem={listItem}
    contextMenuItems={contextMenuItems}
    resolve={null}
  />
);

const Form = PlatformDevice.wrapTaskForm({
  use,
  useConfigure: useConnectModal,
  useNoneSelected,
  Content,
});

const getInitialValues: Task.GetInitialValues<EdgeSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = EDGE_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: "Sparkplug edge node", type: EDGE_TYPE, config: cfg };
};

const onConfigure: Task.OnConfigure<EdgeSchemas["config"]> = async (client, config) => {
  const dev = await client.devices.retrieve({ key: config.device, schemas: SCHEMAS });
  const before = deep.copy(dev.properties);
  const { group, edgeNode } = config;
  try {
    for (const tag of config.tags) {
      if (tag.disabled || !isWritable(tag)) continue;
      tag.commandChannel = await Task.configureCommandChannel(
        client,
        dev.properties.edge,
        {
          propertiesKey: sparkplugPropertiesKey({
            group,
            edgeNode,
            device: "",
            tag: tag.name,
          }),
          channel: tag.commandChannel,
          name: tag.commandChannelName ?? deriveCommandChannelName(tag),
          dataType: fromSparkplugDataType(tag.tagType),
        },
      );
    }
  } finally {
    await PlatformDevice.setProperties(client, dev, before, SCHEMAS);
  }
  return [config, dev.rack];
};

export const Edge = Task.wrapForm({
  Properties,
  Form,
  schemas: EDGE_SCHEMAS,
  deployConfigZ: deployEdgeConfigZ,
  type: "mqtt_sparkplug_edge",
  onConfigure,
});

export const useCreateEdge = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const EdgeSelectable: Selector.Selectable = {
  type: EDGE_TYPE,
  title: "Sparkplug edge node",
  icon: <Icon.Logo.MQTT />,
  useOnSelect: useCreateEdge,
};
