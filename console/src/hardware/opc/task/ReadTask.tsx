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
  Button,
  Channel,
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
} from "@synnaxlabs/pluto";
import { primitiveIsZero } from "@synnaxlabs/x";
import { DataType } from "@synnaxlabs/x/telem";
import { useMutation } from "@tanstack/react-query";
import { nanoid } from "nanoid";
import { type ReactElement, useCallback, useMemo, useState } from "react";
import { v4 as uuid } from "uuid";
import { z } from "zod";

import { CSS } from "@/css";
import { DigitalWriteStateDetails } from "@/hardware/ni/task/types";
import { Device } from "@/hardware/opc/device";
import { Browser } from "@/hardware/opc/device/Browser";
import {
  Read,
  READ_TYPE,
  type ReadChannelConfig,
  type ReadConfig,
  readConfigZ,
  ReadPayload,
  type ReadStateDetails,
  ReadType,
  ZERO_READ_PAYLOAD,
} from "@/hardware/opc/task/types";
import {
  Controls,
  useCreate,
  useObserveState,
  WrappedTaskLayoutProps,
  wrapTaskLayout,
} from "@/hardware/task/common/common";
import { type Layout } from "@/layout";

export const configureReadLayout = (create: boolean = false): Layout.State => ({
  name: "Configure OPC UA Read Task",
  key: uuid(),
  type: READ_TYPE,
  windowKey: READ_TYPE,
  location: "mosaic",
  window: {
    resizable: true,
    size: { width: 1200, height: 900 },
    navTop: true,
  },
  args: { create },
});

export const READ_SELECTABLE: Layout.Selectable = {
  key: READ_TYPE,
  title: "OPC UA Read Task",
  icon: <Icon.Logo.OPC />,
  create: (layoutKey) => ({ ...configureReadLayout(true), key: layoutKey }),
};

const findChannel = (props: Device.Properties, nodeId: string): string | undefined =>
  Object.entries(props.read.channels).find(([, v]) => v === nodeId)?.[0];

const Wrapped = ({
  layoutKey,
  initialValues,
  task,
}: WrappedTaskLayoutProps<Read, ReadPayload>): ReactElement => {
  const client = Synnax.use();
  const [device, setDevice] = useState<device.Device<Device.Properties> | undefined>(
    undefined,
  );

  const schema = useMemo(
    () =>
      z.object({
        name: z.string(),
        config: readConfigZ.superRefine(async (cfg, ctx) => {
          // if (client == null || device == null) return;
          // for (let i = 0; i < cfg.channels.length; i++) {
          //   const { channel, nodeId } = cfg.channels[i];
          //   if (channel === 0 || nodeId.length === 0) continue;
          //   const ch = await client.channels.retrieve(channel);
          //   const node = device.properties.channels?.find((c) => c.nodeId === nodeId);
          //   if (node == null) return;
          //   const nodeDt = new DataType(node.dataType);
          //   if (!nodeDt.canCastTo(ch.dataType))
          //     ctx.addIssue({
          //       code: z.ZodIssueCode.custom,
          //       path: ["channels", i, "nodeId"],
          //       message: `Node data type ${node.dataType} cannot be cast to channel data type ${ch.dataType}`,
          //     });
          //   else if (!nodeDt.canSafelyCastTo(ch.dataType))
          //     ctx.addIssue({
          //       code: z.ZodIssueCode.custom,
          //       path: ["channels", i, "nodeId"],
          //       message: `Node data type ${node.dataType} may not be safely cast to channel data type ${ch.dataType}`,
          //       params: { variant: "warning" },
          //     });
          //   if (cfg.arrayMode && !node.isArray)
          //     ctx.addIssue({
          //       code: z.ZodIssueCode.custom,
          //       path: ["channels", i, "nodeId"],
          //       message: `Cannot sample from a non-array node in array mode`,
          //     });
          // }
        }),
      }),
    [client?.key, device?.key],
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

  const taskState = useObserveState<DigitalWriteStateDetails>(
    methods.setStatus,
    methods.clearStatuses,
    task?.key,
    task?.state,
  );
  const createTask = useCreate<ReadConfig, ReadStateDetails, ReadType>(layoutKey);

  const configure = useMutation({
    mutationKey: [client?.key],
    onError: console.error,
    mutationFn: async () => {
      if (client == null) return;

      const dev = await client.hardware.devices.retrieve<Device.Properties>(
        methods.value().config.device,
      );

      let modified = false;
      let shouldCreateIndex = primitiveIsZero(dev.properties.read.index);
      if (!shouldCreateIndex) {
        try {
          await client.channels.retrieve(dev.properties.read.index);
        } catch (e) {
          if (NotFoundError.matches(e)) shouldCreateIndex = true;
          else throw e;
        }
      }

      if (shouldCreateIndex) {
        console.log("CREATIN INDEX");
        modified = true;
        const idx = await client.channels.create({
          name: `${dev.properties.identifier}_time`,
          dataType: "timestamp",
          isIndex: true,
        });
        dev.properties.read.index = idx.key;
        dev.properties.read.channels = {};
      }

      const toCreate: ReadChannelConfig[] = [];
      for (const ch of methods.value().config.channels) {
        const exKey = findChannel(dev.properties, ch.nodeId);
        if (primitiveIsZero(exKey)) toCreate.push(ch);
        else {
          try {
            const rCh = await client.channels.retrieve(exKey as string);
            if (rCh.name !== ch.name) {
              await client.channels.rename(Number(exKey), ch.name);
            }
          } catch (e) {
            if (NotFoundError.matches(e)) toCreate.push(ch);
            else throw e;
          }
        }
      }

      if (toCreate.length > 0) {
        modified = true;
        const channels = await client.channels.create(
          toCreate.map((c) => ({
            name: `${dev.properties.identifier}_${c.name}`,
            dataType: "float32",
            index: dev.properties.read.index,
          })),
        );
        channels.forEach((c, i) => {
          dev.properties.read.channels[c.key] = toCreate[i].nodeId;
        });
      }

      if (modified)
        await client.hardware.devices.create({
          ...dev,
          properties: dev.properties,
        });

      const config = methods.value().config;
      config.channels = config.channels.map((c) => ({
        ...c,
        channel: findChannel(dev.properties, c.nodeId),
      }));

      createTask({
        key: task?.key,
        name: methods.value().name,
        type: READ_TYPE,
        config,
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

  const arrayMode = Form.useFieldValue<boolean>("config.arrayMode", false, methods);

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
              <Form.Field<number> label="Sample Rate" path="config.sampleRate">
                {(p) => <Input.Numeric {...p} />}
              </Form.Field>
              <Form.SwitchField label="Array Sampling" path="config.arrayMode" />
              <Form.Field<number>
                label={arrayMode ? "Array Size" : "Stream Rate"}
                path={arrayMode ? "config.arraySize" : "config.streamRate"}
              >
                {(p) => <Input.Numeric {...p} />}
              </Form.Field>
            </Align.Space>
          </Align.Space>
          <Align.Space
            direction="x"
            grow
            style={{ overflow: "hidden", height: "500px" }}
          >
            <Align.Space
              className={CSS.B("browser")}
              direction="y"
              grow
              bordered
              rounded
              style={{ overflow: "hidden", height: "100%" }}
              empty
            >
              <Header.Header level="h4">
                <Header.Title weight={500}>Browser</Header.Title>
              </Header.Header>
              <Browser device={device} />
            </Align.Space>
            <ChannelList path="config.channels" device={device} />
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

export interface ChannelListProps {
  path: string;
  device?: device.Device<Device.Properties>;
}

export const ChannelList = ({ path, device }: ChannelListProps): ReactElement => {
  const { value, push, remove } = Form.useFieldArray<ReadChannelConfig>({ path });

  const menuProps = Menu.useContextMenu();

  const onDrop = useCallback(({ items }: Haul.OnDropProps): Haul.Item[] => {
    const dropped = items.filter(
      (i) => i.type === "opc" && i.data?.nodeClass === "Variable",
    );
    push(
      dropped.map((i) => ({
        key: nanoid(),
        name: (i.data?.name as string) ?? "",
        nodeName: (i.data?.nodeName as string) ?? "",
        channel: 0,
        enabled: true,
        nodeId: (i.data?.nodeId as string) ?? "",
      })),
    );
    return dropped;
  }, []);

  const canDrop = useCallback((state: Haul.DraggingState): boolean => {
    const v = state.items.some(
      (i) => i.type === "opc" && i.data?.nodeClass === "Variable",
    );
    return v;
  }, []);

  const props = Haul.useDrop({
    type: "opc.ReadTask",
    canDrop,
    onDrop,
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
        menu={({ keys }: Menu.ContextMenuMenuProps): ReactElement => {
          const handleSelect = (key: string): void => {
            switch (key) {
              case "remove": {
                const indices = keys
                  .map((k) => value.findIndex((v) => v.key === k))
                  .filter((i) => i >= 0);
                remove(indices);
                setSelectedChannels([]);
                setSelectedChannelIndex(null);
                break;
              }
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
        <List.List<string, ReadChannelConfig>
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
          <List.Selector<string, ReadChannelConfig>
            value={selectedChannels}
            allowNone
            allowMultiple
            onChange={(keys, { clickedIndex }) => {
              if (clickedIndex == null) return;
              setSelectedChannels(keys);
              setSelectedChannelIndex(clickedIndex);
            }}
            replaceOnSingle
          >
            <List.Core<string, ReadChannelConfig> grow>
              {({ key, ...props }) => (
                <ChannelListItem
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
      {selectedChannelIndex != null && (
        <ChannelForm
          selectedChannelIndex={selectedChannelIndex}
          deviceProperties={device?.properties}
        />
      )}
    </Align.Space>
  );
};

export const ChannelListItem = ({
  path,
  remove,
  ...props
}: List.ItemProps<string, ReadChannelConfig> & {
  path: string;
  remove?: () => void;
}): ReactElement => {
  const { entry } = props;
  const ctx = Form.useContext();
  const childValues = Form.useChildFieldValues<ReadChannelConfig>({
    path: `${path}.${props.index}`,
    optional: true,
  });
  if (childValues == null) return <></>;
  const channelName = Channel.useName(entry.channel, entry.name);
  let channelColor = undefined;
  if (channelName === "No Synnax Channel") channelColor = "var(--pluto-warning-z)";
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
        <Text.Text level="p" weight={500} shade={9} color={channelColor}>
          {entry.name}
        </Text.Text>
        <Text.Text level="small" weight={350} shade={7} color={opcNodeColor}>
          {entry.name} {opcNode}
        </Text.Text>
      </Align.Space>
      <Button.Toggle
        checkedVariant="outlined"
        uncheckedVariant="outlined"
        value={entry.enabled}
        size="small"
        onClick={(e) => e.stopPropagation()}
        onChange={(v) => ctx.set(`${path}.${props.index}.enabled`, v)}
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
  deviceProperties?: Device.Properties;
}

const ChannelForm = ({ selectedChannelIndex }: ChannelFormProps): ReactElement => {
  const prefix = `config.channels.${selectedChannelIndex}`;
  return (
    <Align.Space
      direction="y"
      grow
      style={{ padding: "2rem", borderTop: "var(--pluto-border)" }}
    >
      <Form.TextField
        path={`${prefix}.name`}
        label="Channel Name"
        inputProps={{ variant: "natural", level: "h3" }}
      />
    </Align.Space>
  );
};

export const ReadTask: Layout.Renderer = wrapTaskLayout(Wrapped, ZERO_READ_PAYLOAD);
