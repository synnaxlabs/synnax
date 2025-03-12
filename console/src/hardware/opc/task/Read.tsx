// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type channel,
  NotFoundError,
  type Synnax,
  type task,
} from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  componentRenderProp,
  Form as PForm,
  type Haul,
} from "@synnaxlabs/pluto";
import { caseconv, DataType } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/opc/device";
import { Form } from "@/hardware/opc/task/Form";
import {
  READ_TYPE,
  type ReadChannel,
  type ReadConfig,
  readConfigZ,
  type ReadStateDetails,
  type ReadType,
  ZERO_READ_PAYLOAD,
} from "@/hardware/opc/task/types";
import { type Selector } from "@/selector";

export const READ_LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  type: READ_TYPE,
  name: ZERO_READ_PAYLOAD.name,
  icon: "Logo.OPC",
};

export const READ_SELECTABLE: Selector.Selectable = {
  key: READ_TYPE,
  title: "OPC UA Read Task",
  icon: <Icon.Logo.OPC />,
  create: async ({ layoutKey }) => ({ ...READ_LAYOUT, key: layoutKey }),
};

const getChannelByNodeID = (props: Device.Properties, nodeId: string): channel.Key =>
  props.read.channels[nodeId] ??
  props.read.channels[caseconv.snakeToCamel(nodeId)] ??
  0;

interface IsIndexItemProps {
  path: string;
  snapshot?: boolean;
}

const IsIndexItem = ({ path }: IsIndexItemProps): ReactElement => (
  <PForm.SwitchField
    path={`${path}.useAsIndex`}
    label="Use as Index"
    hideIfNull
    direction="x"
    align="center"
    showHelpText={false}
    required={false}
    visible={(_, ctx) =>
      DataType.TIMESTAMP.equals(
        ctx.get<string>(`${path}.dataType`, { optional: true })?.value ?? "",
      )
    }
  />
);

const isIndexItem = componentRenderProp(IsIndexItem);

const Properties = (): ReactElement => {
  const arrayMode = PForm.useFieldValue<boolean>("config.arrayMode");
  return (
    <>
      <Device.Select />
      <Align.Space direction="x" grow>
        <Common.Task.Fields.SampleRate />
        <PForm.SwitchField
          label="Array Sampling"
          path="config.arrayMode"
          onChange={(value, { set }) => {
            if (value) set("config.arraySize", 1);
            else set("config.streamRate", 25);
          }}
        />
        {arrayMode ? (
          <PForm.NumericField label="Array Size" path="config.arraySize" />
        ) : (
          <Common.Task.Fields.StreamRate />
        )}
        <Common.Task.Fields.DataSaving />
      </Align.Space>
    </>
  );
};

const convertHaulItemToChannel = ({ data }: Haul.Item): ReadChannel => {
  const nodeId = data?.nodeId as string;
  const name = data?.name as string;
  return {
    key: nodeId,
    name,
    nodeName: name,
    nodeId,
    channel: 0,
    enabled: true,
    useAsIndex: false,
    dataType: (data?.dataType as string) ?? "float32",
  };
};

const TaskForm: FC<Common.Task.FormProps<ReadConfig, ReadStateDetails, ReadType>> = ({
  isSnapshot,
}) => (
  <Form isSnapshot={isSnapshot} convertHaulItemToChannel={convertHaulItemToChannel}>
    {isIndexItem}
  </Form>
);

const getInitialPayload: Common.Task.GetInitialPayload<
  ReadConfig,
  ReadStateDetails,
  ReadType
> = ({ deviceKey }) => ({
  ...ZERO_READ_PAYLOAD,
  config: {
    ...ZERO_READ_PAYLOAD.config,
    device: deviceKey ?? ZERO_READ_PAYLOAD.config.device,
  },
});

interface DetermineIndexChannelArgs {
  client: Synnax;
  config: ReadConfig;
  device: Device.Device;
  taskKey: task.Key;
  taskName: string;
}

interface DetermineIndexChannelResult {
  hasModifiedDevice: boolean;
  indexChannel: channel.Key;
}

const determineIndexChannel = async ({
  client,
  config,
  device,
  taskKey,
  taskName,
}: DetermineIndexChannelArgs): Promise<DetermineIndexChannelResult> => {
  const existingTasks = await client.hardware.tasks.retrieve<
    ReadConfig,
    ReadStateDetails,
    ReadType
  >(device.rack, { types: [READ_TYPE] });
  const existingTask = existingTasks.find(({ key }) => key === taskKey);

  if (existingTask) {
    const channelsForExistingTask = await client.channels.retrieve(
      existingTask.config.channels.map(({ channel }) => channel),
    );
    const indexesForExistingTask = channelsForExistingTask.map(({ index }) => index);
    if (indexesForExistingTask.length)
      return { indexChannel: indexesForExistingTask[0], hasModifiedDevice: false };
  }

  const channelKeysFromExistingTasks: channel.Key[] = existingTasks
    .flatMap(({ config: { channels } }) => channels)
    .flatMap(({ channel }) => channel);
  const indexChannelsFromExistingTasks = await client.channels.retrieve(
    channelKeysFromExistingTasks,
  );
  const indexChannelKeysFromExistingTasks = new Set(
    indexChannelsFromExistingTasks.map(({ index }) => index),
  );

  const devIndexes: channel.Key[] = device.properties.read.indexes;
  const unusedDeviceIndexes = devIndexes.filter(
    (k) => !indexChannelKeysFromExistingTasks.has(k),
  );
  // if there is a useAsIndex in the config
  const indexChannelInTaskConfig = config.channels.find(({ useAsIndex }) => useAsIndex);
  if (indexChannelInTaskConfig) {
    const existingIndex = getChannelByNodeID(
      device.properties,
      indexChannelInTaskConfig.nodeId,
    );
    if (
      devIndexes.includes(existingIndex) &&
      !unusedDeviceIndexes.includes(existingIndex)
    ) {
      // this channel is being used as an index on two different tasks
      const task = existingTasks.find(({ config: { channels } }) =>
        channels.some(({ channel }) => channel === existingIndex),
      );
      const existingTaskName = task?.name;
      throw new Error(
        `${indexChannelInTaskConfig.name} is already being used as an index for ${existingTaskName ?? "an existing OPC UA Read Task"}. Please move all channels from ${taskName} to ${existingTaskName ?? "the existing task"}.`,
      );
    }
    if (existingIndex) return { indexChannel: existingIndex, hasModifiedDevice: false };
    const idx = await client.channels.create({
      name: indexChannelInTaskConfig.name,
      dataType: "timestamp",
      isIndex: true,
    });
    device.properties.read.indexes.push(idx.key);
    device.properties.read.channels[indexChannelInTaskConfig.nodeId] = idx.key;
    return { indexChannel: idx.key, hasModifiedDevice: true };
  }
  if (unusedDeviceIndexes.length)
    return { indexChannel: unusedDeviceIndexes[0], hasModifiedDevice: false };
  const idx = await client.channels.create({
    name: `${device.name} time for ${taskName}`,
    dataType: "timestamp",
    isIndex: true,
  });
  device.properties.read.indexes.push(idx.key);
  return { indexChannel: idx.key, hasModifiedDevice: true };
};

const onConfigure: Common.Task.OnConfigure<ReadConfig> = async (
  client,
  config,
  taskKey,
  name,
) => {
  // Retrieving the device and updating its properties if needed
  const previous = await client.hardware.devices.retrieve<
    Device.Properties,
    Device.Make
  >(config.device);
  const device = await client.hardware.devices.create<Device.Properties, Device.Make>({
    ...previous,
    properties: Device.migrateProperties(previous.properties),
  });

  const { hasModifiedDevice, indexChannel } = await determineIndexChannel({
    client,
    device,
    config,
    taskKey,
    taskName: name,
  });
  let modified = hasModifiedDevice;

  const toCreate: ReadChannel[] = [];
  for (const ch of config.channels) {
    const exKey = getChannelByNodeID(device.properties, ch.nodeId);
    if (!exKey) {
      toCreate.push(ch);
      continue;
    }
    try {
      const rCh = await client.channels.retrieve(exKey);
      if (rCh.index !== indexChannel)
        throw new Error(
          `Channel ${ch.name} already exists on an existing OPC UA read task with a different index channel`,
        );
      if (rCh.name !== ch.name) await client.channels.rename(exKey, ch.name);
    } catch (e) {
      if (NotFoundError.matches(e)) toCreate.push(ch);
      else throw e;
    }
  }
  if (toCreate.length > 0) {
    modified = true;
    const channels = await client.channels.create(
      toCreate.map(({ name, dataType }) => ({
        name,
        dataType,
        index: indexChannel,
      })),
    );
    channels.forEach(
      ({ key }, i) => (device.properties.read.channels[toCreate[i].nodeId] = key),
    );
  }
  config.channels = config.channels.map((c) => ({
    ...c,
    channel: getChannelByNodeID(device.properties, c.nodeId),
  }));
  if (modified) await client.hardware.devices.create(device);
  return [config, device.rack];
};

export const Read = Common.Task.wrapForm({
  type: READ_TYPE,
  Properties,
  Form: TaskForm,
  configSchema: readConfigZ,
  getInitialPayload,
  onConfigure,
});
