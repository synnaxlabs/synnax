// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/service/opc/task/Task.css";

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
import { useCallback, useState } from "react";

import { CSS } from "@/component/css";
import { ChannelName } from "@/component/task/ChannelName";
import { Device as CommonDevice } from "@/component/device";
import { Task } from "@/component/task";
import { Device } from "@/service/opc/device";
import { type Channel } from "@/service/opc/task/types";

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
  const opcNode = nodeId.length > 0 ? nodeId : "No Node Selected";
  let opcNodeColor;
  if (opcNode === "No Node Selected") opcNodeColor = "var(--pluto-warning-z)";
  const { key: channel, id } = getChannelKeyAndID(item);
  return (
    <Select.ListItem {...rest} justify="between" align="center" rightAligned>
      <Flex.Box
        direction="y"
        gap="small"
        className={CSS.BE("channel-name", "container")}
      >
        <ChannelName
          weight={500}
          color={10}
          level="p"
          channel={channel}
          id={id}
          namePath={`${path}.name`}
        />
        <Flex.Box x align="center" gap="tiny">
          <Icon.Variable color={7} />
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
        <Task.EnableDisableButton path={`${path}.enabled`} />
      </Flex.Box>
    </Select.ListItem>
  );
};

const Header = () => (
  <PHeader.Header style={{ height: "4.5rem", flexShrink: 0, flexGrow: 0 }}>
    <PHeader.Title weight={500} color={10} level="p">
      Channels
    </PHeader.Title>
  </PHeader.Header>
);

const EmptyContent = () => (
  <Flex.Box center>
    <Text.Text status="disabled" style={{ display: "inline-block", maxWidth: 300 }}>
      No channels added. Drag a variable{" "}
      <Icon.Variable style={{ fontSize: "2.5rem", transform: "translateY(0.5rem)" }} />{" "}
      from the browser to add a channel to the task.
    </Text.Text>
  </Flex.Box>
);

const CHANNELS_PATH = "config.channels";

const VARIABLE_NODE_CLASS = "Variable";

const isVariableHaulItem = (item: Haul.Item): item is Device.HaulItem =>
  Device.isHaulItem(item) && item.data.nodeClass === VARIABLE_NODE_CLASS;

const canDrop = ({ items }: Haul.DraggingState): boolean =>
  items.some(isVariableHaulItem);

interface ChannelListProps<C extends Channel> extends Pick<
  Task.ChannelListProps<C>,
  "contextMenuItems"
> {
  children: Component.RenderProp<ExtraItemProps>;
  device: Device.Device;
  convertHaulItemToChannel: (item: Device.HaulItem) => C;
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
    type: Device.HAUL_TYPE,
    canDrop,
    onDrop: handleDrop,
  });

  const isDragging = Device.canDropHaulItem(Haul.useDraggingState());

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
      {...haulProps}
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

export const Form = <C extends Channel>({
  convertHaulItemToChannel,
  children = () => null,
  getChannelKeyAndID,
  contextMenuItems,
}: FormProps<C>) => {
  const isSnapshot = Task.useIsSnapshot();
  const connect = Device.useConnectModal();
  return (
    <CommonDevice.Provider
      canConfigure={!isSnapshot}
      onConfigure={(deviceKey) => connect({ deviceKey })}
      schemas={Device.SCHEMAS}
    >
      {({ device }) => (
        <>
          {!isSnapshot && <Device.Browser device={device} />}
          <ChannelList<C>
            device={device}
            convertHaulItemToChannel={convertHaulItemToChannel}
            getChannelKeyAndID={getChannelKeyAndID}
            contextMenuItems={contextMenuItems}
          >
            {children}
          </ChannelList>
        </>
      )}
    </CommonDevice.Provider>
  );
};
