// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/ni/task/DigitalWrite.css";

import { NotFoundError } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Channel, Form, List, Menu, Status, Synnax } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Input } from "@synnaxlabs/pluto/input";
import { Text } from "@synnaxlabs/pluto/text";
import { id, primitiveIsZero } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement, useCallback, useState } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { use } from "@/hardware/device/use";
import { CONFIGURE_LAYOUT } from "@/hardware/ni/device/Configure";
import { type Device, type Properties } from "@/hardware/ni/device/types";
import { CopyButtons, SelectDevice } from "@/hardware/ni/task/common";
import { createLayoutCreator } from "@/hardware/ni/task/createLayoutCreator";
import {
  DIGITAL_WRITE_TYPE,
  type DigitalWrite,
  type DigitalWriteConfig,
  digitalWriteConfigZ,
  type DigitalWriteDetails,
  type DigitalWritePayload,
  type DigitalWriteType,
  type DOChannel,
  ZERO_DIGITAL_WRITE_PAYLOAD,
  ZERO_DO_CHANNEL,
} from "@/hardware/ni/task/types";
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
} from "@/hardware/task/common/desiredState";
import { Layout } from "@/layout";

export const createDigitalWriteLayout = createLayoutCreator<DigitalWritePayload>(
  DIGITAL_WRITE_TYPE,
  "New NI Digital Write Task",
);

export const DIGITAL_WRITE_SELECTABLE: Layout.Selectable = {
  key: DIGITAL_WRITE_TYPE,
  title: "NI Digital Write Task",
  icon: <Icon.Logo.NI />,
  create: (layoutKey) => ({
    ...createDigitalWriteLayout({ create: true }),
    key: layoutKey,
  }),
};

const formSchema = z.object({ name: z.string().min(1), config: digitalWriteConfigZ });

const generateKey: (chan: DOChannel) => string = (chan) => `${chan.port}l${chan.line}`;

const Wrapped = ({
  task,
  initialValues,
  layoutKey,
}: WrappedTaskLayoutProps<DigitalWrite, DigitalWritePayload>): ReactElement => {
  const client = Synnax.use();
  const methods = Form.use({ values: initialValues, schema: formSchema });
  const taskState = useObserveState<DigitalWriteDetails>(
    methods.setStatus,
    methods.clearStatuses,
    task?.key,
    task?.state,
  );
  const isRunning = taskState?.details?.running;
  const initialState =
    isRunning === true ? "running" : isRunning === false ? "paused" : undefined;
  const [desiredState, setDesiredState] = useDesiredState(initialState, task?.key);
  const createTask = useCreate<
    DigitalWriteConfig,
    DigitalWriteDetails,
    DigitalWriteType
  >(layoutKey);
  const handleException = Status.useExceptionHandler();
  const addStatus = Status.useAggregator();
  const configureMutation = useMutation<void, Error, void>({
    onError: (e) => {
      const name = methods.get<string>("name").value ?? "NI digital write task";
      handleException(e, `Failed to configure ${name}`);
    },
    mutationFn: async () => {
      if (!(await methods.validateAsync()) || client == null) return;
      const { name, config } = methods.value();
      const dev = await client.hardware.devices.retrieve<Properties>(config.device);
      let modified = false;
      let shouldCreateStateIndex = primitiveIsZero(
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
        const key = generateKey(channel);
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
          const key = generateKey(statesToCreate[i]);
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
          const key = `${commandsToCreate[i].port}l${commandsToCreate[i].line}`;
          if (!(key in dev.properties.digitalOutput.channels))
            dev.properties.digitalOutput.channels[key] = { state: 0, command: s.key };
          else dev.properties.digitalOutput.channels[key].command = s.key;
        });
      }
      if (modified) await client.hardware.devices.create(dev);
      config.channels = config.channels.map((c) => {
        const key = generateKey(c);
        const pair = dev.properties.digitalOutput.channels[key];
        return { ...c, cmdChannel: pair.command, stateChannel: pair.state };
      });
      methods.set("config.channels", config.channels);
      await createTask({ key: task?.key, name, type: DIGITAL_WRITE_TYPE, config });
      setDesiredState("paused");
    },
  });
  const toggleMutation = useMutation({
    onError: ({ message }) => {
      const action =
        isRunning === true ? "stop" : isRunning === false ? "start" : "toggle";
      const name = methods.get<string>("name").value ?? "NI digital write task";
      addStatus({
        variant: "error",
        message: `Failed to ${action} ${name}`,
        description: message,
      });
    },
    mutationFn: async () => {
      if (client == null) throw new Error("Not connected to Synnax cluster");
      if (task == null) throw new Error("Task is not defined");
      if (isRunning == null) throw new Error("Task state is not defined");
      setDesiredState(isRunning ? "paused" : "running");
      await task?.executeCommand(isRunning ? "stop" : "start");
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
              <CopyButtons
                importClass="DigitalWriteTask"
                taskKey={task?.key}
                getName={() => methods.get<string>("name").value}
                getConfig={() => methods.get<DigitalWriteConfig>("config").value}
              />
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
          layoutKey={layoutKey}
          state={taskState}
          startingOrStopping={
            toggleMutation.isPending ||
            (!checkDesiredStateMatch(desiredState, isRunning) &&
              taskState?.variant === "success")
          }
          snapshot={task?.snapshot}
          configuring={configureMutation.isPending}
          onStartStop={toggleMutation.mutate}
          onConfigure={configureMutation.mutate}
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
  const device = use(formCtx) as Device | undefined;
  const place = Layout.usePlacer();
  if (device == null)
    return (
      <Align.Space grow empty align="center" justify="center">
        <Text.Text level="p">No device selected</Text.Text>
      </Align.Space>
    );
  const handleConfigure = () => place({ ...CONFIGURE_LAYOUT, key: device.key });
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
  device: Device;
}

const ChannelList = ({ path, snapshot, device }: ChannelListProps): ReactElement => {
  const [selected, setSelected] = useState<string[]>([]);
  const { value, push, remove } = Form.useFieldArray<DOChannel>({
    path,
    updateOnChildren: true,
  });
  const handleAdd = useCallback((): void => {
    const availableLine = Math.max(0, ...value.map((v) => v.line)) + 1;
    const zeroDigitalWriteChannel = {
      ...ZERO_DO_CHANNEL,
      key: id.id(),
      line: availableLine,
      port: 0,
    };
    setSelected([zeroDigitalWriteChannel.key]);
    const existingCommandStatePair =
      device.properties.digitalOutput.channels[generateKey(zeroDigitalWriteChannel)];
    push({
      ...zeroDigitalWriteChannel,
      stateChannel: existingCommandStatePair?.state ?? 0,
      cmdChannel: existingCommandStatePair?.command ?? 0,
    });
  }, [value, device, push]);
  const menuProps = Menu.useContextMenu();
  return (
    <Align.Space grow empty direction="y">
      <ChannelListHeader onAdd={handleAdd} snapshot={snapshot} />
      <Align.Space grow empty style={{ height: "100%" }}>
        <Menu.ContextMenu
          menu={({ keys }): ReactElement => (
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
          <List.List<string, DOChannel>
            data={value}
            emptyContent={
              <ChannelListEmptyContent onAdd={handleAdd} snapshot={snapshot} />
            }
          >
            <List.Selector<string, DOChannel>
              value={selected}
              allowMultiple
              onChange={setSelected}
              replaceOnSingle
            >
              <List.Core<string, DOChannel>
                grow
                style={{ height: "calc(100% - 6rem)" }}
              >
                {({ key, entry, ...props }) => (
                  <ChannelListItem
                    key={key}
                    {...props}
                    entry={{ ...entry }}
                    path={`${path}.${props.index}`}
                    snapshot={snapshot}
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

interface ChannelListItemProps extends List.ItemProps<string, DOChannel> {
  path: string;
  snapshot?: boolean;
  device: Device;
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
  const cmdChannelName = Channel.useName(
    entry?.cmdChannel ?? 0,
    NO_COMMAND_CHANNEL_NAME,
  );
  const stateChannelName = Channel.useName(
    entry?.stateChannel ?? 0,
    NO_STATE_CHANNEL_NAME,
  );

  return (
    <List.ItemFrame
      {...props}
      entry={entry}
      style={{ width: "100%" }}
      justify="spaceBetween"
      align="center"
      direction="x"
    >
      <Align.Space direction="x" align="center" justify="spaceEvenly">
        <Align.Pack
          className="port-line-input"
          direction="x"
          align="center"
          style={{ maxWidth: "50rem" }}
        >
          <Form.NumericField
            path={`${path}.port`}
            showLabel={false}
            showHelpText={false}
            inputProps={{ showDragHandle: false }}
            hideIfNull
          />
          <Text.Text level="p">/</Text.Text>
          <Form.NumericField
            path={`${path}.line`}
            showHelpText={false}
            showLabel={false}
            inputProps={{ showDragHandle: false }}
            hideIfNull
          />
        </Align.Pack>
        <Text.Text
          level="small"
          className={CSS.BE("port-line-input", "label")}
          shade={7}
          weight={450}
        >
          Port/Line
        </Text.Text>
      </Align.Space>
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

export const ConfigureDigitalWrite = wrapTaskLayout(
  Wrapped,
  ZERO_DIGITAL_WRITE_PAYLOAD,
);
