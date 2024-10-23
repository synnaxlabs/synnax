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
import {
  Align,
  Channel,
  Form,
  Header,
  Input,
  List,
  Menu,
  Select,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { deep, id, primitiveIsZero } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { ReactElement, useCallback, useState } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { Properties } from "@/hardware/labjack/device/types";
import { SelectDevice } from "@/hardware/labjack/task/common";
import {
  Write,
  WRITE_TYPE,
  WriteChan,
  WritePayload,
  WriteStateDetails,
  WriteTaskConfig,
  writeTaskConfigZ,
  WriteType,
  ZERO_WRITE_CHAN,
  ZERO_WRITE_PAYLOAD,
} from "@/hardware/labjack/task/types";
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

type LayoutArgs = TaskLayoutArgs<WritePayload>;

export const configureWriteLayout = (
  args: LayoutArgs = { create: false },
): Layout.State<TaskLayoutArgs<WritePayload>> => ({
  name: "Configure LabJack Write Task",
  type: WRITE_TYPE,
  key: id.id(),
  icon: "Task",
  windowKey: WRITE_TYPE,
  location: "mosaic",
  args,
});

export const WRITE_SELECTABLE: Layout.Selectable = {
  key: WRITE_TYPE,
  title: "LabJack Write Task",
  icon: <Icon.Task />,
  create: (layoutKey) => ({
    ...configureWriteLayout({ create: true }),
    key: layoutKey,
  }),
};

const Wrapped = ({
  task,
  initialValues,
  layoutKey,
}: WrappedTaskLayoutProps<Write, WritePayload>): ReactElement => {
  const client = Synnax.use();
  const methods = Form.use({
    values: initialValues,
    schema: z.object({
      name: z.string(),
      config: writeTaskConfigZ,
    }),
  });

  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);
  const [selectedChannelIndex, setSelectedChannelIndex] = useState<number | null>(null);

  const taskState = useObserveState<WriteStateDetails>(
    methods.setStatus,
    methods.clearStatuses,
    task?.key,
    task?.state,
  );

  const createTask = useCreate<WriteTaskConfig, WriteStateDetails, WriteType>(
    layoutKey,
  );

  const addStatus = Status.useAggregator();

  const configure = useMutation({
    mutationKey: [client?.key, "configure"],
    onError: (e) => {
      console.error(e);
      addStatus({
        variant: "error",
        message: e.message,
      });
    },
    mutationFn: async () => {
      if (!(await methods.validateAsync()) || client == null) return;
      const { name, config } = methods.value();

      const dev = await client.hardware.devices.retrieve<Properties>(config.deviceKey);

      let modified = false;
      let shouldCreateStateIndex = primitiveIsZero(dev.properties.writeStateIndex);
      if (!shouldCreateStateIndex) {
        try {
          await client.channels.retrieve(dev.properties.writeStateIndex);
        } catch (e) {
          if (NotFoundError.matches(e)) shouldCreateStateIndex = true;
          else throw e;
        }
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

      const commandsToCreate: WriteChan[] = [];
      const statesToCreate: WriteChan[] = [];
      for (const channel of config.channels) {
        const key = `${channel.port}`;
        const exPair = dev.properties[channel.channelType].channels[key];
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
            name: `${dev.properties.identifier}_${c.channelType}_${c.port}_state`,
            index: dev.properties.writeStateIndex,
            dataType: "uint8",
          })),
        );
        states.forEach((s, i) => {
          const statesToCreateC = statesToCreate[i];
          if (
            !(
              statesToCreateC.port in
              dev.properties[statesToCreateC.channelType].channels
            )
          ) {
            dev.properties[statesToCreateC.channelType].channels[statesToCreateC.port] =
              {
                state: s.key,
                command: 0,
              };
          } else
            dev.properties[statesToCreateC.channelType].channels[
              statesToCreateC.port
            ].state = s.key;
        });
      }

      if (commandsToCreate.length > 0) {
        const commandIndexes = await client.channels.create(
          commandsToCreate.map((c) => ({
            name: `${dev.properties.identifier}_${c.channelType}_${c.port}_cmd_time`,
            dataType: "timestamp",
            isIndex: true,
          })),
        );
        const commands = await client.channels.create(
          commandsToCreate.map((c, i) => ({
            name: `${dev.properties.identifier}_${c.channelType}_${c.port}_cmd`,
            index: commandIndexes[i].key,
            dataType: "uint8",
          })),
        );
        commands.forEach((s, i) => {
          const cmdToCreate = commandsToCreate[i];
          if (!(cmdToCreate.port in dev.properties[cmdToCreate.channelType].channels)) {
            dev.properties[cmdToCreate.channelType].channels[cmdToCreate.port] = {
              state: 0,
              command: s.key,
            };
          } else
            dev.properties[cmdToCreate.channelType].channels[cmdToCreate.port].command =
              s.key;
        });
      }

      if (modified)
        await client.hardware.devices.create({
          ...dev,
          properties: dev.properties,
        });

      config.channels = config.channels.map((c) => {
        const pair = dev.properties[c.channelType].channels[c.port];
        return {
          ...c,
          cmdKey: pair.command,
          stateKey: pair.state,
        };
      });
      methods.set("config", config);

      await createTask({
        key: task?.key,
        name,
        type: WRITE_TYPE,
        config,
      });
    },
  });

  const start = useMutation({
    mutationKey: [client?.key],
    mutationFn: async () => {
      if (client == null) return;
      await task?.executeCommand(
        taskState?.details?.running === true ? "stop" : "start",
      );
    },
  });

  return (
    <Align.Space className={CSS.B("task-configure")} direction="y" grow empty>
      <Align.Space>
        <Form.Form {...methods} mode={task?.snapshot ? "preview" : "normal"}>
          <Align.Space direction="x" justify="spaceBetween">
            <Form.Field<string> path="name">
              {(p) => <Input.Text variant="natural" level="h1" {...p} />}
            </Form.Field>
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
              snapshot={task?.snapshot}
              selected={selectedChannels}
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
          snapshot={task?.snapshot}
          startingOrStopping={start.isPending}
          configuring={configure.isPending}
          onConfigure={configure.mutate}
          onStartStop={start.mutate}
        />
      </Align.Space>
    </Align.Space>
  );
};

interface ChannelFormProps {
  selectedChannelIndex: number;
}

const ChannelForm = ({ selectedChannelIndex }: ChannelFormProps): ReactElement => {
  const prefix = `config.channels.${selectedChannelIndex}`; //datatype, location, range, channel type
  return (
    <Align.Space direction="x" grow>
      <Form.NumericField path={`${prefix}.port`} label="Port" grow />
      <Form.Field path={`${prefix}.dataType`} label="Data Type" grow>
        {(p) => <Select.DataType {...p} />}
      </Form.Field>
      <Form.TextField
        path={`${prefix}.channelTypes`}
        label="Channel Type"
        optional
        grow
      />
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
  selected,
  onSelect,
  snapshot,
}: ChannelListProps): ReactElement => {
  const { value, push, remove } = Form.useFieldArray<WriteChan>({ path });
  const handleAdd = (): void => {
    push({
      ...deep.copy(ZERO_WRITE_CHAN),
      key: id.id(),
    });
  };
  const menuProps = Menu.useContextMenu();
  return (
    <Align.Space className={CSS.B("channels")} grow empty>
      <ChannelListHeader onAdd={handleAdd} />
      <Menu.ContextMenu
        menu={({ keys }: Menu.ContextMenuMenuProps) => (
          <ChannelListContextMenu
            path={path}
            keys={keys}
            value={value}
            remove={remove}
            onSelect={onSelect}
            onDuplicate={(indices) => {
              const newChannels = indices.map((i) => ({
                ...value[i],
                key: id.id(),
              }));
              push(newChannels);
            }}
          />
        )}
        {...menuProps}
      >
        <List.List<string, WriteChan>
          data={value}
          emptyContent={<ChannelListEmptyContent onAdd={handleAdd} />}
        >
          <List.Selector<string, WriteChan>
            value={selected}
            allowNone={false}
            allowMultiple
            onChange={(keys, { clickedIndex }) =>
              clickedIndex != null && onSelect(keys, clickedIndex)
            }
            replaceOnSingle
          >
            <List.Core<string, WriteChan> grow>
              {(props) => (
                <ChannelListItem {...props} snapshot={snapshot} path={path} />
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
}: List.ItemProps<string, WriteChan> & {
  path: string;
  snapshot?: boolean;
}): ReactElement => {
  const { entry } = props;
  const ctx = Form.useContext();
  const childValues = Form.useChildFieldValues<WriteChan>({
    path: `${path}.${props.index}`,
    optional: true,
  });
  const cmdChannelName = Channel.useName(childValues?.cmdKey ?? 0, "No Channel");
  const stateChannelName = Channel.useName(childValues?.stateKey ?? 0, "No Channel");

  const stateChannel =
    Form.useField<number>({
      path: `${path}.${props.index}.stateChannel`,
      optional: true,
    })?.status.variant === "success";

  const cmdChannel =
    Form.useField<number>({
      path: `${path}.${props.index}.cmdChannel`,
      optional: true,
    })?.status.variant === "success";

  const locationValid =
    Form.useField<number>({
      path: `${path}.${props.index}.port`,
      optional: true,
    })?.status.variant === "success";

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
          shade={6}
          color={locationValid ? undefined : "var(--pluto-error-z)"}
        >
          {entry.port}
        </Text.Text>
        <Align.Space direction="y">
          <Text.Text
            level="p"
            shade={9}
            color={(() => {
              if (cmdChannelName === "No Channel") return "var(--pluto-warning-m1)";
              else if (cmdChannel) return undefined;
              return "var(--pluto-error-z)";
            })()}
          >
            {cmdChannelName}
          </Text.Text>
          <Text.Text
            level="p"
            shade={9}
            color={(() => {
              if (stateChannelName === "No Channel") return "var(--pluto-warning-m1)";
              else if (stateChannel) return undefined;
              return "var(--pluto-error-z)";
            })()}
          >
            {stateChannelName}
          </Text.Text>
        </Align.Space>
      </Align.Space>
      <EnableDisableButton
        value={childValues.enabled}
        onChange={(v) => ctx?.set(`${path}.${props.index}.enabled`, v)}
        snapshot={snapshot}
      />
    </List.ItemFrame>
  );
};

export const ConfigureWrite = wrapTaskLayout(Wrapped, ZERO_WRITE_PAYLOAD);
