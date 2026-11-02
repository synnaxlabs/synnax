// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/opcua/task/Task.css";

import { type channel } from "@synnaxlabs/client";
import { type Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Header as PHeader } from "@synnaxlabs/lyra/header";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { type Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { deep } from "@synnaxlabs/x";
import { type FC, useState } from "react";

import {
  Browser,
  canDropHaulItem,
  HAUL_TYPE,
  type HaulItem,
  isHaulItem,
} from "@/feature/opcua/device/Browser";
import { use } from "@/feature/opcua/device/queries";
import { type Device } from "@/feature/opcua/device/types";
import { useConnectModal } from "@/feature/opcua/device/useConnectModal";
import { type Channel, nodeIdentity } from "@/feature/opcua/task/types";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { Task } from "@/platform/task";

export interface ExtraItemProps<C extends Channel> {
  path: deep.Path<C>;
}

export interface ChannelKeyAndIDGetter<C extends Channel> {
  (channel: C): { id: string; key: channel.Key };
}

interface ChannelListItemProps<C extends Channel> extends Omit<
  Task.ChannelListItemProps,
  "children"
> {
  children: Component.RenderProp<ExtraItemProps<C>>;
  getChannelKeyAndID: ChannelKeyAndIDGetter<C>;
  path: deep.Path<C>;
}

const ChannelListItem = <C extends Channel>({
  children,
  getChannelKeyAndID,
  path,
  ...rest
}: ChannelListItemProps<C>) => {
  const item = PForm.useFieldValue(path);
  const fields = deep.widen<Channel>(path);
  const nodeIdState = PForm.useFieldState(fields.nodeId, { optional: true });
  if (item == null) return null;
  const { nodeName, nodeId } = item;
  const opcNode = nodeId.length > 0 ? nodeId : "No node selected";
  const issue = nodeIdState?.status;
  const marked = issue?.variant === "error" || issue?.variant === "warning";
  let nodeStatus: Status.Variant | undefined;
  if (marked) nodeStatus = issue?.variant;
  else if (nodeId.length === 0) nodeStatus = "warning";
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
          namePath={fields.name}
        />
        <Flex.Box x align="center" gap="tiny">
          <Icon.Variable color={9} />
          <Tooltip.Dialog hide={!marked}>
            {issue?.message}
            <Text.Text
              level="small"
              weight={350}
              color={9}
              status={nodeStatus}
              gap="small"
              overflow="fade"
            >
              {nodeName} {opcNode}
            </Text.Text>
          </Tooltip.Dialog>
        </Flex.Box>
      </Flex.Box>
      <Flex.Box direction="x" align="center">
        {children({ path })}
        <Task.EnabledCheckbox path={fields.disabled} />
      </Flex.Box>
    </Select.Item>
  );
};

const Header = () => (
  <PHeader.Header className={CSS.B("opc-task-header")}>
    <PHeader.Title weight={500} color={10} level="p">
      Channels
    </PHeader.Title>
  </PHeader.Header>
);

const EmptyContent = () => (
  <Flex.Box center>
    <Text.Text status="disabled" className={CSS.B("opc-task-empty-text")}>
      No channels added. Drag a variable{" "}
      <Icon.Variable className={CSS.B("opc-task-variable-icon")} /> from the browser to
      add a channel to the task.
    </Text.Text>
  </Flex.Box>
);

const VARIABLE_NODE_CLASS = "Variable";

const isVariableHaulItem = (item: Haul.Item): item is HaulItem =>
  isHaulItem(item) && item.data.nodeClass === VARIABLE_NODE_CLASS;

const canDrop = ({ items }: Haul.DraggingState): boolean =>
  items.some(isVariableHaulItem);

interface ChannelListProps<C extends Channel> extends Pick<
  Task.ChannelListProps<C>,
  "contextMenuItems" | "path" | "resolve"
> {
  children: Component.RenderProp<ExtraItemProps<C>>;
  device: Device;
  /** @returns the channel of a dropped node, or null if the task cannot hold it. */
  convertHaulItemToChannel: (item: HaulItem) => C | null;
  getChannelKeyAndID: ChannelKeyAndIDGetter<C>;
}

const ChannelList = <C extends Channel>({
  device,
  children,
  convertHaulItemToChannel,
  getChannelKeyAndID,
  ...rest
}: ChannelListProps<C>) => {
  const ctx = PForm.useContext();
  const { path } = rest;
  const fieldListReturn = PForm.useFieldList(path);
  const { data, push } = fieldListReturn;
  const handleDrop = ({ items }: Haul.OnDropProps): Haul.Item[] => {
    const channels = ctx.get(path).value;
    const dropped = items.filter(isVariableHaulItem);
    const existing = new Set(channels.map(nodeIdentity));
    const toAdd = dropped
      .map(convertHaulItemToChannel)
      .filter((ch): ch is C => ch != null && !existing.has(nodeIdentity(ch)));
    push(toAdd);
    return dropped;
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
    <ChannelListItem<C>
      key={key}
      {...p}
      getChannelKeyAndID={getChannelKeyAndID}
      path={path[p.itemKey]}
    >
      {children}
    </ChannelListItem>
  );
  return (
    <Task.ChannelList
      onSelect={setSelected}
      emptyContent={<EmptyContent />}
      header={<Header />}
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

export interface FormProps<C extends Channel> extends Required<
  Pick<ChannelListProps<C>, "convertHaulItemToChannel" | "contextMenuItems" | "path">
> {
  children?: Component.RenderProp<ExtraItemProps<C>>;
  getChannelKeyAndID: ChannelKeyAndIDGetter<C>;
  /** Returns the bindings the device holds for a channel's node. */
  resolve: (device: Device, channel: C) => Partial<C>;
}

interface BodyProps<C extends Channel>
  extends FormProps<C>, PlatformDevice.TaskFormContentProps<Device> {}

const Body = <C extends Channel>({
  device,
  convertHaulItemToChannel,
  children = () => null,
  getChannelKeyAndID,
  contextMenuItems,
  path,
  resolve,
}: BodyProps<C>) => {
  const isPreview = Task.useIsPreview();
  const resolveOnDevice = (channel: C) => resolve(device, channel);
  return (
    <>
      {!isPreview && <Browser device={device} />}
      <ChannelList<C>
        device={device}
        convertHaulItemToChannel={convertHaulItemToChannel}
        getChannelKeyAndID={getChannelKeyAndID}
        contextMenuItems={contextMenuItems}
        path={path}
        resolve={resolveOnDevice}
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
