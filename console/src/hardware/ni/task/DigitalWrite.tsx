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
import { Align, componentRenderProp } from "@synnaxlabs/pluto";
import { primitiveIsZero } from "@synnaxlabs/x";
import { type FC } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ni/device";
import { createDOChannel } from "@/hardware/ni/task/createChannel";
import {
  DigitalChannelList,
  type NameProps,
} from "@/hardware/ni/task/DigitalChannelList";
import { getDigitalChannelDeviceKey } from "@/hardware/ni/task/getDigitalChannelDeviceKey";
import {
  DIGITAL_WRITE_TYPE,
  type DigitalWriteConfig,
  digitalWriteConfigZ,
  type DigitalWriteStateDetails,
  type DigitalWriteType,
  type DOChannel,
  ZERO_DIGITAL_WRITE_PAYLOAD,
} from "@/hardware/ni/task/types";
import { type Selector } from "@/selector";

export const DIGITAL_WRITE_LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  icon: "Logo.NI",
  name: ZERO_DIGITAL_WRITE_PAYLOAD.name,
  type: DIGITAL_WRITE_TYPE,
};

export const DIGITAL_WRITE_SELECTABLE: Selector.Selectable = {
  create: async ({ layoutKey }) => ({ ...DIGITAL_WRITE_LAYOUT, key: layoutKey }),
  icon: <Icon.Logo.NI />,
  key: DIGITAL_WRITE_TYPE,
  title: "NI Digital Write Task",
};

const Properties = () => (
  <>
    <Device.Select />
    <Align.Space x>
      <Common.Task.Fields.StateUpdateRate />
      <Common.Task.Fields.DataSaving />
    </Align.Space>
  </>
);

const NameComponent = ({
  entry: { cmdChannel, stateChannel },
}: NameProps<DOChannel>) => (
  <Common.Task.WriteChannelNames cmdChannel={cmdChannel} stateChannel={stateChannel} />
);

const name = componentRenderProp(NameComponent);

const Form: FC<
  Common.Task.FormProps<DigitalWriteConfig, DigitalWriteStateDetails, DigitalWriteType>
> = (props) => (
  <DigitalChannelList {...props} createChannel={createDOChannel} name={name} />
);

const getInitialPayload: Common.Task.GetInitialPayload<
  DigitalWriteConfig,
  DigitalWriteStateDetails,
  DigitalWriteType
> = ({ deviceKey }) => ({
  ...ZERO_DIGITAL_WRITE_PAYLOAD,
  config: {
    ...ZERO_DIGITAL_WRITE_PAYLOAD.config,
    device: deviceKey ?? ZERO_DIGITAL_WRITE_PAYLOAD.config.device,
  },
});

const onConfigure: Common.Task.OnConfigure<DigitalWriteConfig> = async (
  client,
  config,
) => {
  const dev = await client.hardware.devices.retrieve<Device.Properties, Device.Make>(
    config.device,
  );
  Common.Device.checkConfigured(dev);
  dev.properties = Device.enrich(dev.model, dev.properties);
  let modified = false;
  let shouldCreateStateIndex = primitiveIsZero(dev.properties.digitalOutput.stateIndex);
  if (!shouldCreateStateIndex)
    try {
      await client.channels.retrieve(dev.properties.digitalOutput.stateIndex);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreateStateIndex = true;
      else throw e;
    }
  if (shouldCreateStateIndex) {
    modified = true;
    const stateIndex = await client.channels.create({
      name: `${dev.properties.identifier}_do_state_time`,
      dataType: "timestamp",
      isIndex: true,
    });
    dev.properties.digitalOutput.stateIndex = stateIndex.key;
    dev.properties.digitalOutput.channels = {};
  }
  const commandsToCreate: DOChannel[] = [];
  const statesToCreate: DOChannel[] = [];
  for (const channel of config.channels) {
    const key = getDigitalChannelDeviceKey(channel);
    const exPair = dev.properties.digitalOutput.channels[key];
    if (exPair == null) {
      commandsToCreate.push(channel);
      statesToCreate.push(channel);
    } else {
      const { state, command } = exPair;
      try {
        await client.channels.retrieve(state);
      } catch (e) {
        if (NotFoundError.matches(e)) statesToCreate.push(channel);
        else throw e;
      }
      try {
        await client.channels.retrieve(command);
      } catch (e) {
        if (NotFoundError.matches(e)) commandsToCreate.push(channel);
        else throw e;
      }
    }
  }
  if (statesToCreate.length > 0) {
    modified = true;
    const states = await client.channels.create(
      statesToCreate.map((c) => ({
        name: `${dev.properties.identifier}_do_${c.port}_${c.line}_state`,
        index: dev.properties.digitalOutput.stateIndex,
        dataType: "uint8",
      })),
    );
    states.forEach((s, i) => {
      const key = getDigitalChannelDeviceKey(statesToCreate[i]);
      if (!(key in dev.properties.digitalOutput.channels))
        dev.properties.digitalOutput.channels[key] = { state: s.key, command: 0 };
      else dev.properties.digitalOutput.channels[key].state = s.key;
    });
  }
  if (commandsToCreate.length > 0) {
    modified = true;
    const commandIndexes = await client.channels.create(
      commandsToCreate.map((c) => ({
        name: `${dev.properties.identifier}_do_${c.port}_${c.line}_cmd_time`,
        dataType: "timestamp",
        isIndex: true,
      })),
    );
    const commands = await client.channels.create(
      commandsToCreate.map((c, i) => ({
        name: `${dev.properties.identifier}_do_${c.port}_${c.line}_cmd`,
        index: commandIndexes[i].key,
        dataType: "uint8",
      })),
    );
    commands.forEach((s, i) => {
      const key = getDigitalChannelDeviceKey(commandsToCreate[i]);
      if (!(key in dev.properties.digitalOutput.channels))
        dev.properties.digitalOutput.channels[key] = { state: 0, command: s.key };
      else dev.properties.digitalOutput.channels[key].command = s.key;
    });
  }
  if (modified) await client.hardware.devices.create(dev);
  config.channels = config.channels.map((c) => {
    const key = getDigitalChannelDeviceKey(c);
    const pair = dev.properties.digitalOutput.channels[key];
    return { ...c, cmdChannel: pair.command, stateChannel: pair.state };
  });
  return [config, dev.rack];
};

export const DigitalWrite = Common.Task.wrapForm({
  Properties,
  Form,
  configSchema: digitalWriteConfigZ,
  getInitialPayload,
  onConfigure,
  type: DIGITAL_WRITE_TYPE,
});
