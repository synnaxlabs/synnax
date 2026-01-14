// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, NotFoundError, type Synnax } from "@synnaxlabs/client";
import { Component, Flex, Form as PForm, type Haul, Icon } from "@synnaxlabs/pluto";
import { caseconv, DataType, primitive } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";
import { type z } from "zod";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/opc/device";
import { type ChannelKeyAndIDGetter, Form } from "@/hardware/opc/task/Form";
import {
  READ_SCHEMAS,
  READ_TYPE,
  type ReadChannel,
  readConfigZ,
  type readStatusDataZ,
  type readTypeZ,
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
    x
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

const isIndexItem = Component.renderProp(IsIndexItem);

const Properties = (): ReactElement => {
  const arrayMode = PForm.useFieldValue<boolean>("config.arrayMode");
  return (
    <>
      <Device.Select />
      <Flex.Box x>
        <Common.Task.Fields.SampleRate />
        <PForm.SwitchField
          label="Array Sampling"
          path="config.arrayMode"
          onChange={(value, { set }) => {
            // always set the array size to 1 for either the default in array mode or an
            // array size of 1 in stream mode.
            set("config.arraySize", 1);
            if (!value) set("config.streamRate", 25);
          }}
        />
        {arrayMode ? (
          <PForm.NumericField
            label="Array Size"
            path="config.arraySize"
            style={{ width: 100 }}
          />
        ) : (
          <Common.Task.Fields.StreamRate />
        )}
        <Common.Task.Fields.DataSaving />
        <Common.Task.Fields.AutoStart />
      </Flex.Box>
    </>
  );
};

const convertHaulItemToChannel = ({ data }: Haul.Item): ReadChannel => {
  if (typeof data?.name !== "string") throw new Error("Invalid name");
  const nodeName = data?.name;
  if (typeof data?.nodeId !== "string")
    throw new Error(`Invalid nodeId for ${nodeName}`);
  const nodeId = data?.nodeId;
  const dataType = typeof data?.dataType === "string" ? data.dataType : "float32";
  return {
    key: nodeId,
    nodeName,
    nodeId,
    channel: 0,
    enabled: true,
    useAsIndex: false,
    dataType,
    name: "",
  };
};

const getChannelKeyAndID: ChannelKeyAndIDGetter<ReadChannel> = ({ channel, key }) => ({
  key: channel,
  id: Common.Task.getChannelNameID(key),
});

const TaskForm: FC<
  Common.Task.FormProps<typeof readTypeZ, typeof readConfigZ, typeof readStatusDataZ>
> = () => (
  <Form
    convertHaulItemToChannel={convertHaulItemToChannel}
    getChannelKeyAndID={getChannelKeyAndID}
    contextMenuItems={Common.Task.readChannelContextMenuItem}
  >
    {isIndexItem}
  </Form>
);

const getInitialValues: Common.Task.GetInitialValues<
  typeof readTypeZ,
  typeof readConfigZ,
  typeof readStatusDataZ
> = ({ deviceKey, config }) => {
  const cfg = config != null ? readConfigZ.parse(config) : ZERO_READ_PAYLOAD.config;
  return {
    ...ZERO_READ_PAYLOAD,
    config: { ...cfg, device: deviceKey ?? cfg.device },
  };
};

interface DetermineIndexChannelArgs {
  client: Synnax;
  config: z.infer<typeof readConfigZ>;
  device: Device.Device;
  taskName: string;
}

const determineIndexChannel = async ({
  client,
  config,
  device,
  taskName,
}: DetermineIndexChannelArgs): Promise<channel.Key> => {
  const indexChannelInTaskConfig = config.channels.find(({ useAsIndex }) => useAsIndex);
  if (indexChannelInTaskConfig) {
    const existingIndex = getChannelByNodeID(
      device.properties,
      indexChannelInTaskConfig.nodeId,
    );
    if (existingIndex)
      try {
        const { isIndex, key, name } = await client.channels.retrieve(existingIndex);
        if (!isIndex)
          throw new Error(
            `${indexChannelInTaskConfig.nodeName} already exist as ${name}, but ${name} is not an index channel. Please remove the useAsIndex flag from ${indexChannelInTaskConfig.nodeName} and reconfigure.`,
          );
        if (!device.properties.read.indexes.includes(key))
          device.properties.read.indexes.push(key);
        device.properties.read.channels[indexChannelInTaskConfig.nodeId] = key;
        return key;
      } catch (e) {
        if (!NotFoundError.matches(e)) throw e;
      }
    const { key } = await client.channels.create({
      name: channel.escapeInvalidName(indexChannelInTaskConfig.nodeName, true),
      dataType: "timestamp",
      isIndex: true,
    });
    device.properties.read.indexes.push(key);
    device.properties.read.channels[indexChannelInTaskConfig.nodeId] = key;
    return key;
  }

  // if one of the channels already exists, just use that channels index channel.
  for (const { nodeId } of config.channels) {
    const existingChannelKey = getChannelByNodeID(device.properties, nodeId);
    if (existingChannelKey)
      try {
        const { index } = await client.channels.retrieve(existingChannelKey);
        if (!device.properties.read.indexes.includes(index))
          device.properties.read.indexes.push(index);
        return index;
      } catch (e) {
        if (!NotFoundError.matches(e)) throw e;
      }
  }

  // there is not an index channel in the task config, so just create a new channel
  const idxCh = await client.channels.create({
    name: `${channel.escapeInvalidName(device.name)}_time_for_${channel.escapeInvalidName(taskName)}`,
    dataType: "timestamp",
    isIndex: true,
  });
  device.properties.read.indexes.push(idxCh.key);
  return idxCh.key;
};

const onConfigure: Common.Task.OnConfigure<typeof readConfigZ> = async (
  client,
  config,
  name,
) => {
  const previous = await client.devices.retrieve<
    typeof Device.propertiesZ,
    typeof Device.makeZ
  >({
    key: config.device,
  });
  const device = await client.devices.create<
    typeof Device.propertiesZ,
    typeof Device.makeZ
  >({
    ...previous,
    properties: Device.migrateProperties(previous.properties),
  });

  const index = await determineIndexChannel({
    client,
    device,
    config,
    taskName: name,
  });

  const toCreate: ReadChannel[] = [];
  for (const ch of config.channels) {
    const exKey = getChannelByNodeID(device.properties, ch.nodeId);
    if (!exKey) {
      toCreate.push(ch);
      continue;
    }
    try {
      const rCh = await client.channels.retrieve(exKey);
      if (rCh.index !== index)
        throw new Error(
          `Channel ${ch.nodeName} already exists as ${rCh.name}. Please move all channels from ${name} to the OPC UA Read Task that reads for ${rCh.name}.`,
        );
    } catch (e) {
      if (NotFoundError.matches(e)) toCreate.push(ch);
      else throw e;
    }
  }
  if (toCreate.length > 0) {
    const channels = await client.channels.create(
      toCreate.map(({ name, nodeName, dataType }) => ({
        dataType,
        name: primitive.isNonZero(name)
          ? name
          : channel.escapeInvalidName(nodeName, true),
        index,
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
  await client.devices.create(device);
  return [config, device.rack];
};

export const Read = Common.Task.wrapForm({
  type: READ_TYPE,
  Properties,
  Form: TaskForm,
  schemas: READ_SCHEMAS,
  getInitialValues,
  onConfigure,
});
