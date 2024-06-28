// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { QueryError } from "@synnaxlabs/client";
import {
  Button,
  Channel,
  Device,
  Form,
  Header,
  List,
  Observe,
  Status,
  Synnax,
} from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Input } from "@synnaxlabs/pluto/input";
import { Text } from "@synnaxlabs/pluto/text";
import { deep, primitiveIsZero } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { nanoid } from "nanoid";
import { type ReactElement, useCallback, useState } from "react";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { CSS } from "@/css";
import { Properties } from "@/hardware/ni/device/types";
import {
  ChannelListEmptyContent,
  ChannelListHeader,
  Controls,
} from "@/hardware/task/common/common";
import {
  AnalogReadStateDetails,
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
} from "@/hardware/ni/task/types";
import { wrapTaskLayout } from "@/hardware/task/TaskWrapper";
import { Layout } from "@/layout";
import { setAltKey, setArgs } from "@/layout/slice";

interface ConfigureDigitalWriteArgs {
  create: boolean;
}

export const configureDigitalWriteLayout = (
  create: boolean = false,
): Layout.State<ConfigureDigitalWriteArgs> => ({
  name: "Configure NI Digital Write Task",
  key: nanoid(),
  type: DIGITAL_WRITE_TYPE,
  windowKey: DIGITAL_WRITE_TYPE,
  location: "mosaic",
  args: { create },
});

interface InternalProps {
  layoutKey: string;
  task?: DigitalWrite;
  initialValues: DigitalWritePayload;
}

const Internal = ({
  task: initialTask,
  initialValues,
  layoutKey,
}: InternalProps): ReactElement => {
  const client = Synnax.use();
  const methods = Form.use({
    values: initialValues,
    schema: z.object({
      name: z.string(),
      config: digitalWriteConfigZ,
    }),
  });

  const dispatch = useDispatch();

  const [task, setTask] = useState(initialTask);
  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);
  const [selectedChannelIndex, setSelectedChannelIndex] = useState<number | null>(null);

  const taskState = Observe.useState({
    key: [task?.key],
    open: async () => await task?.openStateObserver<AnalogReadStateDetails>(),
    initialValue: task?.state,
  });

  const configure = useMutation<void, Error, void>({
    mutationKey: [client?.key, "configure"],
    onError: console.log,
    mutationFn: async () => {
      if (!(await methods.validateAsync()) || client == null) return;
      const rack = await client.hardware.racks.retrieve("sy_node_1_rack");
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
          if (e instanceof QueryError) shouldCreateStateIndex = true;
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
        console.log(exPair, key, dev.properties.digitalOutput.channels);
        if (exPair == null) {
          commandsToCreate.push(channel);
          statesToCreate.push(channel);
        } else {
          try {
            await client.channels.retrieve([exPair.state]);
          } catch (e) {
            if (e instanceof QueryError) statesToCreate.push(channel);
            else throw e;
          }
          try {
            await client.channels.retrieve([exPair.command]);
          } catch (e) {
            if (e instanceof QueryError) commandsToCreate.push(channel);
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
            dataType: "boolean",
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
            dataType: "boolean",
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

      const t = await rack.createTask<
        DigitalWriteConfig,
        DigitalWriteStateDetails,
        DigitalWriteType
      >({
        key: task?.key,
        name,
        type: DIGITAL_WRITE_TYPE,
        config,
      });
      setTask(t);
      dispatch(setAltKey({ key: layoutKey, altKey: t.key }));
      dispatch(
        setArgs<ConfigureDigitalWriteArgs>({ key: layoutKey, args: { create: false } }),
      );
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
        <Form.Form {...methods}>
          <Form.Field<string> path="name">
            {(p) => <Input.Text variant="natural" level="h1" {...p} />}
          </Form.Field>
          <Align.Space direction="x">
            <Form.Field<string>
              path="config.device"
              label="Device"
              onChange={console.log}
              grow
            >
              {(p) => (
                <Device.SelectSingle
                  allowNone={false}
                  grow
                  {...p}
                  searchOptions={{ makes: ["NI"] }}
                />
              )}
            </Form.Field>
            <Form.Field<number> label="State Update Rate" path="config.stateRate">
              {(p) => <Input.Numeric {...p} />}
            </Form.Field>
            <Form.SwitchField label="State Data Saving" path="config.dataSaving" />
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
}

const ChannelList = ({ path, selected, onSelect }: ChannelListProps): ReactElement => {
  const { value, push } = Form.useFieldArray<DOChan>({ path });
  const handleAdd = (): void => {
    const availableLine = Math.max(0, ...value.map((v) => v.line)) + 1;
    push({
      ...deep.copy(ZERO_DO_CHAN),
      port: 0,
      line: availableLine,
      key: nanoid(),
    });
  };
  return (
    <Align.Space className={CSS.B("channels")} grow empty>
      <ChannelListHeader onAdd={handleAdd} />
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
            {(props) => <ChannelListItem {...props} path={path} />}
          </List.Core>
        </List.Selector>
      </List.List>
    </Align.Space>
  );
};

const ChannelListItem = ({
  path,
  ...props
}: List.ItemProps<string, Chan> & {
  path: string;
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
    }).status.variant === "success";

  const stateChannelValid =
    Form.useField<number>({
      path: `${path}.${props.index}.stateChannel`,
    }).status.variant === "success";

  const portValid =
    Form.useField<number>({
      path: `${path}.${props.index}.port`,
    }).status.variant === "success";
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
      <Button.Toggle
        checkedVariant="outlined"
        uncheckedVariant="outlined"
        value={childValues.enabled}
        size="small"
        onClick={(e) => e.stopPropagation()}
        onChange={(v) => {
          ctx.set(`${path}.${props.index}.enabled`, v);
        }}
        tooltip={
          <Text.Text level="small" style={{ maxWidth: 300 }}>
            Data acquisition for this channel is{" "}
            {childValues.enabled ? "enabled" : "disabled"}. Click to
            {childValues.enabled ? " disable" : " enable"} it.
          </Text.Text>
        }
      >
        <Status.Text
          variant={childValues.enabled ? "success" : "disabled"}
          level="small"
          align="center"
        >
          {childValues.enabled ? "Enabled" : "Disabled"}
        </Status.Text>
      </Button.Toggle>
    </List.ItemFrame>
  );
};

export const ConfigureDigitalWrite = wrapTaskLayout<DigitalWrite, DigitalWritePayload>(
  Internal,
  ZERO_DIGITAL_WRITE_PAYLOAD,
);
