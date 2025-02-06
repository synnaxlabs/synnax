// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Form as PForm,
  Haul,
  Header as PHeader,
  List,
  Text,
  useSyncedRef,
} from "@synnaxlabs/pluto";
import { type ReactElement, type ReactNode, useCallback, useState } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/opc/device";
import {
  type ReadChannelConfig,
  type WriteChannelConfig,
} from "@/hardware/opc/task/types";

type ChannelConfig = ReadChannelConfig | WriteChannelConfig;

interface ChildrenProps {
  path: string;
  snapshot: boolean;
}

interface ChannelListItemProps<C extends ChannelConfig>
  extends Common.Task.ChannelListItemProps<C> {
  children: (props: ChildrenProps) => ReactNode | null;
}

const ChannelListItem = ({
  path,
  children,
  isSnapshot,
  ...rest
}: ChannelListItemProps<ChannelConfig>) => {
  const {
    entry: { enabled, name, nodeName, nodeId },
  } = rest;
  console.log("entry", rest.entry);
  const { set } = PForm.useContext();
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
          value={enabled}
          onChange={(v) => set(`${path}.enabled`, v)}
          isSnapshot={isSnapshot}
        />
      </Align.Space>
    </List.ItemFrame>
  );
};

const Header = (): ReactElement => (
  <PHeader.Header level="h4">
    <PHeader.Title weight={500}>Channels</PHeader.Title>
  </PHeader.Header>
);

const EmptyContent = (): ReactElement => (
  <Align.Center>
    <Text.Text shade={6} level="p" style={{ maxWidth: 300 }}>
      No channels added. Drag a variable{" "}
      <Icon.Variable style={{ fontSize: "2.5rem", transform: "translateY(0.5rem)" }} />{" "}
      from the browser to add a channel to the task.
    </Text.Text>
  </Align.Center>
);

const PATH = "config.channels";

interface ChannelListProps<C extends ChannelConfig>
  extends Pick<Common.Task.ChannelListProps<C>, "isSnapshot"> {
  children: (props: ChildrenProps) => ReactNode | null;
  device: Device.Device;
}

const ChannelList = <C extends ChannelConfig>({
  device,
  children,
  ...rest
}: ChannelListProps<C>): ReactElement => {
  const { value, push, remove } = PForm.useFieldArray<C>({
    path: PATH,
    updateOnChildren: true,
  });
  const valueRef = useSyncedRef(value);
  const handleDrop = useCallback(({ items }: Haul.OnDropProps): Haul.Item[] => {
    const dropped = items.filter(
      (i) => i.type === "opc" && i.data?.nodeClass === "Variable",
    );
    const toAdd = dropped
      .filter((v) => !valueRef.current.some((c) => c.nodeId === v.data?.nodeId))
      .map((i) => {
        const nodeId = i.data?.nodeId as string;
        const name = i.data?.name as string;
        return {
          // Todo: different for write, read channels
          key: nodeId,
          name,
          nodeName: name,
          channel: 0,
          enabled: true,
          nodeId,
          useAsIndex: false,
          dataType: (i.data?.dataType as string) ?? "float32",
        };
      });
    // @ts-expect-error fix later
    push(toAdd);
    return dropped;
  }, []);

  const canDrop = useCallback(
    (state: Haul.DraggingState): boolean =>
      state.items.some((i) => i.type === "opc" && i.data?.nodeClass === "Variable"),
    [],
  );

  const haulProps = Haul.useDrop({
    type: "opc", //fix type
    canDrop,
    onDrop: handleDrop,
  });

  const isDragging = Haul.canDropOfType("opc")(Haul.useDraggingState());

  const [selected, setSelected] = useState(value.length > 0 ? [value[0].key] : []);
  return (
    <Common.Task.ChannelList
      {...rest}
      channels={value}
      onSelect={setSelected}
      path={PATH}
      remove={remove}
      emptyContent={<EmptyContent />}
      header={<Header />}
      selected={selected}
      isDragging={isDragging}
      {...haulProps}
    >
      {(p) => <ChannelListItem {...p}>{children}</ChannelListItem>}
    </Common.Task.ChannelList>
  );
};

export interface FormProps {
  isSnapshot: boolean;
  children?: (props: ChildrenProps) => ReactNode | null;
}

export const Form = <C extends ChannelConfig>({
  isSnapshot,
  children = () => null,
}: FormProps): ReactElement => (
  <Common.Device.Provider<Device.Properties, Device.Make>
    canConfigure={!isSnapshot}
    configureLayout={Device.CONFIGURE_LAYOUT}
  >
    {({ device }) => (
      <>
        {!isSnapshot && <Device.Browser device={device} />}
        <ChannelList<C> device={device} isSnapshot={isSnapshot}>
          {children}
        </ChannelList>
      </>
    )}
  </Common.Device.Provider>
);
