// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/opc/task/Task.css";

import { channel, NotFoundError, type Synnax } from "@synnaxlabs/client";
import { Component, Flex, Form as PForm, Icon } from "@synnaxlabs/pluto";
import { caseconv, DataType, errors, primitive } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";

import { type HaulItem } from "@/feature/opc/device/Browser";
import { Select } from "@/feature/opc/device/Select";
import * as Device from "@/feature/opc/device/types";
import { type ChannelKeyAndIDGetter, Form } from "@/feature/opc/task/Form";
import {
  type InputChannel,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadConfig,
  type ReadSchemas,
  ZERO_READ_PAYLOAD,
} from "@/feature/opc/task/types";
import { CSS } from "@/platform/css";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

export const ReadSelectable = Selector.createSelectable({
  type: READ_TYPE,
  title: "OPC UA Read Task",
  icon: <Icon.Logo.OPC />,
  useOnSelect: Task.createOpenTab(READ_TYPE),
});

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
      <Select />
      <Flex.Box x>
        <Task.Fields.SampleRate />
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
            className={CSS.B("opc-array-size-field")}
          />
        ) : (
          <Task.Fields.StreamRate />
        )}
        <Task.Fields.DataSaving />
        <Task.Fields.AutoStart />
      </Flex.Box>
    </>
  );
};

const convertHaulItemToChannel = ({ data }: HaulItem): InputChannel => ({
  key: data.nodeId,
  nodeName: data.name,
  nodeId: data.nodeId,
  channel: 0,
  enabled: true,
  useAsIndex: false,
  dataType: data.dataType,
  name: "",
});

const getChannelKeyAndID: ChannelKeyAndIDGetter<InputChannel> = ({ channel, key }) => ({
  key: channel,
  id: Task.getChannelNameID(key),
});

const TaskForm: FC<Task.FormProps<ReadSchemas>> = () => (
  <Form
    convertHaulItemToChannel={convertHaulItemToChannel}
    getChannelKeyAndID={getChannelKeyAndID}
    contextMenuItems={Task.readChannelContextMenuItem}
  >
    {isIndexItem}
  </Form>
);

const getInitialValues: Task.GetInitialValues<ReadSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg =
    config != null ? READ_SCHEMAS.config.parse(config) : ZERO_READ_PAYLOAD.config;
  return {
    ...ZERO_READ_PAYLOAD,
    config: { ...cfg, device: deviceKey ?? cfg.device },
  };
};

interface DetermineIndexChannelParams {
  client: Synnax;
  config: ReadConfig;
  device: Device.Device;
  taskName: string;
}

const determineIndexChannel = async ({
  client,
  config,
  device,
  taskName,
}: DetermineIndexChannelParams): Promise<channel.Key> => {
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
        if (!NotFoundError.matches(e)) throw errors.fromUnknown(e);
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
        if (!NotFoundError.matches(e)) throw errors.fromUnknown(e);
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

const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (
  client,
  config,
  name,
) => {
  const previous = await client.devices.retrieve({
    key: config.device,
    schemas: Device.SCHEMAS,
  });
  const device = await client.devices.create(
    { ...previous, properties: Device.migrateProperties(previous.properties) },
    Device.SCHEMAS,
  );

  const index = await determineIndexChannel({
    client,
    device,
    config,
    taskName: name,
  });

  const toCreate: InputChannel[] = [];
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
      else throw errors.fromUnknown(e);
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
  await client.devices.create(device, Device.SCHEMAS);
  return [config, device.rack];
};

export const Read = Task.wrapForm({
  type: "opc_read",
  Properties,
  Form: TaskForm,
  schemas: READ_SCHEMAS,
  getInitialValues,
  onConfigure,
});
