// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/opc/task/Task.css";

import {
  type channel,
  DataType,
  NotFoundError,
  type Synnax,
  type task,
} from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Form,
  Haul,
  Header,
  Input,
  List,
  Menu,
  Text,
  useSyncedRef,
} from "@synnaxlabs/pluto";
import { caseconv, primitiveIsZero } from "@synnaxlabs/x";
import { type FC, type ReactElement, useCallback, useState } from "react";

import { CSS } from "@/css";
import { Common } from "@/hardware/common";
import { Device } from "@/hardware/opc/device";
import {
  type Read,
  READ_TYPE,
  type ReadChannelConfig,
  type ReadConfig,
  readConfigZ,
  type ReadStateDetails,
  type ReadType,
  ZERO_READ_PAYLOAD,
} from "@/hardware/opc/task/types";
import { type Layout } from "@/layout";

export const READ_LAYOUT: Common.Task.LayoutBaseState = {
  ...Common.Task.LAYOUT,
  key: READ_TYPE,
  type: READ_TYPE,
  name: ZERO_READ_PAYLOAD.name,
  icon: "Logo.OPC",
};

export const READ_SELECTABLE: Layout.Selectable = {
  key: READ_TYPE,
  title: "OPC UA Read Task",
  icon: <Icon.Logo.OPC />,
  create: (key) => ({ ...READ_LAYOUT, key }),
};

const getChannelByNodeID = (props: Device.Properties, nodeId: string) =>
  props.read.channels[nodeId] ?? props.read.channels[caseconv.snakeToCamel(nodeId)];

interface ChannelListProps {
  path: string;
  device: Device.Device;
  snapshot: boolean;
}

const ChannelList = ({ path, snapshot }: ChannelListProps): ReactElement => {
  const { value, push, remove } = Form.useFieldArray<ReadChannelConfig>({ path });
  const valueRef = useSyncedRef(value);

  const menuProps = Menu.useContextMenu();

  const handleDrop = useCallback(({ items }: Haul.OnDropProps): Haul.Item[] => {
    console.log(items);
    const dropped = items.filter(
      (i) => i.type === "opc" && i.data?.nodeClass === "Variable",
    );
    console.log(dropped);
    const toAdd = dropped
      .filter((v) => !valueRef.current.some((c) => c.nodeId === v.data?.nodeId))
      .map((i) => {
        const nodeId = i.data?.nodeId as string;
        const name = i.data?.name as string;
        return {
          key: nodeId,
          name,
          nodeName: name,
          channel: 0,
          enabled: true,
          nodeId,
          useAsIndex: false,
          dataType: typeof i.data?.dataType === "string" ? i.data?.dataType : "float32",
        };
      });
    push(toAdd);
    return dropped;
  }, []);

  const canDrop = useCallback((state: Haul.DraggingState): boolean => {
    console.log("canDrop", state);
    const v = state.items.some(
      (i) => i.type === "opc" && i.data?.nodeClass === "Variable",
    );
    return v;
  }, []);

  const props = Haul.useDrop({
    type: "opc.ReadTask",
    canDrop,
    onDrop: handleDrop,
  });

  const dragging = Haul.canDropOfType("opc")(Haul.useDraggingState());
  console.log(dragging);

  const [selectedChannels, setSelectedChannels] = useState<string[]>(
    value.length > 0 ? [value[0].key] : [],
  );
  const [selectedChannelIndex, setSelectedChannelIndex] = useState<number | null>(
    value.length > 0 ? 0 : null,
  );

  return (
    <Common.Task.ChannelList
      grow
      className={CSS(CSS.B("channels"), dragging && CSS.B("dragging"))}
      header={
        <Header.Header level="h4">
          <Header.Title weight={500}>Channels</Header.Title>
        </Header.Header>
      }
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
      isSnapshot={snapshot}
      onSelect={(keys, index) => {
        setSelectedChannels(keys);
        setSelectedChannelIndex(index);
      }}
      selected={selectedChannels}
      channels={value}
      path={path}
      remove={remove}
    >
      {() => <></>}
    </Common.Task.ChannelList>
  );

  return (
    <Align.Space
      className={CSS(CSS.B("channels"), dragging && CSS.B("dragging"))}
      grow
      empty
      bordered
      rounded
      background={1}
      {...props}
    >
      <Header.Header level="h4">
        <Header.Title weight={500}>Channels</Header.Title>
      </Header.Header>
      <Menu.ContextMenu
        style={{ maxHeight: value.length > 0 ? "calc(100% - 200px)" : "100%" }}
        menu={({ keys }: Menu.ContextMenuMenuProps): ReactElement => (
          <Common.Task.ChannelListContextMenu
            path={path}
            keys={keys}
            value={value}
            remove={remove}
            onSelect={(k, i) => {
              setSelectedChannels(k);
              setSelectedChannelIndex(i);
            }}
            isSnapshot={snapshot}
          />
        )}
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
                  snapshot={snapshot}
                />
              )}
            </List.Core>
          </List.Selector>
        </List.List>
      </Menu.ContextMenu>
      {value.length > 0 && (
        <ChannelForm selectedChannelIndex={selectedChannelIndex} snapshot={snapshot} />
      )}
    </Align.Space>
  );
};

interface ChannelListItemProps extends List.ItemProps<string, ReadChannelConfig> {
  path: string;
  remove?: () => void;
  snapshot?: boolean;
}

export const ChannelListItem = ({
  path,
  remove,
  snapshot,
  ...props
}: ChannelListItemProps): ReactElement => {
  const { entry } = props;
  const ctx = Form.useContext();
  const childValues = Form.useChildFieldValues<ReadChannelConfig>({
    path: `${path}.${props.index}`,
    optional: true,
  });
  if (childValues == null) return <></>;
  const opcNode =
    childValues.nodeId.length > 0 ? childValues.nodeId : "No Node Selected";
  let opcNodeColor;
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
        {childValues.useAsIndex && (
          <Text.Text level="p" style={{ color: "var(--pluto-success-z)" }}>
            Index
          </Text.Text>
        )}
        <Common.Task.EnableDisableButton
          value={childValues.enabled}
          onChange={(v) => ctx.set(`${path}.${props.index}.enabled`, v)}
          isSnapshot={snapshot ?? false}
        />
      </Align.Space>
    </List.ItemFrame>
  );
};

interface ChannelFormProps {
  selectedChannelIndex?: number | null;
  snapshot?: boolean;
}

const ChannelForm = ({
  selectedChannelIndex,
  snapshot,
}: ChannelFormProps): ReactElement | null => {
  if (selectedChannelIndex == null || selectedChannelIndex == -1) {
    if (snapshot === true) return null;
    return (
      <Align.Center className={CSS.B("channel-form")}>
        <Text.Text level="p" shade={6}>
          Select a channel to configure its properties.
        </Text.Text>
      </Align.Center>
    );
  }
  const prefix = `config.channels.${selectedChannelIndex}`;
  return (
    <Align.Space direction="x" grow className={CSS.B("channel-form")} empty>
      <Form.Field<string>
        path={`${prefix}.name`}
        padHelpText={!snapshot}
        label="Channel Name"
      >
        {(p) => <Input.Text variant="natural" level="h3" {...p} />}
      </Form.Field>
      <Form.SwitchField
        path={`${prefix}.useAsIndex`}
        label="Use as Index"
        visible={(_, ctx) =>
          DataType.TIMESTAMP.equals(
            ctx.get<string>(`${prefix}.dataType`, { optional: true })?.value ?? "",
          )
        }
      />
    </Align.Space>
  );
};

const Properties = (): ReactElement => {
  const arrayMode = Form.useFieldValue<boolean>("config.arrayMode");
  return (
    <>
      <Device.Select />
      <Common.Task.Fields.SampleRate />
      <Form.SwitchField label="Array Sampling" path="config.arrayMode" />
      <Form.Field<number>
        label={arrayMode ? "Array Size" : "Stream Rate"}
        path={arrayMode ? "config.arraySize" : "config.streamRate"}
      >
        {(p) => <Input.Numeric {...p} />}
      </Form.Field>
      <Common.Task.Fields.DataSaving />
    </>
  );
};

const TaskForm: FC<Common.Task.FormProps<ReadConfig, ReadStateDetails, ReadType>> = ({
  isSnapshot,
}) => (
  <Common.Device.Provider<Device.Properties, Device.Make>
    configureLayout={Device.CONFIGURE_LAYOUT}
    isSnapshot={isSnapshot}
  >
    {({ device }) => (
      <Align.Space direction="x">
        {!isSnapshot && <Device.Browser device={device} />}
        <ChannelList path="config.channels" device={device} snapshot={isSnapshot} />
      </Align.Space>
    )}
  </Common.Device.Provider>
);

const zeroPayload: Common.Task.ZeroPayloadFunction<
  ReadConfig,
  ReadStateDetails,
  ReadType
> = (deviceKey) => ({
  ...ZERO_READ_PAYLOAD,
  config: {
    ...ZERO_READ_PAYLOAD.config,
    device: deviceKey ?? ZERO_READ_PAYLOAD.config.device,
  },
});

const onConfigure = async (
  client: Synnax,
  config: ReadConfig,
  taskKey: task.Key,
): Promise<ReadConfig> => {
  // Retrieving the device and updating its properties if needed
  const dev = await client.hardware.devices.retrieve<Device.Properties>(config.device);
  dev.properties = Device.migrateProperties(dev.properties);
  await client.hardware.devices.create(dev);
  // modified determines if we have to configure a device. indexChannel is the key
  // that will be used as an index for the read task.
  let modified = false;
  let indexChannel: channel.Key = 0;
  // getting exiting indexes on the opc device
  let devIndexes: channel.Key[] = [];
  if (!primitiveIsZero(dev.properties.read.indexes))
    try {
      devIndexes = (await client.channels.retrieve(dev.properties.read.indexes)).map(
        (c) => c.key,
      );
    } catch (e) {
      if (NotFoundError.matches(e)) devIndexes = [];
      else throw e;
    }
  // getting the index channels of all opc read tasks channels
  const existingTasks = (await client.hardware.tasks.list()).filter(
    (t) => t.type === READ_TYPE,
  ) as Read[];
  // check if this task already exists
  const existingTask = existingTasks.find((t) => t.key === taskKey);
  // if it does exist, grab the index channel of all of the keys in the task
  if (existingTask) {
    const existingTaskIndexes = (
      await client.channels.retrieve(existingTask.config.channels.map((c) => c.channel))
    ).map((c) => c.index);
    const uniqueIndexes = [...new Set(existingTaskIndexes)];
    if (uniqueIndexes.length === 0)
      throw new Error(`${name} already exists, but no index channel was found`);
    indexChannel = uniqueIndexes[0];
  } else {
    const existingTasksChannels: channel.Key[] = existingTasks
      .flatMap((t) => t.config.channels)
      .flatMap((c) => c.channel);
    const existingTaskIndexes = (
      await client.channels.retrieve(existingTasksChannels)
    ).flatMap((c) => c.index);
    const unusedDeviceIndexes = devIndexes.filter(
      (k) => !existingTaskIndexes.includes(k),
    );
    // if there is a useAsIndex in the config
    const indexChannelConfig = config.channels.find((c) => c.useAsIndex);
    if (indexChannelConfig) {
      const existingIndex = getChannelByNodeID(
        dev.properties,
        indexChannelConfig.nodeId,
      );
      if (
        devIndexes.includes(existingIndex) &&
        !unusedDeviceIndexes.includes(existingIndex)
      ) {
        const task = existingTasks.find((t) =>
          t.config.channels.some((c) => c.channel === existingIndex),
        );
        const taskName = task?.name ?? "an OPC UA read task";
        // this channel is being used as an index on two different tasks
        throw new Error(
          `${indexChannelConfig.name} is already being used as an index for ${taskName}. Please add the channels from this read task to the existing read task`,
        );
      }
      if (primitiveIsZero(existingIndex)) {
        const idx = await client.channels.create({
          name: indexChannelConfig.name,
          dataType: "timestamp",
          isIndex: true,
        });
        dev.properties.read.indexes.push(idx.key);
        dev.properties.read.channels[indexChannelConfig.nodeId] = idx.key;
        modified = true;
        indexChannel = idx.key;
      } else indexChannel = existingIndex;
    } else if (unusedDeviceIndexes.length > 0) indexChannel = unusedDeviceIndexes[0];
    else {
      const idx = await client.channels.create({
        name: `${dev.name} time for ${name}`,
        dataType: "timestamp",
        isIndex: true,
      });
      dev.properties.read.indexes.push(idx.key);
      modified = true;
      indexChannel = idx.key;
    }
  }
  const toCreate: ReadChannelConfig[] = [];
  for (const ch of config.channels) {
    const exKey = getChannelByNodeID(dev.properties, ch.nodeId);
    if (primitiveIsZero(exKey)) toCreate.push(ch);
    else
      try {
        const rCh = await client.channels.retrieve(exKey);
        if (rCh.index !== indexChannel)
          throw new Error(
            `Channel ${ch.name} already exists on an existing OPC UA read task with a different index channel`,
          );

        if (rCh.name !== ch.name) await client.channels.rename(Number(exKey), ch.name);
      } catch (e) {
        if (NotFoundError.matches(e)) toCreate.push(ch);
        else throw e;
      }
  }
  if (toCreate.length > 0) {
    modified = true;
    const channels = await client.channels.create(
      toCreate.map((c) => ({
        name: c.name,
        dataType: c.dataType,
        index: indexChannel,
      })),
    );
    channels.forEach(
      (c, i) => (dev.properties.read.channels[toCreate[i].nodeId] = c.key),
    );
  }
  config.channels = config.channels.map((c) => ({
    ...c,
    channel: getChannelByNodeID(dev.properties, c.nodeId),
  }));
  if (modified) await client.hardware.devices.create(dev);
  return config;
};

export const ReadTask = Common.Task.wrapForm(<Properties />, TaskForm, {
  configSchema: readConfigZ,
  type: READ_TYPE,
  zeroPayload,
  onConfigure,
});
