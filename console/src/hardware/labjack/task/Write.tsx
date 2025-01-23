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
import { Align, Channel, Form, List, Menu, Text } from "@synnaxlabs/pluto";
import { deep, id, primitiveIsZero } from "@synnaxlabs/x";
import { type FC, type ReactElement, useCallback, useState } from "react";

import { CSS } from "@/css";
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

interface ChannelListProps {
  path: string;
  snapshot?: boolean;
  device: Device.Device;
}

const ChannelList = ({ path, snapshot, device }: ChannelListProps): ReactElement => {
  const [selected, setSelected] = useState<string[]>([]);
  const { value, push, remove } = Form.useFieldArray<WriteChannel>({
    path,
    updateOnChildren: true,
  });
  const handleAdd = useCallback(() => {
    const existingCommandStatePair =
      device.properties[ZERO_WRITE_CHANNEL.type].channels[ZERO_WRITE_CHANNEL.port] ??
      Common.Device.ZERO_COMMAND_STATE_PAIR;
    push({
      ...deep.copy(ZERO_WRITE_CHANNEL),
      key: id.id(),
      cmdKey: existingCommandStatePair.command,
      stateKey: existingCommandStatePair.state,
    });
  }, [push, device]);
  const menuProps = Menu.useContextMenu();
  return (
    <Align.Space grow empty direction="y">
      <Common.Task.ChannelListHeader onAdd={handleAdd} snapshot={snapshot} />
      <Align.Space grow empty style={{ height: "100%" }}>
        <Menu.ContextMenu
          menu={({ keys }: Menu.ContextMenuMenuProps) => (
            <Common.Task.ChannelListContextMenu
              path={path}
              keys={keys}
              value={value}
              remove={remove}
              onSelect={(keys) => setSelected(keys)}
              snapshot={snapshot}
            />
          )}
          {...menuProps}
        >
          <List.List<string, WriteChannel>
            data={value}
            emptyContent={
              <Common.Task.ChannelListEmptyContent
                onAdd={handleAdd}
                snapshot={snapshot}
              />
            }
          >
            <List.Selector<string, WriteChannel>
              value={selected}
              allowMultiple
              replaceOnSingle
              onChange={setSelected}
            >
              <List.Core<string, WriteChannel>
                grow
                style={{ height: "calc(100% - 6rem)" }}
              >
                {({ key, entry, ...props }) => (
                  <ChannelListItem
                    key={key}
                    {...props}
                    entry={{ ...entry }}
                    snapshot={snapshot}
                    path={`${path}.${props.index}`}
                    device={device}
                  />
                )}
              </List.Core>
            </List.Selector>
          </List.List>
        </Menu.ContextMenu>
      </Align.Space>
    </Align.Space>
  );
};

interface ChannelListItemProps extends List.ItemProps<string, WriteChannel> {
  path: string;
  snapshot?: boolean;
  device: Device.Device;
}

const NO_COMMAND_CHANNEL_NAME = "No Command Channel";
const NO_STATE_CHANNEL_NAME = "No State Channel";

const ChannelListItem = ({
  path,
  entry,
  snapshot = false,
  device,
  ...props
}: ChannelListItemProps): ReactElement => {
  const ctx = Form.useContext();
  const cmdChannelName = Channel.useName(entry?.cmdKey ?? 0, NO_COMMAND_CHANNEL_NAME);
  const stateChannelName = Channel.useName(entry?.stateKey ?? 0, NO_STATE_CHANNEL_NAME);
  return (
    <List.ItemFrame
      {...props}
      entry={entry}
      style={{ width: "100%" }}
      justify="spaceBetween"
      align="center"
      direction="x"
    >
      <Align.Pack direction="x" align="center">
        <Form.Field<string>
          path={`${path}.port`}
          showLabel={false}
          hideIfNull
          empty
          onChange={(value, { path, get, set }) => {
            const channelPath = path.slice(0, path.lastIndexOf("."));
            const previousChannel = get<WriteChannel>(channelPath).value;
            if (previousChannel.port === value) return;
            const existingCommandStatePair =
              device.properties[previousChannel.type].channels[value] ??
              Common.Device.ZERO_COMMAND_STATE_PAIR;
            set(channelPath, {
              ...previousChannel,
              cmdKey: existingCommandStatePair.command,
              stateKey: existingCommandStatePair.state,
              port: value,
            });
          }}
        >
          {(p) => (
            <Device.SelectPort
              {...p}
              model={device.model as Device.ModelKey}
              portType={entry.type}
              allowNone={false}
              onClick={(e: MouseEvent) => e.stopPropagation()}
              style={{ width: 250 }}
              actions={[
                <Form.Field<OutputChannelType>
                  key="type"
                  path={`${path}.type`}
                  showLabel={false}
                  hideIfNull
                  onChange={(value, { path, get, set }) => {
                    const channelPath = path.slice(0, path.lastIndexOf("."));
                    const previousChannel = get<WriteChannel>(channelPath).value;
                    if (previousChannel.type === value) return;
                    const port =
                      Device.DEVICES[device.model as Device.ModelKey].ports[value][0]
                        .key;
                    const existingCommandStatePair =
                      device.properties[value].channels[port] ??
                      Common.Device.ZERO_COMMAND_STATE_PAIR;
                    set(channelPath, {
                      ...previousChannel,
                      cmdKey: existingCommandStatePair.command,
                      stateKey: existingCommandStatePair.state,
                      type: value,
                    });
                    set(`${channelPath}.port`, port);
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
                </Form.Field>,
              ]}
            />
          )}
        </Form.Field>
      </Align.Pack>
      <Align.Space direction="x" align="center" justify="spaceEvenly">
        <Text.Text
          level="p"
          shade={9}
          color={
            cmdChannelName === NO_COMMAND_CHANNEL_NAME
              ? "var(--pluto-warning-m1)"
              : undefined
          }
        >
          {cmdChannelName}
        </Text.Text>
        <Text.Text
          level="p"
          shade={9}
          color={
            stateChannelName === NO_STATE_CHANNEL_NAME
              ? "var(--pluto-warning-m1)"
              : undefined
          }
        >
          {stateChannelName}
        </Text.Text>
        <Common.Task.EnableDisableButton
          value={entry.enabled}
          onChange={(v) => ctx.set(`${path}.enabled`, v)}
          snapshot={snapshot}
        />
      </Align.Space>
    </List.ItemFrame>
  );
};

const TaskForm: FC<
  Common.Task.FormProps<WriteConfig, WriteStateDetails, WriteType>
> = ({ task }) => (
  <>
    <Align.Space direction="x" className={CSS.B("task-properties")}>
      <Device.Select />
      <Form.NumericField
        label="State Update Rate"
        path="config.stateRate"
        inputProps={{ endContent: "Hz" }}
        grow
      />
      <Form.SwitchField label="State Data Saving" path="config.dataSaving" />
    </Align.Space>
    <Common.Device.Provider<Device.Properties, Device.Make>
      configureLayout={Device.CONFIGURE_LAYOUT}
      snapshot={task?.snapshot}
    >
      {(p) => <ChannelList path="config.channels" snapshot={task?.snapshot} {...p} />}
    </Common.Device.Provider>
  </>
);

export const WriteTask = Common.Task.wrapForm(TaskForm, {
  configSchema: writeConfigZ,
  type: WRITE_TYPE,
  zeroPayload: ZERO_WRITE_PAYLOAD,
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
