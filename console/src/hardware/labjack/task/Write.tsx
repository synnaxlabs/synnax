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
import { Align, Form as PForm, List } from "@synnaxlabs/pluto";
import { deep, id, primitiveIsZero } from "@synnaxlabs/x";
import { type FC, type ReactElement, useCallback, useState } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/labjack/device";
import { SelectOutputChannelType } from "@/hardware/labjack/task/SelectOutputChannelType";
import {
  type OutputChannelType,
  WRITE_TYPE,
  type WriteChannel,
  type WriteConfig,
  writeConfigZ,
  type WriteStateDetails,
  type WriteType,
  ZERO_WRITE_CHANNEL,
  ZERO_WRITE_PAYLOAD,
} from "@/hardware/labjack/task/types";
import { type Layout } from "@/layout";

export const WRITE_LAYOUT: Common.Task.LayoutBaseState = {
  ...Common.Task.LAYOUT,
  type: WRITE_TYPE,
  key: WRITE_TYPE,
  name: ZERO_WRITE_PAYLOAD.name,
  icon: "Logo.LabJack",
};

export const WRITE_SELECTABLE: Layout.Selectable = {
  key: WRITE_TYPE,
  title: "LabJack Write Task",
  icon: <Icon.Logo.LabJack />,
  create: (key) => ({ ...WRITE_LAYOUT, key }),
};

interface ChannelListItemProps extends Common.Task.ChannelListItemProps<WriteChannel> {
  device: Device.Device;
}

const ChannelListItem = ({
  path,
  isSnapshot,
  device,
  ...props
}: ChannelListItemProps): ReactElement => {
  const {
    entry,
    entry: { cmdKey, enabled, stateKey, type, port },
  } = props;
  const { set } = PForm.useContext();
  return (
    <List.ItemFrame
      {...props}
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
              onClick={(e: MouseEvent) => e.stopPropagation()}
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
                  {(p) => (
                    <SelectOutputChannelType
                      {...p}
                      onClick={(e) => e.stopPropagation()}
                      pack={false}
                      size="medium"
                    />
                  )}
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
          value={enabled}
          onChange={(v) => set(`${path}.enabled`, v)}
          isSnapshot={isSnapshot}
        />
      </Align.Space>
    </List.ItemFrame>
  );
};

interface ChannelListProps {
  isSnapshot: boolean;
  device: Device.Device;
}

const ChannelList = ({ isSnapshot, device }: ChannelListProps): ReactElement => {
  const [selected, setSelected] = useState<string[]>([]);
  const generateChannel = useCallback((): WriteChannel => {
    const existingCommandStatePair =
      device.properties[ZERO_WRITE_CHANNEL.type].channels[ZERO_WRITE_CHANNEL.port] ??
      Common.Device.ZERO_COMMAND_STATE_PAIR;
    return {
      ...deep.copy(ZERO_WRITE_CHANNEL),
      key: id.id(),
      cmdKey: existingCommandStatePair.command,
      stateKey: existingCommandStatePair.state,
    };
  }, [device]);
  return (
    <Common.Task.DefaultChannelList<WriteChannel>
      isSnapshot={isSnapshot}
      selected={selected}
      onSelect={setSelected}
      generateChannel={generateChannel}
    >
      {(p) => <ChannelListItem {...p} device={device} />}
    </Common.Task.DefaultChannelList>
  );
};

const Properties = (): ReactElement => (
  <>
    <Device.Select />
    <PForm.NumericField
      label="State Update Rate"
      path="config.stateRate"
      inputProps={{ endContent: "Hz" }}
    />
    <PForm.SwitchField label="State Data Saving" path="config.dataSaving" />
  </>
);

const Form: FC<Common.Task.FormProps<WriteConfig, WriteStateDetails, WriteType>> = ({
  task,
}) => (
  <Common.Device.Provider<Device.Properties, Device.Make, Device.ModelKey>
    configureLayout={Device.CONFIGURE_LAYOUT}
    isSnapshot={task?.snapshot ?? false}
  >
    {(p) => <ChannelList isSnapshot={task?.snapshot ?? false} {...p} />}
  </Common.Device.Provider>
);

const zeroPayload: Common.Task.ZeroPayloadFunction<
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

export const WriteTask = Common.Task.wrapForm(<Properties />, Form, {
  configSchema: writeConfigZ,
  type: WRITE_TYPE,
  zeroPayload,
  onConfigure: async (client, config) => {
    const dev = await client.hardware.devices.retrieve<Device.Properties>(
      config.device,
    );
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
    const commandChannelsToCreate: WriteChannel[] = [];
    const stateChannelsToCreate: WriteChannel[] = [];
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
    return config;
  },
});
