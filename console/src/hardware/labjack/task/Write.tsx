// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, NotFoundError } from "@synnaxlabs/client";
import { Flex, Form as PForm, Icon, List } from "@synnaxlabs/pluto";
import { deep, id, primitive } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/labjack/device";
import { getOpenPort } from "@/hardware/labjack/task/getOpenPort";
import { SelectOutputChannelType } from "@/hardware/labjack/task/SelectOutputChannelType";
import {
  type OutputChannel,
  type OutputChannelType,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  writeConfigZ,
  type writeStatusDataZ,
  type writeTypeZ,
  ZERO_OUTPUT_CHANNEL,
  ZERO_WRITE_PAYLOAD,
} from "@/hardware/labjack/task/types";
import { Selector } from "@/selector";

export const WRITE_LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  type: WRITE_TYPE,
  name: ZERO_WRITE_PAYLOAD.name,
  icon: "Logo.LabJack",
};

export const WriteSelectable: Selector.Selectable = ({ layoutKey, onPlace }) => {
  const handleClick = useCallback(() => {
    onPlace({ ...WRITE_LAYOUT, key: layoutKey });
  }, [onPlace, layoutKey]);
  return (
    <Selector.Item
      key={WRITE_TYPE}
      title="LabJack Write Task"
      icon={<Icon.Logo.LabJack />}
      onClick={handleClick}
    />
  );
};
WriteSelectable.type = WRITE_TYPE;

const Properties = () => (
  <>
    <Device.Select />
    <Flex.Box x>
      <Common.Task.Fields.StateUpdateRate />
      <Common.Task.Fields.DataSaving />
    </Flex.Box>
  </>
);

interface ChannelListItemProps extends Common.Task.ChannelListItemProps {
  device: Device.Device;
}

const ChannelListItem = ({ device, ...rest }: ChannelListItemProps) => {
  const path = `config.channels.${rest.itemKey}`;
  const { set } = PForm.useContext();
  const item = PForm.useFieldValue<OutputChannel>(path);
  const { port, type, cmdChannel, stateChannel } = item;
  return (
    <List.Item {...rest} full="x" justify="between">
      <Flex.Box pack x align="center">
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
              ...item,
              cmdChannel: existingCommandStatePair.command,
              stateChannel: existingCommandStatePair.state,
              port: value,
            });
          }}
        >
          {({ value, onChange }) => (
            <Device.SelectPort
              value={value}
              onChange={onChange}
              model={device.model}
              portType={type}
              allowNone={false}
              onClick={(e) => e.stopPropagation()}
              style={{ width: 250 }}
            >
              <PForm.Field<OutputChannelType>
                key="type"
                path={`${path}.type`}
                showLabel={false}
                hideIfNull
                gap="large"
                onChange={(value) => {
                  if (type === value) return;
                  const port = Device.PORTS[device.model][value][0].key;
                  const existingCommandStatePair =
                    device.properties[value].channels[port] ??
                    Common.Device.ZERO_COMMAND_STATE_PAIR;
                  set(path, {
                    ...item,
                    cmdChannel: existingCommandStatePair.command,
                    stateChannel: existingCommandStatePair.state,
                    type: value,
                    port,
                  });
                }}
                empty
              >
                {({ value, onChange }) => (
                  <SelectOutputChannelType value={value} onChange={onChange} />
                )}
              </PForm.Field>
            </Device.SelectPort>
          )}
        </PForm.Field>
      </Flex.Box>
      <Flex.Box x align="center" justify="evenly">
        <Common.Task.WriteChannelNames
          cmdChannel={cmdChannel}
          itemKey={item.key}
          stateChannel={stateChannel}
          cmdNamePath={`${path}.cmdChannelName`}
          stateNamePath={`${path}.stateChannelName`}
        />
        <Common.Task.EnableDisableButton path={`${path}.enabled`} />
      </Flex.Box>
    </List.Item>
  );
};

const getOpenChannel = (channels: OutputChannel[], device: Device.Device) => {
  if (channels.length === 0)
    return { ...deep.copy(ZERO_OUTPUT_CHANNEL), key: id.create() };
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
    ...Common.Task.WRITE_CHANNEL_OVERRIDE,
    type: port.type,
    key: id.create(),
    port: port.key,
    cmdChannel: existingCommandStatePair.command,
    stateChannel: existingCommandStatePair.state,
  };
};

interface ChannelListProps {
  device: Device.Device;
}

const ChannelList = ({ device }: ChannelListProps) => {
  const createChannel = useCallback(
    (channels: OutputChannel[]) => getOpenChannel(channels, device),
    [device],
  );
  const listItem = useCallback(
    ({ key, ...p }: Common.Task.ChannelListItemProps) => (
      <ChannelListItem key={key} {...p} device={device} />
    ),
    [device],
  );
  return (
    <Common.Task.Layouts.List<OutputChannel>
      createChannel={createChannel}
      listItem={listItem}
      contextMenuItems={Common.Task.writeChannelContextMenuItems}
    />
  );
};

const Form: FC<
  Common.Task.FormProps<typeof writeTypeZ, typeof writeConfigZ, typeof writeStatusDataZ>
> = () => {
  const isSnapshot = Common.Task.useIsSnapshot();
  return (
    <Common.Device.Provider<Device.Properties, Device.Make, Device.Model>
      canConfigure={!isSnapshot}
      configureLayout={Device.CONFIGURE_LAYOUT}
    >
      {({ device }) => <ChannelList device={device} />}
    </Common.Device.Provider>
  );
};

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
  const dev = await client.devices.retrieve<Device.Properties>({
    key: config.device,
  });
  Common.Device.checkConfigured(dev);
  let modified = false;
  let shouldCreateStateIndex = primitive.isZero(dev.properties.writeStateIndex);
  if (!shouldCreateStateIndex)
    try {
      await client.channels.retrieve(dev.properties.writeStateIndex);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreateStateIndex = true;
      else throw e;
    }
  const identifier = channel.escapeInvalidName(dev.properties.identifier);
  try {
    if (shouldCreateStateIndex) {
      modified = true;
      const stateIndex = await client.channels.create({
        name: `${identifier}_write_state_time`,
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
        stateChannelsToCreate.map(({ port, type, stateChannelName }) => ({
          name: primitive.isNonZero(stateChannelName)
            ? stateChannelName
            : `${identifier}_${port}_state`,
          index: dev.properties.writeStateIndex,
          dataType: type === "AO" ? "float32" : "uint8",
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
        commandChannelsToCreate.map(({ port, cmdChannelName }) => ({
          name: primitive.isNonZero(cmdChannelName)
            ? `${cmdChannelName}_time`
            : `${identifier}_${port}_cmd_time`,
          dataType: "timestamp",
          isIndex: true,
        })),
      );
      const commandChannels = await client.channels.create(
        commandChannelsToCreate.map(({ cmdChannelName, port, type }, i) => ({
          name: primitive.isNonZero(cmdChannelName)
            ? cmdChannelName
            : `${identifier}_${port}_cmd`,
          index: commandIndexes[i].key,
          dataType: type === "AO" ? "float32" : "uint8",
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
  } finally {
    if (modified) await client.devices.create(dev);
  }
  config.channels = config.channels.map((c) => {
    const pair = dev.properties[c.type].channels[c.port];
    return { ...c, cmdChannel: pair.command, stateChannel: pair.state };
  });
  return [config, dev.rack];
};

export const Write = Common.Task.wrapForm({
  Properties,
  Form,
  schemas: WRITE_SCHEMAS,
  type: WRITE_TYPE,
  getInitialValues,
  onConfigure,
});
