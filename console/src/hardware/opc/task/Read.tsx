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
import { Align, Form as PForm } from "@synnaxlabs/pluto";
import { caseconv, DataType, primitiveIsZero } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/opc/device";
import { Form } from "@/hardware/opc/task/Form";
import {
  READ_TYPE,
  type ReadChannelConfig,
  type ReadConfig,
  readConfigZ,
  type ReadStateDetails,
  type ReadTask,
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

interface IsIndexItemProps {
  path: string;
  snapshot?: boolean;
}

const IsIndexItem = ({ path }: IsIndexItemProps): ReactElement => (
  <PForm.SwitchField
    path={`${path}.useAsIndex`}
    label="Use as Index"
    visible={(_, ctx) =>
      DataType.TIMESTAMP.equals(
        ctx.get<string>(`${path}.dataType`, { optional: true })?.value ?? "",
      )
    }
  />
);

const Properties = (): ReactElement => {
  const arrayMode = PForm.useFieldValue<boolean>("config.arrayMode");
  return (
    <>
      <Device.Select />
      <Align.Space direction="x" grow>
        <Common.Task.Fields.SampleRate />
        <PForm.SwitchField label="Array Sampling" path="config.arrayMode" />
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

const TaskForm: FC<Common.Task.FormProps<ReadConfig, ReadStateDetails, ReadType>> = ({
  isSnapshot,
}) => (
  <Form isSnapshot={isSnapshot}>
    {({ path, snapshot }) => <IsIndexItem path={path} snapshot={snapshot} />}
  </Form>
);
const getInitialPayload: Common.Task.GetInitialPayload<
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
  name: string,
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
  ) as ReadTask[];
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

export const Read = Common.Task.wrapForm(() => <Properties />, TaskForm, {
  configSchema: readConfigZ,
  type: READ_TYPE,
  getInitialPayload,
  onConfigure,
});
