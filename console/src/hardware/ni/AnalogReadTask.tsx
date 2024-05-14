// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState, type ReactElement, useCallback, useRef } from "react";

import {
  Form,
  Select,
  Header,
  Synnax,
  Nav,
  Button,
  useAsyncEffect,
  Status,
  Device,
} from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Input } from "@synnaxlabs/pluto/input";
import { Text } from "@synnaxlabs/pluto/text";

import { CSS } from "@/css";
import {
  AIChan,
  AIChanType,
  AnalogReadTaskConfig,
  analogReadTaskConfigZ,
  AnalogReadTaskState,
  AnalogReadTaskStateDetails,
  type LinearScaleType,
} from "@/hardware/ni/types";

import "@/hardware/ni/AnalogReadTask.css";
import { Layout } from "@/layout";
import { useMutation, useQuery } from "@tanstack/react-query";
import { task } from "@synnaxlabs/client";
import { z } from "zod";
import { Icon } from "@synnaxlabs/media";
import { ANALOG_INPUT_FORMS, SelectChannelTypeField } from "./ChannelForms";
import { Channel } from "@synnaxlabs/pluto";
import { List } from "@synnaxlabs/pluto/list";
import { deep, xy } from "@synnaxlabs/x";
import { nanoid } from "nanoid";

import {
  AI_CHANNEL_TYPE_NAMES,
  CHANNEL_TYPE_DISPLAY,
  ZERO_AI_CHANNELS,
  type NIChannel,
} from "@/hardware/ni/types";

export const analogReadTaskLayout: Layout.LayoutState = {
  name: "Configure NI Analog Read Task",
  key: "niAnalogReadTask",
  type: "niAnalogReadTask",
  windowKey: "niAnalogReadTask",
  location: "window",
  window: {
    resizable: false,
    size: { width: 1200, height: 900 },
    navTop: true,
  },
};

export const AnalogReadTask: Layout.Renderer = ({ layoutKey }) => {
  const client = Synnax.use();
  const fetchTask = useQuery<AnalogReadTaskInternalProps>({
    queryKey: [layoutKey, client?.key],
    queryFn: async () => {
      if (client == null || layoutKey == analogReadTaskLayout.key)
        return {
          initialValues: {
            key: "niAnalogReadTask",
            type: "niAnalogReadTask",
            name: "NI Analog Read Task",
            config: {
              device: "",
              sampleRate: 50,
              streamRate: 25,
              channels: [],
            },
          },
        };
      const t = await client.hardware.tasks.retrieve<
        AnalogReadTaskConfig,
        AnalogReadTaskStateDetails
      >(layoutKey, { includeState: true });
      return { initialValues: t, task: t };
    },
  });
  if (fetchTask.isLoading) return <></>;
  if (fetchTask.isError) return <></>;
  return (
    <AnalogReadTaskInternal {...(fetchTask.data as AnalogReadTaskInternalProps)} />
  );
};

export interface AnalogReadTaskInternalProps {
  task?: task.Task<AnalogReadTaskConfig, AnalogReadTaskStateDetails>;
  initialValues: task.TaskPayload<AnalogReadTaskConfig, AnalogReadTaskStateDetails>;
}

const AnalogReadTaskInternal = ({
  task: pTask,
  initialValues,
}: AnalogReadTaskInternalProps): ReactElement | null => {
  const client = Synnax.use();
  const methods = Form.use({
    values: initialValues,
    schema: z.object({
      name: z.string(),
      config: analogReadTaskConfigZ,
    }),
  });

  const [task, setTask] = useState(pTask);
  const [taskState, setTaskState] = useState<AnalogReadTaskState | null>(
    initialValues?.state ?? null,
  );

  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);
  const [selectedChannelIndex, setSelectedChannelIndex] = useState<number | null>(null);

  const stateObserverRef =
    useRef<task.StateObservable<AnalogReadTaskStateDetails> | null>(null);

  useAsyncEffect(async () => {
    if (client == null || task == null) return;
    stateObserverRef.current =
      await task.openStateObserver<AnalogReadTaskStateDetails>();
    stateObserverRef.current.onChange((s) => {
      setTaskState(s);
    });
    return async () => await stateObserverRef.current?.close().catch(console.error);
  }, [client?.key, task?.key, setTaskState]);

  const configure = useMutation({
    mutationKey: [client?.key, "configure"],
    mutationFn: async () => {
      if (!(await methods.validateAsync()) || client == null) return;
      const rack = await client.hardware.racks.retrieve("sy_node_1_rack");
      const { name, config } = methods.value();
      setTask(
        await rack.createTask<AnalogReadTaskConfig>({
          key: task?.key,
          name,
          type: "niAnalogReader",
          config,
        }),
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
    <Align.Space className={CSS.B("ni-analog-read-task")} direction="y" grow empty>
      <Align.Space className={CSS.B("content")} grow>
        <Form.Form {...methods}>
          <Align.Space direction="x">
            <Form.Field<string> path="name">
              {(p) => <Input.Text variant="natural" level="h1" {...p} />}
            </Form.Field>
          </Align.Space>
          <Align.Space direction="x">
            <Form.Field<string> path="config.device" label="Device" grow>
              {(p) => (
                <Device.SelectSingle
                  allowNone={false}
                  grow
                  {...p}
                  searchOptions={{ makes: ["NI"] }}
                />
              )}
            </Form.Field>
            <Form.Field<number> label="Sample Rate" path="config.sampleRate">
              {(p) => <Input.Numeric {...p} />}
            </Form.Field>
            <Form.Field<number> label="Stream Rate" path="config.streamRate">
              {(p) => <Input.Numeric {...p} />}
            </Form.Field>
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
              <Header.Header level="h3">
                <Header.Title weight={500}>Channel Details</Header.Title>
              </Header.Header>
              <Align.Space className={CSS.B("details")}>
                {selectedChannelIndex != null && (
                  <ChannelForm selectedChannelIndex={selectedChannelIndex} />
                )}
              </Align.Space>
            </Align.Space>
          </Align.Space>
        </Form.Form>
      </Align.Space>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.Start style={{ paddingLeft: "2rem" }}>
          <Text.Text level="p">{JSON.stringify(taskState)}</Text.Text>
        </Nav.Bar.Start>
        <Nav.Bar.End style={{ paddingRight: "2rem" }}>
          <Button.Icon
            loading={start.isPending}
            disabled={start.isPending || taskState == null}
            onClick={() => start.mutate()}
            variant="outlined"
          >
            {taskState?.details?.running === true ? <Icon.Pause /> : <Icon.Play />}
          </Button.Icon>
          <Button.Button
            loading={configure.isPending}
            disabled={configure.isPending}
            onClick={() => configure.mutate()}
          >
            Configure
          </Button.Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Align.Space>
  );
};

interface ChannelFormProps {
  selectedChannelIndex: number;
}

const ChannelForm = ({ selectedChannelIndex }: ChannelFormProps): ReactElement => {
  if (selectedChannelIndex == -1) return <></>;
  const prefix = `config.channels.${selectedChannelIndex}`;
  const ctx = Form.useContext();
  const type = ctx.get<AIChanType>({ path: `${prefix}.type` });
  const TypeForm = ANALOG_INPUT_FORMS[type.value];
  const [counter, setCounter] = useState(0);
  Form.useFieldListener<AIChanType>({
    path: `${prefix}.type`,
    onChange: (v) => setCounter((c) => c + 1),
  });

  return (
    <>
      <Align.Space direction="y" className={CSS.B("channel-form-content")} empty>
        <SelectChannelTypeField path={prefix} inputProps={{ allowNone: false }} />
        <TypeForm prefix={prefix} />
      </Align.Space>
    </>
  );
};

interface ChannelListProps {
  path: string;
  onSelect: (keys: string[], index: number) => void;
  selected: string[];
}

const ChannelList = ({ path, selected, onSelect }: ChannelListProps): ReactElement => {
  const { value, push } = Form.useFieldArray<NIChannel>({ path });
  const handleAdd = (): void => {
    const availablePort = Math.max(0, ...value.map((v) => v.port)) + 1;
    push({
      ...deep.copy(ZERO_AI_CHANNELS["ai_accel"]),
      port: availablePort,
      key: nanoid(),
    });
  };
  return (
    <Align.Space className={CSS.B("channels")} grow empty>
      <Header.Header level="h3">
        <Header.Title weight={500}>Channels</Header.Title>
        <Header.Actions>
          {[
            {
              key: "add",
              onClick: handleAdd,
              children: <Icon.Add />,
              size: "large",
            },
          ]}
        </Header.Actions>
      </Header.Header>
      <List.List<string, NIChannel> data={value}>
        <List.Selector<string, NIChannel>
          value={selected}
          allowNone={false}
          allowMultiple={true}
          onChange={(keys, { clickedIndex }) =>
            clickedIndex != null && onSelect(keys, clickedIndex)
          }
          replaceOnSingle
        >
          <List.Core<string, NIChannel> grow>
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
}: List.ItemProps<string, NIChannel> & {
  path: string;
}): ReactElement => {
  const { entry } = props;
  const hasLine = "line" in entry;
  const ctx = Form.useContext();
  const childValues = Form.useChildFieldValues<AIChan>({
    path: `${path}.${props.index}`,
    optional: true,
  });
  const channelName = Channel.useName(childValues?.channel ?? 0, "No Synnax Channel");
  const channelValid =
    Form.useField<number>({
      path: `${path}.${props.index}.channel`,
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
      <Align.Space direction="y" size="small">
        <Align.Space direction="x">
          <Text.Text
            level="p"
            weight={500}
            shade={6}
            style={{ width: "3rem" }}
            color={portValid ? undefined : "var(--pluto-error-z)"}
          >
            {childValues.port} {hasLine && `/${entry.line}`}
          </Text.Text>
          <Text.Text
            level="p"
            weight={500}
            shade={9}
            color={(() => {
              if (channelName === "No Synnax Channel") return "var(--pluto-warning-z)";
              else if (channelValid) return undefined;
              return "var(--pluto-error-z)";
            })()}
          >
            {channelName}
          </Text.Text>
        </Align.Space>
        <Text.Text level="p" shade={6}>
          {AI_CHANNEL_TYPE_NAMES[childValues.type]}
        </Text.Text>
      </Align.Space>
      <Button.Toggle
        checkedVariant="outlined"
        uncheckedVariant="outlined"
        value={childValues.enabled}
        size="small"
        onClick={(e) => e.stopPropagation()}
        onChange={(v) => {
          console.log("setting enabled", v);
          ctx.set({ path: `${path}.${props.index}.enabled`, value: v });
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
