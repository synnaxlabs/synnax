// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, NotFoundError } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Channel,
  Form,
  Header,
  Input,
  List,
  Menu,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { deep, id, primitiveIsZero } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { ReactElement, useCallback, useState } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { SelectInputChannelType, SelectPort } from "@/hardware/labjack/device/Select";
import {
  ChannelType,
  InputChannelType,
  ModelKey,
  Properties,
} from "@/hardware/labjack/device/types";
import { SelectDevice } from "@/hardware/labjack/task/common";
import {
  Read,
  READ_TYPE,
  ReadChan,
  ReadPayload,
  ReadStateDetails,
  ReadTaskConfig,
  readTaskConfigZ,
  ReadType,
  ZERO_READ_CHAN,
  ZERO_READ_PAYLOAD,
} from "@/hardware/labjack/task/types";
import { useDevice } from "@/hardware/ni/task/common";
import {
  ChannelListContextMenu,
  ChannelListEmptyContent,
  ChannelListHeader,
  Controls,
  EnableDisableButton,
  TaskLayoutArgs,
  useCreate,
  useObserveState,
  WrappedTaskLayoutProps,
  wrapTaskLayout,
} from "@/hardware/task/common/common";
import { Layout } from "@/layout";

type LayoutArgs = TaskLayoutArgs<ReadPayload>;

export const configureReadLayout = (
  args: LayoutArgs = { create: false },
): Layout.State<TaskLayoutArgs<ReadPayload>> => ({
  name: "Configure LabJack Read Task",
  type: READ_TYPE,
  key: id.id(),
  icon: "Logo.LabJack",
  windowKey: READ_TYPE,
  location: "mosaic",
  args,
});

export const READ_SELECTABLE: Layout.Selectable = {
  key: READ_TYPE,
  title: "LabJack Read Task",
  icon: <Icon.Logo.LabJack />,
  create: (layoutKey) => ({
    ...configureReadLayout({ create: true }),
    key: layoutKey,
  }),
};

const Wrapped = ({
  task,
  initialValues,
  layoutKey,
}: WrappedTaskLayoutProps<Read, ReadPayload>): ReactElement => {
  const client = Synnax.use();
  const methods = Form.use({
    values: initialValues,
    schema: z.object({
      name: z.string(),
      config: readTaskConfigZ,
    }),
  });

  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);
  const [selectedChannelIndex, setSelectedChannelIndex] = useState<number | null>(null);

  const taskState = useObserveState<ReadStateDetails>(
    methods.setStatus,
    methods.clearStatuses,
    task?.key,
    task?.state,
  );

  const createTask = useCreate<ReadTaskConfig, ReadStateDetails, ReadType>(layoutKey);

  const addStatus = Status.useAggregator();

  const configure = useMutation({
    mutationKey: [client?.key, "configure"],
    onError: (e) =>
      addStatus({
        variant: "error",
        message: e.message,
      }),
    mutationFn: async () => {
      if (!(await methods.validateAsync()) || client == null) return;
      const { name, config } = methods.value();

      const dev = await client.hardware.devices.retrieve<Properties>(config.deviceKey);
      let shouldCreateIndex = false;
      if (dev.properties.readIndex)
        try {
          await client.channels.retrieve(dev.properties.readIndex);
          config.indexKeys = [dev.properties.readIndex];
        } catch (e) {
          if (NotFoundError.matches(e)) shouldCreateIndex = true;
          else throw e;
        }
      else shouldCreateIndex = true;

      let modified = false;

      if (shouldCreateIndex) {
        modified = true;
        const index = await client.channels.create({
          name: `${dev.properties.identifier}_time`,
          dataType: "timestamp",
          isIndex: true,
        });
        dev.properties.readIndex = index.key;
        config.indexKeys = [index.key];
      }

      const toCreate: ReadChan[] = [];
      for (const c of config.channels) {
        let existingKey = 0;
        const existing = dev.properties[c.type].channels[c.port];
        if (typeof existing === "number") existingKey = existing;
        else if (existing == null) existingKey = 0;
        else existingKey = existing.state;

        // check if the channel is in properties
        if (primitiveIsZero(existingKey)) toCreate.push(c);
        else
          try {
            await client.channels.retrieve(existingKey.toString());
          } catch (e) {
            if (NotFoundError.matches(e)) toCreate.push(c);
            else throw e;
          }
      }

      if (toCreate.length > 0) {
        modified = true;
        const channels = await client.channels.create(
          toCreate.map((c) => ({
            name: `${dev.properties.identifier}_${c.port}`,
            dataType: `${c.dataType}`,
            index: dev.properties.readIndex,
          })),
        );
        channels.forEach((c, i) => {
          const toCreateC = toCreate[i];
          dev.properties[toCreateC.type].channels[toCreateC.port] = c.key;
        });
      }

      if (modified)
        await client.hardware.devices.create({
          ...dev,
          properties: dev.properties,
        });

      config.channels.forEach((c) => {
        c.channel = dev.properties[c.type].channels[c.port];
      });
      await createTask({ key: task?.key, name, type: READ_TYPE, config });
    },
  });

  const start = useMutation({
    mutationKey: [client?.key],
    mutationFn: async () => {
      if (client == null) return;
      await task?.executeCommand(
        taskState?.details?.running === true ? "stop" : "start",
      );
    },
  });

  const dev = useDevice(methods);

  return (
    <Align.Space className={CSS.B("task-configure")} direction="y" grow empty>
      <Align.Space>
        <Form.Form {...methods} mode={task?.snapshot ? "preview" : "normal"}>
          <Align.Space direction="x" justify="spaceBetween">
            <Form.Field<string> path="name">
              {(p) => <Input.Text variant="natural" level="h1" {...p} />}
            </Form.Field>
          </Align.Space>
          <Align.Space direction="x" className={CSS.B("task-properties")}>
            <SelectDevice />
            <Align.Space direction="x">
              <Form.NumericField
                label="Sample Rate"
                path="config.sampleRate"
                inputProps={{ endContent: "Hz" }}
              />
              <Form.NumericField
                label="Stream Rate"
                path="config.streamRate"
                inputProps={{ endContent: "Hz" }}
              />
              <Form.SwitchField label="Data Saving" path="config.dataSaving" />
            </Align.Space>
          </Align.Space>
          <Align.Space
            direction="x"
            className={CSS.B("channel-form-container")}
            bordered
            rounded
            grow
            empty
          >
            <ChannelList
              path="config.channels"
              snapshot={task?.snapshot}
              selected={selectedChannels}
              onSelect={useCallback(
                (v, i) => {
                  setSelectedChannels(v);
                  setSelectedChannelIndex(i);
                },
                [setSelectedChannels, setSelectedChannelIndex],
              )}
            />
            <Align.Space className={CSS.B("channel-form")} direction="y" grow>
              <Header.Header level="h4">
                <Header.Title weight={500}>Details</Header.Title>
              </Header.Header>
              <Align.Space className={CSS.B("details")}>
                {selectedChannelIndex != null && (
                  <ChannelForm
                    selectedChannelIndex={selectedChannelIndex}
                    device={dev}
                  />
                )}
              </Align.Space>
            </Align.Space>
          </Align.Space>
        </Form.Form>
        <Controls
          state={taskState}
          snapshot={task?.snapshot}
          startingOrStopping={start.isPending}
          configuring={configure.isPending}
          onConfigure={configure.mutate}
          onStartStop={start.mutate}
        />
      </Align.Space>
    </Align.Space>
  );
};

interface ChannelFormProps {
  selectedChannelIndex: number;
  device?: device.Device;
}

const ChannelForm = ({
  selectedChannelIndex,
  device,
}: ChannelFormProps): ReactElement => {
  const prefix = `config.channels.${selectedChannelIndex}`;
  const channelType = Form.useFieldValue<ChannelType>(`${prefix}.type`, true) ?? "AI";

  return (
    <Align.Space direction="y" empty>
      <Align.Space direction="x" grow>
        <Form.Field<InputChannelType> path={`${prefix}.type`} label="Type">
          {(p) => <SelectInputChannelType grow {...p} />}
        </Form.Field>
        <Form.Field<string> path={`${prefix}.port`} grow>
          {(p) => (
            <SelectPort
              {...p}
              model={(device?.model ?? "LJM_dtT4") as ModelKey}
              channelType={channelType}
            />
          )}
        </Form.Field>
      </Align.Space>
      <Form.NumericField path={`${prefix}.range`} optional label="Voltage Range" grow />
      <Align.Space direction="x" grow>
        <Input.Item label="Slope" required grow>
          <Input.Numeric value={1} onChange={console.log} />
        </Input.Item>
        <Input.Item label="Offset" required grow>
          <Input.Numeric value={0} onChange={console.log} />
        </Input.Item>
      </Align.Space>
    </Align.Space>
  );
};

interface ChannelListProps {
  path: string;
  onSelect: (keys: string[], index: number) => void;
  selected: string[];
  snapshot?: boolean;
}

const ChannelList = ({
  path,
  selected,
  onSelect,
  snapshot,
}: ChannelListProps): ReactElement => {
  const { value, push, remove } = Form.useFieldArray<ReadChan>({ path });
  const handleAdd = (): void => {
    push({
      ...deep.copy(ZERO_READ_CHAN),
      key: id.id(),
    });
  };
  const menuProps = Menu.useContextMenu();
  return (
    <Align.Space className={CSS.B("channels")} grow empty>
      <ChannelListHeader onAdd={handleAdd} />
      <Menu.ContextMenu
        menu={({ keys }: Menu.ContextMenuMenuProps) => (
          <ChannelListContextMenu
            path={path}
            keys={keys}
            value={value}
            remove={remove}
            onSelect={onSelect}
            onDuplicate={(indices) => {
              const newChannels = indices.map((i) => ({
                ...value[i],
                key: id.id(),
              }));
              push(newChannels);
            }}
          />
        )}
        {...menuProps}
      >
        <List.List<string, ReadChan>
          data={value}
          emptyContent={<ChannelListEmptyContent onAdd={handleAdd} />}
        >
          <List.Selector<string, ReadChan>
            value={selected}
            allowNone={false}
            allowMultiple
            onChange={(keys, { clickedIndex }) =>
              clickedIndex != null && onSelect(keys, clickedIndex)
            }
            replaceOnSingle
          >
            <List.Core<string, ReadChan> grow>
              {(props) => (
                <ChannelListItem {...props} snapshot={snapshot} path={path} />
              )}
            </List.Core>
          </List.Selector>
        </List.List>
      </Menu.ContextMenu>
    </Align.Space>
  );
};

const ChannelListItem = ({
  path,
  snapshot = false,
  ...props
}: List.ItemProps<string, ReadChan> & {
  path: string;
  snapshot?: boolean;
}): ReactElement => {
  const { entry } = props;
  const ctx = Form.useContext();
  const childValues = Form.useChildFieldValues<ReadChan>({
    path: `${path}.${props.index}`,
    optional: true,
  });
  const channelName = Channel.useName(childValues?.channel ?? 0, "No Channel");

  const channelValid =
    Form.useField<number>({
      path: `${path}.${props.index}.channel`,
      optional: true,
    })?.status.variant === "success";

  if (childValues == null) return <></>;
  return (
    <List.ItemFrame
      {...props}
      entry={childValues}
      justify="spaceBetween"
      align="center"
    >
      <Align.Space direction="x" size="small">
        <Text.Text
          level="p"
          shade={6}
          // color={locationValid ? undefined : "var(--pluto-error-z)"}
        >
          {entry.port}
        </Text.Text>
        <Align.Space direction="y">
          <Text.Text
            level="p"
            shade={9}
            color={(() => {
              if (channelName === "No Channel") return "var(--pluto-warning-m1)";
              else if (channelValid) return undefined;
              return "var(--pluto-error-z)";
            })()}
          >
            {channelName}
          </Text.Text>
        </Align.Space>
      </Align.Space>
      <EnableDisableButton
        value={childValues.enabled}
        onChange={(v) => ctx?.set(`${path}.${props.index}.enabled`, v)}
        snapshot={snapshot}
      />
    </List.ItemFrame>
  );
};

export const ConfigureRead = wrapTaskLayout(Wrapped, ZERO_READ_PAYLOAD);
