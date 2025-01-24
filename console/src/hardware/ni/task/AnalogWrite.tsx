// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Form, List, Text } from "@synnaxlabs/pluto";
import { deep, id, primitiveIsZero } from "@synnaxlabs/x";
import { type FC, type ReactElement, useCallback, useState } from "react";

import { CSS } from "@/css";
import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ni/device";
import { AO_CHANNEL_FORMS } from "@/hardware/ni/task/AOChannelForms";
import { SelectAOChannelTypeField } from "@/hardware/ni/task/SelectAOChannelTypeField";
import {
  ANALOG_WRITE_TYPE,
  type AnalogWrite,
  type AnalogWriteConfig,
  analogWriteConfigZ,
  type AnalogWriteDetails,
  type AnalogWritePayload,
  type AnalogWriteType,
  AO_CHANNEL_TYPE_NAMES,
  type AOChannel,
  type AOChannelType,
  ZERO_ANALOG_WRITE_PAYLOAD,
  ZERO_AO_CHANNELS,
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

interface ChannelListItemProps extends Common.Task.ChannelListItemProps<AOChannel> {}

const ChannelListItem = ({
  path,
  isSnapshot,
  ...props
}: ChannelListItemProps): ReactElement => {
  const {
    entry: { port, enabled, type },
  } = props;
  const { set } = Form.useContext();
  return (
    <List.ItemFrame {...props} justify="spaceBetween" align="center">
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
  const type = Form.useFieldValue<AOChannelType>(`${path}.type`, true);
  if (type == null) return <></>;
  const TypeForm = AO_CHANNEL_FORMS[type];
  return (
    <>
      <Align.Space direction="y" className={CSS.B("channel-form-content")} empty>
        <SelectAOChannelTypeField path={path} inputProps={{ allowNone: false }} />
        <TypeForm prefix={path} />
      </Align.Space>
    </>
  );
};

interface ChannelDetailsProps {
  selectedChannelIndex: number;
}

const ChannelDetails = ({
  selectedChannelIndex,
}: ChannelDetailsProps): ReactElement => (
  <Common.Task.ChannelDetails>
    {selectedChannelIndex !== -1 && (
      <ChannelForm path={`config.channels.${selectedChannelIndex}`} />
    )}
  </Common.Task.ChannelDetails>
);

const availablePortFinder = (channels: AOChannel[]): (() => number) => {
  const exclude = new Set(channels.map((v) => v.port));
  return () => {
    let i = 0;
    while (exclude.has(i)) i++;
    exclude.add(i);
    return i;
  };
};

interface ChannelsFormProps {
  task: AnalogWrite | AnalogWritePayload;
}

const ChannelsForm = ({ task }: ChannelsFormProps): ReactElement => {
  const initialChannels = task.config.channels;
  const [selected, setSelected] = useState<string[]>(
    initialChannels.length ? [initialChannels[0].key] : [],
  );
  const [selectedIndex, setSelectedIndex] = useState<number>(
    initialChannels.length ? 0 : -1,
  );
  const handleSelect = useCallback(
    (v: string[], i: number) => {
      setSelected(v);
      setSelectedIndex(i);
    },
    [setSelected, setSelectedIndex],
  );
  const generateChannels = useCallback(
    (channels: AOChannel[]): AOChannel => ({
      ...deep.copy(ZERO_AO_CHANNELS.ao_voltage),
      port: availablePortFinder(channels)(),
      key: id.id(),
    }),
    [],
  );
  return (
    <>
      <Common.Task.DefaultChannelList<AOChannel>
        path="config.channels"
        isSnapshot={task.snapshot ?? false}
        selected={selected}
        onSelect={handleSelect}
        generateChannel={generateChannels}
      >
        {(props) => <ChannelListItem {...props} />}
      </Common.Task.DefaultChannelList>
      <ChannelDetails selectedChannelIndex={selectedIndex} />
    </>
  );
};

const Properties = (): ReactElement => (
  <>
    <Device.Select />
    <Form.NumericField
      label="State Update Rate"
      path="config.stateRate"
      inputProps={{ endContent: "Hz" }}
    />
    <Form.SwitchField label="State Data Saving" path="config.dataSaving" />
  </>
);

const TaskForm: FC<
  Common.Task.FormProps<AnalogWriteConfig, AnalogWriteDetails, AnalogWriteType>
> = ({ task }) => (
  <Common.Device.Provider<Device.Properties, Device.Make>
    configureLayout={Device.CONFIGURE_LAYOUT}
    isSnapshot={task?.snapshot ?? false}
  >
    {() => <ChannelsForm task={task} />}
  </Common.Device.Provider>
);

export const AnalogWriteTask = Common.Task.wrapForm(<Properties />, TaskForm, {
  configSchema: analogWriteConfigZ,
  type: ANALOG_WRITE_TYPE,
  zeroPayload: ZERO_ANALOG_WRITE_PAYLOAD,
  onConfigure: async (client, config) => {
    const dev = await client.hardware.devices.retrieve<Device.Properties, Device.Make>(
      config.device,
    );
    let modified = false;
    let shouldCreateStateIndex = primitiveIsZero(
      dev.properties.analogOutput.stateIndex,
    );
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
  },
});
