// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Channel,
  Device,
  Form,
  Header,
  Input,
  List,
  Nav,
  Status,
  Synnax,
  Text,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import { nanoid } from "nanoid";
import { ReactElement, useCallback, useRef, useState } from "react";
import { v4 as uuid } from "uuid";
import { z } from "zod";

import { CSS } from "@/css";
import {
  AnalogReadStateDetails,
  Chan,
  DIChan,
  DIGITAL_READ_TYPE,
  DigitalRead,
  DigitalReadConfig,
  digitalReadConfigZ,
  DigitalReadPayload,
  DigitalReadStateDetails,
  DigitalReadType,
  ZERO_DI_CHAN,
  ZERO_DIGITAL_READ_PAYLOAD,
} from "@/hardware/ni/task/types";
import { Layout } from "@/layout";

export const configureDigitalReadLayout = (): Layout.State => ({
  name: "Configure NI Digital Read Task",
  type: DIGITAL_READ_TYPE,
  key: uuid(),
  windowKey: DIGITAL_READ_TYPE,
  location: "window",
  window: {
    resizable: false,
    size: { width: 1200, height: 900 },
    navTop: true,
  },
});

export const ConfigureDigitalRead: Layout.Renderer = ({ layoutKey }) => {
  const client = Synnax.use();
  const fetchTask = useQuery<InternalProps>({
    queryKey: [layoutKey, client?.key],
    queryFn: async () => {
      if (client == null || layoutKey == configureDigitalReadLayout.key)
        return { initialValues: deep.copy(ZERO_DIGITAL_READ_PAYLOAD) };
      const t = await client.hardware.tasks.retrieve<
        DigitalReadConfig,
        DigitalReadStateDetails,
        DigitalReadType
      >(layoutKey, { includeState: true });
      return { initialValues: t, task: t };
    },
  });
  if (fetchTask.isLoading) return <></>;
  if (fetchTask.isError) return <></>;
  return <Internal {...(fetchTask.data as InternalProps)} />;
};

interface InternalProps {
  task?: DigitalRead;
  initialValues: DigitalReadPayload;
}

const Internal = ({ task: pTask, initialValues }: InternalProps): ReactElement => {
  const client = Synnax.use();
  const methods = Form.use({
    values: initialValues,
    schema: z.object({
      name: z.string(),
      config: digitalReadConfigZ,
    }),
  });

  const [task, setTask] = useState(pTask);
  const [taskState, setTaskState] = useState<task.State<AnalogReadStateDetails> | null>(
    initialValues?.state ?? null,
  );
  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);
  const [selectedChannelIndex, setSelectedChannelIndex] = useState<number | null>(null);
  const stateObserverRef = useRef<task.StateObservable<DigitalReadStateDetails> | null>(
    null,
  );
  useAsyncEffect(async () => {
    if (client == null || task == null) return;
    stateObserverRef.current = await task.openStateObserver<DigitalReadStateDetails>();
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
      const t = await rack.createTask<
        DigitalReadConfig,
        DigitalReadStateDetails,
        DigitalReadType
      >({
        key: task?.key,
        name,
        type: DIGITAL_READ_TYPE,
        config,
      });
      setTask(t);
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
            <Form.NumericField label="Sample Rate" path="config.sampleRate" />
            <Form.NumericField label="Stream Rate" path="config.streamRate" />
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
  return (
    <Align.Space direction="y" className={CSS.B("channel-form-content")} empty>
      <Align.Space direction="x" grow>
        <Form.NumericField path={`${prefix}.port`} label="Port" grow />
        <Form.NumericField path={`${prefix}.line`} label="Line" grow />
      </Align.Space>
    </Align.Space>
  );
};

interface ChannelListProps {
  path: string;
  onSelect: (keys: string[], index: number) => void;
  selected: string[];
}

const ChannelList = ({ path, selected, onSelect }: ChannelListProps): ReactElement => {
  const { value, push } = Form.useFieldArray<DIChan>({ path });
  const handleAdd = (): void => {
    const availableLine = Math.max(0, ...value.map((v) => v.line)) + 1;
    push({
      ...deep.copy(ZERO_DI_CHAN),
      port: 0,
      line: availableLine,
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
      <List.List<string, Chan> data={value}>
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
  const childValues = Form.useChildFieldValues<DIChan>({
    path: `${path}.${props.index}`,
    optional: true,
  });
  const channelName = Channel.useName(childValues?.channel ?? 0, "No Channel");

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
              if (channelName === "No Channel") return "var(--pluto-warning-z)";
              else if (channelValid) return undefined;
              return "var(--pluto-error-z)";
            })()}
          >
            {channelName}
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
