// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, NotFoundError, task as clientTask } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Channel,
  Form,
  Header,
  List,
  Menu,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { deep, id, primitiveIsZero } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type FC, type ReactElement, useCallback, useState } from "react";

import { CSS } from "@/css";
import { Common } from "@/hardware/common";
import { Device } from "@/hardware/labjack/device";
import { CustomScaleForm } from "@/hardware/labjack/task/CustomScaleForm";
import { SelectInputChannelTypeField } from "@/hardware/labjack/task/SelectInputChannelTypeField";
import { ThermocoupleForm } from "@/hardware/labjack/task/ThermocoupleForm";
import {
  type ChannelType,
  type InputChannelType,
  inputChannelZ,
  READ_TYPE,
  type ReadChannel,
  type ReadConfig,
  readConfigZ,
  type ReadStateDetails,
  type ReadType,
  thermocoupleChannelZ,
  ZERO_READ_CHANNEL,
  ZERO_READ_PAYLOAD,
  ZERO_THERMOCOUPLE_CHANNEL,
} from "@/hardware/labjack/task/types";
import { type Layout } from "@/layout";

export const READ_LAYOUT: Common.Task.LayoutBaseState = {
  ...Common.Task.LAYOUT,
  key: READ_TYPE,
  type: READ_TYPE,
  name: ZERO_READ_PAYLOAD.name,
  icon: "Logo.LabJack",
};

export const READ_SELECTABLE: Layout.Selectable = {
  key: READ_TYPE,
  title: "LabJack Read Task",
  icon: <Icon.Logo.LabJack />,
  create: (key) => ({ ...READ_LAYOUT, key }),
};

interface ChannelFormProps {
  selectedChannelIndex?: number | null;
  device?: device.Device;
}

const ChannelForm = ({
  selectedChannelIndex,
  device,
}: ChannelFormProps): ReactElement => {
  const prefix = `config.channels.${selectedChannelIndex}`;
  const channelType = (Form.useFieldValue<ChannelType>(`${prefix}.type`, true) ??
    "AI") as "AI" | "DI" | "TC";
  const model = (device?.model ?? "LJM_dtT4") as Device.ModelKey;
  if (selectedChannelIndex === -1) return <></>;
  return (
    <Align.Space direction="y" size="small">
      <Align.Space direction="x" grow>
        <SelectInputChannelTypeField
          path={prefix}
          onChange={(value, { get, path, set }) => {
            const prevType = get<InputChannelType>(path).value;
            if (prevType === value) return;
            const next = deep.copy(
              value === "TC" ? ZERO_THERMOCOUPLE_CHANNEL : ZERO_READ_CHANNEL,
            );
            const parentPath = path.slice(0, path.lastIndexOf("."));
            const prevParent = get<ReadChannel>(parentPath).value;
            const schema = value === "TC" ? thermocoupleChannelZ : inputChannelZ;
            const port =
              Device.DEVICES[model].ports[value === "TC" ? "AI" : value][0].key;
            set(parentPath, {
              ...deep.overrideValidItems(next, prevParent, schema),
              type: next.type,
            });
            // Need to explicitly set port to cause select port field to rerender
            set(`${parentPath}.port`, port);
          }}
          inputProps={{ allowNone: false }}
          grow
        />
        <Form.Field<string> path={`${prefix}.port`} grow hideIfNull>
          {(p) => (
            <Device.SelectPort
              {...p}
              model={model}
              portType={channelType === "TC" ? "AI" : channelType}
            />
          )}
        </Form.Field>
      </Align.Space>
      <Form.NumericField
        path={`${prefix}.range`}
        optional
        label="Max Voltage"
        inputProps={{ endContent: "V" }}
        grow
      />
      <ThermocoupleForm model={model} prefix={prefix} />
      <CustomScaleForm prefix={prefix} />
    </Align.Space>
  );
};

interface ChannelListProps {
  path: string;
  onSelect: (keys: string[], index: number) => void;
  selected: string[];
  snapshot?: boolean;
  onTare: (keys: number[]) => void;
  state?: clientTask.State<{ running?: boolean; message?: string }>;
}

const ChannelList = ({
  path,
  selected,
  onSelect,
  snapshot,
  state,
  onTare,
}: ChannelListProps): ReactElement => {
  const { value, push, remove } = Form.useFieldArray<ReadChannel>({ path });
  const handleAdd = (): void => push({ ...deep.copy(ZERO_READ_CHANNEL), key: id.id() });
  const menuProps = Menu.useContextMenu();
  return (
    <Align.Space className={CSS.B("channels")} grow empty>
      <Common.Task.ChannelListHeader onAdd={handleAdd} snapshot={snapshot} />
      <Menu.ContextMenu
        menu={({ keys }: Menu.ContextMenuMenuProps) => (
          <Common.Task.ChannelListContextMenu
            path={path}
            keys={keys}
            value={value}
            remove={remove}
            onSelect={onSelect}
            snapshot={snapshot}
            allowTare={
              value.some((v) => v.type === "AI") && state?.details?.running === true
            }
            onTare={onTare}
            onDuplicate={(indices) => {
              const newChannels = indices.map((i) => ({ ...value[i], key: id.id() }));
              push(newChannels);
            }}
          />
        )}
        {...menuProps}
      >
        <List.List<string, ReadChannel>
          data={value}
          emptyContent={
            <Common.Task.ChannelListEmptyContent
              onAdd={handleAdd}
              snapshot={snapshot}
            />
          }
        >
          <List.Selector<string, ReadChannel>
            value={selected}
            allowNone={false}
            allowMultiple
            onChange={(keys, { clickedIndex }) =>
              clickedIndex != null && onSelect(keys, clickedIndex)
            }
            replaceOnSingle
          >
            <List.Core<string, ReadChannel> grow>
              {({ key, ...props }) => (
                <ChannelListItem
                  key={key}
                  {...props}
                  snapshot={snapshot}
                  path={path}
                  state={state}
                  onTare={(key) => onTare([key])}
                />
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
  onTare,
  state,
  ...props
}: List.ItemProps<string, ReadChannel> & {
  path: string;
  snapshot?: boolean;
  onTare?: (channelKey: number) => void;
  state?: clientTask.State<{ running?: boolean; message?: string }>;
}): ReactElement => {
  const { entry } = props;
  const ctx = Form.useContext();
  // TODO: Fix bug in useChildFieldValues
  const channels = Form.useChildFieldValues<ReadChannel[]>({ path });
  const childValues = channels?.[props.index];
  // const childValues = Form.useChildFieldValues<ReadChan>({
  //   path: `${path}.${props.index}`,
  //   optional: true,
  // });
  const channelName = Channel.useName(childValues?.channel ?? 0, "No Channel");
  const channelValid =
    Form.useField<number>({
      path: `${path}.${props.index}.channel`,
      optional: true,
    })?.status.variant === "success";
  if (childValues == null) return <></>;
  const color =
    channelName === "No Channel"
      ? "var(--pluto-warning-m1)"
      : channelValid
        ? undefined
        : "var(--pluto-error-z)";
  const showTareButton =
    childValues.channel != null && onTare != null && childValues.type === "AI";
  const tareIsDisabled =
    !childValues.enabled || snapshot || state?.details?.running !== true;
  return (
    <List.ItemFrame
      {...props}
      entry={childValues}
      justify="spaceBetween"
      align="center"
    >
      <Align.Space direction="x" size="small">
        <Text.Text level="p" shade={6}>
          {entry.port}
        </Text.Text>
        <Align.Space direction="y">
          <Text.Text level="p" shade={9} color={color}>
            {channelName}
          </Text.Text>
        </Align.Space>
      </Align.Space>
      <Align.Pack direction="x" align="center" size="small">
        {showTareButton && (
          <Common.Task.TareButton
            disabled={tareIsDisabled}
            onClick={() => onTare(childValues.channel as number)}
          />
        )}
        <Common.Task.EnableDisableButton
          value={childValues.enabled}
          onChange={(v) => ctx?.set(`${path}.${props.index}.enabled`, v)}
          snapshot={snapshot}
        />
      </Align.Pack>
    </List.ItemFrame>
  );
};

const TaskForm: FC<Common.Task.FormProps<ReadConfig, ReadStateDetails, ReadType>> = ({
  task,
  methods,
  taskState,
}) => {
  const [selectedChannels, setSelectedChannels] = useState<string[]>(
    task.config.channels.length ? [task.config.channels[0].key] : [],
  );
  const [selectedChannelIndex, setSelectedChannelIndex] = useState<number | null>(
    task.config.channels.length > 0 ? 0 : null,
  );
  const handleException = Status.useExceptionHandler();
  const client = Synnax.use();
  const handleTare = useMutation({
    onError: (e) => handleException(e, "Failed to tare channels"),
    mutationFn: async (keys: number[]) => {
      if (client == null) return;
      if (!(task instanceof clientTask.Task)) return;
      await task.executeCommand("tare", { keys });
    },
  }).mutate;
  const dev = Common.Device.use<Device.Properties, Device.Make>(methods);
  return (
    <>
      <Align.Space direction="x" className={CSS.B("task-properties")}>
        <Device.Select />
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
          onTare={handleTare}
          state={taskState}
        />
        <Align.Space className={CSS.B("channel-form")} direction="y" grow>
          <Header.Header level="h4">
            <Header.Title weight={500}>Details</Header.Title>
          </Header.Header>
          <Align.Space className={CSS.B("details")}>
            {selectedChannelIndex != null && (
              <ChannelForm selectedChannelIndex={selectedChannelIndex} device={dev} />
            )}
          </Align.Space>
        </Align.Space>
      </Align.Space>
    </>
  );
};

export const ReadTask = Common.Task.wrapForm(TaskForm, {
  configSchema: readConfigZ,
  type: READ_TYPE,
  zeroPayload: ZERO_READ_PAYLOAD,
  onConfigure: async (client, config) => {
    const dev = await client.hardware.devices.retrieve<Device.Properties>(
      config.device,
    );
    let shouldCreateIndex = false;
    if (dev.properties.readIndex)
      try {
        await client.channels.retrieve(dev.properties.readIndex);
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
    }
    const toCreate: ReadChannel[] = [];
    for (const c of config.channels) {
      const type = c.type === "TC" ? "AI" : c.type;
      const existing = dev.properties[type].channels[c.port];
      // check if the channel is in properties
      if (primitiveIsZero(existing)) toCreate.push(c);
      else
        try {
          await client.channels.retrieve(existing.toString());
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
          dataType: c.type === "DI" ? "uint8" : "float32",
          index: dev.properties.readIndex,
        })),
      );
      channels.forEach((c, i) => {
        const toCreateC = toCreate[i];
        const type = toCreateC.type === "TC" ? "AI" : toCreateC.type;
        dev.properties[type].channels[toCreateC.port] = c.key;
      });
    }
    if (modified) await client.hardware.devices.create(dev);
    config.channels.forEach((c) => {
      const type = c.type === "TC" ? "AI" : c.type;
      c.channel = dev.properties[type].channels[c.port];
    });
    return config;
  },
});
