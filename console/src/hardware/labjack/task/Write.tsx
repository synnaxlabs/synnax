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
import {
  Align,
  Channel,
  Form,
  Input,
  List,
  Menu,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { deep, id, primitiveIsZero } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement, useCallback, useState } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { useDevice } from "@/hardware/device/useDevice";
import { createConfigureLayout } from "@/hardware/labjack/device/Configure";
import { SelectOutputChannelType, SelectPort } from "@/hardware/labjack/device/Select";
import {
  type ConfiguredDevice,
  type Device,
  DEVICES,
  type OutputChannelType,
  ZERO_COMMAND_STATE_PAIR,
} from "@/hardware/labjack/device/types";
import { SelectDevice } from "@/hardware/labjack/task/common";
import { createLayoutCreator } from "@/hardware/labjack/task/createLayoutCreator";
import {
  type Write,
  WRITE_TYPE,
  type WriteChannel,
  type WriteConfig,
  writeConfigZ,
  type WritePayload,
  type WriteStateDetails,
  type WriteType,
  ZERO_WRITE_CHANNEL,
  ZERO_WRITE_PAYLOAD,
} from "@/hardware/labjack/task/types";
import {
  ChannelListContextMenu,
  ChannelListEmptyContent,
  ChannelListHeader,
  Controls,
  EnableDisableButton,
  ParentRangeButton,
  useCreate,
  useObserveState,
  type WrappedTaskLayoutProps,
  wrapTaskLayout,
} from "@/hardware/task/common/common";
import {
  checkDesiredStateMatch,
  useDesiredState,
} from "@/hardware/task/common/useDesiredState";
import { Layout } from "@/layout";

export const createWriteLayout = createLayoutCreator<WritePayload>(
  WRITE_TYPE,
  "New LabJack Write Task",
);

export const WRITE_SELECTABLE: Layout.Selectable = {
  key: WRITE_TYPE,
  title: "LabJack Write Task",
  icon: <Icon.Logo.LabJack />,
  create: (layoutKey) => ({
    ...createWriteLayout({ create: true }),
    key: layoutKey,
  }),
};

const formSchema = z.object({ name: z.string().min(1), config: writeConfigZ });

const Wrapped = ({
  task,
  initialValues,
  layoutKey,
}: WrappedTaskLayoutProps<Write, WritePayload>): ReactElement => {
  const client = Synnax.use();
  const methods = Form.use({ values: initialValues, schema: formSchema });
  const taskState = useObserveState<WriteStateDetails>(
    methods.setStatus,
    methods.clearStatuses,
    task?.key,
    task?.state,
  );
  const isRunning = taskState?.details?.running;
  const initialState =
    isRunning === true ? "running" : isRunning === false ? "paused" : undefined;
  const [desiredState, setDesiredState] = useDesiredState(initialState, task?.key);
  const createTask = useCreate<WriteConfig, WriteStateDetails, WriteType>(layoutKey);
  const handleException = Status.useExceptionHandler();
  const configure = useMutation({
    onError: (e) => handleException(e, "Failed to configure write task"),
    mutationFn: async () => {
      if (!(await methods.validateAsync()) || client == null) return;
      const { name, config } = methods.value();
      const dev = (await client.hardware.devices.retrieve(
        config.device,
      )) as ConfiguredDevice;
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
        return {
          ...c,
          cmdKey: pair.command,
          stateKey: pair.state,
        };
      });
      methods.set("config.channels", config.channels);
      await createTask({
        key: task?.key,
        name,
        type: WRITE_TYPE,
        config,
      });
      setDesiredState("paused");
    },
  });
  const toggleMutation = useMutation({
    onError: (e) => {
      const action =
        isRunning === true ? "stop" : isRunning === false ? "start" : "toggle";
      const name = methods.get<string>("name").value ?? "write task";
      handleException(e, `Failed to ${action} ${name}`);
    },
    mutationFn: async () => {
      if (client == null) throw new Error("Not connected to Synnax cluster");
      if (task == null) throw new Error("Task is not defined");
      if (isRunning == null) throw new Error("Task state is not defined");
      setDesiredState(isRunning ? "paused" : "running");
      await task.executeCommand(isRunning ? "stop" : "start");
    },
  });
  return (
    <Align.Space className={CSS.B("task-configure")} direction="y" grow empty>
      <Align.Space>
        <Form.Form {...methods} mode={task?.snapshot ? "preview" : "normal"}>
          <Align.Space direction="y" empty>
            <Align.Space direction="x" justify="spaceBetween">
              <Form.Field<string> path="name" padHelpText={!task?.snapshot}>
                {(p) => <Input.Text variant="natural" level="h2" {...p} />}
              </Form.Field>
            </Align.Space>
            <ParentRangeButton taskKey={task?.key} />
            <Align.Space direction="x" className={CSS.B("task-properties")}>
              <SelectDevice />
              <Align.Space direction="x">
                <Form.NumericField
                  label="State Update Rate"
                  path="config.stateRate"
                  inputProps={{ endContent: "Hz" }}
                  grow
                />
                <Form.SwitchField label="State Data Saving" path="config.dataSaving" />
              </Align.Space>
            </Align.Space>
          </Align.Space>
          <Align.Space
            direction="x"
            className={CSS.B("channel-form-container")}
            bordered
            rounded
            grow
            empty
          >
            <MainContent snapshot={task?.snapshot} />
          </Align.Space>
        </Form.Form>
        <Controls
          state={taskState}
          layoutKey={layoutKey}
          snapshot={task?.snapshot}
          startingOrStopping={
            toggleMutation.isPending ||
            (!checkDesiredStateMatch(desiredState, isRunning) &&
              taskState?.variant === "success")
          }
          configuring={configure.isPending}
          onConfigure={configure.mutate}
          onStartStop={toggleMutation.mutate}
        />
      </Align.Space>
    </Align.Space>
  );
};

interface MainContentProps {
  snapshot?: boolean;
}

const MainContent = ({ snapshot }: MainContentProps): ReactElement => {
  const formCtx = Form.useContext();
  const device = useDevice(formCtx) as Device | undefined;
  const place = Layout.usePlacer();
  if (device == null)
    return (
      <Align.Space grow empty align="center" justify="center">
        <Text.Text level="p">No device selected</Text.Text>
      </Align.Space>
    );
  const handleConfigure = () => place(createConfigureLayout(device.key, {}));
  if (!device.configured)
    return (
      <Align.Space grow align="center" justify="center" direction="y">
        <Text.Text level="p">{`${device.name} is not configured.`}</Text.Text>
        {snapshot !== true && (
          <Text.Link level="p" onClick={handleConfigure}>
            {`Configure ${device.name}.`}
          </Text.Link>
        )}
      </Align.Space>
    );
  return <ChannelList path="config.channels" snapshot={snapshot} device={device} />;
};

interface ChannelListProps {
  path: string;
  snapshot?: boolean;
  device: ConfiguredDevice;
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
      ZERO_COMMAND_STATE_PAIR;
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
      <ChannelListHeader onAdd={handleAdd} snapshot={snapshot} />
      <Align.Space grow empty style={{ height: "100%" }}>
        <Menu.ContextMenu
          menu={({ keys }: Menu.ContextMenuMenuProps) => (
            <ChannelListContextMenu
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
              <ChannelListEmptyContent onAdd={handleAdd} snapshot={snapshot} />
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
  device: ConfiguredDevice;
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
              ZERO_COMMAND_STATE_PAIR;
            set(channelPath, {
              ...previousChannel,
              cmdKey: existingCommandStatePair.command,
              stateKey: existingCommandStatePair.state,
              port: value,
            });
          }}
        >
          {(p) => (
            <SelectPort
              {...p}
              model={device.model}
              channelType={entry.type}
              allowNone={false}
              onClick={(e) => e.stopPropagation()}
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
                    const port = DEVICES[device.model].ports[value][0].key;
                    const existingCommandStatePair =
                      device.properties[value].channels[port] ??
                      ZERO_COMMAND_STATE_PAIR;
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
        <EnableDisableButton
          value={entry.enabled}
          onChange={(v) => ctx.set(`${path}.enabled`, v)}
          snapshot={snapshot}
        />
      </Align.Space>
    </List.ItemFrame>
  );
};

export const ConfigureWrite = wrapTaskLayout(Wrapped, ZERO_WRITE_PAYLOAD);
