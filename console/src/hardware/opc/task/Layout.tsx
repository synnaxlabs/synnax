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
  Form,
  Haul,
  Header as PHeader,
  Text,
  useSyncedRef,
} from "@synnaxlabs/pluto";
import { type FC, type ReactElement, useCallback, useState } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/opc/device";
import {
  type ReadChannelConfig,
  type WriteChannelConfig,
} from "@/hardware/opc/task/types";

type ChannelConfig = ReadChannelConfig | WriteChannelConfig;

interface ChannelListItemProps<C extends ChannelConfig>
  extends Common.Task.ChannelListItemProps<C> {
  path: string;
  remove?: () => void;
  snapshot?: boolean;
}

interface ChannelListProps
  extends Pick<Common.Task.ChannelListProps<ChannelConfig>, "isSnapshot"> {
  children: (props: ChannelListItemProps<ChannelConfig>) => ReactElement;
  device: Device.Device;
}

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

const ChannelList = ({ device, ...rest }: ChannelListProps) => {
  const { value, push, remove } = Form.useFieldArray<ChannelConfig>({ path: PATH });
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
    push(toAdd);
    return dropped;
  }, []);

  const canDrop = useCallback((state: Haul.DraggingState): boolean => {
    const v = state.items.some(
      (i) => i.type === "opc" && i.data?.nodeClass === "Variable",
    );
    return v;
  }, []);

  const haulProps = Haul.useDrop({
    type: "opc.ReadTask", //fix type
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
      {({ path, remove, snapshot, ...rest }) => (
        <ChannelListItem {...rest} path={path} remove={remove} snapshot={snapshot} />
      )}
    </Common.Task.ChannelList>
  );
};

type FormType = FC<Common.Task.FormProps>;

const TaskForm: FormType = ({ isSnapshot }) => (
  <Common.Device.Provider<Device.Properties, Device.Make>
    configureLayout={Device.CONFIGURE_LAYOUT}
    isSnapshot={isSnapshot}
  >
    {({ device }) => (
      <>
        {!isSnapshot && <Device.Browser device={device} />}
        <ChannelList device={device} isSnapshot={isSnapshot}></ChannelList>
      </>
    )}
  </Common.Device.Provider>
);
