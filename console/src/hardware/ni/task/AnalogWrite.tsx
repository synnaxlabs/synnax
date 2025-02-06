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
import { Align, Form as PForm, List, Text } from "@synnaxlabs/pluto";
import { deep, id, primitiveIsZero } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ni/device";
import { AO_CHANNEL_FORMS } from "@/hardware/ni/task/AOChannelForms";
import { findPort } from "@/hardware/ni/task/findPort";
import { SelectAOChannelTypeField } from "@/hardware/ni/task/SelectAOChannelTypeField";
import {
  ANALOG_WRITE_TYPE,
  type AnalogWriteConfig,
  analogWriteConfigZ,
  type AnalogWriteDetails,
  type AnalogWriteType,
  AO_CHANNEL_TYPE_NAMES,
  type AOChannel,
  type AOChannelType,
  ZERO_ANALOG_WRITE_PAYLOAD,
  ZERO_AO_CHANNEL,
} from "@/hardware/ni/task/types";
import { type Layout } from "@/layout";

export const ANALOG_WRITE_LAYOUT: Common.Task.LayoutBaseState = {
  ...Common.Task.LAYOUT,
  key: ANALOG_WRITE_TYPE,
  type: ANALOG_WRITE_TYPE,
  name: ZERO_ANALOG_WRITE_PAYLOAD.name,
  icon: "Logo.LabJack",
};

export const ANALOG_WRITE_SELECTABLE: Layout.Selectable = {
  key: ANALOG_WRITE_TYPE,
  title: "NI Analog Write Task",
  icon: <Icon.Logo.NI />,
  create: (key) => ({ ...ANALOG_WRITE_LAYOUT, key }),
};

const Properties = (): ReactElement => (
  <>
    <Device.Select />
    <Common.Task.Fields.StateUpdateRate />
    <Common.Task.Fields.DataSaving />
  </>
);

interface ChannelListItemProps extends Common.Task.ChannelListItemProps<AOChannel> {}

const ChannelListItem = ({
  path,
  isSnapshot,
  ...rest
}: ChannelListItemProps): ReactElement => {
  const {
    entry: { port, enabled, type },
  } = rest;
  const { set } = PForm.useContext();
  return (
    <List.ItemFrame {...rest} justify="spaceBetween" align="center">
      <Align.Space direction="x">
        <Text.Text level="p" shade={6}>
          {port}
        </Text.Text>
        <Text.Text level="p" shade={9}>
          {AO_CHANNEL_TYPE_NAMES[type]}
        </Text.Text>
      </Align.Space>
      <Common.Task.EnableDisableButton
        value={enabled}
        onChange={(v) => set(`${path}.enabled`, v)}
        isSnapshot={isSnapshot}
      />
    </List.ItemFrame>
  );
};

interface ChannelFormProps {
  path: string;
}

const ChannelForm = ({ path }: ChannelFormProps): ReactElement => {
  const type = PForm.useFieldValue<AOChannelType>(`${path}.type`);
  const TypeForm = AO_CHANNEL_FORMS[type];
  return (
    <>
      <SelectAOChannelTypeField path={path} inputProps={{ allowNone: false }} />
      <TypeForm prefix={path} />
    </>
  );
};

const generateChannel = (channels: AOChannel[], index: number): AOChannel =>
  index === -1
    ? { ...deep.copy(ZERO_AO_CHANNEL), key: id.id() }
    : { ...deep.copy(channels[index]), port: findPort(channels), key: id.id() };

const Form: FC<
  Common.Task.FormProps<AnalogWriteConfig, AnalogWriteDetails, AnalogWriteType>
> = ({ task, isSnapshot }) => (
  <Common.Task.Layouts.ListAndDetails
    listItem={ChannelListItem}
    details={ChannelForm}
    generateChannel={generateChannel}
    isSnapshot={isSnapshot}
    initalChannels={task.config.channels}
  />
);

const getInitialPayload: Common.Task.GetInitialPayload<
  AnalogWriteConfig,
  AnalogWriteDetails,
  AnalogWriteType
> = (deviceKey) => ({
  ...ZERO_ANALOG_WRITE_PAYLOAD,
  config: {
    ...ZERO_ANALOG_WRITE_PAYLOAD.config,
    device: deviceKey ?? ZERO_ANALOG_WRITE_PAYLOAD.config.device,
  },
});

const onConfigure = async (
  client: Synnax,
  config: AnalogWriteConfig,
): Promise<AnalogWriteConfig> => {
  const dev = await client.hardware.devices.retrieve<Device.Properties, Device.Make>(
    config.device,
  );
  let modified = false;
  let shouldCreateStateIndex = primitiveIsZero(dev.properties.analogOutput.stateIndex);
  if (!shouldCreateStateIndex)
    try {
      await client.channels.retrieve(dev.properties.analogOutput.stateIndex);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreateStateIndex = true;
      else throw e;
    }
  if (shouldCreateStateIndex) {
    modified = true;
    const stateIndex = await client.channels.create({
      name: `${dev.properties.identifier}_ao_state_time`,
      dataType: "timestamp",
      isIndex: true,
    });
    dev.properties.analogOutput.stateIndex = stateIndex.key;
    dev.properties.analogOutput.channels = {};
  }
  const commandsToCreate: AOChannel[] = [];
  const statesToCreate: AOChannel[] = [];
  for (const channel of config.channels) {
    const exPair = dev.properties.analogOutput.channels[channel.port.toString()];
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
        name: `${dev.properties.identifier}_ao_${c.port}_state`,
        index: dev.properties.analogOutput.stateIndex,
        dataType: "float32",
      })),
    );
    states.forEach((s, i) => {
      const key = statesToCreate[i].port.toString();
      if (!(key in dev.properties.analogOutput.channels))
        dev.properties.analogOutput.channels[key] = { state: s.key, command: 0 };
      else dev.properties.analogOutput.channels[key].state = s.key;
    });
  }
  if (commandsToCreate.length > 0) {
    modified = true;
    const commandIndexes = await client.channels.create(
      commandsToCreate.map((c) => ({
        name: `${dev.properties.identifier}_ao_${c.port}_cmd_time`,
        dataType: "timestamp",
        isIndex: true,
      })),
    );
    const commands = await client.channels.create(
      commandsToCreate.map((c, i) => ({
        name: `${dev.properties.identifier}_ao_${c.port}_cmd`,
        index: commandIndexes[i].key,
        dataType: "float32",
      })),
    );
    commands.forEach((s, i) => {
      const key = commandsToCreate[i].port.toString();
      if (!(key in dev.properties.analogOutput.channels))
        dev.properties.analogOutput.channels[key] = { state: 0, command: s.key };
      else dev.properties.analogOutput.channels[key].command = s.key;
    });
  }
  if (modified) await client.hardware.devices.create(dev);
  config.channels = config.channels.map((c) => {
    const pair = dev.properties.analogOutput.channels[c.port.toString()];
    return { ...c, cmdChannel: pair.command, stateChannel: pair.state };
  });
  return config;
};

export const AnalogWriteTask = Common.Task.wrapForm(<Properties />, Form, {
  configSchema: analogWriteConfigZ,
  type: ANALOG_WRITE_TYPE,
  getInitialPayload,
  onConfigure,
});
