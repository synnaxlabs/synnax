// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, QueryError } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Button, Form, Header, Menu, Status, Synnax } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Input } from "@synnaxlabs/pluto/input";
import { List } from "@synnaxlabs/pluto/list";
import { Text } from "@synnaxlabs/pluto/text";
import { deep, id, primitiveIsZero } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement, useCallback, useState } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { enrich } from "@/hardware/ni/device/enrich/enrich";
import { Properties } from "@/hardware/ni/device/types";
import { SelectDevice } from "@/hardware/ni/task/common";
import {
  AI_CHANNEL_TYPE_NAMES,
  AIChan,
  AIChanType,
  ANALOG_READ_TYPE,
  AnalogRead as AnalogRead,
  AnalogReadPayload as AnalogReadPayload,
  AnalogReadStateDetails as AnalogReadStateDetails,
  AnalogReadTaskConfig as AnalogReadConfig,
  analogReadTaskConfigZ,
  AnalogReadType,
  type Chan,
  ZERO_AI_CHANNELS,
  ZERO_ANALOG_READ_PAYLOAD,
} from "@/hardware/ni/task/types";
import {
  ChannelListContextMenu,
  ChannelListEmptyContent,
  ChannelListHeader,
  Controls,
  EnableDisableButton,
  useCreate,
  useObserveState,
  WrappedTaskLayoutProps,
  wrapTaskLayout,
} from "@/hardware/task/common/common";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";
import { Layout } from "@/layout";

import { ANALOG_INPUT_FORMS, SelectChannelTypeField } from "./ChannelForms";

export const configureAnalogReadLayout = (create: boolean = false): Layout.State => ({
  name: "Configure NI Analog Read Task",
  key: id.id(),
  type: ANALOG_READ_TYPE,
  windowKey: ANALOG_READ_TYPE,
  location: "mosaic",
  args: { create },
});

export const ANALOG_READ_SELECTABLE: Layout.Selectable = {
  key: ANALOG_READ_TYPE,
  title: "NI Analog Read Task",
  icon: <Icon.Logo.NI />,
  create: (layoutKey) => ({ ...configureAnalogReadLayout(true), key: layoutKey }),
};

const Wrapped = ({
  task,
  initialValues,
  layoutKey,
}: WrappedTaskLayoutProps<AnalogRead, AnalogReadPayload>): ReactElement => {
  const client = Synnax.use();
  const methods = Form.use({
    values: initialValues,
    schema: z.object({
      name: z.string(),
      config: analogReadTaskConfigZ,
    }),
  });

  const [selectedChannels, setSelectedChannels] = useState<string[]>(
    initialValues.config.channels.length ? [initialValues.config.channels[0].key] : [],
  );
  const [selectedChannelIndex, setSelectedChannelIndex] = useState<number | null>(
    initialValues.config.channels.length > 0 ? 0 : null,
  );

  const taskState = useObserveState<AnalogReadStateDetails>(
    methods.setStatus,
    methods.clearStatuses,
    task?.key,
    task?.state,
  );

  const createTask = useCreate<
    AnalogReadConfig,
    AnalogReadStateDetails,
    AnalogReadType
  >(layoutKey);

  const addStatus = Status.useAggregator();

  const configure = useMutation<void, Error, void, unknown>({
    mutationKey: [client?.key, "configure"],
    onError: ({ message }) =>
      addStatus({
        variant: "error",
        message,
      }),
    mutationFn: async () => {
      if (!(await methods.validateAsync()) || client == null) return;
      const { name, config } = methods.value();

      const dev = await client.hardware.devices.retrieve<Properties>(config.device);
      dev.properties = enrich(dev.model, dev.properties);

      let modified = false;
      let shouldCreateIndex = primitiveIsZero(dev.properties.analogInput.index);
      if (!shouldCreateIndex) {
        try {
          await client.channels.retrieve(dev.properties.analogInput.index);
        } catch (e) {
          if (NotFoundError.matches(e)) shouldCreateIndex = true;
          else throw e;
        }
      }

      if (shouldCreateIndex) {
        modified = true;
        const aiIndex = await client.channels.create({
          name: `${dev.properties.identifier}_ai_time`,
          dataType: "timestamp",
          isIndex: true,
        });
        dev.properties.analogInput.index = aiIndex.key;
        dev.properties.analogInput.channels = {};
      }

      const toCreate: AIChan[] = [];
      for (const channel of config.channels) {
        // check if the channel is in properties
        const exKey = dev.properties.analogInput.channels[channel.port.toString()];
        if (primitiveIsZero(exKey)) toCreate.push(channel);
        else {
          try {
            await client.channels.retrieve(exKey.toString());
          } catch (e) {
            if (QueryError.matches(e)) toCreate.push(channel);
            else throw e;
          }
        }
      }

      if (toCreate.length > 0) {
        modified = true;
        const channels = await client.channels.create(
          toCreate.map((c) => ({
            name: `${dev.properties.identifier}_ai_${c.port}`,
            dataType: "float32", // TODO: also support float64
            index: dev.properties.analogInput.index,
          })),
        );
        channels.forEach((c, i) => {
          dev.properties.analogInput.channels[toCreate[i].port.toString()] = c.key;
        });
      }

      if (modified)
        await client.hardware.devices.create({
          ...dev,
          properties: dev.properties,
        });

      config.channels.forEach((c) => {
        c.channel = dev.properties.analogInput.channels[c.port.toString()];
      });
      if (dev == null) return;
      await createTask({
        key: task?.key,
        name,
        type: ANALOG_READ_TYPE,
        config,
      });
    },
  });

  const startOrStop = useMutation({
    mutationKey: [client?.key, "start"],
    mutationFn: async () => {
      if (client == null) return;
      await task?.executeCommand(
        taskState?.details?.running === true ? "stop" : "start",
      );
    },
  });

  const copy = useCopyToClipboard();
  const handleCopyPythonCode = () => {
    const name = methods.get("name").value;
    copy(
      `
      from synnax.hardware.ni import AnalogReadTask
      # Retrieve ${name}
      task = AnalogReadTask(client.hardware.tasks.retrieve(key=${task?.key}))
      `,
      `Python code for ${name}`,
    );
  };

  return (
    <Align.Space className={CSS.B("task-configure")} direction="y" grow empty>
      <Align.Space grow>
        <Form.Form {...methods}>
          <Align.Space direction="x" justify="spaceBetween">
            <Form.Field<string> path="name">
              {(p) => <Input.Text variant="natural" level="h1" {...p} />}
            </Form.Field>
            <Button.Icon
              tooltip={`Copy Python code for ${methods.get("name").value}`}
              tooltipLocation="left"
              variant="text"
              onClick={handleCopyPythonCode}
            >
              <Icon.Python style={{ color: "var(--pluto-gray-l7)" }} />
            </Button.Icon>
          </Align.Space>
          <Align.Space direction="x" className={CSS.B("task-properties")}>
            <SelectDevice />
            <Align.Space direction="x">
              <Form.Field<number> label="Sample Rate" path="config.sampleRate">
                {(p) => <Input.Numeric {...p} />}
              </Form.Field>
              <Form.Field<number> label="Stream Rate" path="config.streamRate">
                {(p) => <Input.Numeric {...p} />}
              </Form.Field>
              <Form.SwitchField path="config.dataSaving" label="Data Saving" />
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
                <Header.Title weight={500} wrap={false}>
                  Details
                </Header.Title>
              </Header.Header>
              <Align.Space className={CSS.B("details")}>
                {selectedChannelIndex != null && (
                  <ChannelForm selectedChannelIndex={selectedChannelIndex} />
                )}
              </Align.Space>
            </Align.Space>
          </Align.Space>
        </Form.Form>
        <Controls
          state={taskState}
          startingOrStopping={startOrStop.isPending}
          configuring={configure.isPending}
          onStartStop={startOrStop.mutate}
          onConfigure={configure.mutate}
          snapshot={task?.snapshot}
        />
      </Align.Space>
    </Align.Space>
  );
};

interface ChannelFormProps {
  selectedChannelIndex: number;
}

const ChannelForm = ({ selectedChannelIndex }: ChannelFormProps): ReactElement => {
  const prefix = `config.channels.${selectedChannelIndex}`;
  const type = Form.useFieldValue<AIChanType>(`${prefix}.type`, true);
  if (type == null) return <></>;
  const TypeForm = ANALOG_INPUT_FORMS[type];
  if (selectedChannelIndex == -1) return <></>;

  return (
    <>
      <Align.Space direction="y" className={CSS.B("channel-form-content")} empty>
        <SelectChannelTypeField path={prefix} inputProps={{ allowNone: false }} />
        <TypeForm prefix={prefix} />
      </Align.Space>
    </>
  );
};

interface ChannelListProps {
  path: string;
  onSelect: (keys: string[], index: number) => void;
  selected: string[];
}

const availablePortFinder = (channels: Chan[]): (() => number) => {
  const exclude = new Set(channels.map((v) => v.port));
  return () => {
    let i = 0;
    while (exclude.has(i)) i++;
    exclude.add(i);
    return i;
  };
};

const ChannelList = ({ path, selected, onSelect }: ChannelListProps): ReactElement => {
  const { value, push, remove } = Form.useFieldArray<Chan>({ path });
  const handleAdd = (): void => {
    const key = id.id();
    push({
      ...deep.copy(ZERO_AI_CHANNELS["ai_voltage"]),
      port: availablePortFinder(value)(),
      key,
    });
    onSelect([key], value.length);
  };
  const menuProps = Menu.useContextMenu();
  return (
    <Align.Space className={CSS.B("channels")} grow empty>
      <ChannelListHeader onAdd={handleAdd} />
      <Menu.ContextMenu
        menu={({ keys }: Menu.ContextMenuMenuProps): ReactElement => (
          <ChannelListContextMenu
            path={path}
            keys={keys}
            value={value}
            remove={remove}
            onSelect={onSelect}
            onDuplicate={(indices) => {
              const pf = availablePortFinder(value);
              push(
                indices.map((i) => ({
                  ...deep.copy(value[i]),
                  channel: 0,
                  port: pf(),
                  key: id.id(),
                })),
              );
            }}
          />
        )}
        {...menuProps}
      >
        <List.List<string, Chan>
          data={value}
          emptyContent={<ChannelListEmptyContent onAdd={handleAdd} />}
        >
          <List.Selector<string, Chan>
            value={selected}
            allowNone={false}
            allowMultiple={true}
            onChange={(keys, { clickedIndex }) =>
              clickedIndex != null && onSelect(keys, clickedIndex)
            }
            replaceOnSingle
          >
            <List.Core<string, Chan> grow>
              {(props) => <ChannelListItem {...props} path={path} />}
            </List.Core>
          </List.Selector>
        </List.List>
      </Menu.ContextMenu>
    </Align.Space>
  );
};

const ChannelListItem = ({
  path: basePath,
  ...props
}: List.ItemProps<string, Chan> & {
  path: string;
}): ReactElement => {
  const ctx = Form.useContext();
  const path = `${basePath}.${props.index}`;
  const childValues = Form.useChildFieldValues<AIChan>({ path, optional: true });
  if (childValues == null) return <></>;
  const portValid = Form.useFieldValid(`${path}.port`);
  return (
    <List.ItemFrame
      {...props}
      entry={childValues}
      justify="spaceBetween"
      align="center"
    >
      <Align.Space direction="y" size="small">
        <Align.Space direction="x">
          <Text.Text
            level="p"
            weight={500}
            shade={6}
            style={{ width: "3rem" }}
            color={portValid ? undefined : "var(--pluto-error-z)"}
          >
            {childValues.port}
          </Text.Text>
          <Text.Text level="p" weight={500} shade={9}>
            {AI_CHANNEL_TYPE_NAMES[childValues.type]}
          </Text.Text>
        </Align.Space>
      </Align.Space>
      <EnableDisableButton
        value={childValues.enabled}
        onChange={(v) => ctx.set(`${path}.enabled`, v)}
      />
    </List.ItemFrame>
  );
};

export const ConfigureAnalogRead = wrapTaskLayout(Wrapped, ZERO_ANALOG_READ_PAYLOAD);
