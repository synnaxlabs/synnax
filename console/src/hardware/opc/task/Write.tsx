// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { type Haul } from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";
import { type FC } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/opc/device";
import { Form } from "@/hardware/opc/task/Form";
import {
  WRITE_TYPE,
  type WriteChannel,
  type WriteConfig,
  writeConfigZ,
  type WriteStateDetails,
  type WriteType,
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
    <Common.Task.Fields.DataSaving />
    <Common.Task.Fields.AutoStart />
  </>
);

const convertHaulItemToChannel = ({ data }: Haul.Item): WriteChannel => {
  const nodeId = data?.nodeId as string;
  const name = data?.name as string;
  return {
    key: nodeId,
    name,
    nodeName: name,
    nodeId,
    cmdChannel: 0,
    enabled: true,
    dataType: (data?.dataType as string) ?? "float32",
  };
};

const TaskForm: FC<
  Common.Task.FormProps<WriteConfig, WriteStateDetails, WriteType>
> = ({ isSnapshot }) => (
  <Form isSnapshot={isSnapshot} convertHaulItemToChannel={convertHaulItemToChannel} />
);

const getChannelByNodeID = (props: Device.Properties, nodeId: string) =>
  props.write.channels[nodeId] ?? props.write.channels[caseconv.snakeToCamel(nodeId)];

const getInitialPayload: Common.Task.GetInitialPayload<
  WriteConfig,
  WriteStateDetails,
  WriteType
> = ({ deviceKey }) => ({
  ...ZERO_WRITE_PAYLOAD,
  config: {
    ...ZERO_WRITE_PAYLOAD.config,
    device: deviceKey ?? ZERO_WRITE_PAYLOAD.config.device,
  },
});

const onConfigure: Common.Task.OnConfigure<WriteConfig> = async (client, config) => {
  const dev = await client.hardware.devices.retrieve<Device.Properties, Device.Make>(
    config.device,
  );
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
      commandsToCreate.map(({ name }) => ({
        name: `${name}_cmd_time`,
        dataType: "timestamp",
        isIndex: true,
      })),
    );
    const commands = await client.channels.create(
      commandsToCreate.map(({ dataType, name }, i) => ({
        name: `${name}_cmd`,
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
  configSchema: writeConfigZ,
  type: WRITE_TYPE,
  getInitialPayload,
  onConfigure,
});
