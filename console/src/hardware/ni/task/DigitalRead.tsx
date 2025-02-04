// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
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
import { type ReactElement, useCallback, useState } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { enrich } from "@/hardware/ni/device/enrich/enrich";
import { type Properties } from "@/hardware/ni/device/types";
import { CopyButtons, SelectDevice } from "@/hardware/ni/task/common";
import { createLayoutCreator } from "@/hardware/ni/task/createLayoutCreator";
import {
  type Chan,
  type DIChan,
  DIGITAL_READ_TYPE,
  type DigitalRead,
  type DigitalReadConfig,
  digitalReadConfigZ,
  type DigitalReadDetails,
  type DigitalReadPayload,
  type DigitalReadType,
  ZERO_DI_CHAN,
  ZERO_DIGITAL_READ_PAYLOAD,
} from "@/hardware/ni/task/types";
import {
  ChannelListContextMenu,
  ChannelListEmptyContent,
  ChannelListHeader,
  Controls,
  EnableDisableButton,
  ParentRangeButton,
  useCreate,
  useObserveState,
  type WrappedTaskLayoutProps,
  wrapTaskLayout,
} from "@/hardware/task/common/common";
import {
  checkDesiredStateMatch,
  useDesiredState,
} from "@/hardware/task/common/useDesiredState";
import { type Layout } from "@/layout";

export const createDigitalReadLayout = createLayoutCreator<DigitalReadPayload>(
  DIGITAL_READ_TYPE,
  "New NI Digital Read Task",
);

export const DIGITAL_READ_SELECTABLE: Layout.Selectable = {
  key: DIGITAL_READ_TYPE,
  title: "NI Digital Read Task",
  icon: <Icon.Logo.NI />,
  create: (layoutKey) => ({
    ...createDigitalReadLayout({ create: true }),
    key: layoutKey,
  }),
};

const Wrapped = ({
  task,
  initialValues,
  layoutKey,
}: WrappedTaskLayoutProps<DigitalRead, DigitalReadPayload>): ReactElement => {
  const client = Synnax.use();
  const methods = Form.use({
    values: initialValues,
    schema: z.object({
      name: z.string(),
      config: digitalReadConfigZ,
    }),
  });

  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);
  const [selectedChannelIndex, setSelectedChannelIndex] = useState<number | null>(null);

  const taskState = useObserveState<DigitalReadDetails>(
    methods.setStatus,
    methods.clearStatuses,
    task?.key,
    task?.state,
  );
  const running = taskState?.details?.running;
  const initialState =
    running === true ? "running" : running === false ? "paused" : undefined;
  const [desiredState, setDesiredState] = useDesiredState(initialState, task?.key);

  const createTask = useCreate<DigitalReadConfig, DigitalReadDetails, DigitalReadType>(
    layoutKey,
  );

  const handleException = Status.useExceptionHandler();

  const configure = useMutation({
    onError: (e) => handleException(e, "Failed to configure NI Digital Read Task"),
    mutationFn: async () => {
      if (!(await methods.validateAsync()) || client == null) return;
      const { name, config } = methods.value();

      const dev = await client.hardware.devices.retrieve<Properties>(config.device);
      dev.properties = enrich(dev.model, dev.properties);

      let modified = false;
      let shouldCreateIndex = primitiveIsZero(dev.properties.digitalInput.index);
      if (!shouldCreateIndex)
        try {
          await client.channels.retrieve(dev.properties.digitalInput.index);
        } catch (e) {
          if (NotFoundError.matches(e)) shouldCreateIndex = true;
          else throw e;
        }

      if (shouldCreateIndex) {
        modified = true;
        const aiIndex = await client.channels.create({
          name: `${dev.properties.identifier}_di_time`,
          dataType: "timestamp",
          isIndex: true,
        });
        dev.properties.digitalInput.index = aiIndex.key;
        dev.properties.digitalInput.channels = {};
      }

      const toCreate: DIChan[] = [];
      for (const channel of config.channels) {
        const key = `${channel.port}l${channel.line}`;
        // check if the channel is in properties
        const exKey = dev.properties.digitalInput.channels[key];
        if (primitiveIsZero(exKey)) toCreate.push(channel);
        else
          try {
            await client.channels.retrieve(exKey.toString());
          } catch (e) {
            if (NotFoundError.matches(e)) toCreate.push(channel);
            else throw e;
          }
      }

      if (toCreate.length > 0) {
        modified = true;
        const channels = await client.channels.create(
          toCreate.map((c) => ({
            name: `${dev.properties.identifier}_di_${c.port}_${c.line}`,
            dataType: "uint8",
            index: dev.properties.digitalInput.index,
          })),
        );
        channels.forEach((c, i) => {
          const key = `${toCreate[i].port}l${toCreate[i].line}`;
          dev.properties.digitalInput.channels[key] = c.key;
        });
      }

      if (modified)
        await client.hardware.devices.create({
          ...dev,
          properties: dev.properties,
        });

      config.channels.forEach((c) => {
        const key = `${c.port}l${c.line}`;
        c.channel = dev.properties.digitalInput.channels[key];
      });
      await createTask({
        key: task?.key,
        name,
        type: DIGITAL_READ_TYPE,
        config,
      });
      setDesiredState("paused");
    },
  });

  const start = useMutation({
    mutationFn: async () => {
      if (client == null) return;
      const isRunning = running === true;
      setDesiredState(isRunning ? "paused" : "running");
      await task?.executeCommand(isRunning ? "stop" : "start");
    },
  });

  return (
    <Align.Space className={CSS.B("task-configure")} direction="y" grow empty>
      <Align.Space>
        <Form.Form {...methods} mode={task?.snapshot ? "preview" : "normal"}>
          <Align.Space direction="x" justify="spaceBetween">
            <Form.Field<string> path="name" padHelpText={!task?.snapshot}>
              {(p) => <Input.Text variant="natural" level="h1" {...p} />}
            </Form.Field>
            <CopyButtons
              importClass="DigitalReadTask"
              taskKey={task?.key}
              getName={() => methods.get<string>("name").value}
              getConfig={() => methods.get("config").value}
            />
          </Align.Space>
          <ParentRangeButton taskKey={task?.key} />
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
                  <ChannelForm selectedChannelIndex={selectedChannelIndex} />
                )}
              </Align.Space>
            </Align.Space>
          </Align.Space>
        </Form.Form>
        <Controls
          layoutKey={layoutKey}
          state={taskState}
          snapshot={task?.snapshot}
          startingOrStopping={
            start.isPending ||
            (!checkDesiredStateMatch(desiredState, running) &&
              taskState?.variant === "success")
          }
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
}

const ChannelForm = ({ selectedChannelIndex }: ChannelFormProps): ReactElement => {
  if (selectedChannelIndex == -1) return <></>;
  const prefix = `config.channels.${selectedChannelIndex}`;
  return (
    <Align.Space direction="x" grow>
      <Form.NumericField path={`${prefix}.port`} label="Port" grow />
      <Form.NumericField path={`${prefix}.line`} label="Line" grow />
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
  const { value, push, remove } = Form.useFieldArray<DIChan>({ path });
  const handleAdd = (): void => {
    const availableLine = Math.max(0, ...value.map((v) => v.line)) + 1;
    push({
      ...deep.copy(ZERO_DI_CHAN),
      port: 0,
      line: availableLine,
      key: id.id(),
    });
  };
  const menuProps = Menu.useContextMenu();
  return (
    <Align.Space className={CSS.B("channels")} grow empty>
      <ChannelListHeader onAdd={handleAdd} snapshot={snapshot} />
      <Menu.ContextMenu
        menu={({ keys }: Menu.ContextMenuMenuProps) => (
          <ChannelListContextMenu
            path={path}
            keys={keys}
            value={value}
            remove={remove}
            snapshot={snapshot}
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
        <List.List<string, Chan>
          data={value}
          emptyContent={
            <ChannelListEmptyContent onAdd={handleAdd} snapshot={snapshot} />
          }
        >
          <List.Selector<string, Chan>
            value={selected}
            allowNone={false}
            allowMultiple
            onChange={(keys, { clickedIndex }) =>
              clickedIndex != null && onSelect(keys, clickedIndex)
            }
            replaceOnSingle
          >
            <List.Core<string, Chan> grow>
              {({ key, ...props }) => (
                <ChannelListItem key={key} {...props} snapshot={snapshot} path={path} />
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
}: List.ItemProps<string, Chan> & {
  path: string;
  snapshot?: boolean;
}): ReactElement => {
  const { entry } = props;
  const hasLine = "line" in entry;
  const ctx = Form.useContext();
  const childValues = Form.useChildFieldValues<DIChan>({
    path: `${path}.${props.index}`,
    optional: true,
  });
  const channelName = Channel.useName(childValues?.channel ?? 0, "No Channel");

  const channelValid =
    Form.useField<number>({
      path: `${path}.${props.index}.channel`,
      optional: true,
    })?.status?.variant === "success";

  const portValid =
    Form.useField<number>({
      path: `${path}.${props.index}.port`,
      optional: true,
    })?.status?.variant === "success";
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
          weight={500}
          shade={6}
          style={{ width: "4rem" }}
          color={portValid ? undefined : "var(--pluto-error-z)"}
        >
          {childValues.port}
          {hasLine && `/${entry.line}`}
        </Text.Text>
        <Align.Space direction="y">
          <Text.Text
            level="p"
            weight={500}
            shade={9}
            color={(() => {
              if (channelName === "No Channel") return "var(--pluto-warning-m1)";
              if (channelValid) return undefined;
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

export const ConfigureDigitalRead = wrapTaskLayout(Wrapped, ZERO_DIGITAL_READ_PAYLOAD);
