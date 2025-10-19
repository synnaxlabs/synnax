// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { Component, type Haul, Icon, Text } from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";
import { type FC } from "react";

import { ContextMenu } from "@/components";
import { Common } from "@/hardware/common";
import { Device } from "@/hardware/opc/device";
import { type ChannelKeyAndIDGetter, Form } from "@/hardware/opc/task/Form";
import {
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteChannel,
  writeConfigZ,
  type writeStatusDataZ,
  type writeTypeZ,
  ZERO_WRITE_PAYLOAD,
} from "@/hardware/opc/task/types";
import { type Selector } from "@/selector";

export const WRITE_LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  type: WRITE_TYPE,
  name: ZERO_WRITE_PAYLOAD.name,
  icon: "Logo.OPC",
};

export const WRITE_SELECTABLE: Selector.Selectable = {
  key: WRITE_TYPE,
  title: "OPC UA Write Task",
  icon: <Icon.Logo.OPC />,
  create: async ({ layoutKey }) => ({ ...WRITE_LAYOUT, key: layoutKey }),
};

const Properties = () => (
  <>
    <Device.Select />
    <Common.Task.Fields.AutoStart />
  </>
);

const convertHaulItemToChannel = ({ data }: Haul.Item): WriteChannel => {
  if (typeof data?.name !== "string") throw new Error("Invalid name");
  const nodeName = data?.name;
  if (typeof data?.nodeId !== "string")
    throw new Error(`Invalid nodeId for ${nodeName}`);
  const nodeId = data?.nodeId;
  const dataType = typeof data?.dataType === "string" ? data.dataType : "float32";
  return {
    key: nodeId,
    nodeName,
    nodeId,
    cmdChannel: 0,
    enabled: true,
    dataType,
  };
};

const getChannelKeyAndID: ChannelKeyAndIDGetter<WriteChannel> = ({
  cmdChannel,
  key,
}) => ({
  key: cmdChannel,
  id: Common.Task.getChannelNameID(key, "cmd"),
});

interface ContextMenuItemProps extends Common.Task.ContextMenuItemProps<WriteChannel> {}

const ContextMenuItem: React.FC<ContextMenuItemProps> = ({ channels, keys }) => {
  if (keys.length !== 1) return null;
  const key = keys[0];
  const cmdChannel = channels.find((ch) => ch.key === key)?.cmdChannel;
  if (cmdChannel == null || cmdChannel == 0) return null;
  const handleRename = () => Text.edit(Common.Task.getChannelNameID(key, "cmd"));
  return <ContextMenu.RenameItem onClick={handleRename} showBottomDivider />;
};

const contextMenuItems = Component.renderProp(ContextMenuItem);

const TaskForm: FC<
  Common.Task.FormProps<typeof writeTypeZ, typeof writeConfigZ, typeof writeStatusDataZ>
> = () => (
  <Form
    convertHaulItemToChannel={convertHaulItemToChannel}
    getChannelKeyAndID={getChannelKeyAndID}
    contextMenuItems={contextMenuItems}
  />
);

const getChannelByNodeID = (props: Device.Properties, nodeId: string) =>
  props.write.channels[nodeId] ?? props.write.channels[caseconv.snakeToCamel(nodeId)];

const getInitialValues: Common.Task.GetInitialValues<
  typeof writeTypeZ,
  typeof writeConfigZ,
  typeof writeStatusDataZ
> = ({ deviceKey, config }) => {
  const cfg = config != null ? writeConfigZ.parse(config) : ZERO_WRITE_PAYLOAD.config;
  return {
    ...ZERO_WRITE_PAYLOAD,
    config: { ...cfg, device: deviceKey ?? cfg.device },
  };
};

const onConfigure: Common.Task.OnConfigure<typeof writeConfigZ> = async (
  client,
  config,
) => {
  const dev = await client.hardware.devices.retrieve<Device.Properties, Device.Make>({
    key: config.device,
  });
  dev.properties = Device.migrateProperties(dev.properties);
  const commandsToCreate: WriteChannel[] = [];
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
      else throw e;
    }
  }
  if (commandsToCreate.length > 0) {
    if (
      dev.properties.write.channels == null ||
      Array.isArray(dev.properties.write.channels)
    )
      dev.properties.write.channels = {};
    const commandIndexes = await client.channels.create(
      commandsToCreate.map(({ nodeName }) => ({
        name: `${nodeName}_cmd_time`,
        dataType: "timestamp",
        isIndex: true,
      })),
    );
    const commands = await client.channels.create(
      commandsToCreate.map(({ dataType, nodeName }, i) => ({
        name: `${nodeName}_cmd`,
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
  await client.hardware.devices.create(dev);
  return [config, dev.rack];
};

export const Write = Common.Task.wrapForm({
  Properties,
  Form: TaskForm,
  schemas: WRITE_SCHEMAS,
  type: WRITE_TYPE,
  getInitialValues,
  onConfigure,
});
