// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/opc/task/ReadTask.css";

import { device, NotFoundError } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Device as PDevice,
  Form,
  Haul,
  Header,
  Input,
  List,
  Menu,
  Status,
  Synnax,
  Text,
  useAsyncEffect,
  useSyncedRef,
} from "@synnaxlabs/pluto";
import { caseconv, primitiveIsZero } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement, useCallback, useState } from "react";
import { v4 as uuid } from "uuid";
import { z } from "zod";

import { CSS } from "@/css";
import { Device } from "@/hardware/opc/device";
import { Browser } from "@/hardware/opc/device/Browser";
import { createConfigureLayout } from "@/hardware/opc/device/Configure";
import {
  Write,
  WRITE_TYPE,
  type WriteChannelConfig,
  type WriteConfig,
  writeConfigZ,
  WritePayload,
  type WriteStateDetails,
  WriteType,
  ZERO_WRITE_PAYLOAD,
} from "@/hardware/opc/task/types";
import {
  ChannelListContextMenu,
  Controls,
  EnableDisableButton,
  TaskLayoutArgs,
  useCreate,
  useObserveState,
  WrappedTaskLayoutProps,
  wrapTaskLayout,
} from "@/hardware/task/common/common";
import { Layout } from "@/layout";

export const configureWriteLayout = (
  args: TaskLayoutArgs<WritePayload> = { create: false },
): Layout.State => ({
  name: "Configure OPC UA Write Task",
  key: uuid(),
  type: WRITE_TYPE,
  windowKey: WRITE_TYPE,
  location: "mosaic",
  icon: "Logo.OPC",
  window: {
    resizable: true,
    size: { width: 1200, height: 900 },
    navTop: true,
  },
  args,
});

export const WRITE_SELECTABLE: Layout.Selectable = {
  key: WRITE_TYPE,
  title: "OPC UA Write Task",
  icon: <Icon.Logo.OPC />,
  create: (layoutKey) => ({
    ...configureWriteLayout({ create: true }),
    key: layoutKey,
  }),
};

const schema = z.object({
  name: z.string(),
  config: writeConfigZ,
});

const getChannelByNodeID = (props: Device.Properties, nodeId: string) =>
  props.write.channels[nodeId] ?? props.write.channels[caseconv.snakeToCamel(nodeId)];

const Wrapped = ({
  layoutKey,
  initialValues,
  task,
}: WrappedTaskLayoutProps<Write, WritePayload>): ReactElement => {
  const client = Synnax.use();
  const addStatus = Status.useAggregator();
  const [device, setDevice] = useState<device.Device<Device.Properties> | undefined>(
    undefined,
  );

  const methods = Form.use({ schema, values: initialValues });

  useAsyncEffect(async () => {
    if (client == null) return;
    const dev = methods.value().config.device;
    if (dev === "") return;
    const d = await client.hardware.devices.retrieve<Device.Properties>(dev);
    setDevice(d);
  }, [client?.key]);

  Form.useFieldListener<string, typeof schema>({
    ctx: methods,
    path: "config.device",
    onChange: useCallback(
      (fs) => {
        if (!fs.touched || fs.status.variant !== "success" || client == null) return;
        client.hardware.devices
          .retrieve<Device.Properties>(fs.value)
          .then((d) => setDevice(d))
          .catch(console.error);
      },
      [client?.key, setDevice],
    ),
  });

  const taskState = useObserveState<WriteStateDetails>(
    methods.setStatus,
    methods.clearStatuses,
    task?.key,
    task?.state,
  );
  const createTask = useCreate<WriteConfig, WriteStateDetails, WriteType>(layoutKey);

  const configure = useMutation<void>({
    mutationKey: [client?.key],
    mutationFn: async () => {
      if (!(await methods.validate) || client == null) return;
      const { config, name } = methods.value();

      const dev = await client.hardware.devices.retrieve<Device.Properties>(
        config.device,
      );

      let modified = false;

      const commandsToCreate: WriteChannelConfig[] = [];
      for (const channel of config.channels) {
        const key = getChannelByNodeID(dev.properties, channel.nodeId);
        if (primitiveIsZero(key)) {
          commandsToCreate.push(channel);
        } else {
          try {
            await client.channels.retrieve(key);
          } catch (e) {
            if (NotFoundError.matches(e)) commandsToCreate.push(channel);
            else throw e;
          }
        }
      }

      if (commandsToCreate.length > 0) {
        modified = true;
        if (
          dev.properties.write.channels == null ||
          Array.isArray(dev.properties.write.channels)
        )
          dev.properties.write.channels = {};
        const commandIndexes = await client.channels.create(
          commandsToCreate.map((c) => ({
            name: `${c.name}_cmd_time`,
            dataType: "timestamp",
            isIndex: true,
          })),
        );
        const commands = await client.channels.create(
          commandsToCreate.map((c, i) => ({
            name: `${c.name}_cmd`,
            dataType: c.dataType,
            index: commandIndexes[i].key,
          })),
        );
        commands.forEach((c, i) => {
          const key = commandsToCreate[i].nodeId;
          dev.properties.write.channels[key] = c.key;
        });
      }

      config.channels = config.channels.map((c) => ({
        ...c,
        channel: getChannelByNodeID(dev.properties, c.nodeId),
      }));

      if (modified)
        await client.hardware.devices.create({
          ...dev,
          properties: dev.properties,
        });

      await createTask({
        key: task?.key,
        name,
        type: WRITE_TYPE,
        config,
      });
    },
    onError: (e) => {
      addStatus({
        variant: "error",
        message: "Failed to configure task",
        description: e.message,
      });
    },
  });

  const start = useMutation({
    mutationKey: [client?.key, "start"],
    mutationFn: async () => {
      if (task == null) return;
      await task.executeCommand(taskState?.details?.running == true ? "stop" : "start");
    },
  });

  const placer = Layout.usePlacer();

  return (
    <Align.Space
      className={CSS(CSS.B("task-configure"), CSS.B("opcua"))}
      direction="y"
      grow
      empty
    >
      <Align.Space direction="y" grow>
        <Form.Form {...methods}>
          <Align.Space direction="x">
            <Form.Field<string> path="name" label="Name">
              {(p) => <Input.Text variant="natural" level="h1" {...p} />}
            </Form.Field>
          </Align.Space>
          <Align.Space direction="x" className={CSS.B("task-properties")}>
            <Form.Field<string>
              path="config.device"
              label="OPC UA Server"
              style={{ width: "100%" }}
            >
              {(p) => (
                <PDevice.SelectSingle
                  {...p}
                  allowNone={false}
                  searchOptions={{ makes: ["opc"] }}
                  emptyContent={
                    <Align.Center>
                      <Text.Text shade={6} level="p">
                        No OPC UA servers found.
                      </Text.Text>
                      <Text.Link
                        level="p"
                        onClick={() => placer(createConfigureLayout())}
                      >
                        Connect a new server.
                      </Text.Link>
                    </Align.Center>
                  }
                />
              )}
            </Form.Field>
            <Align.Space direction="x">
              <Form.Field<boolean>
                label="Data Saving"
                path="config.dataSaving"
                optional
              >
                {(p) => <Input.Switch {...p} />}
              </Form.Field>
            </Align.Space>
          </Align.Space>
          <Align.Space
            direction="x"
            grow
            style={{ overflow: "hidden", height: "500px" }}
          >
            <Browser device={device} />
            <WriterChannelList path="config.channels" device={device} />
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

export interface WriterChannelLisProps {
  path: string;
  device?: device.Device<Device.Properties>;
}

export const WriterChannelList = ({
  path,
  device,
}: WriterChannelLisProps): ReactElement => {
  const { value, push, remove } = Form.useFieldArray<WriteChannelConfig>({ path });
  const valueRef = useSyncedRef(value);

  const menuProps = Menu.useContextMenu();

  const handleDrop = useCallback(({ items }: Haul.OnDropProps): Haul.Item[] => {
    const dropped = items.filter(
      (i) => i.type === "opc" && i.data?.nodeClass === "Variable",
    );
    const toAdd = dropped
      .filter((v) => !valueRef.current.some((c) => c.nodeId === v.data?.nodeId))
      .map((i) => {
        const nodeId = i.data?.nodeId as string;
        const name = i.data?.name as string;
        return {
          key: nodeId,
          name,
          nodeName: name,
          cmdChannel: 0,
          enabled: true,
          nodeId,
          dataType: (i.data?.dataType as string) ?? "float32",
        };
      });
    push(toAdd);
    return dropped;
  }, []);

  const canDrop = useCallback((state: Haul.DraggingState): boolean => {
    const v = state.items.some(
      (i) => i.type === "opc" && i.data?.nodeClass === "Variable",
    );
    return v;
  }, []);

  const props = Haul.useDrop({
    type: "opc.WriteTask",
    canDrop,
    onDrop: handleDrop,
  });

  const dragging = Haul.canDropOfType("opc")(Haul.useDraggingState());

  const [selectedChannels, setSelectedChannels] = useState<string[]>(
    value.length > 0 ? [value[0].key] : [],
  );
  const [selectedChannelIndex, setSelectedChannelIndex] = useState<number | null>(
    value.length > 0 ? 0 : null,
  );

  return (
    <Align.Space
      className={CSS(CSS.B("channels"), dragging && CSS.B("dragging"))}
      grow
      empty
      bordered
      rounded
      {...props}
    >
      <Header.Header level="h4">
        <Header.Title weight={500}>Channels</Header.Title>
      </Header.Header>
      <Menu.ContextMenu
        menu={({ keys }: Menu.ContextMenuMenuProps): ReactElement => (
          <ChannelListContextMenu
            path={path}
            keys={keys}
            value={value}
            remove={remove}
            onSelect={(k, i) => {
              setSelectedChannels(k);
              setSelectedChannelIndex(i);
            }}
          />
        )}
        {...menuProps}
      >
        <List.List<string, WriteChannelConfig>
          data={value}
          emptyContent={
            <Align.Center>
              <Text.Text shade={6} level="p" style={{ maxWidth: 300 }}>
                No channels added. Drag a variable{" "}
                <Icon.Variable
                  style={{ fontSize: "2.5rem", transform: "translateY(0.5rem)" }}
                />{" "}
                from the browser to add a channel to the task.
              </Text.Text>
            </Align.Center>
          }
        >
          <List.Selector<string, WriteChannelConfig>
            value={selectedChannels}
            allowNone={false}
            autoSelectOnNone={false}
            allowMultiple
            onChange={(keys, { clickedIndex }) => {
              if (clickedIndex == null) return;
              setSelectedChannels(keys);
              setSelectedChannelIndex(clickedIndex);
            }}
            replaceOnSingle
          >
            <List.Core<string, WriteChannelConfig> grow>
              {({ key, ...props }) => (
                <WriterChannelListItem
                  key={key}
                  {...props}
                  path={path}
                  remove={() => {
                    const indices = selectedChannels
                      .map((k) => value.findIndex((v) => v.key === k))
                      .filter((i) => i >= 0);
                    remove(indices);
                    setSelectedChannels([]);
                    setSelectedChannelIndex(null);
                  }}
                />
              )}
            </List.Core>
          </List.Selector>
        </List.List>
      </Menu.ContextMenu>
      {value.length > 0 && (
        <ChannelForm
          selectedChannelIndex={selectedChannelIndex}
          deviceProperties={device?.properties}
        />
      )}
    </Align.Space>
  );
};

export const WriterChannelListItem = ({
  path,
  remove,
  ...props
}: List.ItemProps<string, WriteChannelConfig> & {
  path: string;
  remove?: () => void;
}): ReactElement => {
  const { entry } = props;
  const ctx = Form.useContext();
  const childValues = Form.useChildFieldValues<WriteChannelConfig>({
    path: `${path}.${props.index}`,
    optional: true,
  });
  if (childValues == null) return <></>;
  const opcNode =
    childValues.nodeId.length > 0 ? childValues.nodeId : "No Node Selected";
  let opcNodeColor = undefined;
  if (opcNode === "No Node Selected") opcNodeColor = "var(--pluto-warning-z)";

  return (
    <List.ItemFrame
      {...props}
      entry={childValues}
      justify="spaceBetween"
      align="center"
      onKeyDown={(e) => ["Delete", "Backspace"].includes(e.key) && remove?.()}
    >
      <Align.Space direction="y" size="small">
        <Text.WithIcon
          startIcon={<Icon.Channel style={{ color: "var(--pluto-gray-l7)" }} />}
          level="p"
          weight={500}
          shade={9}
          align="end"
        >
          {entry.name}
        </Text.WithIcon>
        <Text.WithIcon
          startIcon={<Icon.Variable style={{ color: "var(--pluto-gray-l7)" }} />}
          level="small"
          weight={350}
          shade={7}
          color={opcNodeColor}
          size="small"
        >
          {entry.nodeName} {opcNode}
        </Text.WithIcon>
      </Align.Space>
      <Align.Space direction="x" align="center">
        <EnableDisableButton
          value={childValues.enabled}
          onChange={(v) => ctx.set(`${path}.${props.index}.enabled`, v)}
        />
      </Align.Space>
    </List.ItemFrame>
  );
};

interface ChannelFormProps {
  selectedChannelIndex?: number | null;
  deviceProperties?: Device.Properties;
}

const ChannelForm = ({ selectedChannelIndex }: ChannelFormProps): ReactElement => {
  if (selectedChannelIndex == null || selectedChannelIndex == -1)
    return (
      <Align.Center className={CSS.B("channel-form")}>
        <Text.Text level="p" shade={6}>
          Select a channel to configure its properties.
        </Text.Text>
      </Align.Center>
    );
  const prefix = `config.channels.${selectedChannelIndex}`;
  return (
    <Align.Space direction="y" grow className={CSS.B("channel-form")} empty>
      <Form.TextField
        path={`${prefix}.name`}
        label="Channel Name"
        inputProps={{ variant: "natural", level: "h3" }}
      />
    </Align.Space>
  );
};

export const WriteTask: Layout.Renderer = wrapTaskLayout(Wrapped, ZERO_WRITE_PAYLOAD);
