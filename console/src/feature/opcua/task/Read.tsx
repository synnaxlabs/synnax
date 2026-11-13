// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/opcua/task/Task.css";

import { channel, NotFoundError, type opcua, type Synnax } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select as PSelect } from "@synnaxlabs/lyra/select";
import { caseconv, DataType, deep, errors, primitive } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";

import { Select } from "@/feature/opcua/device/Select";
import {
  type Device,
  MAKE,
  migrateProperties,
  type Properties as DeviceProperties,
  SCHEMAS,
} from "@/feature/opcua/device/types";
import {
  type ChannelKeyAndIDGetter,
  createForm,
  type ExtraItemProps,
} from "@/feature/opcua/task/Form";
import {
  deployReadConfigZ,
  nodeIdentity,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadChannel,
  type ReadConfig,
  type ReadSchemas,
  type ScannedNode,
} from "@/feature/opcua/task/types";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const path = deep.path<Task.Values<ReadSchemas>>();

type ChannelSource = Pick<ReadChannel, "nodeId" | "field">;

const findField = (props: DeviceProperties, { nodeId, field }: ChannelSource) =>
  props.read.fields.find((f) => f.nodeId === nodeId && f.field === field);

const getChannel = (props: DeviceProperties, source: ChannelSource) =>
  source.field === ""
    ? (props.read.channels[source.nodeId] ??
      props.read.channels[caseconv.snakeToCamel(source.nodeId)] ??
      0)
    : (findField(props, source)?.channel ?? 0);

const setChannel = (
  props: DeviceProperties,
  source: ChannelSource,
  key: channel.Key,
) => {
  const { nodeId, field } = source;
  if (field === "") {
    props.read.channels[nodeId] = key;
    return;
  }
  const existing = findField(props, source);
  if (existing != null) existing.channel = key;
  else props.read.fields.push({ nodeId, field, channel: key });
};

const IsIndexItem = ({ path }: ExtraItemProps<ReadChannel>): ReactElement => {
  const dataType = PForm.useFieldValue(path.dataType, { optional: true });
  return (
    <PForm.CheckboxField
      path={path.isIndex}
      label="Use as index"
      x
      align="center"
      showHelpText={false}
      visible={DataType.TIMESTAMP.equals(dataType ?? "")}
    />
  );
};

const isIndexItem = Component.renderProp(IsIndexItem);

const Properties = (): ReactElement => {
  const samplingType = PForm.useFieldValue(path.config.sampling.type);
  return (
    <>
      <Select />
      <Flex.Box x wrap>
        <PForm.Field<opcua.Sampling>
          label="Sampling mode"
          path={path.config.sampling}
          onChange={({ type }, { set }) => {
            if (type === "unary") set(path.config.streamRate, 25);
          }}
        >
          {({ value, onChange }) => (
            <PSelect.Buttons<opcua.SamplingType>
              value={value.type}
              onChange={(type: opcua.SamplingType) =>
                onChange(type === "array" ? { type, size: 1 } : { type })
              }
            >
              <PSelect.Item itemKey="unary">Scalar</PSelect.Item>
              <PSelect.Item itemKey="array">Array</PSelect.Item>
            </PSelect.Buttons>
          )}
        </PForm.Field>
        <Task.Fields.SampleRate />
        {samplingType === "array" ? (
          <PForm.NumericField
            label="Array size"
            path={deep.variant(path.config.sampling, "type", "array").size}
            className={CSS.B("opc-array-size-field")}
          />
        ) : (
          <Task.Fields.StreamRate />
        )}
        <Task.Fields.NaNFill />
      </Flex.Box>
    </>
  );
};

const convertNodeToChannel = (node: ScannedNode): ReadChannel => ({
  key: node.field === "" ? node.nodeId : nodeIdentity(node),
  nodeName: node.name,
  nodeId: node.nodeId,
  field: node.field,
  channel: 0,
  disabled: false,
  isIndex: false,
  dataType: node.dataType,
  name: "",
});

const getChannelKeyAndID: ChannelKeyAndIDGetter<ReadChannel> = ({ channel, key }) => ({
  key: channel,
  id: Task.getChannelNameID(key),
});

const resolve = (device: Device, ch: ReadChannel) => ({
  channel: getChannel(device.properties, ch),
});

const TaskForm: FC = createForm<ReadChannel>({
  path: path.config.channels,
  convertNodeToChannel,
  getChannelKeyAndID,
  contextMenuItems: Task.readChannelContextMenuItem,
  children: isIndexItem,
  resolve,
  indexed: true,
  tareable: true,
});

const getInitialValues: Task.GetInitialValues<ReadSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = READ_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: "OPC UA read task", type: READ_TYPE, config: cfg };
};

interface DetermineIndexChannelParams {
  client: Synnax;
  config: ReadConfig;
  device: Device;
  taskName: string;
}

const determineIndexChannel = async ({
  client,
  config,
  device,
  taskName,
}: DetermineIndexChannelParams): Promise<channel.Key> => {
  const indexChannelInTaskConfig = config.channels.find(({ isIndex }) => isIndex);
  if (indexChannelInTaskConfig) {
    const existingIndex = getChannel(device.properties, indexChannelInTaskConfig);
    if (existingIndex)
      try {
        const { isIndex, key, name } = await client.channels.retrieve(existingIndex);
        if (!isIndex)
          throw new Error(
            `${indexChannelInTaskConfig.nodeName} already exists as ${name}, which is not an index channel. Disable "use as index" on ${indexChannelInTaskConfig.nodeName} and reconfigure.`,
          );
        if (!device.properties.read.indexes.includes(key))
          device.properties.read.indexes.push(key);
        setChannel(device.properties, indexChannelInTaskConfig, key);
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
    setChannel(device.properties, indexChannelInTaskConfig, key);
    return key;
  }

  // if one of the channels already exists, just use that channels index channel.
  for (const ch of config.channels) {
    const existingChannelKey = getChannel(device.properties, ch);
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
    schemas: SCHEMAS,
  });
  const before = deep.copy(previous.properties);
  const device = { ...previous, properties: migrateProperties(previous.properties) };

  const index = await determineIndexChannel({
    client,
    device,
    config,
    taskName: name,
  });

  const toCreate: ReadChannel[] = [];
  for (const ch of config.channels) {
    const exKey = getChannel(device.properties, ch);
    if (!exKey) {
      toCreate.push(ch);
      continue;
    }
    try {
      const rCh = await client.channels.retrieve(exKey);
      if (rCh.index !== index)
        throw new Error(
          `Channel ${ch.nodeName} already exists as ${rCh.name}. Move all channels from ${name} to the task that reads ${rCh.name}.`,
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
    channels.forEach(({ key }, i) => setChannel(device.properties, toCreate[i], key));
  }
  config.channels = config.channels.map((c) => ({
    ...c,
    channel: getChannel(device.properties, c),
  }));
  await PlatformDevice.setProperties(client, device, before, SCHEMAS);
  return [config, device.rack];
};

/** @returns the channels that a plot of the read task shows. */
export const readPlotChannels: Task.PlotChannels = (config) =>
  Task.enabledChannels(READ_SCHEMAS.config.parse(config).channels);

export const Read = Task.wrapForm({
  type: "opc_read",
  Properties,
  Form: TaskForm,
  schemas: READ_SCHEMAS,
  deployConfigZ: deployReadConfigZ,
  onConfigure,
  plotChannels: readPlotChannels,
});

export const useCreateRead = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const ReadSelectable: Selector.Selectable = {
  type: READ_TYPE,
  title: "OPC UA read task",
  label: "Read",
  icon: <Icon.Logo.OPCUA />,
  useOnSelect: useCreateRead,
};
