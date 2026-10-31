// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/opcua/task/Task.css";

import { channel, DisconnectedError, status, type Synnax } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { type Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Header as PHeader } from "@synnaxlabs/lyra/header";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Synnax as PSynnax } from "@synnaxlabs/pluto";
import { id, primitive, TimeSpan } from "@synnaxlabs/x";
import { type FC, useState } from "react";

import { Browser } from "@/feature/opcua/device/Browser";
import {
  canDropHaulItem,
  HAUL_TYPE,
  type HaulItem,
  isHaulItem,
} from "@/feature/opcua/device/haul";
import { use } from "@/feature/opcua/device/queries";
import { executeScanCommand } from "@/feature/opcua/device/scanCommand";
import { type Device } from "@/feature/opcua/device/types";
import { useConnectModal } from "@/feature/opcua/device/useConnectModal";
import { CSV } from "@/feature/opcua/task/csv";
import {
  BROWSE_VARIABLES_COMMAND_TYPE,
  browseVariablesResponseZ,
  type Channel,
  nodeIdentity,
  type ScannedNode,
} from "@/feature/opcua/task/types";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { Modals } from "@/platform/modals";
import { Runtime } from "@/platform/runtime";
import { Task } from "@/platform/task";

export interface ExtraItemProps {
  path: string;
}

export interface ChannelKeyAndIDGetter<C extends Channel> {
  (channel: C): { id: string; key: channel.Key };
}

interface ChannelListItemProps<C extends Channel> extends Omit<
  Task.ChannelListItemProps,
  "children"
> {
  children: Component.RenderProp<ExtraItemProps>;
  getChannelKeyAndID: ChannelKeyAndIDGetter<C>;
}

const ChannelListItem = <C extends Channel>({
  children,
  getChannelKeyAndID,
  ...rest
}: ChannelListItemProps<C>) => {
  const path = `config.channels.${rest.itemKey}`;
  const item = PForm.useFieldValue<C>(path);
  if (item == null) return null;
  const { nodeName, nodeId } = item;
  const opcNode = nodeId.length > 0 ? nodeId : "No node selected";
  let opcNodeColor;
  if (opcNode === "No node selected") opcNodeColor = "var(--pluto-warning-z)";
  const { key: channel, id } = getChannelKeyAndID(item);
  return (
    <Select.Item {...rest} justify="between" align="center" rightAligned>
      <Flex.Box
        direction="y"
        gap="small"
        className={CSS.BE("channel-name", "container")}
      >
        <Task.ChannelName
          weight={500}
          color={10}
          level="p"
          channel={channel}
          id={id}
          namePath={`${path}.name`}
        />
        <Flex.Box x align="center" gap="tiny">
          <Icon.Variable color={9} />
          <Text.Text
            level="small"
            weight={350}
            color={opcNodeColor ?? 9}
            gap="small"
            overflow="fade"
          >
            {nodeName} {opcNode}
          </Text.Text>
        </Flex.Box>
      </Flex.Box>
      <Flex.Box direction="x" align="center">
        {children({ path })}
        <Task.EnabledCheckbox path={`${path}.disabled`} />
      </Flex.Box>
    </Select.Item>
  );
};

const CHANNELS_PATH = "config.channels";

const EXPORT_LABEL = "Export channels to CSV";

const IMPORT_LABEL = "Import channels from CSV";

interface HeaderProps<C extends Channel> extends Pick<
  ChannelListProps<C>,
  "convertNodeToChannel" | "indexed"
> {
  onImport: (channels: C[]) => void;
}

const Header = <C extends Channel>({
  convertNodeToChannel,
  indexed,
  onImport,
}: HeaderProps<C>) => {
  const ctx = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const download = Runtime.useDownload();
  const handleError = Status.useErrorHandler();
  const getChannels = () => ctx.get<C[]>(CHANNELS_PATH).value;
  const handleExport = () =>
    handleError(
      async () =>
        await download({
          stream: Runtime.toStream(
            new TextEncoder().encode(CSV.encode(getChannels(), { indexed })),
          ),
          name: `${ctx.get<string>("name").value}_channels`,
          extension: "csv",
        }),
      "Failed to export channels",
    );
  const handleImport = () =>
    handleError(async () => {
      const file = await Runtime.pickFiles({ title: IMPORT_LABEL, extension: "csv" });
      if (file == null) return;
      const text = new TextDecoder().decode(await Runtime.toBytes(await file.read()));
      onImport(
        CSV.decode(text, {
          existing: getChannels(),
          convert: convertNodeToChannel,
          indexed,
        }),
      );
    }, "Failed to import channels");
  return (
    <PHeader.Header className={CSS.B("opc-task-header")}>
      <PHeader.Title weight={500} color={10} level="p">
        Channels
      </PHeader.Title>
      <PHeader.Actions>
        {!isPreview && (
          <Button.Button
            onClick={handleImport}
            tooltip={IMPORT_LABEL}
            aria-label={IMPORT_LABEL}
            size="medium"
            variant="text"
          >
            <Icon.Import />
          </Button.Button>
        )}
        <Button.Button
          onClick={handleExport}
          tooltip={EXPORT_LABEL}
          aria-label={EXPORT_LABEL}
          size="medium"
          variant="text"
        >
          <Icon.Export />
        </Button.Button>
      </PHeader.Actions>
    </PHeader.Header>
  );
};

const EmptyContent = () => (
  <Flex.Box center>
    <Text.Text status="disabled" className={CSS.B("opc-task-empty-text")}>
      No channels added. Drag a variable{" "}
      <Icon.Variable className={CSS.B("opc-task-variable-icon")} /> or a folder from the
      browser to add channels to the task.
    </Text.Text>
  </Flex.Box>
);

const VARIABLE_NODE_CLASS = "Variable";
const OBJECT_NODE_CLASS = "Object";

const isVariableHaulItem = (item: Haul.Item): item is HaulItem =>
  isHaulItem(item) && item.data.nodeClass === VARIABLE_NODE_CLASS;

const isFolderHaulItem = (item: Haul.Item): item is HaulItem =>
  isHaulItem(item) && item.data.nodeClass === OBJECT_NODE_CLASS;

const canDrop = ({ items }: Haul.DraggingState): boolean =>
  items.some((item) => isVariableHaulItem(item) || isFolderHaulItem(item));

const SEARCH_TIMEOUT = TimeSpan.seconds(30);

const SEARCH_LIMIT_NOTE =
  "The search stopped at a limit before it found every variable. Drag folders further down to add the rest.";

const NAMING_NOTE =
  "Each channel takes its name from the folder and the path to the variable.";

const MOVED_NOTE = "The task moved to another server during the search.";

interface Search {
  nodes: ScannedNode[];
  truncated: boolean;
  skipped: string[];
}

/** @returns the variables under the folders, named from the folder and their path. */
const findVariables = async (
  client: Synnax,
  device: Device,
  folders: ScannedNode[],
): Promise<Search> => {
  const search: Search = { nodes: [], truncated: false, skipped: [] };
  // The Driver runs one scan command at a time, so a search sent before the last one
  // ends would spend its timeout in the queue.
  for (const folder of folders) {
    const { channels, truncated, skipped } = await executeScanCommand(
      client,
      device.rack,
      {
        type: BROWSE_VARIABLES_COMMAND_TYPE,
        args: { connection: device.properties.connection, node_id: folder.nodeId },
        schema: browseVariablesResponseZ,
        timeout: SEARCH_TIMEOUT,
      },
    );
    channels.forEach(({ path, ...node }) =>
      search.nodes.push({ ...node, name: [folder.name, ...path].join("_") }),
    );
    search.truncated ||= truncated;
    search.skipped.push(...skipped);
  }
  return search;
};

const deployName = ({ name, nodeName }: Channel): string =>
  channel.escapeInvalidName(primitive.isNonZero(name) ? name : nodeName, true);

// The Core accepts two channels with one name, so a new channel gets a number at the
// end of its node name when the task already has the name.
const withFreeName = <C extends Channel>(ch: C, names: Set<string>): C => {
  let free = ch;
  for (let i = 2; names.has(deployName(free)); i++)
    free = { ...ch, nodeName: `${ch.nodeName}_${i}` };
  return free;
};

/** @returns why the found variables are not every variable, or null if they are. */
const describeMissing = ({ truncated, skipped }: Search): string | null => {
  const notes: string[] = [];
  if (truncated) notes.push(SEARCH_LIMIT_NOTE);
  const count = skipped.length;
  if (count > 0)
    notes.push(
      `The server did not browse ${count} ${count === 1 ? "object" : "objects"}, so their variables are missing. First failure: ${skipped[0]}`,
    );
  return notes.length === 0 ? null : notes.join(" ");
};

interface ChannelListProps<C extends Channel> extends Pick<
  Task.ChannelListProps<C>,
  "contextMenuItems" | "resolve"
> {
  children: Component.RenderProp<ExtraItemProps>;
  device: Device;
  /** @returns the channel of a node, or null if the task cannot hold it. */
  convertNodeToChannel: (node: ScannedNode) => C | null;
  getChannelKeyAndID: ChannelKeyAndIDGetter<C>;
  /** True when a channel can be the task's index, which adds a CSV index column. */
  indexed?: boolean;
}

const ChannelList = <C extends Channel>({
  device,
  children,
  convertNodeToChannel,
  getChannelKeyAndID,
  indexed,
  ...rest
}: ChannelListProps<C>) => {
  const ctx = PForm.useContext();
  const fieldListReturn = PForm.useFieldList<C["key"], C>(CHANNELS_PATH);
  const { data, push } = fieldListReturn;
  const client = PSynnax.use();
  const confirm = Modals.useConfirm();
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  const newChannels = (nodes: ScannedNode[]): C[] => {
    const channels = ctx.get<C[]>(CHANNELS_PATH).value;
    const held = new Set(channels.map(nodeIdentity));
    const names = new Set(channels.map(deployName));
    return nodes.map(convertNodeToChannel).flatMap((ch) => {
      if (ch == null || held.has(nodeIdentity(ch))) return [];
      held.add(nodeIdentity(ch));
      const free = withFreeName(ch, names);
      names.add(deployName(free));
      return [free];
    });
  };
  const addFolders = (folders: HaulItem[]) =>
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      const key = id.create();
      const name = folders.length === 1 ? folders[0].data.name : "";
      addStatus({ key, variant: "loading", name, message: "Searching for variables" });
      // Another device mounts another body, but this search still writes to the form.
      const moved = (): boolean => {
        if (ctx.get<string>("config.device").value === device.key) return false;
        addStatus({
          key,
          variant: "warning",
          name,
          message: "Added no variables",
          description: MOVED_NOTE,
        });
        return true;
      };
      let search: Search;
      try {
        search = await findVariables(
          client,
          device,
          folders.map(({ data }) => data),
        );
      } catch (e) {
        addStatus({
          ...status.fromException(e, "Failed to search for variables"),
          key,
          name,
        });
        return;
      }
      if (moved()) return;
      const found = search.nodes;
      const missing = describeMissing(search);
      const target = folders.length === 1 ? "the folder" : "the folders";
      const count = newChannels(found).length;
      if (count === 0) {
        const reason =
          found.length === 0
            ? `No variables are under ${target}.`
            : `The task already has every variable under ${target}.`;
        addStatus({
          key,
          variant: "warning",
          name,
          message: "Found no new variables",
          description: missing == null ? reason : `${reason} ${missing}`,
        });
        return;
      }
      const noun = count === 1 ? "variable" : "variables";
      addStatus({
        key,
        variant: "success",
        name,
        message: `Found ${count} new ${noun}`,
      });
      const source = name === "" ? `${folders.length} folders` : name;
      const confirmed = await confirm({
        message: `Add ${count} ${noun} from ${source}?`,
        description: missing ?? NAMING_NOTE,
        confirm: { label: "Add", variant: "success" },
      });
      if (confirmed !== true || moved()) return;
      push(newChannels(found));
    }, "Failed to add the variables of a folder");
  const handleDrop = ({ items }: Haul.OnDropProps): Haul.Item[] => {
    const variables = items.filter(isVariableHaulItem);
    const folders = items.filter(isFolderHaulItem);
    push(newChannels(variables.map(({ data }) => data)));
    if (folders.length > 0) addFolders(folders);
    return [...variables, ...folders];
  };

  const haulProps = Haul.useDrop({
    type: HAUL_TYPE,
    canDrop,
    onDrop: handleDrop,
  });

  // The browser hides in preview, but a second opcua tab's browser can still source
  // drags, so the drop target goes inert too.
  const isPreview = Task.useIsPreview();
  const draggingState = Haul.useDraggingState();
  const isDragging = !isPreview && canDropHaulItem(draggingState);

  const [selected, setSelected] = useState(data.length > 0 ? [data[0]] : []);
  const listItem = ({ key, ...p }: Task.ChannelListItemProps) => (
    <ChannelListItem<C> key={key} {...p} getChannelKeyAndID={getChannelKeyAndID}>
      {children}
    </ChannelListItem>
  );
  return (
    <Task.ChannelList
      onSelect={setSelected}
      path={CHANNELS_PATH}
      emptyContent={<EmptyContent />}
      header={
        <Header<C>
          convertNodeToChannel={convertNodeToChannel}
          indexed={indexed}
          onImport={push}
        />
      }
      selected={selected}
      isDragging={isDragging}
      listItem={listItem}
      grow
      {...rest}
      {...(isPreview ? {} : haulProps)}
      {...fieldListReturn}
    />
  );
};

export interface FormProps<C extends Channel>
  extends
    Required<Pick<ChannelListProps<C>, "convertNodeToChannel" | "contextMenuItems">>,
    Pick<ChannelListProps<C>, "indexed"> {
  children?: Component.RenderProp<ExtraItemProps>;
  getChannelKeyAndID: ChannelKeyAndIDGetter<C>;
  /** Returns the bindings the device holds for a channel's node. */
  resolve: (device: Device, channel: C) => Partial<C>;
}

interface BodyProps<C extends Channel>
  extends FormProps<C>, PlatformDevice.TaskFormContentProps<Device> {}

const Body = <C extends Channel>({
  device,
  convertNodeToChannel,
  children = () => null,
  getChannelKeyAndID,
  contextMenuItems,
  resolve,
  indexed,
}: BodyProps<C>) => {
  const isPreview = Task.useIsPreview();
  const resolveOnDevice = (channel: C) => resolve(device, channel);
  return (
    <>
      {!isPreview && <Browser device={device} />}
      <ChannelList<C>
        device={device}
        convertNodeToChannel={convertNodeToChannel}
        getChannelKeyAndID={getChannelKeyAndID}
        contextMenuItems={contextMenuItems}
        resolve={resolveOnDevice}
        indexed={indexed}
      >
        {children}
      </ChannelList>
    </>
  );
};

export const createForm = <C extends Channel>(props: FormProps<C>): FC<{}> => {
  const Content = ({ device }: PlatformDevice.TaskFormContentProps<Device>) => (
    <Body<C> device={device} {...props} />
  );
  Content.displayName = "OPCTaskForm";
  return PlatformDevice.wrapTaskForm({
    use,
    useConfigure: useConnectModal,
    Content,
  });
};
