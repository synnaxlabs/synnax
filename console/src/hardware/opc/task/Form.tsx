// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/opc/task/Form.css";

import { type channel } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  componentRenderProp,
  Form as PForm,
  Haul,
  Header as PHeader,
  List,
  Menu,
  type RenderProp,
  Text,
  useSyncedRef,
} from "@synnaxlabs/pluto";
import { useCallback, useState } from "react";

import { Common } from "@/hardware/common";
import { ChannelName } from "@/hardware/common/task/ChannelName";
import { Device } from "@/hardware/opc/device";
import { type Channel } from "@/hardware/opc/task/types";

export interface ExtraItemProps {
  path: string;
  snapshot: boolean;
}

export interface ChannelKeyAndIDGetter<C extends Channel> {
  (channel: C): { id: string; key: channel.Key };
}

interface ChannelListItemProps<C extends Channel>
  extends Common.Task.ChannelListItemProps<C> {
  children: RenderProp<ExtraItemProps>;
  getChannelKeyAndID: ChannelKeyAndIDGetter<C>;
}

const ChannelListItem = <C extends Channel>({
  path,
  children,
  isSnapshot,
  getChannelKeyAndID,
  ...rest
}: ChannelListItemProps<C>) => {
  const {
    entry: { nodeName, nodeId },
  } = rest;
  const opcNode = nodeId.length > 0 ? nodeId : "No Node Selected";
  let opcNodeColor;
  if (opcNode === "No Node Selected") opcNodeColor = "var(--pluto-warning-z)";
  const { key: channel, id } = getChannelKeyAndID(rest.entry);
  return (
    <List.ItemFrame {...rest} justify="spaceBetween" align="center">
      <Align.Space direction="y" size="small">
        <ChannelName level="p" weight={500} shade={9} channel={channel} id={id} />
        <Text.WithIcon
          startIcon={<Icon.Variable style={{ color: "var(--pluto-gray-l7)" }} />}
          level="small"
          weight={350}
          shade={7}
          color={opcNodeColor}
          size="small"
        >
          {nodeName} {opcNode}
        </Text.WithIcon>
      </Align.Space>
      <Align.Space direction="x" align="center">
        {children({ path, snapshot: isSnapshot })}
        <Common.Task.EnableDisableButton
          path={`${path}.enabled`}
          isSnapshot={isSnapshot}
        />
      </Align.Space>
    </List.ItemFrame>
  );
};

const Header = () => (
  <PHeader.Header level="p">
    <PHeader.Title weight={500} shade={8}>
      Channels
    </PHeader.Title>
  </PHeader.Header>
);

const EmptyContent = () => (
  <Align.Center>
    <Text.Text shade={6} level="p" style={{ maxWidth: 300 }}>
      No channels added. Drag a variable{" "}
      <Icon.Variable style={{ fontSize: "2.5rem", transform: "translateY(0.5rem)" }} />{" "}
      from the browser to add a channel to the task.
    </Text.Text>
  </Align.Center>
);

const CHANNELS_PATH = "config.channels";

const VARIABLE_NODE_CLASS = "Variable";

const filterHaulItem = (item: Haul.Item): boolean =>
  item.type === Device.HAUL_TYPE && item.data?.nodeClass === VARIABLE_NODE_CLASS;

const canDrop = ({ items }: Haul.DraggingState): boolean => items.some(filterHaulItem);

export const generateContextMenuItemRenderProps = <C extends Channel>(
  getChannelKeyAndID: ChannelKeyAndIDGetter<C>,
): RenderProp<Common.Task.ContextMenuItemProps<C>> => {
  const C = ({ channels, keys }: Common.Task.ContextMenuItemProps<C>) => {
    if (keys.length !== 1) return null;
    const key = keys[0];
    const channel = channels.find((ch) => ch.key === key);
    if (!channel) return null;
    const { key: channelKey, id } = getChannelKeyAndID(channel);
    if (!channelKey) return null;
    const handleRename = () => Text.edit(id);
    return (
      <>
        <Menu.Item itemKey="rename" startIcon={<Icon.Rename />} onClick={handleRename}>
          Rename
        </Menu.Item>
      </>
    );
  };
  return componentRenderProp(C);
};

interface ChannelListProps<C extends Channel>
  extends Pick<Common.Task.ChannelListProps<C>, "isSnapshot" | "contextMenuItems"> {
  children: RenderProp<ExtraItemProps>;
  device: Device.Device;
  convertHaulItemToChannel: (item: Haul.Item) => C;
  getChannelKeyAndID: ChannelKeyAndIDGetter<C>;
}

const ChannelList = <C extends Channel>({
  device,
  children,
  convertHaulItemToChannel,
  getChannelKeyAndID,
  ...rest
}: ChannelListProps<C>) => {
  const { value, push, remove } = PForm.useFieldArray<C>({
    path: CHANNELS_PATH,
    updateOnChildren: true,
  });
  const valueRef = useSyncedRef(value);
  const handleDrop = useCallback(
    ({ items }: Haul.OnDropProps): Haul.Item[] => {
      const dropped = items.filter(filterHaulItem);
      const toAdd = dropped
        .filter(
          ({ data }) => !valueRef.current.some(({ nodeId }) => nodeId === data?.nodeId),
        )
        .map(convertHaulItemToChannel);
      push(toAdd);
      return dropped;
    },
    [push],
  );

  const haulProps = Haul.useDrop({
    type: Device.HAUL_TYPE,
    canDrop,
    onDrop: handleDrop,
  });

  const isDragging = Haul.canDropOfType(Device.HAUL_TYPE)(Haul.useDraggingState());

  const [selected, setSelected] = useState(value.length > 0 ? [value[0].key] : []);
  const listItem = useCallback(
    ({ key, ...p }: Common.Task.ChannelListItemProps<C>) => (
      <ChannelListItem<C> key={key} {...p} getChannelKeyAndID={getChannelKeyAndID}>
        {children}
      </ChannelListItem>
    ),
    [children],
  );
  return (
    <Common.Task.ChannelList
      {...rest}
      channels={value}
      onSelect={setSelected}
      path={CHANNELS_PATH}
      remove={remove}
      emptyContent={<EmptyContent />}
      header={<Header />}
      selected={selected}
      isDragging={isDragging}
      listItem={listItem}
      grow
      {...haulProps}
    />
  );
};

export interface FormProps<C extends Channel> {
  isSnapshot: boolean;
  children?: RenderProp<ExtraItemProps>;
  convertHaulItemToChannel: (item: Haul.Item) => C;
  getChannelKeyAndID: ChannelKeyAndIDGetter<C>;
}

export const Form = <C extends Channel>({
  isSnapshot,
  convertHaulItemToChannel,
  children = () => null,
  getChannelKeyAndID,
}: FormProps<C>) => {
  const contextMenuItems = useCallback(
    generateContextMenuItemRenderProps(getChannelKeyAndID),
    [getChannelKeyAndID],
  );
  return (
    <Common.Device.Provider<Device.Properties, Device.Make>
      canConfigure={!isSnapshot}
      configureLayout={Device.CONNECT_LAYOUT}
    >
      {({ device }) => (
        <>
          {!isSnapshot && <Device.Browser device={device} />}
          <ChannelList<C>
            device={device}
            isSnapshot={isSnapshot}
            convertHaulItemToChannel={convertHaulItemToChannel}
            getChannelKeyAndID={getChannelKeyAndID}
            contextMenuItems={contextMenuItems}
          >
            {children}
          </ChannelList>
        </>
      )}
    </Common.Device.Provider>
  );
};
