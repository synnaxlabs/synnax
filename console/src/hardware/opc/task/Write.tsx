// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, type Synnax } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { caseconv, primitiveIsZero } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/opc/device";
import { Form } from "@/hardware/opc/task/Form";
import {
  WRITE_TYPE,
  type WriteChannelConfig,
  type WriteConfig,
  writeConfigZ,
  type WriteStateDetails,
  type WriteType,
  ZERO_WRITE_PAYLOAD,
} from "@/hardware/opc/task/types";
import { type Layout } from "@/layout";

export const WRITE_LAYOUT: Common.Task.LayoutBaseState = {
  ...Common.Task.LAYOUT,
  key: WRITE_TYPE,
  type: WRITE_TYPE,
  name: ZERO_WRITE_PAYLOAD.name,
  icon: "Logo.OPC",
};

export const WRITE_SELECTABLE: Layout.Selectable = {
  key: WRITE_TYPE,
  title: "OPC UA Write Task",
  icon: <Icon.Logo.OPC />,
  create: (key) => ({ ...WRITE_LAYOUT, key }),
};

const Properties = (): ReactElement => (
  <>
    <Device.Select />
    <Common.Task.Fields.DataSaving />
  </>
);

const getChannelByNodeID = (props: Device.Properties, nodeId: string) =>
  props.write.channels[nodeId] ?? props.write.channels[caseconv.snakeToCamel(nodeId)];

const getInitialPayload: Common.Task.GetInitialPayload<
  WriteConfig,
  WriteStateDetails,
  WriteType
> = (deviceKey) => ({
  ...ZERO_WRITE_PAYLOAD,
  config: {
    ...ZERO_WRITE_PAYLOAD.config,
    device: deviceKey ?? ZERO_WRITE_PAYLOAD.config.device,
  },
});

const onConfigure = async (
  client: Synnax,
  config: WriteConfig,
): Promise<WriteConfig> => {
  const dev = await client.hardware.devices.retrieve<Device.Properties>(config.device);
  let modified = false;
  const commandsToCreate: WriteChannelConfig[] = [];
  for (const channel of config.channels) {
    const key = getChannelByNodeID(dev.properties, channel.nodeId);
    if (primitiveIsZero(key)) commandsToCreate.push(channel);
    else
      try {
        await client.channels.retrieve(key);
      } catch (e) {
        if (NotFoundError.matches(e)) commandsToCreate.push(channel);
        else throw e;
      }
  }
  if (commandsToCreate.length > 0) {
    modified = true;
    if (
      dev.properties.write.channels == null ||
      Array.isArray(dev.properties.write.channels)
    )
      dev.properties.write.channels = {};
    const commandIndexes = await client.channels.create(
      commandsToCreate.map((c) => ({
        name: `${c.name}_cmd_time`,
        dataType: "timestamp",
        isIndex: true,
      })),
    );
    const commands = await client.channels.create(
      commandsToCreate.map((c, i) => ({
        name: `${c.name}_cmd`,
        dataType: c.dataType,
        index: commandIndexes[i].key,
      })),
    );
    commands.forEach((c, i) => {
      const key = commandsToCreate[i].nodeId;
      dev.properties.write.channels[key] = c.key;
    });
  }
  config.channels = config.channels.map((c) => ({
    ...c,
    channel: getChannelByNodeID(dev.properties, c.nodeId),
  }));
  if (modified) await client.hardware.devices.create(dev);
  return config;
};

export const WriteTask = Common.Task.wrapForm(
  <Properties />,
  ({ isSnapshot }) => <Form isSnapshot={isSnapshot} />,
  { configSchema: writeConfigZ, type: WRITE_TYPE, getInitialPayload, onConfigure },
);
