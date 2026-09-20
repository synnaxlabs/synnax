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
import {
  type Component,
  Flex,
  Form as PForm,
  Haul,
  Header as PHeader,
  Icon,
  Select,
  Text,
} from "@synnaxlabs/pluto";
import { type FC, useCallback, useState } from "react";

import {
  Browser,
  canDropHaulItem,
  HAUL_TYPE,
  type HaulItem,
  isHaulItem,
} from "@/feature/opcua/device/Browser";
import { use } from "@/feature/opcua/device/queries";
import type * as Device from "@/feature/opcua/device/types";
import { useConnectModal } from "@/feature/opcua/device/useConnectModal";
import { type Channel } from "@/feature/opcua/task/types";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
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
    <Select.ListItem {...rest} justify="between" align="center" rightAligned>
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
            overflow="ellipsis"
          >
            {nodeName} {opcNode}
          </Text.Text>
        </Flex.Box>
      </Flex.Box>
      <Flex.Box direction="x" align="center">
        {children({ path })}
        <Task.EnableDisableButton path={`${path}.disabled`} />
      </Flex.Box>
    </Select.ListItem>
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

const CHANNELS_PATH = "config.channels";

const VARIABLE_NODE_CLASS = "Variable";

const isVariableHaulItem = (item: Haul.Item): item is HaulItem =>
  isHaulItem(item) && item.data.nodeClass === VARIABLE_NODE_CLASS;

const canDrop = ({ items }: Haul.DraggingState): boolean =>
  items.some(isVariableHaulItem);

interface ChannelListProps<C extends Channel> extends Pick<
  Task.ChannelListProps<C>,
  "contextMenuItems"
> {
  children: Component.RenderProp<ExtraItemProps>;
  device: Device.Device;
  convertHaulItemToChannel: (item: HaulItem) => C;
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
  const fieldListReturn = PForm.useFieldList<C["key"], C>(CHANNELS_PATH);
  const { data, push } = fieldListReturn;
  const handleDrop = useCallback(
    ({ items }: Haul.OnDropProps): Haul.Item[] => {
      const channels = ctx.get<C[]>(CHANNELS_PATH).value;
      const dropped = items.filter(isVariableHaulItem);
      const toAdd = dropped
        .filter(({ data }) => !channels.some(({ nodeId }) => nodeId === data.nodeId))
        .map(convertHaulItemToChannel);
      push(toAdd);
      return dropped;
    },
    [push],
  );

  const haulProps = Haul.useDrop({
    type: HAUL_TYPE,
    canDrop,
    onDrop: handleDrop,
  });

  // The browser hides in preview, but a second opcua tab's browser can still source
  // drags, so the drop target goes inert too.
  const isPreview = Task.useIsPreview();
  const isDragging = !isPreview && canDropHaulItem(Haul.useDraggingState());

  const [selected, setSelected] = useState(data.length > 0 ? [data[0]] : []);
  const listItem = useCallback(
    ({ key, ...p }: Task.ChannelListItemProps) => (
      <ChannelListItem<C> key={key} {...p} getChannelKeyAndID={getChannelKeyAndID}>
        {children}
      </ChannelListItem>
    ),
    [children],
  );
  return (
    <Task.ChannelList
      onSelect={setSelected}
      path={CHANNELS_PATH}
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
  Pick<ChannelListProps<C>, "convertHaulItemToChannel" | "contextMenuItems">
> {
  children?: Component.RenderProp<ExtraItemProps>;
  getChannelKeyAndID: ChannelKeyAndIDGetter<C>;
}

interface BodyProps<C extends Channel>
  extends FormProps<C>, PlatformDevice.TaskFormContentProps<Device.Device> {}

const Body = <C extends Channel>({
  device,
  convertHaulItemToChannel,
  children = () => null,
  getChannelKeyAndID,
  contextMenuItems,
}: BodyProps<C>) => {
  const isPreview = Task.useIsPreview();
  return (
    <>
      {!isPreview && <Browser device={device} />}
      <ChannelList<C>
        device={device}
        convertHaulItemToChannel={convertHaulItemToChannel}
        getChannelKeyAndID={getChannelKeyAndID}
        contextMenuItems={contextMenuItems}
      >
        {children}
      </ChannelList>
    </>
  );
};

export const createForm = <C extends Channel>(props: FormProps<C>): FC<{}> => {
  const Content = ({ device }: PlatformDevice.TaskFormContentProps<Device.Device>) => (
    <Body<C> device={device} {...props} />
  );
  Content.displayName = "OPCTaskForm";
  return PlatformDevice.wrapTaskForm({
    use,
    useConfigure: useConnectModal,
    Content,
  });
};
