// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/opc/task/Task.css";

import { type device, NotFoundError } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Form as PForm,
  Haul,
  Header,
  Input,
  List,
  Text,
  useSyncedRef,
} from "@synnaxlabs/pluto";
import { caseconv, primitiveIsZero } from "@synnaxlabs/x";
import { type FC, type ReactElement, useCallback, useState } from "react";

import { CSS } from "@/css";
import { Common } from "@/hardware/common";
import { Device } from "@/hardware/opc/device";
import {
  WRITE_TYPE,
  type WriteChannelConfig,
  type WriteConfig,
  writeConfigZ,
  type WriteStateDetails,
  type WriteType,
  ZERO_WRITE_PAYLOAD,
} from "@/hardware/opc/task/types";
import { type Layout } from "@/layout";

export const WRITE_LAYOUT: Common.Task.LayoutBaseState = {
  ...Common.Task.LAYOUT,
  key: WRITE_TYPE,
  type: WRITE_TYPE,
  name: ZERO_WRITE_PAYLOAD.name,
  icon: "Logo.OPC",
};

export const WRITE_SELECTABLE: Layout.Selectable = {
  key: WRITE_TYPE,
  title: "OPC UA Write Task",
  icon: <Icon.Logo.OPC />,
  create: (key) => ({ ...WRITE_LAYOUT, key }),
};

interface ChannelListItemProps extends List.ItemProps<string, WriteChannelConfig> {
  path: string;
  remove?: () => void;
  snapshot?: boolean;
}

const ChannelListItem = ({
  path,
  remove,
  snapshot,
  ...props
}: ChannelListItemProps): ReactElement => {
  const { entry } = props;
  const ctx = PForm.useContext();
  const childValues = PForm.useChildFieldValues<WriteChannelConfig>({
    path: `${path}.${props.index}`,
    optional: true,
  });
  if (childValues == null) return <></>;
  const opcNode =
    childValues.nodeId.length > 0 ? childValues.nodeId : "No Node Selected";
  let opcNodeColor;
  if (opcNode === "No Node Selected") opcNodeColor = "var(--pluto-warning-z)";

  return (
    <List.ItemFrame
      {...props}
      entry={childValues}
      justify="spaceBetween"
      align="center"
      onKeyDown={(e) => ["Delete", "Backspace"].includes(e.key) && remove?.()}
    >
      <Align.Space direction="y" size="small">
        <Text.WithIcon
          startIcon={<Icon.Channel style={{ color: "var(--pluto-gray-l7)" }} />}
          level="p"
          weight={500}
          shade={9}
          align="end"
        >
          {entry.name}
        </Text.WithIcon>
        <Text.WithIcon
          startIcon={<Icon.Variable style={{ color: "var(--pluto-gray-l7)" }} />}
          level="small"
          weight={350}
          shade={7}
          color={opcNodeColor}
          size="small"
        >
          {entry.nodeName} {opcNode}
        </Text.WithIcon>
      </Align.Space>
      <Align.Space direction="x" align="center">
        <Common.Task.EnableDisableButton
          value={childValues.enabled}
          onChange={(v) => ctx.set(`${path}.${props.index}.enabled`, v)}
          snapshot={snapshot}
        />
      </Align.Space>
    </List.ItemFrame>
  );
};

interface ChannelFormProps {
  selectedChannelIndex?: number | null;
  snapshot?: boolean;
}

const ChannelForm = ({
  selectedChannelIndex,
  snapshot,
}: ChannelFormProps): ReactElement | null => {
  if (selectedChannelIndex == null || selectedChannelIndex == -1) {
    if (snapshot === true) return null;
    return (
      <Align.Center className={CSS.B("channel-form")}>
        <Text.Text level="p" shade={6}>
          Select a channel to configure its properties.
        </Text.Text>
      </Align.Center>
    );
  }
  const prefix = `config.channels.${selectedChannelIndex}`;
  return (
    <Align.Space direction="y" grow className={CSS.B("channel-form")} empty>
      <PForm.Field<string>
        path={`${prefix}.name`}
        padHelpText={!snapshot}
        label="Channel Name"
      >
        {(p) => <Input.Text variant="natural" level="h3" {...p} />}
      </PForm.Field>
    </Align.Space>
  );
};

const getChannelByNodeID = (props: Device.Properties, nodeId: string) =>
  props.write.channels[nodeId] ?? props.write.channels[caseconv.snakeToCamel(nodeId)];

interface ChannelListProps {
  path: string;
  device?: device.Device<Device.Properties>;
  snapshot?: boolean;
}

const ChannelList = ({ path, snapshot }: ChannelListProps): ReactElement => {
  const { value, push, remove } = PForm.useFieldArray<WriteChannelConfig>({ path });
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
          key: nodeId,
          name,
          nodeName: name,
          cmdChannel: 0,
          enabled: true,
          nodeId,
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

  const props = Haul.useDrop({ type: "opc.WriteTask", canDrop, onDrop: handleDrop });

  const dragging = Haul.canDropOfType("opc")(Haul.useDraggingState());

  const [selectedChannels, setSelectedChannels] = useState<string[]>(
    value.length > 0 ? [value[0].key] : [],
  );
  const [selectedChannelIndex, setSelectedChannelIndex] = useState<number | null>(
    value.length > 0 ? 0 : null,
  );

  return (
    <>
      <Common.Task.ChannelList<WriteChannelConfig>
        path={path}
        snapshot={snapshot}
        className={CSS(CSS.B("channels"), dragging && CSS.B("dragging"))}
        grow
        empty
        bordered
        rounded
        {...props}
        header={() => (
          <Header.Header level="h4">
            <Header.Title weight={500}>Channels</Header.Title>
          </Header.Header>
        )}
        emptyContent={() => (
          <Align.Center>
            <Text.Text shade={6} level="p" style={{ maxWidth: 300 }}>
              No channels added. Drag a variable{" "}
              <Icon.Variable
                style={{ fontSize: "2.5rem", transform: "translateY(0.5rem)" }}
              />{" "}
              from the browser to add a channel to the task.
            </Text.Text>
          </Align.Center>
        )}
        selected={selectedChannels}
        onSelect={(keys, index) => {
          setSelectedChannels(keys);
          setSelectedChannelIndex(index);
        }}
      >
        {(props) => (
          <ChannelListItem
            {...props}
            path={path}
            remove={() => {
              const indices = selectedChannels
                .map((k) => value.findIndex((v) => v.key === k))
                .filter((i) => i >= 0);
              remove(indices);
              setSelectedChannels([]);
              setSelectedChannelIndex(null);
            }}
            snapshot={snapshot}
          />
        )}
      </Common.Task.ChannelList>
      {value.length > 0 && (
        <ChannelForm selectedChannelIndex={selectedChannelIndex} snapshot={snapshot} />
      )}
    </>
  );
};

const Form: FC<Common.Task.FormProps<WriteConfig, WriteStateDetails, WriteType>> = ({
  methods,
  task,
}) => {
  const device = Common.Device.use<Device.Properties>(methods);
  return (
    <>
      <Align.Space direction="x" className={CSS.B("task-properties")}>
        <Device.Select />
        <Align.Space direction="x">
          <PForm.Field<boolean> label="Data Saving" path="config.dataSaving" optional>
            {(p) => <Input.Switch {...p} />}
          </PForm.Field>
        </Align.Space>
      </Align.Space>
      <Align.Space direction="x" grow style={{ overflow: "hidden", height: "500px" }}>
        {task.snapshot !== true && <Device.Browser device={device} />}
        <ChannelList path="config.channels" device={device} snapshot={task.snapshot} />
      </Align.Space>
    </>
  );
};

export const WriteTask = Common.Task.wrapForm(Form, {
  configSchema: writeConfigZ,
  type: WRITE_TYPE,
  zeroPayload: ZERO_WRITE_PAYLOAD,
  onConfigure: async (client, config) => {
    const dev = await client.hardware.devices.retrieve<Device.Properties>(
      config.device,
    );

    let modified = false;

    const commandsToCreate: WriteChannelConfig[] = [];
    for (const channel of config.channels) {
      const key = getChannelByNodeID(dev.properties, channel.nodeId);
      if (primitiveIsZero(key)) commandsToCreate.push(channel);
      else
        try {
          await client.channels.retrieve(key);
        } catch (e) {
          if (NotFoundError.matches(e)) commandsToCreate.push(channel);
          else throw e;
        }
    }

    if (commandsToCreate.length > 0) {
      modified = true;
      if (
        dev.properties.write.channels == null ||
        Array.isArray(dev.properties.write.channels)
      )
        dev.properties.write.channels = {};
      const commandIndexes = await client.channels.create(
        commandsToCreate.map((c) => ({
          name: `${c.name}_cmd_time`,
          dataType: "timestamp",
          isIndex: true,
        })),
      );
      const commands = await client.channels.create(
        commandsToCreate.map((c, i) => ({
          name: `${c.name}_cmd`,
          dataType: c.dataType,
          index: commandIndexes[i].key,
        })),
      );
      commands.forEach((c, i) => {
        const key = commandsToCreate[i].nodeId;
        dev.properties.write.channels[key] = c.key;
      });
    }

    config.channels = config.channels.map((c) => ({
      ...c,
      channel: getChannelByNodeID(dev.properties, c.nodeId),
    }));

    if (modified) await client.hardware.devices.create(dev);
    return config;
  },
});
