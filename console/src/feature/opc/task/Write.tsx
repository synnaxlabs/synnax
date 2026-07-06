// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, NotFoundError } from "@synnaxlabs/client";
import { Component, Icon, Menu, Text } from "@synnaxlabs/pluto";
import { caseconv, errors, primitive } from "@synnaxlabs/x";
import { type FC } from "react";

import { type HaulItem } from "@/feature/opc/device/Browser";
import { Select } from "@/feature/opc/device/Select";
import * as Device from "@/feature/opc/device/types";
import { type ChannelKeyAndIDGetter, Form } from "@/feature/opc/task/Form";
import {
  type OutputChannel,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteSchemas,
  ZERO_WRITE_PAYLOAD,
} from "@/feature/opc/task/types";
import { ContextMenu } from "@/platform/context-menu";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

export const WRITE_LAYOUT: Task.Layout = {
  ...Task.LAYOUT,
  type: WRITE_TYPE,
  name: ZERO_WRITE_PAYLOAD.name,
  icon: "Logo.OPC",
};

export const WriteSelectable = Selector.createSimpleItem({
  title: "OPC UA Write Task",
  icon: <Icon.Logo.OPC />,
  layout: WRITE_LAYOUT,
});

const Properties = () => (
  <>
    <Select />
    <Task.Fields.AutoStart />
  </>
);

const convertHaulItemToChannel = ({ data }: HaulItem): OutputChannel => ({
  key: data.nodeId,
  nodeName: data.name,
  nodeId: data.nodeId,
  name: "",
  cmdChannel: 0,
  enabled: true,
  dataType: data.dataType,
});

const getChannelKeyAndID: ChannelKeyAndIDGetter<OutputChannel> = ({
  cmdChannel,
  key,
}) => ({
  key: cmdChannel,
  id: Task.getChannelNameID(key, "cmd"),
});

interface ContextMenuItemProps extends Task.ContextMenuItemProps<OutputChannel> {}

const ContextMenuItem: React.FC<ContextMenuItemProps> = ({ channels, keys }) => {
  if (keys.length !== 1) return null;
  const key = keys[0];
  const cmdChannel = channels.find((ch) => ch.key === key)?.cmdChannel;
  if (cmdChannel == null) return null;
  const handleRename = () => Text.edit(Task.getChannelNameID(key, "cmd"));
  return (
    <>
      <ContextMenu.RenameItem onClick={handleRename} />
      <Menu.Divider />
    </>
  );
};

const contextMenuItems = Component.renderProp(ContextMenuItem);

const TaskForm: FC<Task.FormProps<WriteSchemas>> = () => (
  <Form
    convertHaulItemToChannel={convertHaulItemToChannel}
    getChannelKeyAndID={getChannelKeyAndID}
    contextMenuItems={contextMenuItems}
  />
);

const getChannelByNodeID = (props: Device.Properties, nodeId: string) =>
  props.write.channels[nodeId] ?? props.write.channels[caseconv.snakeToCamel(nodeId)];

const getInitialValues: Task.GetInitialValues<WriteSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg =
    config != null ? WRITE_SCHEMAS.config.parse(config) : ZERO_WRITE_PAYLOAD.config;
  return {
    ...ZERO_WRITE_PAYLOAD,
    config: { ...cfg, device: deviceKey ?? cfg.device },
  };
};

const onConfigure: Task.OnConfigure<WriteSchemas["config"]> = async (
  client,
  config,
) => {
  let dev = await client.devices.retrieve({
    key: config.device,
    schemas: Device.SCHEMAS,
  });
  try {
    dev = { ...dev, properties: Device.migrateProperties(dev.properties) };
    const commandsToCreate: OutputChannel[] = [];
    for (const channel of config.channels) {
      const key = getChannelByNodeID(dev.properties, channel.nodeId);
      if (!key) {
        commandsToCreate.push(channel);
        continue;
      }
      try {
        await client.channels.retrieve(key);
      } catch (e) {
        if (NotFoundError.matches(e)) commandsToCreate.push(channel);
        else throw errors.fromUnknown(e);
      }
    }
    if (commandsToCreate.length > 0) {
      if (
        dev.properties.write.channels == null ||
        Array.isArray(dev.properties.write.channels)
      )
        dev.properties.write.channels = {};
      const commandIndexes = await client.channels.create(
        commandsToCreate.map(({ name, nodeName }) => ({
          name: primitive.isNonZero(name)
            ? `${name}_time`
            : `${channel.escapeInvalidName(nodeName)}_cmd_time`,
          dataType: "timestamp",
          isIndex: true,
        })),
      );
      const commands = await client.channels.create(
        commandsToCreate.map(({ dataType, name, nodeName }, i) => ({
          name: primitive.isNonZero(name)
            ? name
            : `${channel.escapeInvalidName(nodeName)}_cmd`,
          dataType,
          index: commandIndexes[i].key,
        })),
      );
      commands.forEach(({ key }, i) => {
        const nodeID = commandsToCreate[i].nodeId;
        dev.properties.write.channels[nodeID] = key;
      });
    }
    config.channels = config.channels.map((c) => ({
      ...c,
      cmdChannel: getChannelByNodeID(dev.properties, c.nodeId),
    }));
  } finally {
    await client.devices.create(dev, Device.SCHEMAS);
  }
  return [config, dev.rack];
};

export const Write = Task.wrapForm({
  Properties,
  Form: TaskForm,
  schemas: WRITE_SCHEMAS,
  type: "opc_write",
  getInitialValues,
  onConfigure,
});
