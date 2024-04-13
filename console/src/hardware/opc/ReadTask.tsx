// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useEffect, useRef, useState, type ReactElement } from "react";

import { type task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Channel,
  Device,
  Form,
  Header,
  Input,
  List,
  Select,
  Status,
  Synnax,
  Text,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { useMutation, useQuery } from "@tanstack/react-query";
import { nanoid } from "nanoid";

import { CSS } from "@/css";
import {
  type ReadTaskChannelConfig,
  readTaskConfigZ,
  type DeviceProperties,
  type ReadTaskState,
  type ReadTaskStateDetails,
  type ReadTaskConfig,
} from "@/hardware/opc/types";
import { type Layout } from "@/layout";

export const readTaskLayout: Layout.LayoutState = {
  name: "Configure OPC UA Read Task",
  key: "readopcTask",
  type: "readopcTask",
  windowKey: "readopcTask",
  location: "mosaic",
};

export const ReadTask = (): ReactElement => {
  const client = Synnax.use();
  const [task, setTask] = useState<task.Task | undefined>(undefined);
  const [taskState, setTaskState] = useState<ReadTaskState | null>(null);
  const methods = Form.use({
    schema: readTaskConfigZ,
    values: {
      device: "",
      sampleRate: 50,
      streamRate: 25,
      channels: [],
    },
  });

  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);
  const [selectedChannelIndex, setSelectedChannelIndex] = useState<number | null>(null);

  const stateObserverRef = useRef<task.StateObservable<ReadTaskStateDetails> | null>(
    null,
  );

  useAsyncEffect(async () => {
    if (client == null || task == null) return;
    stateObserverRef.current = await task.openStateObserver<ReadTaskStateDetails>();
    console.log("BB");
    stateObserverRef.current.onChange((s) => {
      console.log("BB");
      setTaskState(s);
    });
    return () => stateObserverRef.current?.close().catch(console.error);
  }, [client?.key, task?.key, setTaskState]);

  useEffect(() => {
    console.log(taskState);
  }, [taskState]);

  const configure = useMutation({
    mutationKey: [client?.key],
    mutationFn: async () => {
      if (client == null) return;
      const rack = await client.hardware.racks.retrieve("sy_node_1_rack");
      try {
        const v = methods.value();
        const dev = await client.hardware.devices.retrieve<DeviceProperties>(v.device);
        setTask(
          await rack.createTask<ReadTaskConfig>({
            key: task?.key,
            name: "opc Read Task",
            type: "opcReader",
            config: {
              ...v,
              channels: v.channels.map((c) => ({
                ...c,
                namespace: dev.properties.channels.find((d) => d.nodeId === c.node)
                  ?.namespace,
              })),
            },
          }),
        );
      } catch (e) {
        console.error(e);
      }
    },
  });

  return (
    <Align.Space className={CSS.B("opc-read-task")} direction="y" grow empty>
      <Form.Form {...methods}>
        <Align.Space direction="x">
          <Form.Field<string> path="device" label="Device">
            {(p) => (
              <Device.SelectSingle
                {...p}
                allowNone={false}
                searchOptions={{ makes: ["opc"] }}
              />
            )}
          </Form.Field>
          <Form.Field<number> label="Sample Rate" path="sampleRate">
            {(p) => <Input.Numeric {...p} />}
          </Form.Field>
          <Form.Field<number> label="Stream Rate" path="streamRate">
            {(p) => <Input.Numeric {...p} />}
          </Form.Field>
        </Align.Space>
        <Align.Space direction="x">
          <ChannelList
            path="channels"
            selected={selectedChannels}
            onSelect={useCallback(
              (v, i) => {
                setSelectedChannels(v);
                setSelectedChannelIndex(i);
              },
              [setSelectedChannels, setSelectedChannelIndex],
            )}
          />
        </Align.Space>
        {selectedChannelIndex != null && (
          <ChannelForm selectedChannelIndex={selectedChannelIndex} />
        )}
      </Form.Form>
      <Button.Button onClick={() => configure.mutate()}>Configure</Button.Button>
    </Align.Space>
  );
};

export interface ChannelListProps {
  path: string;
  onSelect: (keys: string[], index: number) => void;
  selected: string[];
}

export const ChannelList = ({
  path,
  selected,
  onSelect,
}: ChannelListProps): ReactElement => {
  const { value, push } = Form.useFieldArray<ReadTaskChannelConfig>({ path });

  const handleAdd = (): void => {
    push({
      key: nanoid(),
      channel: 0,
      node: "",
      enabled: true,
    });
  };

  return (
    <Align.Space className={CSS.B("channels")} grow empty>
      <Header.Header level="h3">
        <Header.Title weight={500}>Channels</Header.Title>
        <Header.Actions>
          {[
            {
              key: "add",
              onClick: handleAdd,
              children: <Icon.Add />,
            },
          ]}
        </Header.Actions>
      </Header.Header>
      <List.List<string, ReadTaskChannelConfig> data={value}>
        <List.Selector<string, ReadTaskChannelConfig>
          value={selected}
          allowNone={false}
          allowMultiple={true}
          onChange={(keys, { clickedIndex }) =>
            clickedIndex != null && onSelect(keys, clickedIndex)
          }
          replaceOnSingle
        >
          <List.Core<string, ReadTaskChannelConfig> grow>
            {(props) => <ChannelListItem {...props} path={path} />}
          </List.Core>
        </List.Selector>
      </List.List>
    </Align.Space>
  );
};

export const ChannelListItem = ({
  path,
  ...props
}: List.ItemProps<string, ReadTaskChannelConfig> & {
  path: string;
}): ReactElement => {
  const { entry } = props;
  const ctx = Form.useContext();
  const childValues = Form.useChildFieldValues<ReadTaskChannelConfig>({
    path: `${path}.${props.index}`,
    optional: true,
  });
  if (childValues == null) return <></>;
  const channelName = Channel.useName(entry.channel);
  return (
    <List.ItemFrame
      {...props}
      entry={childValues}
      justify="spaceBetween"
      align="center"
    >
      <Align.Space direction="y" size="small">
        <Align.Space direction="x">
          <Text.Text level="p" weight={500} shade={9}>
            {channelName}
          </Text.Text>
        </Align.Space>
      </Align.Space>
      <Button.Toggle
        checkedVariant="outlined"
        uncheckedVariant="outlined"
        value={entry.enabled}
        size="small"
        onClick={(e) => e.stopPropagation()}
        onChange={(v) => {
          ctx.set({ path: `${path}.${props.index}.enabled`, value: v });
        }}
        tooltip={
          <Text.Text level="small" style={{ maxWidth: 300 }}>
            Data acquisition for this channel is{" "}
            {entry.enabled ? "enabled" : "disabled"}. Click to
            {entry.enabled ? " disable" : " enable"} it.
          </Text.Text>
        }
      >
        <Status.Text
          variant={entry.enabled ? "success" : "disabled"}
          level="small"
          align="center"
        >
          {entry.enabled ? "Enabled" : "Disabled"}
        </Status.Text>
      </Button.Toggle>
    </List.ItemFrame>
  );
};

interface ChannelFormProps {
  selectedChannelIndex: number;
}

const ChannelForm = ({ selectedChannelIndex }: ChannelFormProps): ReactElement => {
  const prefix = `channels.${selectedChannelIndex}`;
  return (
    <Align.Space direction="y">
      <Form.Field<number> path={`${prefix}.channel`} label="Channel">
        {(p) => <Channel.SelectSingle {...p} />}
      </Form.Field>
      <Form.Field<string> path={`${prefix}.node`} label="Node">
        {(p) => <SelectNode {...p} />}
      </Form.Field>
    </Align.Space>
  );
};

interface UANodeEntry {
  key: string;
  name: string;
  dataType: string;
}

const SELECT_NODE_COLUMNS: Array<List.ColumnSpec<string, UANodeEntry>> = [
  {
    name: "Name",
    key: "name",
  },
  {
    name: "Data Type",
    key: "dataType",
  },
];

interface SelectNodeProps
  extends Omit<Select.SingleProps<string, UANodeEntry>, "columns" | "data"> {}

const SelectNode = (props: SelectNodeProps): ReactElement => {
  const client = Synnax.use();
  const form = Form.useContext();
  const nodes = useQuery({
    queryKey: [client?.key],
    queryFn: async () => {
      if (client == null) return;
      const dev = form.get<string>({ path: "device" });
      if (dev.status.variant === "success" && dev.value.length > 0) {
        const device = await client.hardware.devices.retrieve<DeviceProperties>(
          dev.value,
        );
        return device.properties.channels.map((c) => ({
          key: c.nodeId,
          name: c.nodeId,
          dataType: c.dataType,
        }));
      }

      return [];
    },
  });

  Form.useFieldListener("device", () => {
    void nodes.refetch();
  });

  return (
    <Select.Single<string, UANodeEntry>
      columns={SELECT_NODE_COLUMNS}
      data={nodes.data}
      {...props}
    />
  );
};
