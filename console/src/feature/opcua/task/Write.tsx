// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/opcua/task/Task.css";

import { channel, NotFoundError, opcua, type Synnax } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Header } from "@synnaxlabs/lyra/header";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Channel } from "@synnaxlabs/pluto";
import { caseconv, DataType, errors, id, primitive } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";

import { Select } from "@/feature/opcua/device/Select";
import {
  type Device,
  migrateProperties,
  type Properties as DeviceProperties,
  SCHEMAS,
} from "@/feature/opcua/device/types";
import { findFreeName } from "@/feature/opcua/task/findFreeName";
import {
  type ChannelKeyAndIDGetter,
  createForm,
  type ExtraItemProps,
} from "@/feature/opcua/task/Form";
import {
  deployWriteConfigZ,
  type InputArgument,
  isMethod,
  type MethodArgument,
  type NodeDetails,
  type ScannedNode,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteChannel,
  type WriteSchemas,
} from "@/feature/opcua/task/types";
import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const Properties = () => (
  <>
    <Select />
    <Task.Fields.AutoStart />
  </>
);

// A command channel that only starts a method takes a value that the call ignores.
const TRIGGER_DATA_TYPE = DataType.UINT8.toString();

/** @returns the type of the argument that takes the command, if it has one. */
const commandDataType = (args: MethodArgument[]): string | undefined => {
  const type = args.find(({ source }) => source === "command")?.dataType;
  return primitive.isNonZero(type) ? type : undefined;
};

const convertArgument = (
  { name, dataType }: InputArgument,
  i: number,
): MethodArgument =>
  i === 0 && dataType !== ""
    ? { source: "command", name, dataType }
    : { source: "value", value: "", name, dataType };

// A write goes to a whole node, so a member of a structure is not a write channel.
const convertNodeToChannel = (
  node: ScannedNode,
  details?: NodeDetails,
): WriteChannel | null => {
  if (node.field !== "") return null;
  const base = {
    nodeName: node.name,
    nodeId: node.nodeId,
    name: "",
    cmdChannel: 0,
    disabled: false,
  };
  if (node.nodeClass === "Method")
    return opcua.methodWriteChannelZ.parse({
      ...base,
      key: id.create(),
      type: "method",
      objectId: node.parent ?? "",
      arguments: (details?.inputArguments ?? []).map(convertArgument),
    });
  if (node.nodeClass !== "Variable") return null;
  return { ...base, key: node.nodeId, type: "node", dataType: node.dataType };
};

const VALUE_INPUT_PROPS = { placeholder: "Value" } as const;

interface ArgumentProps extends ExtraItemProps {
  index: number;
}

const Argument = ({ path, index }: ArgumentProps): ReactElement | null => {
  const argPath = `${path}.arguments.${index}`;
  const arg = PForm.useFieldValue<MethodArgument>(argPath, { optional: true });
  if (arg == null) return null;
  const command = arg.source === "command";
  const handleSourceChange = (
    source: opcua.ArgumentSource,
    { get, set }: PForm.ContextValue,
  ) => {
    const args = get<MethodArgument[]>(`${path}.arguments`).value.map(
      ({ name, dataType, ...a }, i): MethodArgument =>
        i === index && source === "command"
          ? { source, name, dataType }
          : { source: "value", value: "value" in a ? a.value : "", name, dataType },
    );
    set(`${path}.arguments`, args);
  };
  const label = arg.dataType === "" ? arg.name : `${arg.name} (${arg.dataType})`;
  return (
    <Flex.Box x align="start" gap="small">
      {command ? (
        <Input.Item label={label} grow padHelpText>
          <Text.Text level="small" color={9}>
            Each command gives the value.
          </Text.Text>
        </Input.Item>
      ) : (
        <PForm.TextField
          path={`${argPath}.value`}
          label={label}
          grow
          inputProps={VALUE_INPUT_PROPS}
        />
      )}
      <PForm.Field<opcua.ArgumentSource>
        path={`${argPath}.source`}
        label="Command"
        onChange={handleSourceChange}
      >
        {({ value, onChange }) => (
          <Input.Switch
            value={value === "command"}
            onChange={(next) => onChange(next ? "command" : "value")}
          />
        )}
      </PForm.Field>
    </Flex.Box>
  );
};

const OBJECT_INPUT_PROPS = { placeholder: "ns=2;s=Pump" } as const;

interface ReplacementProps {
  bound: channel.Channel;
  dataType: string;
}

const Replacement = ({ bound, dataType }: ReplacementProps): ReactElement => (
  <Text.Text level="small" status="warning">
    {`The command takes ${dataType}, but ${bound.name} is ${bound.dataType.toString()}. Configuring the task replaces ${bound.name} with a new channel.`}
  </Text.Text>
);

const MethodDetails = ({ path }: ExtraItemProps): ReactElement | null => {
  const ch = PForm.useFieldValue<WriteChannel>(path, { optional: true });
  const method = ch != null && isMethod(ch) ? ch : null;
  const query =
    method != null && method.cmdChannel !== 0 ? { key: method.cmdChannel } : null;
  const { data: bound } = Channel.useResult(query);
  if (method == null) return null;
  const dataType = commandDataType(method.arguments);
  return (
    <Flex.Box y empty className={CSS.B("opc-method")}>
      <Header.Header>
        <Header.Title weight={500} color={10} level="small">
          {`Call ${method.nodeName}`}
        </Header.Title>
      </Header.Header>
      <Flex.Box y gap="small" className={CSS.BE("opc-method", "fields")}>
        <PForm.TextField
          path={`${path}.objectId`}
          label="Object"
          inputProps={OBJECT_INPUT_PROPS}
        />
        {method.arguments.length === 0 ? (
          <Text.Text level="small" color={9}>
            The method takes no arguments. Any command calls it.
          </Text.Text>
        ) : (
          method.arguments.map((_, i) => <Argument key={i} path={path} index={i} />)
        )}
        {bound != null && dataType != null && !bound.dataType.equals(dataType) && (
          <Replacement bound={bound} dataType={dataType} />
        )}
      </Flex.Box>
    </Flex.Box>
  );
};

const details = Component.renderProp(MethodDetails);

const getChannelKeyAndID: ChannelKeyAndIDGetter<WriteChannel> = ({
  cmdChannel,
  key,
}) => ({
  key: cmdChannel,
  id: Task.getChannelNameID(key, "cmd"),
});

interface ContextMenuItemProps extends Task.ContextMenuItemProps<WriteChannel> {}

const ContextMenuItem: React.FC<ContextMenuItemProps> = ({ channels, keys }) => {
  const canRename = Access.useUpdateGranted(channel.TYPE_ONTOLOGY_ID);
  if (keys.length !== 1 || !canRename) return null;
  const key = keys[0];
  const cmdChannel = channels.find((ch) => ch.key === key)?.cmdChannel;
  if (cmdChannel == null) return null;
  const handleRename = () => Text.edit(Task.getChannelNameID(key, "cmd"));
  return (
    <>
      <ContextMenu.RenameItem onClick={handleRename} />
      <Menu.Divider />
    </>
  );
};

const contextMenuItems = Component.renderProp(ContextMenuItem);

const getChannelByNodeID = (props: DeviceProperties, nodeId: string) =>
  props.write.channels[nodeId] ?? props.write.channels[caseconv.snakeToCamel(nodeId)];

// The device binds a command channel to each node. A method channel holds its own.
const resolve = (device: Device, ch: WriteChannel): Partial<WriteChannel> =>
  isMethod(ch)
    ? {}
    : { cmdChannel: getChannelByNodeID(device.properties, ch.nodeId) ?? 0 };

const TaskForm: FC = createForm<WriteChannel>({
  convertNodeToChannel,
  getChannelKeyAndID,
  contextMenuItems,
  resolve,
  details,
});

const getInitialValues: Task.GetInitialValues<WriteSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = WRITE_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: "OPC UA write task", type: WRITE_TYPE, config: cfg };
};

const defaultCommandName = ({ nodeName }: WriteChannel): string =>
  `${channel.escapeInvalidName(nodeName)}_cmd`;

const commandName = (ch: WriteChannel): string =>
  primitive.isNonZero(ch.name) ? ch.name : defaultCommandName(ch);

interface Command {
  name: string;
  dataType: string;
}

/** @returns the command channels it creates, each with its index. */
const createCommands = async (
  client: Synnax,
  commands: Command[],
): Promise<channel.Channel[]> => {
  if (commands.length === 0) return [];
  const indexes = await client.channels.create(
    commands.map(({ name }) => ({
      name: `${name}_time`,
      dataType: "timestamp",
      isIndex: true,
    })),
  );
  return await client.channels.create(
    commands.map(({ name, dataType }, i) => ({
      name,
      dataType,
      index: indexes[i].key,
    })),
  );
};

/** Binds a new command channel to each node that the device has no channel for. */
const bindNodeCommands = async (
  client: Synnax,
  dev: Device,
  nodes: opcua.NodeWriteChannel[],
): Promise<void> => {
  const unbound: opcua.NodeWriteChannel[] = [];
  for (const node of nodes) {
    const key = getChannelByNodeID(dev.properties, node.nodeId);
    if (key)
      try {
        await client.channels.retrieve(key);
        continue;
      } catch (e) {
        if (!NotFoundError.matches(e)) throw errors.fromUnknown(e);
      }
    unbound.push(node);
  }
  const created = await createCommands(
    client,
    unbound.map((node) => ({ name: commandName(node), dataType: node.dataType })),
  );
  if (created.length === 0) return;
  if (
    dev.properties.write.channels == null ||
    Array.isArray(dev.properties.write.channels)
  )
    dev.properties.write.channels = {};
  created.forEach(({ key }, i) => {
    dev.properties.write.channels[unbound[i].nodeId] = key;
  });
};

/**
 * @returns the new command channel of each method that has no channel, or whose command
 * now takes another type.
 */
const createMethodCommands = async (
  client: Synnax,
  methods: opcua.MethodWriteChannel[],
): Promise<Map<string, channel.Key>> => {
  const keys: string[] = [];
  const commands: Command[] = [];
  // The Core holds one channel per name, and a method can take many channels, so these
  // take free names. A name that the user set, and its index name, is never free.
  const reserved = new Set(
    methods
      .map(({ name }) => name)
      .filter(primitive.isNonZero)
      .flatMap((name) => [name, `${name}_time`]),
  );
  for (const method of methods) {
    const dataType = commandDataType(method.arguments);
    let replaced: string | undefined;
    if (method.cmdChannel !== 0)
      try {
        const existing = await client.channels.retrieve(method.cmdChannel);
        if (dataType == null || existing.dataType.equals(dataType)) continue;
        replaced = existing.name;
      } catch (e) {
        if (!NotFoundError.matches(e)) throw errors.fromUnknown(e);
      }
    let name = commandName(method);
    if (replaced != null || !primitive.isNonZero(method.name))
      name = await findFreeName(client, replaced ?? name, reserved);
    keys.push(method.key);
    commands.push({ name, dataType: dataType ?? TRIGGER_DATA_TYPE });
  }
  const created = await createCommands(client, commands);
  return new Map(created.map(({ key }, i) => [keys[i], key]));
};

const onConfigure: Task.OnConfigure<WriteSchemas["config"]> = async (
  client,
  config,
) => {
  let dev = await client.devices.retrieve({
    key: config.device,
    schemas: SCHEMAS,
  });
  try {
    dev = { ...dev, properties: migrateProperties(dev.properties) };
    const methods = config.channels.filter(isMethod);
    await bindNodeCommands(
      client,
      dev,
      config.channels.filter((ch): ch is opcua.NodeWriteChannel => ch.type === "node"),
    );
    const created = await createMethodCommands(client, methods);
    config.channels = config.channels.map((ch) =>
      isMethod(ch)
        ? { ...ch, cmdChannel: created.get(ch.key) ?? ch.cmdChannel }
        : { ...ch, cmdChannel: getChannelByNodeID(dev.properties, ch.nodeId) },
    );
  } finally {
    await client.devices.create(dev, SCHEMAS);
  }
  return [config, dev.rack];
};

export const Write = Task.wrapForm({
  Properties,
  Form: TaskForm,
  schemas: WRITE_SCHEMAS,
  deployConfigZ: deployWriteConfigZ,
  type: "opc_write",
  getInitialValues,
  onConfigure,
});

export const useCreateWrite = Task.createUseCreate({
  getInitialValues,
});

export const WriteSelectable = Selector.createSelectable({
  type: WRITE_TYPE,
  title: "OPC UA write task",
  icon: <Icon.Logo.OPCUA />,
  useOnSelect: useCreateWrite,
});
