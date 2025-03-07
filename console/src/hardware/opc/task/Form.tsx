// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/opc/task/Form.css";

import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Form as PForm,
  Haul,
  Header as PHeader,
  List,
  type RenderProp,
  Text,
  useSyncedRef,
} from "@synnaxlabs/pluto";
import { useCallback, useState } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/opc/device";
import { type Channel } from "@/hardware/opc/task/types";

export interface ExtraItemProps {
  path: string;
  snapshot: boolean;
}

interface ChannelListItemProps<C extends Channel>
  extends Common.Task.ChannelListItemProps<C> {
  children: RenderProp<ExtraItemProps>;
}

const ChannelListItem = ({
  path,
  children,
  isSnapshot,
  ...rest
}: ChannelListItemProps<Channel>) => {
  const {
    entry: { name, nodeName, nodeId },
  } = rest;
  const opcNode = nodeId.length > 0 ? nodeId : "No Node Selected";
  let opcNodeColor;
  if (opcNode === "No Node Selected") opcNodeColor = "var(--pluto-warning-z)";
  return (
    <List.ItemFrame {...rest} justify="spaceBetween" align="center">
      <Align.Space direction="y" size="small">
        <Text.WithIcon
          startIcon={<Icon.Channel style={{ color: "var(--pluto-gray-l7)" }} />}
          level="p"
          weight={500}
          shade={9}
          align="end"
        >
          {name}
        </Text.WithIcon>
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

interface ChannelListProps<C extends Channel>
  extends Pick<Common.Task.ChannelListProps<C>, "isSnapshot"> {
  children: RenderProp<ExtraItemProps>;
  device: Device.Device;
  convertHaulItemToChannel: (item: Haul.Item) => C;
}

const filterHaulItem = (item: Haul.Item): boolean =>
  item.type === Device.HAUL_TYPE && item.data?.nodeClass === "Variable";

const canDrop = ({ items }: Haul.DraggingState): boolean => items.some(filterHaulItem);

const ChannelList = <C extends Channel>({
  device,
  children,
  convertHaulItemToChannel,
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
      <ChannelListItem key={key} {...p}>
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
}

export const Form = <C extends Channel>({
  isSnapshot,
  convertHaulItemToChannel,
  children = () => null,
}: FormProps<C>) => (
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
        >
          {children}
        </ChannelList>
      </>
    )}
  </Common.Device.Provider>
);
