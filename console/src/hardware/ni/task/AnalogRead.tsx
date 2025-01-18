// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, QueryError, type task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Button, Form, Header, Menu, Status, Synnax } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Input } from "@synnaxlabs/pluto/input";
import { List } from "@synnaxlabs/pluto/list";
import { Text } from "@synnaxlabs/pluto/text";
import { binary, deep, id, type migrate, primitiveIsZero, unique } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement, useCallback, useState } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { enrich } from "@/hardware/ni/device/enrich/enrich";
import { type Properties } from "@/hardware/ni/device/types";
import {
  ANALOG_INPUT_FORMS,
  SelectChannelTypeField,
} from "@/hardware/ni/task/ChannelForms";
import { CopyButtons } from "@/hardware/ni/task/common";
import { createLayoutCreator } from "@/hardware/ni/task/createLayoutCreator";
import {
  AI_CHANNEL_TYPE_NAMES,
  type AIChannel,
  type AIChannelType,
  ANALOG_READ_TYPE,
  type AnalogRead,
  type AnalogReadConfig,
  analogReadConfigZ,
  type AnalogReadDetails,
  type AnalogReadPayload,
  type AnalogReadType,
  migrateAnalogReadConfig,
  ZERO_AI_CHANNELS,
  ZERO_ANALOG_READ_PAYLOAD,
} from "@/hardware/ni/task/types";
import {
  ChannelListContextMenu,
  ChannelListEmptyContent,
  ChannelListHeader,
  Controls,
  EnableDisableButton,
  ParentRangeButton,
  TareButton,
  useCreate,
  useObserveState,
  type WrappedTaskLayoutProps,
  wrapTaskLayout,
} from "@/hardware/task/common/common";
import {
  checkDesiredStateMatch,
  useDesiredState,
} from "@/hardware/task/common/useDesiredState";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";
import { type Layout } from "@/layout";

export const createAnalogReadLayout = createLayoutCreator<AnalogReadPayload>(
  ANALOG_READ_TYPE,
  "New NI Analog Read Task",
);

export const ANALOG_READ_SELECTABLE: Layout.Selectable = {
  key: ANALOG_READ_TYPE,
  title: "NI Analog Read Task",
  icon: <Icon.Logo.NI />,
  create: (layoutKey) => ({
    ...createAnalogReadLayout({ create: true }),
    key: layoutKey,
  }),
};

const schema = z.object({ name: z.string(), config: analogReadConfigZ });

const Wrapped = ({
  task,
  initialValues,
  layoutKey,
}: WrappedTaskLayoutProps<AnalogRead, AnalogReadPayload>): ReactElement => {
  const client = Synnax.use();
  const methods = Form.use({ values: initialValues, schema });

  const [selectedChannels, setSelectedChannels] = useState<string[]>(
    initialValues.config.channels.length ? [initialValues.config.channels[0].key] : [],
  );
  const [selectedChannelIndex, setSelectedChannelIndex] = useState<number | null>(
    initialValues.config.channels.length > 0 ? 0 : null,
  );

  const taskState = useObserveState<AnalogReadDetails>(
    methods.setStatus,
    methods.clearStatuses,
    task?.key,
    task?.state,
  );
  const running = taskState?.details?.running;
  const initialState =
    running === true ? "running" : running === false ? "paused" : undefined;
  const [desiredState, setDesiredState] = useDesiredState(initialState, task?.key);

  const createTask = useCreate<AnalogReadConfig, AnalogReadDetails, AnalogReadType>(
    layoutKey,
  );

  const handleException = Status.useExceptionHandler();

  const configure = useMutation<void, Error, void, unknown>({
    onError: (e) => handleException(e, "Failed to configure NI Analog Read Task}"),
    mutationFn: async () => {
      if (!(await methods.validateAsync()) || client == null) return;
      const { name, config } = methods.value();
      const devices = unique.unique(config.channels.map((c) => c.device));

      for (const devKey of devices) {
        const dev = await client.hardware.devices.retrieve<Properties>(devKey);
        dev.properties = enrich(dev.model, dev.properties);

        let modified = false;
        let shouldCreateIndex = primitiveIsZero(dev.properties.analogInput.index);
        if (!shouldCreateIndex)
          try {
            await client.channels.retrieve(dev.properties.analogInput.index);
          } catch (e) {
            if (NotFoundError.matches(e)) shouldCreateIndex = true;
            else throw e;
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

        const toCreate: AIChannel[] = [];
        for (const channel of config.channels) {
          if (channel.device !== dev.key) continue;
          // check if the channel is in properties
          const exKey = dev.properties.analogInput.channels[channel.port.toString()];
          if (primitiveIsZero(exKey)) toCreate.push(channel);
          else
            try {
              await client.channels.retrieve(exKey.toString());
            } catch (e) {
              if (QueryError.matches(e)) toCreate.push(channel);
              else throw e;
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
          channels.forEach(
            (c, i) =>
              (dev.properties.analogInput.channels[toCreate[i].port.toString()] =
                c.key),
          );
        }

        if (modified)
          await client.hardware.devices.create({
            ...dev,
            properties: dev.properties,
          });

        config.channels.forEach((c) => {
          if (c.device !== dev.key) return;
          c.channel = dev.properties.analogInput.channels[c.port.toString()];
        });
      }
      await createTask({
        key: task?.key,
        name,
        type: ANALOG_READ_TYPE,
        config,
      });
      setDesiredState("paused");
    },
  });

  const startOrStop = useMutation({
    mutationFn: async () => {
      if (client == null) return;
      const isRunning = running === true;
      setDesiredState(isRunning ? "paused" : "running");
      await task?.executeCommand(isRunning ? "stop" : "start");
    },
  });

  const handleTare = useMutation({
    onError: (e) => handleException(e, "Failed to tare channels"),
    mutationFn: async (keys: number[]) => {
      if (client == null) return;
      await task?.executeCommand("tare", { keys });
    },
  }).mutate;

  return (
    <Align.Space className={CSS.B("task-configure")} direction="y" grow empty>
      <Align.Space grow>
        <Form.Form {...methods} mode={task?.snapshot ? "preview" : "normal"}>
          <Align.Space direction="x" justify="spaceBetween">
            <Form.Field<string> path="name" padHelpText={!task?.snapshot}>
              {(p) => (
                <Input.Text
                  variant={task?.snapshot ? "preview" : "natural"}
                  level="h1"
                  {...p}
                />
              )}
            </Form.Field>
            <CopyButtons
              importClass="AnalogReadTask"
              taskKey={task?.key}
              getName={() => methods.get<string>("name").value}
              getConfig={() => methods.get("config").value}
            />
          </Align.Space>
          <ParentRangeButton taskKey={task?.key} />
          <Align.Space direction="x" className={CSS.B("task-properties")}>
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
              snapshot={task?.snapshot}
              path="config.channels"
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
            <ChannelDetails selectedChannelIndex={selectedChannelIndex} />
          </Align.Space>
        </Form.Form>
        <Controls
          layoutKey={layoutKey}
          state={taskState}
          startingOrStopping={
            startOrStop.isPending ||
            (!checkDesiredStateMatch(desiredState, running) &&
              taskState?.variant === "success")
          }
          configuring={configure.isPending}
          onStartStop={startOrStop.mutate}
          onConfigure={configure.mutate}
          snapshot={task?.snapshot}
        />
      </Align.Space>
    </Align.Space>
  );
};

interface ChannelDetailsProps {
  selectedChannelIndex?: number | null;
}

const ChannelDetails = ({
  selectedChannelIndex,
}: ChannelDetailsProps): ReactElement => {
  const ctx = Form.useContext();
  const copy = useCopyToClipboard();
  const handleCopyChannelDetails = () => {
    if (selectedChannelIndex == null) return;
    copy(
      binary.JSON_CODEC.encodeString(
        ctx.get(`config.channels.${selectedChannelIndex}`).value,
      ),
      "Channel details",
    );
  };

  return (
    <Align.Space className={CSS.B("channel-form")} direction="y" grow>
      <Header.Header level="h4">
        <Header.Title weight={500} wrap={false}>
          Details
        </Header.Title>
        <Header.Actions>
          <Button.Icon
            tooltip="Copy channel details as JSON"
            tooltipLocation="left"
            variant="text"
            onClick={handleCopyChannelDetails}
          >
            <Icon.JSON style={{ color: "var(--pluto-gray-l7)" }} />
          </Button.Icon>
        </Header.Actions>
      </Header.Header>
      <Align.Space className={CSS.B("details")}>
        {selectedChannelIndex != null && (
          <ChannelForm selectedChannelIndex={selectedChannelIndex} />
        )}
      </Align.Space>
    </Align.Space>
  );
};

interface ChannelFormProps {
  selectedChannelIndex: number;
}

const ChannelForm = ({ selectedChannelIndex }: ChannelFormProps): ReactElement => {
  const prefix = `config.channels.${selectedChannelIndex}`;
  const type = Form.useFieldValue<AIChannelType>(`${prefix}.type`, true);
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
  snapshot?: boolean;
  onTare: (keys: number[]) => void;
  state?: task.State<{ running?: boolean; message?: string }>;
}

const availablePortFinder = (channels: AIChannel[]): (() => number) => {
  const exclude = new Set(channels.map((v) => v.port));
  return () => {
    let i = 0;
    while (exclude.has(i)) i++;
    exclude.add(i);
    return i;
  };
};

const ChannelList = ({
  path,
  snapshot,
  selected,
  onSelect,
  state,
  onTare,
}: ChannelListProps): ReactElement => {
  const { value, push, remove } = Form.useFieldArray<AIChannel>({ path });
  const handleAdd = (): void => {
    const key = id.id();
    push({
      ...deep.copy(ZERO_AI_CHANNELS.ai_voltage),
      port: availablePortFinder(value)(),
      key,
    });
    onSelect([key], value.length);
  };
  const menuProps = Menu.useContextMenu();
  return (
    <Align.Space className={CSS.B("channels")} grow empty>
      <ChannelListHeader onAdd={handleAdd} snapshot={snapshot} />
      <Menu.ContextMenu
        menu={({ keys }: Menu.ContextMenuMenuProps): ReactElement => (
          <ChannelListContextMenu
            path={path}
            keys={keys}
            value={value}
            remove={remove}
            onSelect={onSelect}
            snapshot={snapshot}
            onTare={onTare}
            allowTare={state?.details?.running === true}
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
        <List.List<string, AIChannel>
          data={value}
          emptyContent={
            <ChannelListEmptyContent onAdd={handleAdd} snapshot={snapshot} />
          }
        >
          <List.Selector<string, AIChannel>
            value={selected}
            allowNone={false}
            allowMultiple
            onChange={(keys, { clickedIndex }) =>
              clickedIndex != null && onSelect(keys, clickedIndex)
            }
            replaceOnSingle
          >
            <List.Core<string, AIChannel> grow>
              {({ key: i, ...props }) => (
                <ChannelListItem
                  {...props}
                  key={i}
                  path={path}
                  snapshot={snapshot}
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
  path: basePath,
  snapshot = false,
  onTare,
  state,
  ...props
}: List.ItemProps<string, AIChannel> & {
  path: string;
  snapshot?: boolean;
  onTare?: (channelKey: number) => void;
  state?: task.State<{ running?: boolean; message?: string }>;
}): ReactElement => {
  const ctx = Form.useContext();
  const path = `${basePath}.${props.index}`;
  const portValid = Form.useFieldValid(`${path}.port`);

  // TODO: fix bug so I can refactor this to original code
  const channels = Form.useChildFieldValues<AIChannel[]>({ path: basePath });
  if (channels == null || props?.index == null) return <></>;
  const childValues = channels[props.index];
  // const childValues = Form.useChildFieldValues<AIChan>({ path, optional: true });

  if (childValues == null) return <></>;
  const showTareButton = childValues.channel != null && onTare != null;
  const tareIsDisabled =
    !childValues.enabled || snapshot || state?.details?.running !== true;
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
      <Align.Pack direction="x" align="center" size="small">
        {showTareButton && (
          <TareButton
            disabled={tareIsDisabled}
            onClick={() => onTare(childValues.channel)}
          />
        )}
        <EnableDisableButton
          value={childValues.enabled}
          onChange={(v) => ctx.set(`${path}.enabled`, v)}
          snapshot={snapshot}
        />
      </Align.Pack>
    </List.ItemFrame>
  );
};

export const ConfigureAnalogRead = wrapTaskLayout(
  Wrapped,
  ZERO_ANALOG_READ_PAYLOAD,
  migrateAnalogReadConfig as migrate.Migrator,
);
