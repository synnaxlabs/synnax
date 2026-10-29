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
import { Header } from "@synnaxlabs/lyra/header";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { Access } from "@synnaxlabs/pluto";
import { caseconv, DataType, errors, id, primitive } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";

import { type HaulItem } from "@/feature/opcua/device/Browser";
import { Select } from "@/feature/opcua/device/Select";
import {
  type Device,
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
  deployWriteConfigZ,
  isMethod,
  type MethodArgument,
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

const commandDataType = (args: MethodArgument[]): string =>
  args.find(({ source }) => source === "command")?.dataType ?? TRIGGER_DATA_TYPE;

// A write goes to a whole node, so a member of a structure is not a write channel.
const convertHaulItemToChannel = ({ data }: HaulItem): WriteChannel | null => {
  if (data.field !== "") return null;
  const base = { nodeName: data.name, nodeId: data.nodeId, name: "" };
  if (data.nodeClass === "Method") {
    // The first argument takes the command, unless its type has no Synnax type.
    const args = (data.inputArguments ?? []).map(
      ({ name, dataType }, i): MethodArgument => ({
        source: i === 0 && dataType !== "" ? "command" : "value",
        value: "",
        name,
        dataType,
      }),
    );
    return {
      ...base,
      key: id.create(),
      cmdChannel: 0,
      disabled: false,
      type: "method",
      objectId: data.parent ?? "",
      arguments: args,
      dataType: commandDataType(args),
    };
  }
  if (data.nodeClass !== "Variable") return null;
  return {
    ...base,
    key: data.nodeId,
    cmdChannel: 0,
    disabled: false,
    type: "node",
    objectId: "",
    arguments: [],
    dataType: data.dataType,
  };
};

interface ArgumentProps extends ExtraItemProps {
  index: number;
}

const Argument = ({ path, index }: ArgumentProps): ReactElement | null => {
  const argPath = `${path}.arguments.${index}`;
  const arg = PForm.useFieldValue<MethodArgument>(argPath, { optional: true });
  if (arg == null) return null;
  const command = arg.source === "command";
  // One argument at most takes the command, and the command channel takes its type.
  const handleSourceChange = (
    source: opcua.ArgumentSource,
    { get, set }: PForm.ContextValue,
  ) => {
    const args = get<MethodArgument[]>(`${path}.arguments`).value.map((a, i) => ({
      ...a,
      source: i === index ? source : "value",
    }));
    set(`${path}.arguments`, args);
    set(`${path}.dataType`, commandDataType(args));
  };
  return (
    <Flex.Box x align="start" gap="small">
      <PForm.TextField
        path={`${argPath}.value`}
        label={arg.dataType === "" ? arg.name : `${arg.name} (${arg.dataType})`}
        grow
        inputProps={{
          disabled: command,
          placeholder: command ? "Value of the command" : "Value",
        }}
      />
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

const MethodDetails = ({ path }: ExtraItemProps): ReactElement | null => {
  const ch = PForm.useFieldValue<WriteChannel>(path, { optional: true });
  if (ch == null || !isMethod(ch)) return null;
  return (
    <Flex.Box y empty className={CSS.B("opc-method")}>
      <Header.Header>
        <Header.Title weight={500} color={10} level="small">
          {`Call ${ch.nodeName}`}
        </Header.Title>
      </Header.Header>
      <Flex.Box y gap="small" className={CSS.BE("opc-method", "fields")}>
        <PForm.TextField
          path={`${path}.objectId`}
          label="Object"
          inputProps={OBJECT_INPUT_PROPS}
        />
        {ch.arguments.length === 0 ? (
          <Text.Text level="small" color={9}>
            The method takes no arguments. Any command calls it.
          </Text.Text>
        ) : (
          ch.arguments.map((_, i) => <Argument key={i} path={path} index={i} />)
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
  convertHaulItemToChannel,
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

/**
 * @returns base, or base with the lowest numeric suffix, such that no channel and no
 * name in reserved holds it or its index name. Adds the name to reserved.
 */
const findFreeName = async (
  client: Synnax,
  base: string,
  reserved: Set<string>,
): Promise<string> => {
  for (let n = 1; ; n++) {
    const name = n === 1 ? base : `${base}_${n}`;
    if (reserved.has(name)) continue;
    const taken = await client.channels.retrieve([name, `${name}_time`]);
    if (taken.length > 0) continue;
    reserved.add(name);
    return name;
  }
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
    const commandsToCreate: WriteChannel[] = [];
    // The Core holds one channel per name. A method can take many channels, and a
    // channel whose command changes type replaces its channel, so these take free
    // names.
    const names = new Map<string, string>();
    const reserved = new Set<string>();
    for (const channel of config.channels) {
      const method = isMethod(channel);
      const key = method
        ? channel.cmdChannel
        : getChannelByNodeID(dev.properties, channel.nodeId);
      let replaced: string | undefined;
      if (key)
        try {
          const existing = await client.channels.retrieve(key);
          // A method channel's type follows the argument that takes the command.
          if (!method || existing.dataType.equals(channel.dataType)) continue;
          replaced = existing.name;
        } catch (e) {
          if (!NotFoundError.matches(e)) throw errors.fromUnknown(e);
        }

      commandsToCreate.push(channel);
      if (!method || (replaced == null && primitive.isNonZero(channel.name))) continue;
      const base = replaced ?? defaultCommandName(channel);
      names.set(channel.key, await findFreeName(client, base, reserved));
    }
    const commandName = (ch: WriteChannel): string =>
      names.get(ch.key) ??
      (primitive.isNonZero(ch.name) ? ch.name : defaultCommandName(ch));
    const created = new Map<string, channel.Key>();
    if (commandsToCreate.length > 0) {
      if (
        dev.properties.write.channels == null ||
        Array.isArray(dev.properties.write.channels)
      )
        dev.properties.write.channels = {};
      const commandIndexes = await client.channels.create(
        commandsToCreate.map((ch) => ({
          name: `${commandName(ch)}_time`,
          dataType: "timestamp",
          isIndex: true,
        })),
      );
      const commands = await client.channels.create(
        commandsToCreate.map((ch, i) => ({
          name: commandName(ch),
          dataType: ch.dataType,
          index: commandIndexes[i].key,
        })),
      );
      commands.forEach(({ key }, i) => {
        const ch = commandsToCreate[i];
        if (isMethod(ch)) created.set(ch.key, key);
        else dev.properties.write.channels[ch.nodeId] = key;
      });
    }
    config.channels = config.channels.map((c) => ({
      ...c,
      cmdChannel: isMethod(c)
        ? (created.get(c.key) ?? c.cmdChannel)
        : getChannelByNodeID(dev.properties, c.nodeId),
    }));
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
