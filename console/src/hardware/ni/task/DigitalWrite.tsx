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
import { Channel, Form, Header, List, Menu, Status, Synnax } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Input } from "@synnaxlabs/pluto/input";
import { Text } from "@synnaxlabs/pluto/text";
import { deep, id, primitiveIsZero } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement, useCallback, useState } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { Properties } from "@/hardware/ni/device/types";
import { CopyButtons, SelectDevice } from "@/hardware/ni/task/common";
import {
  Chan,
  DIGITAL_WRITE_TYPE,
  DigitalWrite,
  DigitalWriteConfig,
  digitalWriteConfigZ,
  DigitalWritePayload,
  DigitalWriteStateDetails,
  DigitalWriteType,
  DOChan,
  ZERO_DIGITAL_WRITE_PAYLOAD,
  ZERO_DO_CHAN,
} from "@/hardware/ni/task/migrations";
import {
  ChannelListContextMenu,
  ChannelListEmptyContent,
  ChannelListHeader,
  Controls,
  EnableDisableButton,
  TaskLayoutArgs,
  useCreate,
  useObserveState,
  WrappedTaskLayoutProps,
  wrapTaskLayout,
} from "@/hardware/task/common/common";
import { Layout } from "@/layout";

export const configureDigitalWriteLayout = (
  args: TaskLayoutArgs<DigitalWritePayload> = { create: true },
): Layout.State<TaskLayoutArgs<DigitalWritePayload>> => ({
  name: "Configure NI Digital Write Task",
  key: id.id(),
  icon: "Logo.NI",
  type: DIGITAL_WRITE_TYPE,
  windowKey: DIGITAL_WRITE_TYPE,
  location: "mosaic",
  args,
});

export const DIGITAL_WRITE_SELECTABLE: Layout.Selectable = {
  key: DIGITAL_WRITE_TYPE,
  title: "NI Digital Write Task",
  icon: <Icon.Logo.NI />,
  create: (layoutKey) => ({
    ...configureDigitalWriteLayout({ create: true }),
    key: layoutKey,
  }),
};

const Wrapped = ({
  task,
  initialValues,
  layoutKey,
}: WrappedTaskLayoutProps<DigitalWrite, DigitalWritePayload>): ReactElement => {
  const client = Synnax.use();
  const methods = Form.use({
    values: initialValues,
    schema: z.object({
      name: z.string(),
      config: digitalWriteConfigZ,
    }),
  });

  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);
  const [selectedChannelIndex, setSelectedChannelIndex] = useState<number | null>(null);

  const taskState = useObserveState<DigitalWriteStateDetails>(
    methods.setStatus,
    methods.clearStatuses,
    task?.key,
    task?.state,
  );

  const createTask = useCreate<
    DigitalWriteConfig,
    DigitalWriteStateDetails,
    DigitalWriteType
  >(layoutKey);

  const addStatus = Status.useAggregator();

  const configure = useMutation<void, Error, void>({
    mutationKey: [client?.key, "configure"],
    onError: ({ message }) =>
      addStatus({
        variant: "error",
        message,
      }),
    mutationFn: async () => {
      if (!(await methods.validateAsync()) || client == null) return;
      const { name, config } = methods.value();

      const dev = await client.hardware.devices.retrieve<Properties>(config.device);

      let modified = false;
      let shouldCreateStateIndex = primitiveIsZero(
        dev.properties.digitalOutput.stateIndex,
      );
      if (!shouldCreateStateIndex) {
        try {
          await client.channels.retrieve(dev.properties.digitalOutput.stateIndex);
        } catch (e) {
          if (NotFoundError.matches(e)) shouldCreateStateIndex = true;
          else throw e;
        }
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

      const commandsToCreate: DOChan[] = [];
      const statesToCreate: DOChan[] = [];
      for (const channel of config.channels) {
        const key = `${channel.port}l${channel.line}`;
        const exPair = dev.properties.digitalOutput.channels[key];
        if (exPair == null) {
          commandsToCreate.push(channel);
          statesToCreate.push(channel);
        } else {
          try {
            await client.channels.retrieve([exPair.state]);
          } catch (e) {
            if (NotFoundError.matches(e)) statesToCreate.push(channel);
            else throw e;
          }
          try {
            await client.channels.retrieve([exPair.command]);
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
          const key = `${statesToCreate[i].port}l${statesToCreate[i].line}`;
          if (!(key in dev.properties.digitalOutput.channels)) {
            dev.properties.digitalOutput.channels[key] = { state: s.key, command: 0 };
          } else dev.properties.digitalOutput.channels[key].state = s.key;
        });
      }

      if (commandsToCreate.length > 0) {
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
          if (!(key in dev.properties.digitalOutput.channels)) {
            dev.properties.digitalOutput.channels[key] = { state: 0, command: s.key };
          } else dev.properties.digitalOutput.channels[key].command = s.key;
        });
      }

      if (modified)
        await client.hardware.devices.create({
          ...dev,
          properties: dev.properties,
        });

      config.channels = config.channels.map((c) => {
        const key = `${c.port}l${c.line}`;
        const pair = dev.properties.digitalOutput.channels[key];
        return {
          ...c,
          cmdChannel: pair.command,
          stateChannel: pair.state,
        };
      });
      methods.set("config", config);

      await createTask({
        key: task?.key,
        name,
        type: DIGITAL_WRITE_TYPE,
        config,
      });
    },
  });

  const start = useMutation({
    mutationKey: [client?.key, "start"],
    mutationFn: async () => {
      if (client == null) return;
      await task?.executeCommand(
        taskState?.details?.running === true ? "stop" : "start",
      );
    },
  });

  return (
    <Align.Space className={CSS.B("task-configure")} direction="y" grow empty>
      <Align.Space grow>
        <Form.Form {...methods} mode={task?.snapshot ? "preview" : "normal"}>
          <Align.Space direction="x" justify="spaceBetween">
            <Form.Field<string> path="name">
              {(p) => <Input.Text variant="natural" level="h1" {...p} />}
            </Form.Field>
            <CopyButtons
              importClass="DigitalWriteTask"
              taskKey={task?.key}
              getName={() => methods.get<string>("name").value}
              getConfig={() => methods.get<DigitalWriteConfig>("config").value}
            />
          </Align.Space>
          <Align.Space direction="x" className={CSS.B("task-properties")}>
            <SelectDevice />
            <Align.Space direction="x">
              <Form.Field<number>
                label="State Update Rate"
                path="config.stateRate"
                grow
              >
                {(p) => <Input.Numeric {...p} />}
              </Form.Field>
              <Form.SwitchField label="State Data Saving" path="config.dataSaving" />
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
            <ChannelList
              path="config.channels"
              selected={selectedChannels}
              snapshot={task?.snapshot}
              onSelect={useCallback(
                (v, i) => {
                  setSelectedChannels(v);
                  setSelectedChannelIndex(i);
                },
                [setSelectedChannels, setSelectedChannelIndex],
              )}
            />
            <Align.Space className={CSS.B("channel-form")} direction="y" grow>
              <Header.Header level="h4">
                <Header.Title weight={500}>Details</Header.Title>
              </Header.Header>
              <Align.Space className={CSS.B("details")}>
                {selectedChannelIndex != null && (
                  <ChannelForm selectedChannelIndex={selectedChannelIndex} />
                )}
              </Align.Space>
            </Align.Space>
          </Align.Space>
        </Form.Form>
        <Controls
          state={taskState}
          startingOrStopping={start.isPending}
          snapshot={task?.snapshot}
          configuring={configure.isPending}
          onStartStop={start.mutate}
          onConfigure={configure.mutate}
        />
      </Align.Space>
    </Align.Space>
  );
};

interface ChannelFormProps {
  selectedChannelIndex: number;
}

const ChannelForm = ({ selectedChannelIndex }: ChannelFormProps): ReactElement => {
  if (selectedChannelIndex == -1) return <></>;
  const prefix = `config.channels.${selectedChannelIndex}`;
  return (
    <Align.Space direction="y" className={CSS.B("channel-form-content")} empty>
      <Form.NumericField path={`${prefix}.port`} label="Port" grow />
      <Form.NumericField path={`${prefix}.line`} label="Line" grow />
    </Align.Space>
  );
};

interface ChannelListProps {
  path: string;
  onSelect: (keys: string[], index: number) => void;
  selected: string[];
  snapshot?: boolean;
}

const ChannelList = ({
  path,
  snapshot,
  selected,
  onSelect,
}: ChannelListProps): ReactElement => {
  const { value, push, remove } = Form.useFieldArray<DOChan>({ path });
  const handleAdd = (): void => {
    const availableLine = Math.max(0, ...value.map((v) => v.line)) + 1;
    push({
      ...deep.copy(ZERO_DO_CHAN),
      port: 0,
      line: availableLine,
      key: id.id(),
    });
  };
  const menuProps = Menu.useContextMenu();
  return (
    <Align.Space className={CSS.B("channels")} grow empty>
      <ChannelListHeader onAdd={handleAdd} />
      <Menu.ContextMenu
        menu={({ keys }): ReactElement => (
          <ChannelListContextMenu
            path={path}
            keys={keys}
            value={value}
            remove={remove}
            onSelect={onSelect}
            onDuplicate={(indices): void => {
              push(
                indices.map((i) => ({
                  ...deep.copy(value[i]),
                  stateChannel: 0,
                  cmdChannel: 0,
                  key: id.id(),
                })),
              );
            }}
          />
        )}
        {...menuProps}
      >
        <List.List<string, Chan>
          data={value}
          emptyContent={<ChannelListEmptyContent onAdd={handleAdd} />}
        >
          <List.Selector<string, Chan>
            value={selected}
            allowNone={false}
            allowMultiple
            onChange={(keys, { clickedIndex }) =>
              clickedIndex != null && onSelect(keys, clickedIndex)
            }
            replaceOnSingle
          >
            <List.Core<string, Chan> grow>
              {(props) => (
                <ChannelListItem {...props} path={path} snapshot={snapshot} />
              )}
            </List.Core>
          </List.Selector>
        </List.List>
      </Menu.ContextMenu>
    </Align.Space>
  );
};

const ChannelListItem = ({
  path,
  snapshot = false,
  ...props
}: List.ItemProps<string, Chan> & {
  path: string;
  snapshot?: boolean;
}): ReactElement => {
  const { entry } = props;
  const hasLine = "line" in entry;
  const ctx = Form.useContext();
  const childValues = Form.useChildFieldValues<DOChan>({
    path: `${path}.${props.index}`,
    optional: true,
  });
  const cmdChannelName = Channel.useName(
    childValues?.cmdChannel ?? 0,
    "No Command Channel",
  );
  const stateChannelName = Channel.useName(
    childValues?.stateChannel ?? 0,
    "No State Channel",
  );

  const cmdChannelValid =
    Form.useField<number>({
      path: `${path}.${props.index}.cmdChannel`,
      optional: true,
    })?.status?.variant === "success";

  const stateChannelValid =
    Form.useField<number>({
      path: `${path}.${props.index}.stateChannel`,
      optional: true,
    })?.status?.variant === "success";

  const portValid =
    Form.useField<number>({
      path: `${path}.${props.index}.port`,
      optional: true,
    })?.status?.variant === "success";
  if (childValues == null) return <></>;

  return (
    <List.ItemFrame
      {...props}
      entry={childValues}
      justify="spaceBetween"
      align="center"
    >
      <Align.Space direction="x" size="small">
        <Text.Text
          level="p"
          weight={500}
          shade={6}
          style={{ width: "4rem" }}
          color={portValid ? undefined : "var(--pluto-error-z)"}
        >
          {childValues.port}
          {hasLine && `/${entry.line}`}
        </Text.Text>
        <Align.Space direction="y">
          <Text.Text
            level="p"
            weight={500}
            shade={9}
            color={(() => {
              if (cmdChannelName === "No Synnax Channel")
                return "var(--pluto-warning-z)";
              else if (cmdChannelValid) return undefined;
              return "var(--pluto-error-z)";
            })()}
          >
            {cmdChannelName}
          </Text.Text>
          <Text.Text
            level="small"
            weight={500}
            shade={6}
            color={(() => {
              if (stateChannelName === "No Synnax Channel")
                return "var(--pluto-warning-z)";
              else if (stateChannelValid) return undefined;
              return "var(--pluto-error-z)";
            })()}
          >
            {stateChannelName}
          </Text.Text>
        </Align.Space>
      </Align.Space>
      <EnableDisableButton
        value={childValues.enabled}
        onChange={(v) => ctx.set(`${path}.${props.index}.enabled`, v)}
        snapshot={snapshot}
      />
    </List.ItemFrame>
  );
};

export const ConfigureDigitalWrite = wrapTaskLayout(
  Wrapped,
  ZERO_DIGITAL_WRITE_PAYLOAD,
);
