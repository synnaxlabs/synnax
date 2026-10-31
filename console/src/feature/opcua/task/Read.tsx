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
import { caseconv, DataType, errors, primitive, record } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";

import { Select } from "@/feature/opcua/device/Select";
import {
  type Device,
  migrateProperties,
  type Properties as DeviceProperties,
  SCHEMAS,
} from "@/feature/opcua/device/types";
import { type ChannelKeyAndIDGetter, createForm } from "@/feature/opcua/task/Form";
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
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

type ChannelSource = Pick<ReadChannel, "nodeId" | "field">;

const getChannel = (props: DeviceProperties, { nodeId, field }: ChannelSource) =>
  field === ""
    ? (props.read.channels[nodeId] ??
      props.read.channels[caseconv.snakeToCamel(nodeId)] ??
      0)
    : (props.read.fields[nodeId]?.[field] ?? 0);

const setChannel = (
  props: DeviceProperties,
  { nodeId, field }: ChannelSource,
  key: channel.Key,
) => {
  if (field === "") props.read.channels[nodeId] = key;
  else (props.read.fields[nodeId] ??= {})[field] = key;
};

interface IsIndexItemProps {
  path: string;
  snapshot?: boolean;
}

const IsIndexItem = ({ path }: IsIndexItemProps): ReactElement => {
  const dataType = PForm.useFieldValue<string>(`${path}.dataType`, { optional: true });
  return (
    <PForm.SwitchField
      path={`${path}.isIndex`}
      label="Use as index"
      x
      align="center"
      showHelpText={false}
      visible={DataType.TIMESTAMP.equals(dataType ?? "")}
    />
  );
};

const isIndexItem = Component.renderProp(IsIndexItem);

const HISTORY_BACKFILL_INPUT_PROPS = {
  tooltip: "Fill the gap of a stop or an outage from the history of each node",
} as const;

const MODE_NAMES: Record<opcua.ReadMode, string> = {
  sampled: "Sampled",
  on_change: "On change",
};

const ModeButtons = (p: PSelect.ButtonsProps<opcua.ReadMode>) => (
  <PSelect.Buttons {...p}>
    {record.entries(MODE_NAMES).map(([mode, name]) => (
      <PSelect.Item key={mode} itemKey={mode}>
        {name}
      </PSelect.Item>
    ))}
  </PSelect.Buttons>
);

const renderMode = Component.renderProp(ModeButtons);

const Properties = (): ReactElement => {
  const arrayMode = PForm.useFieldValue<boolean>("config.arrayMode");
  const onChangeMode =
    PForm.useFieldValue<opcua.ReadMode>("config.mode") === "on_change";
  return (
    <>
      <Select />
      <Flex.Box x>
        <PForm.Field<opcua.ReadMode>
          path="config.mode"
          label="Mode"
          onChange={(mode, { get, set }) => {
            // On change mode reads each value at its own time, never as an array.
            if (mode !== "on_change" || !get<boolean>("config.arrayMode").value) return;
            set("config.arrayMode", false);
            set("config.arraySize", 1);
            set("config.streamRate", 25);
          }}
        >
          {renderMode}
        </PForm.Field>
        <Task.Fields.SampleRate />
        {!onChangeMode && (
          <PForm.SwitchField
            label="Array sampling"
            path="config.arrayMode"
            onChange={(value, { set }) => {
              // always set the array size to 1 for either the default in array mode or
              // an array size of 1 in stream mode.
              set("config.arraySize", 1);
              if (!value) set("config.streamRate", 25);
            }}
          />
        )}
        {arrayMode ? (
          <PForm.NumericField
            label="Array size"
            path="config.arraySize"
            className={CSS.B("opc-array-size-field")}
          />
        ) : (
          <Task.Fields.StreamRate />
        )}
        <Task.Fields.DataSaving />
        <Task.Fields.AutoStart />
        {onChangeMode ? (
          <PForm.SwitchField
            label="History backfill"
            path="config.historyBackfillEnabled"
            inputProps={HISTORY_BACKFILL_INPUT_PROPS}
          />
        ) : (
          <Task.Fields.NaNFill />
        )}
      </Flex.Box>
    </>
  );
};

const convertNodeToChannel = (node: ScannedNode): ReadChannel | null =>
  node.nodeClass !== "Variable"
    ? null
    : {
        key: node.field === "" ? node.nodeId : nodeIdentity(node),
        nodeName: node.name,
        nodeId: node.nodeId,
        field: node.field,
        channel: 0,
        disabled: false,
        isIndex: false,
        dataType: node.dataType,
        name: "",
      };

const getChannelKeyAndID: ChannelKeyAndIDGetter<ReadChannel> = ({ channel, key }) => ({
  key: channel,
  id: Task.getChannelNameID(key),
});

const resolve = (device: Device, ch: ReadChannel) => ({
  channel: getChannel(device.properties, ch),
});

const TaskForm: FC = createForm<ReadChannel>({
  convertNodeToChannel,
  getChannelKeyAndID,
  contextMenuItems: Task.readChannelContextMenuItem,
  children: isIndexItem,
  resolve,
  indexed: true,
});

const getInitialValues: Task.GetInitialValues<ReadSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = READ_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: "OPC UA read task", type: READ_TYPE, config: cfg };
};

const channelName = ({ name, nodeName }: ReadChannel) =>
  primitive.isNonZero(name) ? name : channel.escapeInvalidName(nodeName, true);

/**
 * Returns the index that the channels of a sampled task share: the channel of its index
 * node, or else the shared index of a channel that the device already binds. Returns
 * null when the task needs a new index.
 */
const findSharedIndex = async (
  client: Synnax,
  config: ReadConfig,
  device: Device,
): Promise<channel.Key | null> => {
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
        return key;
      } catch (e) {
        if (!NotFoundError.matches(e)) throw errors.fromUnknown(e);
      }
    return null;
  }

  // The index of an On change channel belongs to that channel alone.
  for (const ch of config.channels) {
    const existingChannelKey = getChannel(device.properties, ch);
    if (existingChannelKey)
      try {
        const { index } = await client.channels.retrieve(existingChannelKey);
        if (device.properties.read.indexes.includes(index)) return index;
      } catch (e) {
        if (!NotFoundError.matches(e)) throw errors.fromUnknown(e);
      }
  }
  return null;
};

/** Creates the index of a sampled task, as the channel of its index node if it has one. */
const createSharedIndex = async (
  client: Synnax,
  config: ReadConfig,
  device: Device,
  taskName: string,
): Promise<channel.Key> => {
  const indexChannelInTaskConfig = config.channels.find(({ isIndex }) => isIndex);
  const { key } = await client.channels.create({
    name:
      indexChannelInTaskConfig != null
        ? channel.escapeInvalidName(indexChannelInTaskConfig.nodeName, true)
        : `${channel.escapeInvalidName(device.name)}_time_for_${channel.escapeInvalidName(taskName)}`,
    dataType: "timestamp",
    isIndex: true,
  });
  device.properties.read.indexes.push(key);
  if (indexChannelInTaskConfig != null)
    setChannel(device.properties, indexChannelInTaskConfig, key);
  return key;
};

/**
 * Binds each channel of a sampled task to a channel on the index that they share.
 * Checks each channel that exists before it creates anything, so a rejected deploy
 * leaves no new channels.
 */
const bindSharedIndex = async (
  client: Synnax,
  config: ReadConfig,
  device: Device,
  name: string,
): Promise<void> => {
  const index = await findSharedIndex(client, config, device);
  const toCreate: ReadChannel[] = [];
  for (const ch of config.channels) {
    if (ch.isIndex) continue;
    const exKey = getChannel(device.properties, ch);
    if (!exKey) {
      toCreate.push(ch);
      continue;
    }
    try {
      const rCh = await client.channels.retrieve(exKey);
      // A channel that exists is never on a new index.
      if (rCh.index !== index)
        throw new Error(
          `Channel ${ch.nodeName} already exists as ${rCh.name}. Move all channels from ${name} to the task that reads ${rCh.name}.`,
        );
    } catch (e) {
      if (NotFoundError.matches(e)) toCreate.push(ch);
      else throw errors.fromUnknown(e);
    }
  }
  const shared = index ?? (await createSharedIndex(client, config, device, name));
  if (toCreate.length === 0) return;
  const channels = await client.channels.create(
    toCreate.map((ch) => ({
      dataType: ch.dataType,
      name: channelName(ch),
      index: shared,
    })),
  );
  channels.forEach(({ key }, i) => setChannel(device.properties, toCreate[i], key));
};

/**
 * Binds each channel of an On change task to a channel with an index of its own, since
 * each channel writes at the times that its node gives.
 */
const bindOwnIndexes = async (
  client: Synnax,
  config: ReadConfig,
  device: Device,
): Promise<void> => {
  const toCreate: ReadChannel[] = [];
  for (const ch of config.channels) {
    const exKey = getChannel(device.properties, ch);
    if (!exKey) {
      toCreate.push(ch);
      continue;
    }
    try {
      const rCh = await client.channels.retrieve(exKey);
      if (device.properties.read.indexes.includes(rCh.index))
        throw new Error(
          `Channel ${ch.nodeName} already exists as ${rCh.name}, which shares an index with other channels. On change mode needs a channel with its own index. Read ${ch.nodeName} in the task that reads ${rCh.name}.`,
        );
    } catch (e) {
      if (NotFoundError.matches(e)) toCreate.push(ch);
      else throw errors.fromUnknown(e);
    }
  }
  if (toCreate.length === 0) return;
  const indexes = await client.channels.create(
    toCreate.map((ch) => ({
      name: `${channelName(ch)}_time`,
      dataType: "timestamp",
      isIndex: true,
    })),
  );
  const channels = await client.channels.create(
    toCreate.map((ch, i) => ({
      dataType: ch.dataType,
      name: channelName(ch),
      index: indexes[i].key,
    })),
  );
  channels.forEach(({ key }, i) => setChannel(device.properties, toCreate[i], key));
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
  const device = await client.devices.create(
    { ...previous, properties: migrateProperties(previous.properties) },
    SCHEMAS,
  );
  if (config.mode === "on_change") await bindOwnIndexes(client, config, device);
  else await bindSharedIndex(client, config, device, name);
  config.channels = config.channels.map((c) => ({
    ...c,
    channel: getChannel(device.properties, c),
  }));
  await client.devices.create(device, SCHEMAS);
  return [config, device.rack];
};

export const Read = Task.wrapForm({
  type: "opc_read",
  Properties,
  Form: TaskForm,
  schemas: READ_SCHEMAS,
  deployConfigZ: deployReadConfigZ,
  getInitialValues,
  onConfigure,
});

export const useCreateRead = Task.createUseCreate({
  getInitialValues,
});

export const ReadSelectable = Selector.createSelectable({
  type: READ_TYPE,
  title: "OPC UA read task",
  icon: <Icon.Logo.OPCUA />,
  useOnSelect: useCreateRead,
});
