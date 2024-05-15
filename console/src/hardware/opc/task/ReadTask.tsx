// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactElement,
} from "react";

import { DataType } from "@synnaxlabs/x/telem";
import { device, type task } from "@synnaxlabs/client";
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
  Select,
  Status,
  Synnax,
  Text,
  useAsyncEffect,
  Nav,
  Menu,
} from "@synnaxlabs/pluto";
import { useMutation, useQuery } from "@tanstack/react-query";
import { nanoid } from "nanoid";

import { CSS } from "@/css";
import { SelectNodeRemote } from "@/hardware/opc/SelectNode";
import {
  type ReadTaskChannelConfig,
  readTaskConfigZ,
  type ReadTaskState,
  type ReadTaskStateDetails,
  type ReadTaskConfig,
} from "@/hardware/opc/types";
import { type Layout } from "@/layout";
import { z } from "zod";

import "@/hardware/opc/task/ReadTask.css";

export const readTaskLayout: Layout.LayoutState = {
  name: "Configure OPC UA Read Task",
  key: "readopcTask",
  type: "readopcTask",
  windowKey: "readopcTask",
  location: "window",
  window: {
    resizable: false,
    size: { width: 1200, height: 900 },
    navTop: true,
  },
};

export const ReadTask: Layout.Renderer = ({ layoutKey }) => {
  const client = Synnax.use();
  const fetchTask = useQuery<ReadTaskInternalProps>({
    queryKey: [client?.key, "task", layoutKey],
    queryFn: async () => {
      if (client == null || layoutKey == readTaskLayout.key)
        return {
          initialValues: {
            key: "readopcTask",
            type: "opcReader",
            name: "OPC Read Task",
            config: {
              device: "",
              sampleRate: 50,
              streamRate: 25,
              channels: [],
            },
          },
        };
      const t = await client.hardware.tasks.retrieve<ReadTaskConfig, ReadTaskState>(
        layoutKey,
        { includeState: true },
      );
      return { initialValues: t, task: t };
    },
  });
  if (fetchTask.isLoading) return <></>;
  if (fetchTask.isError) return <></>;
  return <ReadTaskInternal {...fetchTask.data} />;
};

interface ReadTaskInternalProps {
  task?: task.Task<ReadTaskConfig, ReadTaskState>;
  initialValues: task.TaskPayload<ReadTaskConfig, ReadTaskState>;
}

const ReadTaskInternal = ({
  initialValues,
  task: pTask,
}: ReadTaskInternalProps): ReactElement => {
  const client = Synnax.use();
  const [task, setTask] = useState(pTask);
  const [taskState, setTaskState] = useState<ReadTaskState | null>(
    initialValues.state ?? null,
  );
  const [device, setDevice] = useState<device.Device<DeviceProperties> | null>(null);

  const schema = useMemo(
    () =>
      z.object({
        name: z.string(),
        config: readTaskConfigZ.superRefine(async (cfg, ctx) => {
          if (client == null || device == null) return;
          for (let i = 0; i < cfg.channels.length; i++) {
            const { channel, nodeId } = cfg.channels[i];
            if (channel === 0 || nodeId.length === 0) continue;
            const ch = await client.channels.retrieve(channel);
            const node = device.properties.channels.find((c) => c.nodeId === nodeId);
            if (node == null) return;
            const nodeDt = new DataType(node.dataType);
            if (!nodeDt.canCastTo(ch.dataType)) {
              ctx.addIssue({
                code: z.ZodIssueCode.custom,
                path: ["channels", i, "nodeId"],
                message: `Node data type ${node.dataType} cannot be cast to channel data type ${ch.dataType}`,
              });
            } else if (!nodeDt.canSafelyCastTo(ch.dataType)) {
              ctx.addIssue({
                code: z.ZodIssueCode.custom,
                path: ["channels", i, "nodeId"],
                message: `Node data type ${node.dataType} may not be safely cast to channel data type ${ch.dataType}`,
                params: { variant: "warning" },
              });
            }
          }
        }),
      }),
    [client?.key, device?.key],
  );

  const methods = Form.use({
    schema,
    values: initialValues,
  });

  Form.useFieldListener<string, typeof schema>({
    ctx: methods,
    path: "config.device",
    onChange: useCallback(
      (fs) => {
        if (!fs.touched || fs.status.variant !== "success" || client == null) return;
        client.hardware.devices
          .retrieve<DeviceProperties>(fs.value)
          .then((d) => setDevice(d))
          .catch(console.error);
      },
      [client?.key, setDevice],
    ),
  });

  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);
  const [selectedChannelIndex, setSelectedChannelIndex] = useState<number | null>(null);

  const stateObserverRef = useRef<task.StateObservable<ReadTaskStateDetails> | null>(
    null,
  );

  useAsyncEffect(async () => {
    if (client == null || task == null) return;
    stateObserverRef.current = await task.openStateObserver<ReadTaskStateDetails>();
    stateObserverRef.current.onChange((s) => {
      setTaskState(s);
    });
    return async () => await stateObserverRef.current?.close().catch(console.error);
  }, [client?.key, task?.key, setTaskState]);

  useEffect(() => {
    console.log(taskState);
  }, [taskState]);

  const configure = useMutation({
    mutationKey: [client?.key],
    mutationFn: async () => {
      if (!(await methods.validateAsync()) || client == null) return;
      const rack = await client.hardware.racks.retrieve("sy_node_1_rack");
      setTask(
        await rack.createTask<ReadTaskConfig>({
          key: task?.key,
          name: methods.value().name,
          type: "opcReader",
          config: methods.value().config,
        }),
      );
    },
  });

  const start = useMutation({
    mutationKey: [client?.key, "start"],
    mutationFn: async () => {
      if (task == null) return;
      await task.executeCommand(taskState?.details?.running == true ? "stop" : "start");
    },
  });

  return (
    <Align.Space className={CSS.B("opc-read-task")} direction="y" grow empty>
      <Align.Space className={CSS.B("content")} direction="y" grow>
        <Form.Form {...methods}>
          <Align.Space direction="x">
            <Form.Field<string> path="name" label="Name">
              {(p) => <Input.Text variant="natural" level="h1" {...p} />}
            </Form.Field>
          </Align.Space>
          <Align.Space direction="x">
            <Form.Field<string> path="config.device" label="Device" grow>
              {(p) => (
                <Device.SelectSingle
                  {...p}
                  allowNone={false}
                  searchOptions={{ makes: ["opc"] }}
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
            className={CSS.B("channel-form-container")}
            direction="x"
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
                  if (v.length > 0) setSelectedChannelIndex(i);
                  else setSelectedChannelIndex(null);
                  setSelectedChannels(v);
                },
                [setSelectedChannels, setSelectedChannelIndex],
              )}
            />
            <Align.Space className={CSS.B("channel-form")} direction="y" grow>
              <Header.Header level="h3">
                <Header.Title weight={500}>Channel Details</Header.Title>
              </Header.Header>
              <Align.Space direction="y" className={CSS.B("channel-form-content")} grow>
                {selectedChannelIndex != null && (
                  <ChannelForm selectedChannelIndex={selectedChannelIndex} />
                )}
              </Align.Space>
            </Align.Space>
          </Align.Space>
        </Form.Form>
      </Align.Space>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.End style={{ paddingRight: "2rem" }}>
          <Button.ToggleIcon
            loading={start.isPending}
            disabled={start.isPending || taskState == null}
            value={taskState?.details.running ?? false}
            onChange={() => start.mutate()}
          >
            {taskState?.details.running ? <Icon.Pause /> : <Icon.Play />}
          </Button.ToggleIcon>
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

export interface ChannelListProps {
  path: string;
  onSelect: (keys: string[], index: number) => void;
  selected: string[];
}

export const ChannelList = ({
  path,
  selected,
  onSelect,
}: ChannelListProps): ReactElement => {
  const { value, push, remove } = Form.useFieldArray<ReadTaskChannelConfig>({ path });

  const menuProps = Menu.useContextMenu();

  const handleAdd = (): void => {
    push({
      key: nanoid(),
      channel: 0,
      nodeId: "",
      enabled: true,
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
            },
          ]}
        </Header.Actions>
      </Header.Header>
      <Menu.ContextMenu
        menu={({ keys }: Menu.ContextMenuMenuProps): ReactElement => {
          const handleSelect = (key: string): void => {
            switch (key) {
              case "remove":
                const indices = keys
                  .map((k) => value.findIndex((v) => v.key === k))
                  .filter((i) => i >= 0);
                remove(indices);
                onSelect([], 0);
                break;
            }
          };

          return (
            <Menu.Menu onChange={handleSelect} level="small">
              <Menu.Item startIcon={<Icon.Close />} itemKey="remove">
                Remove
              </Menu.Item>
            </Menu.Menu>
          );
        }}
        {...menuProps}
      >
        <List.List<string, ReadTaskChannelConfig> data={value}>
          <List.Selector<string, ReadTaskChannelConfig>
            value={selected}
            allowNone={false}
            allowMultiple={true}
            onChange={(keys, { clickedIndex }) =>
              clickedIndex != null && onSelect(keys, clickedIndex)
            }
            replaceOnSingle
          >
            <List.Core<string, ReadTaskChannelConfig> grow>
              {(props) => (
                <ChannelListItem
                  {...props}
                  path={path}
                  remove={() => {
                    const indices = selected
                      .map((k) => value.findIndex((v) => v.key === k))
                      .filter((i) => i >= 0);
                    remove(indices);
                    onSelect([], 0);
                  }}
                />
              )}
            </List.Core>
          </List.Selector>
        </List.List>
      </Menu.ContextMenu>
    </Align.Space>
  );
};

export const ChannelListItem = ({
  path,
  ...props
}: List.ItemProps<string, ReadTaskChannelConfig> & {
  path: string;
  remove?: () => void;
}): ReactElement => {
  const { entry } = props;
  const ctx = Form.useContext();
  const childValues = Form.useChildFieldValues<ReadTaskChannelConfig>({
    path: `${path}.${props.index}`,
    optional: true,
  });
  if (childValues == null) return <></>;
  const channelName = Channel.useName(entry.channel);
  return (
    <List.ItemFrame
      {...props}
      entry={childValues}
      justify="spaceBetween"
      align="center"
      onKeyDown={(e) => {
        if (["Delete", "Backspace"].includes(e.key)) props.remove?.();
      }}
    >
      <Align.Space direction="y" size="small">
        <Align.Space direction="x">
          <Text.Text level="p" weight={500} shade={9}>
            {channelName}
          </Text.Text>
        </Align.Space>
      </Align.Space>
      <Button.Toggle
        checkedVariant="outlined"
        uncheckedVariant="outlined"
        value={entry.enabled}
        size="small"
        onClick={(e) => e.stopPropagation()}
        onChange={(v) => {
          ctx.set({ path: `${path}.${props.index}.enabled`, value: v });
        }}
        tooltip={
          <Text.Text level="small" style={{ maxWidth: 300 }}>
            Data acquisition for this channel is{" "}
            {entry.enabled ? "enabled" : "disabled"}. Click to
            {entry.enabled ? " disable" : " enable"} it.
          </Text.Text>
        }
      >
        <Status.Text
          variant={entry.enabled ? "success" : "disabled"}
          level="small"
          align="center"
        >
          {entry.enabled ? "Enabled" : "Disabled"}
        </Status.Text>
      </Button.Toggle>
    </List.ItemFrame>
  );
};

interface ChannelFormProps {
  selectedChannelIndex: number;
}

const ChannelForm = ({ selectedChannelIndex }: ChannelFormProps): ReactElement => {
  const prefix = `config.channels.${selectedChannelIndex}`;
  const dev = Form.useField<string>({ path: "config.device" }).value;
  return (
    <>
      <Form.Field<number> path={`${prefix}.channel`} label="Synnax Channel" hideIfNull>
        {(p) => <Channel.SelectSingle allowNone={false} {...p} />}
      </Form.Field>
      <Form.Field<string> path={`${prefix}.nodeId`} label="OPC Node" hideIfNull>
        {(p) => <SelectNodeRemote allowNone={false} device={dev} {...p} />}
      </Form.Field>
    </>
  );
};
