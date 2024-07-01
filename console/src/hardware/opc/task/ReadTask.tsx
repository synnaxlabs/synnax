// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, device } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Channel,
  Device as PDevice,
  Form,
  Header,
  Input,
  List,
  Menu,
  Status,
  Synnax,
  Text,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { DataType } from "@synnaxlabs/x/telem";
import { useMutation } from "@tanstack/react-query";
import { nanoid } from "nanoid";
import { type ReactElement, useCallback, useMemo, useState } from "react";
import { v4 as uuid } from "uuid";
import { z } from "zod";

import { CSS } from "@/css";
import { DigitalWriteStateDetails } from "@/hardware/ni/task/types";
import { Device } from "@/hardware/opc/device";
import { SelectNodeRemote } from "@/hardware/opc/device/SelectNode";
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
  ChannelListEmptyContent,
  ChannelListHeader,
  Controls,
  useCreate,
  useObserveState,
} from "@/hardware/task/common/common";
import { wrapTaskLayout } from "@/hardware/task/TaskWrapper";
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

interface InternalProps {
  layoutKey: string;
  task?: Read;
  initialValues: ReadPayload;
}

const Internal = ({ layoutKey, initialValues, task }: InternalProps): ReactElement => {
  const client = Synnax.use();
  const [device, setDevice] = useState<device.Device<Device.Properties> | null>(null);

  const schema = useMemo(
    () =>
      z.object({
        name: z.string(),
        config: readConfigZ.superRefine(async (cfg, ctx) => {
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
            if (cfg.arrayMode && !node.isArray) {
              ctx.addIssue({
                code: z.ZodIssueCode.custom,
                path: ["channels", i, "nodeId"],
                message: `Cannot sample from a non-array node in array mode`,
              });
            }
          }
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

  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);
  const [selectedChannelIndex, setSelectedChannelIndex] = useState<number | null>(null);

  const taskState = useObserveState<DigitalWriteStateDetails>(
    methods.setStatus,
    task?.key,
    task?.state,
  );
  const createTask = useCreate<ReadConfig, ReadStateDetails, ReadType>(layoutKey);

  const configure = useMutation({
    mutationKey: [client?.key],
    mutationFn: async () => {
      if (!(await methods.validateAsync()) || client == null) return;
      createTask({
        key: task?.key,
        name: methods.value().name,
        type: READ_TYPE,
        config: readConfigZ.parse(methods.value().config),
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
    <Align.Space className={CSS.B("task-configure")} direction="y" grow empty>
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
              label="Device"
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
              <Header.Header level="h4">
                <Header.Title weight={500}>Details</Header.Title>
              </Header.Header>
              <Align.Space direction="y" className={CSS.B("details")} grow>
                {selectedChannelIndex != null && (
                  <ChannelForm
                    key={selectedChannelIndex}
                    deviceProperties={device?.properties}
                    selectedChannelIndex={selectedChannelIndex}
                  />
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
  const { value, push, remove } = Form.useFieldArray<ReadChannelConfig>({ path });

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
      <ChannelListHeader onAdd={handleAdd} />
      <Menu.ContextMenu
        menu={({ keys }: Menu.ContextMenuMenuProps): ReactElement => {
          const handleSelect = (key: string): void => {
            switch (key) {
              case "remove": {
                const indices = keys
                  .map((k) => value.findIndex((v) => v.key === k))
                  .filter((i) => i >= 0);
                remove(indices);
                onSelect([], 0);
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
          emptyContent={<ChannelListEmptyContent onAdd={handleAdd} />}
        >
          <List.Selector<string, ReadChannelConfig>
            value={selected}
            allowNone
            allowMultiple
            onChange={(keys, { clickedIndex }) =>
              clickedIndex != null && onSelect(keys, clickedIndex)
            }
            replaceOnSingle
          >
            <List.Core<string, ReadChannelConfig> grow>
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
  const channelName = Channel.useName(entry.channel, "No Synnax Channel");
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
          {channelName}
        </Text.Text>
        <Text.Text level="small" weight={350} shade={7} color={opcNodeColor}>
          {opcNode}
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

const ChannelForm = ({
  selectedChannelIndex,
  deviceProperties,
}: ChannelFormProps): ReactElement => {
  const prefix = `config.channels.${selectedChannelIndex}`;
  const dev = Form.useField<string>({ path: "config.device" }).value;
  const channelPath = `${prefix}.channel`;
  const channelValue = Form.useFieldValue<number>(channelPath);
  const [channelRec, setChannelRec] = useState<channel.Key>(0);
  const channelRecName = Channel.useName(channelRec, "");
  const ctx = Form.useContext();
  return (
    <>
      <Form.Field<string>
        path={`${prefix}.nodeId`}
        label="OPC Node"
        onChange={(v) => {
          if (deviceProperties == null) return;
          const defaultChan = deviceProperties.channels.find(
            (c) => c.nodeId === v,
          )?.synnaxChannel;
          if (defaultChan != null) setChannelRec(defaultChan);
        }}
        hideIfNull
      >
        {(p) => <SelectNodeRemote allowNone={false} device={dev} {...p} />}
      </Form.Field>
      <Form.Field<number>
        path={channelPath}
        label="Synnax Channel"
        hideIfNull
        padHelpText={false}
      >
        {(p) => <Channel.SelectSingle allowNone={false} {...p} />}
      </Form.Field>
      {channelRecName.length > 0 && channelValue !== channelRec && (
        <Align.Space direction="x" size="small">
          <Button.Icon
            variant="text"
            size="small"
            onClick={() => ctx.set(channelPath, channelRec)}
            tooltip={"Apply recommended channel"}
          >
            <Icon.Bolt style={{ color: "var(--pluto-gray-l6)" }} />
          </Button.Icon>
          <Button.Button
            variant="suggestion"
            size="small"
            style={{ width: "fit-content" }}
            startIcon={<Icon.Channel />}
            onClick={() => ctx.set(channelPath, channelRec)}
            tooltip={"Apply recommended channel"}
          >
            {channelRecName}
          </Button.Button>
        </Align.Space>
      )}
    </>
  );
};

export const ReadTask: Layout.Renderer = wrapTaskLayout<Read, ReadPayload>(
  Internal,
  ZERO_READ_PAYLOAD,
);
