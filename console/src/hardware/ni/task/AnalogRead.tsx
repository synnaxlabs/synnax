// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/ni/task/AnalogRead.css";

import { QueryError, task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Button,
  Device,
  Form,
  Header,
  Menu,
  Status,
  Synnax,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { Channel } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Input } from "@synnaxlabs/pluto/input";
import { List } from "@synnaxlabs/pluto/list";
import { Text } from "@synnaxlabs/pluto/text";
import { deep, primitiveIsZero } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import { nanoid } from "nanoid";
import { type ReactElement, useCallback, useRef, useState } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { NI } from "@/hardware/ni";
import { enrich } from "@/hardware/ni/device/enrich/enrich";
import { Properties } from "@/hardware/ni/device/types";
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
  AnalogReadTaskState,
  AnalogReadType,
  type Chan,
  ZERO_AI_CHANNELS,
  ZERO_ANALOG_READ_PAYLOAD,
} from "@/hardware/ni/task/types";
import { Layout } from "@/layout";

import { ANALOG_INPUT_FORMS, NameField, SelectChannelTypeField } from "./ChannelForms";

export const configureAnalogReadLayout: Layout.State = {
  name: "Configure NI Analog Read Task",
  key: ANALOG_READ_TYPE,
  type: ANALOG_READ_TYPE,
  windowKey: ANALOG_READ_TYPE,
  location: "mosaic",
};

export const ConfigureAnalogRead: Layout.Renderer = ({ layoutKey }) => {
  const client = Synnax.use();
  const fetchTask = useQuery<InternalProps>({
    queryKey: [layoutKey, client?.key],
    queryFn: async () => {
      if (client == null || layoutKey == configureAnalogReadLayout.key)
        return { initialValues: deep.copy(ZERO_ANALOG_READ_PAYLOAD) };
      const t = await client.hardware.tasks.retrieve<
        AnalogReadConfig,
        AnalogReadStateDetails,
        AnalogReadType
      >(layoutKey, { includeState: true });
      return { initialValues: t, initialTask: t };
    },
  });
  if (fetchTask.isLoading) return <></>;
  if (fetchTask.isError) return <></>;
  return <Internal {...(fetchTask.data as InternalProps)} />;
};

interface InternalProps {
  initialTask?: AnalogRead;
  initialValues: AnalogReadPayload;
}

const Internal = ({ initialTask, initialValues }: InternalProps): ReactElement => {
  const client = Synnax.use();
  const methods = Form.use({
    values: initialValues,
    schema: z.object({
      name: z.string(),
      config: analogReadTaskConfigZ,
    }),
  });

  const [task, setTask] = useState(initialTask);
  const [taskState, setTaskState] = useState<AnalogReadTaskState | null>(
    initialValues?.state ?? null,
  );

  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);
  const [selectedChannelIndex, setSelectedChannelIndex] = useState<number | null>(null);

  const stateObserverRef = useRef<task.StateObservable<AnalogReadStateDetails> | null>(
    null,
  );

  useAsyncEffect(async () => {
    if (client == null || task == null) return;
    stateObserverRef.current = await task.openStateObserver<AnalogReadStateDetails>();
    stateObserverRef.current.onChange((s) => setTaskState(s));
    return async () => await stateObserverRef.current?.close().catch(console.error);
  }, [client?.key, task?.key, setTaskState]);

  const configure = useMutation({
    mutationKey: [client?.key, "configure"],
    onError: console.error,
    mutationFn: async () => {
      if (!(await methods.validateAsync()) || client == null) return;
      const rack = await client.hardware.racks.retrieve("sy_node_1_rack");
      const { name, config } = methods.value();

      const dev = await client.hardware.devices.retrieve<Properties>(config.device);
      dev.properties = enrich(dev.model, dev.properties);

      let modified = false;
      let shouldCreateIndex = primitiveIsZero(dev.properties.analogInput.index);
      if (!shouldCreateIndex) {
        try {
          await client.channels.retrieve(dev.properties.analogInput.index);
        } catch (e) {
          if (e instanceof QueryError) shouldCreateIndex = true;
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
            if (e instanceof QueryError) toCreate.push(channel);
            else throw e;
          }
        }
      }

      if (toCreate.length > 0) {
        modified = true;
        const channels = await client.channels.create(
          toCreate.map((c) => ({
            name: `${dev.properties.identifier}_ai_${c.port}`,
            dataType: "float32",
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
      methods.set("config", config);
      if (dev == null) return;

      const t = await rack.createTask<
        AnalogReadConfig,
        AnalogReadStateDetails,
        AnalogReadType
      >({
        key: task?.key,
        name,
        type: ANALOG_READ_TYPE,
        config,
      });
      setTask(t);
    },
  });

  const start = useMutation({
    mutationKey: [client?.key, "start"],
    mutationFn: async () => {
      if (client == null) return;
      await task?.executeCommand(
        taskState?.details?.running === true ? "stop" : "start",
      );
    },
  });

  const placer = Layout.usePlacer();

  const handleDeviceChange = async (v: string) => {
    if (client == null) return;
    const { configured } = await client.hardware.devices.retrieve<Properties>(v);
    if (configured) return;
    placer(NI.Device.createConfigureLayout(v, {}));
  };

  return (
    <Align.Space className={CSS.B("ni-analog-read-task")} direction="y" grow empty>
      <Align.Space className={CSS.B("content")} grow>
        <Form.Form {...methods}>
          <Align.Space direction="x">
            <Form.Field<string> path="name">
              {(p) => <Input.Text variant="natural" level="h1" {...p} />}
            </Form.Field>
          </Align.Space>
          <Align.Space direction="x">
            <Form.Field<string>
              path="config.device"
              label="Device"
              grow
              onChange={handleDeviceChange}
            >
              {(p) => (
                <Device.SelectSingle
                  allowNone={false}
                  grow
                  {...p}
                  searchOptions={{ makes: ["NI"] }}
                />
              )}
            </Form.Field>
            <Form.Field<number> label="Sample Rate" path="config.sampleRate">
              {(p) => <Input.Numeric {...p} />}
            </Form.Field>
            <Form.Field<number> label="Stream Rate" path="config.streamRate">
              {(p) => <Input.Numeric {...p} />}
            </Form.Field>
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
              <Header.Header level="h3">
                <Header.Title weight={500}>Channel Details</Header.Title>
              </Header.Header>
              <Align.Space className={CSS.B("details")}>
                {selectedChannelIndex != null && (
                  <ChannelForm selectedChannelIndex={selectedChannelIndex} />
                )}
              </Align.Space>
            </Align.Space>
          </Align.Space>
        </Form.Form>
        <Align.Space
          direction="x"
          style={{
            borderRadius: "1rem",
            border: "var(--pluto-border)",
            padding: "2rem",
          }}
          justify="end"
        >
          <Button.Icon
            loading={start.isPending}
            disabled={start.isPending || taskState == null}
            onClick={() => start.mutate()}
            variant="outlined"
          >
            {taskState?.details?.running === true ? <Icon.Pause /> : <Icon.Play />}
          </Button.Icon>
          <Button.Button
            loading={configure.isPending}
            disabled={configure.isPending}
            onClick={() => configure.mutate()}
          >
            Configure
          </Button.Button>
        </Align.Space>
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
        <NameField path={prefix} />
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
  const { value, push, remove, set } = Form.useFieldArray<Chan>({ path });
  const handleAdd = (): void => {
    push({
      ...deep.copy(ZERO_AI_CHANNELS["ai_voltage"]),
      port: availablePortFinder(value)(),
      key: nanoid(),
    });
  };
  const menuProps = Menu.useContextMenu();
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
              size: "large",
            },
          ]}
        </Header.Actions>
      </Header.Header>
      <Menu.ContextMenu
        menu={({ keys }: Menu.ContextMenuMenuProps): ReactElement => {
          const indices = keys.map((k) => value.findIndex((v) => v.key === k));
          const handleRemove = () => {
            remove(indices);
            onSelect([], -1);
          };
          const handleDuplicate = () => {
            const pf = availablePortFinder(value);
            push(
              indices.map((i) => ({
                ...deep.copy(value[i]),
                channel: 0,
                port: pf(),
                key: nanoid(),
              })),
            );
          };
          const handleDisable = () =>
            set((v) =>
              v.map((c, i) => {
                if (!indices.includes(i)) return c;
                return { ...c, enabled: false };
              }),
            );
          const handleEnable = () =>
            set((v) =>
              v.map((c, i) => {
                if (!indices.includes(i)) return c;
                return { ...c, enabled: true };
              }),
            );
          const allowDisable = indices.some((i) => value[i].enabled);
          const allowEnable = indices.some((i) => !value[i].enabled);
          return (
            <Menu.Menu
              onChange={{
                remove: handleRemove,
                duplicate: handleDuplicate,
                disable: handleDisable,
                enable: handleEnable,
              }}
              level="small"
            >
              <Menu.Item itemKey="remove" startIcon={<Icon.Close />}>
                Remove
              </Menu.Item>
              <Menu.Item itemKey="duplicate" startIcon={<Icon.Copy />}>
                Duplicate
              </Menu.Item>
              <Menu.Divider />
              {allowDisable && (
                <Menu.Item itemKey="disable" startIcon={<Icon.Disable />}>
                  Disable
                </Menu.Item>
              )}
              {allowEnable && (
                <Menu.Item itemKey="enable" startIcon={<Icon.Enable />}>
                  Enable
                </Menu.Item>
              )}
              <Menu.Divider />
              <Menu.Item itemKey="plot" startIcon={<Icon.Visualize />}>
                Plot Live Data
              </Menu.Item>
            </Menu.Menu>
          );
        }}
        {...menuProps}
      >
        <List.List<string, Chan> data={value}>
          <List.Selector<string, Chan>
            value={selected}
            allowNone={false}
            allowMultiple={true}
            onChange={(keys, { clickedIndex }) => {
              console.log(keys);

              clickedIndex != null && onSelect(keys, clickedIndex);
            }}
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
  const channelName = Channel.useName(childValues?.channel ?? 0, "No Synnax Channel");
  const channelValid = Form.useFieldValid(`${path}.channel`);
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
        <Text.Text level="p" shade={6}></Text.Text>
      </Align.Space>
      <Align.Space direction="x" size="small">
        <Align.Space direction="x" size="small" className={CSS.B("hover-actions")}>
          <Button.Icon size="small" variant="outlined" tooltip="Plot Live Data">
            <Icon.Visualize color="var(--pluto-gray-l7)" />
          </Button.Icon>
        </Align.Space>
        <Button.Toggle
          checkedVariant="outlined"
          uncheckedVariant="outlined"
          value={childValues.enabled}
          size="small"
          onClick={(e) => e.stopPropagation()}
          onChange={(v) => {
            ctx.set(`${path}.enabled`, v);
          }}
          tooltip={
            <Text.Text level="small" style={{ maxWidth: 300 }}>
              Data acquisition for this channel is{" "}
              {childValues.enabled ? "enabled" : "disabled"}. Click to
              {childValues.enabled ? " disable" : " enable"} it.
            </Text.Text>
          }
        >
          <Status.Text
            variant={childValues.enabled ? "success" : "disabled"}
            level="small"
            align="center"
          >
            {childValues.enabled ? "Enabled" : "Disabled"}
          </Status.Text>
        </Button.Toggle>
      </Align.Space>
    </List.ItemFrame>
  );
};
