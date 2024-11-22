// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, NotFoundError, type task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Channel,
  Form,
  Header,
  Input,
  List,
  Menu,
  Select,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { deep, id, type KeyedNamed, primitiveIsZero } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type FC, type ReactElement, useCallback, useState } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { useDevice } from "@/hardware/device/useDevice";
import {
  SelectInputChannelTypeField,
  SelectPort,
} from "@/hardware/labjack/device/Select";
import {
  type ChannelType,
  DEVICES,
  type InputChannelType,
  type ModelKey,
  type Properties,
} from "@/hardware/labjack/device/types";
import { SelectDevice } from "@/hardware/labjack/task/common";
import {
  inputChan,
  type Read,
  READ_TYPE,
  type ReadChan,
  type ReadPayload,
  type ReadStateDetails,
  type ReadTaskConfig,
  readTaskConfigZ,
  type ReadType,
  type Scale,
  SCALE_SCHEMAS,
  type ScaleType,
  type TemperatureUnits,
  thermocoupleChanZ,
  ZERO_READ_CHAN,
  ZERO_READ_PAYLOAD,
  ZERO_SCALES,
  ZERO_THERMOCOUPLE_CHAN,
} from "@/hardware/labjack/task/types";
import {
  ChannelListContextMenu,
  ChannelListEmptyContent,
  ChannelListHeader,
  Controls,
  EnableDisableButton,
  TareButton,
  type TaskLayoutArgs,
  useCreate,
  useObserveState,
  type WrappedTaskLayoutProps,
  wrapTaskLayout,
} from "@/hardware/task/common/common";
import { LabJackThermocoupleTypeField } from "@/hardware/task/common/thermocouple";
import { type Layout } from "@/layout";

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
    schema: z.object({ name: z.string(), config: readTaskConfigZ }),
  });
  const [selectedChannels, setSelectedChannels] = useState<string[]>(
    initialValues.config.channels.length ? [initialValues.config.channels[0].key] : [],
  );
  const [selectedChannelIndex, setSelectedChannelIndex] = useState<number | null>(
    initialValues.config.channels.length > 0 ? 0 : null,
  );
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
    onError: (e) => addStatus({ variant: "error", message: e.message }),
    mutationFn: async () => {
      if (!(await methods.validateAsync()) || client == null) return;
      const { name, config } = methods.value();
      const dev = await client.hardware.devices.retrieve<Properties>(config.device);
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
      const toCreate: ReadChan[] = [];
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
      if (modified)
        await client.hardware.devices.create({
          ...dev,
          properties: dev.properties,
        });
      config.channels.forEach((c) => {
        const type = c.type === "TC" ? "AI" : c.type;
        c.channel = dev.properties[type].channels[c.port];
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
  const handleTare = useMutation({
    mutationKey: [client?.key],
    onError: (e) => addStatus({ variant: "error", message: e.message }),
    mutationFn: async (keys: number[]) => {
      if (client == null) return;
      await task?.executeCommand("tare", { keys });
    },
  }).mutate;
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
              onTare={handleTare}
              state={taskState}
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
          layoutKey={layoutKey}
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
  const model = (device?.model ?? "LJM_dtT4") as ModelKey;
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
              value === "TC" ? ZERO_THERMOCOUPLE_CHAN : ZERO_READ_CHAN,
            );
            const parentPath = path.slice(0, path.lastIndexOf("."));
            const prevParent = get<ReadChan>(parentPath).value;
            const schema = value === "TC" ? thermocoupleChanZ : inputChan;
            const port = DEVICES[model].ports[value === "TC" ? "AI" : value][0].key;
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
          {(p) => <SelectPort {...p} model={model} channelType={channelType} />}
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
  state?: task.State<{ running?: boolean; message?: string }>;
}

const ChannelList = ({
  path,
  selected,
  onSelect,
  snapshot,
  state,
  onTare,
}: ChannelListProps): ReactElement => {
  const { value, push, remove } = Form.useFieldArray<ReadChan>({ path });
  const handleAdd = (): void => push({ ...deep.copy(ZERO_READ_CHAN), key: id.id() });
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
}: List.ItemProps<string, ReadChan> & {
  path: string;
  snapshot?: boolean;
  onTare?: (channelKey: number) => void;
  state?: task.State<{ running?: boolean; message?: string }>;
}): ReactElement => {
  const { entry } = props;
  const ctx = Form.useContext();
  // TODO: Fix bug in useChildFieldValues
  const channels = Form.useChildFieldValues<ReadChan[]>({ path });
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
          <TareButton
            disabled={tareIsDisabled}
            onClick={() => onTare(childValues.channel as number)}
          />
        )}
        <EnableDisableButton
          value={childValues.enabled}
          onChange={(v) => ctx?.set(`${path}.${props.index}.enabled`, v)}
          snapshot={snapshot}
        />
      </Align.Pack>
    </List.ItemFrame>
  );
};

export const ConfigureRead = wrapTaskLayout(Wrapped, ZERO_READ_PAYLOAD);

export const SelectScaleTypeField = Form.buildDropdownButtonSelectField<
  ScaleType,
  KeyedNamed<ScaleType>
>({
  fieldKey: "type",
  fieldProps: {
    label: "Scale",
    onChange: (value, { get, set, path }) => {
      const prevType = get<ScaleType>(path).value;
      if (prevType === value) return;
      const next = deep.copy(ZERO_SCALES[value]);
      const parentPath = path.slice(0, path.lastIndexOf("."));
      const prevParent = get<Scale>(parentPath).value;
      set(parentPath, {
        ...deep.overrideValidItems(next, prevParent, SCALE_SCHEMAS[value]),
        type: next.type,
      });
    },
  },
  inputProps: {
    entryRenderKey: "name",
    columns: [{ key: "name", name: "Name" }],
    data: [
      { key: "none", name: "None" },
      { key: "linear", name: "Linear" },
    ],
  },
});

export interface FormProps {
  prefix: string;
  fieldKey?: string;
  label?: string;
}

const SCALE_FORMS: Record<ScaleType, FC<FormProps>> = {
  linear: ({ prefix }) => (
    <Align.Space direction="x" grow>
      <Form.NumericField path={`${prefix}.slope`} label="Slope" grow />
      <Form.NumericField path={`${prefix}.offset`} label="Offset" grow />
    </Align.Space>
  ),
  none: () => <></>,
};

export const CustomScaleForm = ({ prefix }: FormProps): ReactElement | null => {
  const path = `${prefix}.scale`;
  const channelType = Form.useFieldValue<ChannelType>(`${prefix}.type`, true);
  const scaleType = Form.useFieldValue<ScaleType>(`${path}.type`, true);
  if (channelType !== "AI" || scaleType == null) return null;
  const FormComponent = SCALE_FORMS[scaleType];
  return (
    <>
      <SelectScaleTypeField path={path} />
      <FormComponent prefix={path} />
    </>
  );
};

interface ThermocoupleFormProps extends FormProps {
  model: ModelKey;
}

const ThermocoupleForm = ({
  prefix,
  model,
}: ThermocoupleFormProps): ReactElement | null => {
  const channelType = Form.useFieldValue<ChannelType>(`${prefix}.type`, true);
  if (channelType !== "TC") return null;
  return (
    <Align.Space direction="y" grow>
      <Align.Space direction="x" grow>
        <LabJackThermocoupleTypeField path={prefix} grow />
        <TemperatureUnitsField path={prefix} grow />
      </Align.Space>
      <Align.Space direction="x" grow>
        <Form.NumericField path={`${prefix}.posChan`} label="Positive Channel" grow />
        <Form.NumericField path={`${prefix}.negChan`} label="Negative Channel" grow />
      </Align.Space>
      <Align.Space direction="x" grow>
        <Form.Field<string>
          path={`${prefix}.cjcSource`}
          grow
          hideIfNull
          label="CJC Source"
        >
          {(p) => <SelectCJCSourceField {...p} model={model} />}
        </Form.Field>
        <Form.NumericField path={`${prefix}.cjcSlope`} label="CJC Slope" grow />
        <Form.NumericField path={`${prefix}.cjcOffset`} label="CJC Offset" grow />
      </Align.Space>
    </Align.Space>
  );
};

interface SelectCJCSourceProps extends Select.SingleProps<string, CJCSourceType> {
  model: ModelKey;
}

interface CJCSourceType {
  key: string;
}

const SelectCJCSourceField = ({ model, ...props }: SelectCJCSourceProps) => {
  const ports: CJCSourceType[] = DEVICES[model].ports.AI;
  const data = [
    { key: "TEMPERATURE_DEVICE_K" },
    { key: "TEMPERATURE_AIR_K" },
    ...ports,
  ];
  return (
    <Select.Single<string, CJCSourceType>
      data={data}
      columns={[{ key: "key", name: "CJC Source" }]}
      allowNone={false}
      entryRenderKey="key"
      {...props}
    />
  );
};

const TemperatureUnitsField = Form.buildDropdownButtonSelectField<
  TemperatureUnits,
  KeyedNamed<TemperatureUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Temperature Units" },
  inputProps: {
    entryRenderKey: "name",
    columns: [{ key: "name", name: "Name" }],
    data: [
      { key: "C", name: "Celsius" },
      { key: "F", name: "Fahrenheit" },
      { key: "K", name: "Kelvin" },
    ],
  },
});
