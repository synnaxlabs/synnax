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
import { Align, Form as PForm, List } from "@synnaxlabs/pluto";
import { deep, id, primitiveIsZero } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/labjack/device";
import { getOpenPort } from "@/hardware/labjack/task/getOpenPort";
import { SelectOutputChannelType } from "@/hardware/labjack/task/SelectOutputChannelType";
import {
  type OutputChannel,
  type OutputChannelType,
  WRITE_TYPE,
  type WriteConfig,
  writeConfigZ,
  type WriteStateDetails,
  type WriteType,
  ZERO_OUTPUT_CHANNEL,
  ZERO_WRITE_PAYLOAD,
} from "@/hardware/labjack/task/types";
import { type Selector } from "@/selector";

export const WRITE_LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  type: WRITE_TYPE,
  name: ZERO_WRITE_PAYLOAD.name,
  icon: "Logo.LabJack",
};

export const WRITE_SELECTABLE: Selector.Selectable = {
  key: WRITE_TYPE,
  title: "LabJack Write Task",
  icon: <Icon.Logo.LabJack />,
  create: async ({ layoutKey }) => ({ ...WRITE_LAYOUT, key: layoutKey }),
};

const Properties = () => (
  <>
    <Device.Select />
    <Align.Space direction="x" grow>
      <Common.Task.Fields.StateUpdateRate />
      <Common.Task.Fields.DataSaving />
    </Align.Space>
  </>
);

interface ChannelListItemProps extends Common.Task.ChannelListItemProps<OutputChannel> {
  device: Device.Device;
}

const ChannelListItem = ({
  path,
  isSnapshot,
  device,
  ...rest
}: ChannelListItemProps) => {
  const {
    entry,
    entry: { cmdKey, stateKey, type, port },
  } = rest;
  const { set } = PForm.useContext();
  return (
    <List.ItemFrame
      {...rest}
      style={{ width: "100%" }}
      justify="spaceBetween"
      align="center"
      direction="x"
    >
      <Align.Pack direction="x" align="center">
        <PForm.Field<string>
          path={`${path}.port`}
          showLabel={false}
          hideIfNull
          onChange={(value) => {
            if (port === value) return;
            const existingCommandStatePair =
              device.properties[type].channels[value] ??
              Common.Device.ZERO_COMMAND_STATE_PAIR;
            set(path, {
              ...entry,
              cmdKey: existingCommandStatePair.command,
              stateKey: existingCommandStatePair.state,
              port: value,
            });
          }}
        >
          {(p) => (
            <Device.SelectPort
              {...p}
              model={device.model}
              portType={type}
              allowNone={false}
              onClick={(e) => e.stopPropagation()}
              style={{ width: 250 }}
              actions={[
                <PForm.Field<OutputChannelType>
                  key="type"
                  path={`${path}.type`}
                  showLabel={false}
                  hideIfNull
                  onChange={(value) => {
                    if (type === value) return;
                    const port = Device.DEVICES[device.model].ports[value][0].key;
                    const existingCommandStatePair =
                      device.properties[value].channels[port] ??
                      Common.Device.ZERO_COMMAND_STATE_PAIR;
                    set(path, {
                      ...entry,
                      cmdKey: existingCommandStatePair.command,
                      stateKey: existingCommandStatePair.state,
                      type: value,
                      port,
                    });
                  }}
                  empty
                >
                  {(p) => <SelectOutputChannelType {...p} />}
                </PForm.Field>,
              ]}
            />
          )}
        </PForm.Field>
      </Align.Pack>
      <Align.Space direction="x" align="center" justify="spaceEvenly">
        <Common.Task.ChannelName channel={cmdKey} defaultName="No Command Channel" />
        <Common.Task.ChannelName channel={stateKey} defaultName="No State Channel" />
        <Common.Task.EnableDisableButton
          path={`${path}.enabled`}
          isSnapshot={isSnapshot}
        />
      </Align.Space>
    </List.ItemFrame>
  );
};

const getOpenChannel = (channels: OutputChannel[], device: Device.Device) => {
  if (channels.length === 0) return { ...deep.copy(ZERO_OUTPUT_CHANNEL), key: id.id() };
  const last = channels[channels.length - 1];
  const backupType =
    last.type === Device.DO_PORT_TYPE ? Device.AO_PORT_TYPE : Device.DO_PORT_TYPE;
  const port = getOpenPort(channels, device.model, [last.type, backupType]);
  if (port == null) return null;
  const existingCommandStatePair =
    device.properties[port.type].channels[port.key] ??
    Common.Device.ZERO_COMMAND_STATE_PAIR;
  return {
    ...deep.copy(last),
    type: port.type,
    key: id.id(),
    port: port.key,
    cmdKey: existingCommandStatePair.command,
    stateKey: existingCommandStatePair.state,
  };
};

interface ChannelListProps {
  device: Device.Device;
  isSnapshot: boolean;
}

const ChannelList = ({ device, isSnapshot }: ChannelListProps) => {
  const generateChannel = useCallback(
    (channels: OutputChannel[]) => getOpenChannel(channels, device),
    [device],
  );
  return (
    <Common.Task.Layouts.List<OutputChannel>
      isSnapshot={isSnapshot}
      generateChannel={generateChannel}
      ListItem={(p) => <ChannelListItem {...p} device={device} />}
    />
  );
};

const Form: FC<Common.Task.FormProps<WriteConfig, WriteStateDetails, WriteType>> = ({
  isSnapshot,
}) => (
  <Common.Device.Provider<Device.Properties, Device.Make, Device.Model>
    canConfigure={!isSnapshot}
    configureLayout={Device.CONFIGURE_LAYOUT}
  >
    {({ device }) => <ChannelList device={device} isSnapshot={isSnapshot} />}
  </Common.Device.Provider>
);

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
  const dev = await client.hardware.devices.retrieve<Device.Properties>(config.device);
  let modified = false;
  let shouldCreateStateIndex = primitiveIsZero(dev.properties.writeStateIndex);
  if (!shouldCreateStateIndex)
    try {
      await client.channels.retrieve(dev.properties.writeStateIndex);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreateStateIndex = true;
      else throw e;
    }
  if (shouldCreateStateIndex) {
    modified = true;
    const stateIndex = await client.channels.create({
      name: `${dev.properties.identifier}_o_state_time`,
      dataType: "timestamp",
      isIndex: true,
    });
    dev.properties.writeStateIndex = stateIndex.key;
    dev.properties.DO.channels = {};
    dev.properties.AO.channels = {};
  }
  const commandChannelsToCreate: OutputChannel[] = [];
  const stateChannelsToCreate: OutputChannel[] = [];
  for (const channel of config.channels) {
    const key = channel.port;
    const existingPair = dev.properties[channel.type].channels[key];
    if (existingPair == null) {
      commandChannelsToCreate.push(channel);
      stateChannelsToCreate.push(channel);
    } else {
      const { state, command } = existingPair;
      try {
        await client.channels.retrieve(state);
      } catch (e) {
        if (NotFoundError.matches(e)) stateChannelsToCreate.push(channel);
        else throw e;
      }
      try {
        await client.channels.retrieve(command);
      } catch (e) {
        if (NotFoundError.matches(e)) commandChannelsToCreate.push(channel);
        else throw e;
      }
    }
  }
  if (stateChannelsToCreate.length > 0) {
    modified = true;
    const stateChannels = await client.channels.create(
      stateChannelsToCreate.map((c) => ({
        name: `${dev.properties.identifier}_${c.type}_${c.port}_state`,
        index: dev.properties.writeStateIndex,
        dataType: c.type === "AO" ? "float32" : "uint8",
      })),
    );
    stateChannels.forEach((c, i) => {
      const statesToCreateC = stateChannelsToCreate[i];
      const port = statesToCreateC.port;
      if (!(port in dev.properties[statesToCreateC.type].channels))
        dev.properties[statesToCreateC.type].channels[port] = {
          state: c.key,
          command: 0,
        };
      else dev.properties[statesToCreateC.type].channels[port].state = c.key;
    });
  }
  if (commandChannelsToCreate.length > 0) {
    modified = true;
    const commandIndexes = await client.channels.create(
      commandChannelsToCreate.map((c) => ({
        name: `${dev.properties.identifier}_${c.type}_${c.port}_cmd_time`,
        dataType: "timestamp",
        isIndex: true,
      })),
    );
    const commandChannels = await client.channels.create(
      commandChannelsToCreate.map((c, i) => ({
        name: `${dev.properties.identifier}_${c.type}_${c.port}_cmd`,
        index: commandIndexes[i].key,
        dataType: c.type === "AO" ? "float32" : "uint8",
      })),
    );
    commandChannels.forEach((c, i) => {
      const cmdToCreate = commandChannelsToCreate[i];
      const port = cmdToCreate.port;
      if (!(port in dev.properties[cmdToCreate.type].channels))
        dev.properties[cmdToCreate.type].channels[port] = {
          state: 0,
          command: c.key,
        };
      else dev.properties[cmdToCreate.type].channels[port].command = c.key;
    });
  }
  if (modified) await client.hardware.devices.create(dev);
  config.channels = config.channels.map((c) => {
    const pair = dev.properties[c.type].channels[c.port];
    return { ...c, cmdKey: pair.command, stateKey: pair.state };
  });
  return [config, dev.rack];
};

export const Write = Common.Task.wrapForm(() => <Properties />, Form, {
  configSchema: writeConfigZ,
  type: WRITE_TYPE,
  getInitialPayload,
  onConfigure,
});
