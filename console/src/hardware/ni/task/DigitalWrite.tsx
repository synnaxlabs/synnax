// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { Component, Flex, Icon } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type FC } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ni/device";
import { createDOChannel } from "@/hardware/ni/task/createChannel";
import {
  DigitalChannelList,
  type DigitalNameComponentProps,
} from "@/hardware/ni/task/DigitalChannelList";
import { getDigitalChannelDeviceKey } from "@/hardware/ni/task/getDigitalChannelDeviceKey";
import {
  DIGITAL_WRITE_SCHEMAS,
  DIGITAL_WRITE_TYPE,
  digitalWriteConfigZ,
  type digitalWriteStatusDataZ,
  type digitalWriteTypeZ,
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
    <Flex.Box x>
      <Common.Task.Fields.StateUpdateRate />
      <Common.Task.Fields.DataSaving />
      <Common.Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

interface NameComponentProps extends DigitalNameComponentProps<DOChannel> {}

const NameComponent = (props: NameComponentProps) => (
  <Common.Task.WriteChannelNames {...props} />
);

const name = Component.renderProp(NameComponent);

const Form: FC<
  Common.Task.FormProps<
    typeof digitalWriteTypeZ,
    typeof digitalWriteConfigZ,
    typeof digitalWriteStatusDataZ
  >
> = (props) => (
  <DigitalChannelList
    {...props}
    createChannel={createDOChannel}
    name={name}
    contextMenuItems={Common.Task.writeChannelContextMenuItems}
  />
);

const getInitialValues: Common.Task.GetInitialValues<
  typeof digitalWriteTypeZ,
  typeof digitalWriteConfigZ,
  typeof digitalWriteStatusDataZ
> = ({ deviceKey, config }) => {
  const cfg =
    config != null
      ? digitalWriteConfigZ.parse(config)
      : ZERO_DIGITAL_WRITE_PAYLOAD.config;
  return {
    ...ZERO_DIGITAL_WRITE_PAYLOAD,
    config: { ...cfg, device: deviceKey ?? cfg.device },
  };
};

const onConfigure: Common.Task.OnConfigure<typeof digitalWriteConfigZ> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve<Device.Properties, Device.Make>({
    key: config.device,
  });
  Common.Device.checkConfigured(dev);
  dev.properties = Device.enrich(dev.model, dev.properties);
  let modified = false;
  let shouldCreateStateIndex = primitive.isZero(
    dev.properties.digitalOutput.stateIndex,
  );
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
  if (modified) await client.devices.create(dev);
  config.channels = config.channels.map((c) => {
    const key = getDigitalChannelDeviceKey(c);
    const pair = dev.properties.digitalOutput.channels[key];
    return { ...c, cmdChannel: pair.command, stateChannel: pair.state };
  });
  return [config, dev.rack];
};

export const DigitalWrite = Common.Task.wrapForm({
  initialStatusData: null,
  Properties,
  Form,
  schemas: DIGITAL_WRITE_SCHEMAS,
  getInitialValues,
  onConfigure,
  type: DIGITAL_WRITE_TYPE,
});
