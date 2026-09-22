// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/mqtt/task/Form.css";

import {
  channel,
  DisconnectedError,
  mqtt,
  type Synnax as Client,
} from "@synnaxlabs/client";
import {
  Access,
  Channel as PChannel,
  Component,
  Flex,
  Form as PForm,
  Icon,
  Menu,
  Select,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { use } from "@/feature/mqtt/device/queries";
import { Select as SelectDevice } from "@/feature/mqtt/device/Select";
import { SCHEMAS } from "@/feature/mqtt/device/types";
import { useConnectModal } from "@/feature/mqtt/device/useConnectModal";
import { fromSparkplugDataType } from "@/feature/mqtt/task/sparkplug";
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
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const Properties = () => (
  <>
    <SelectDevice />
    <SparkplugNodeFields path="config" />
    <Flex.Box x grow>
      <PForm.NumericField
        path="config.authority"
        label="Authority"
        helpText="The control authority of the writes that commands make"
        inputProps={AUTHORITY_INPUT_PROPS}
      />
      <Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

const AUTHORITY_INPUT_PROPS = { bounds: { lower: 0, upper: 255 } } as const;

const TAGS_PATH = "config.tags";

const CHANNEL_QUERY: channel.RetrieveOptions = { internal: false };

const renderSelectChannel = Component.renderProp((p: PChannel.SelectSingleProps) => (
  <PChannel.SelectSingle {...p} initialQuery={CHANNEL_QUERY} location="bottom" />
));

const TagListItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = `${TAGS_PATH}.${itemKey}`;
  const commandChannel = PForm.useFieldValue<channel.Key>(`${path}.commandChannel`);
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();

  const handleChannelChange = useCallback(
    (key: channel.Key, { get, set }: PForm.ContextValue) => {
      if (get<string>(`${path}.name`).value !== "") return;
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
        const ch = await client.channels.retrieve(key);
        set(`${path}.name`, ch.name);
      }, "Failed to name the tag");
    },
    [client, handleError, path],
  );

  return (
    <Select.ListItem {...props} justify="between" align="center" x>
      <PForm.Field<channel.Key>
        path={`${path}.channel`}
        showLabel={false}
        padHelpText={false}
        onChange={handleChannelChange}
      >
        {renderSelectChannel}
      </PForm.Field>
      <PForm.TextField
        path={`${path}.name`}
        showLabel={false}
        padHelpText={false}
        inputProps={NAME_INPUT_PROPS}
        grow
      />
      <SparkplugTypeField
        path={`${path}.sparkplugType`}
        showLabel={false}
        showHelpText={false}
        className={CSS.B("field-data-type")}
      />
      <Flex.Box x align="center" grow justify="end">
        {commandChannel === 0 ? (
          <Text.Text level="small" color={9}>
            No command channel
          </Text.Text>
        ) : (
          // The tag holds no name for its command channel, so the path resolves to
          // nothing and the name comes from the Core alone.
          <Task.ChannelName
            channel={commandChannel}
            namePath={`${path}.commandChannelName`}
            defaultName="No command channel"
            id={Task.getChannelNameID(itemKey, "cmd")}
          />
        )}
        <Task.EnableDisableButton path={`${path}.disabled`} />
      </Flex.Box>
    </Select.ListItem>
  );
};

const NAME_INPUT_PROPS = { placeholder: "oven/temperature" } as const;

const listItem = Component.renderProp(TagListItem);

const createCommandChannel = async (
  client: Client,
  tag: EdgeTag,
): Promise<channel.Channel> => {
  if (tag.name === "") throw new Error("Name the tag before adding a command channel");
  const name = channel.escapeInvalidName(`${tag.name}_cmd`);
  return await Task.createChannel(
    client,
    name,
    fromSparkplugDataType(tag.sparkplugType),
  );
};

const ContextMenuItem = ({ channels, keys }: Task.ContextMenuItemProps<EdgeTag>) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const { set } = PForm.useContext();
  const canCreate = Access.useCreateGranted(channel.TYPE_ONTOLOGY_ID);
  const canRename = Access.useUpdateGranted(channel.TYPE_ONTOLOGY_ID);
  const withoutCommands = channels.filter(({ commandChannel }) => commandChannel === 0);
  const withCommands = channels.filter(({ commandChannel }) => commandChannel !== 0);

  const handleAdd = () =>
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      for (const tag of withoutCommands) {
        const ch = await createCommandChannel(client, tag);
        set(`${TAGS_PATH}.${tag.key}.commandChannel`, ch.key);
      }
    }, "Failed to add command channel");

  const handleRemove = () =>
    withCommands.forEach(({ key }) => set(`${TAGS_PATH}.${key}.commandChannel`, 0));

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
  ...(tags.length > 0 && { sparkplugType: tags[tags.length - 1].sparkplugType }),
});

const Content = () => (
  <Task.Views.List<EdgeTag>
    path={TAGS_PATH}
    createChannel={createTag}
    listItem={listItem}
    contextMenuItems={contextMenuItems}
  />
);

const Form = PlatformDevice.wrapTaskForm({
  use,
  useConfigure: useConnectModal,
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

// A deleted command channel drops to zero so the deploy does not reference it.
const onConfigure: Task.OnConfigure<EdgeSchemas["config"]> = async (client, config) => {
  const dev = await client.devices.retrieve({ key: config.device, schemas: SCHEMAS });
  for (const tag of config.tags) {
    if (tag.commandChannel === 0) continue;
    if (!(await Task.channelExists(client, tag.commandChannel))) tag.commandChannel = 0;
  }
  return [config, dev.rack];
};

export const Edge = Task.wrapForm({
  Properties,
  Form,
  schemas: EDGE_SCHEMAS,
  deployConfigZ: deployEdgeConfigZ,
  type: EDGE_TYPE,
  getInitialValues,
  onConfigure,
});

export const useCreateEdge = Task.createUseCreate({
  getInitialValues,
});

export const EdgeSelectable = Selector.createSelectable({
  type: EDGE_TYPE,
  title: "Sparkplug edge node",
  icon: <Icon.Logo.MQTT />,
  useOnSelect: useCreateEdge,
});
