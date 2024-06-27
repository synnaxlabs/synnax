// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { QueryError, task } from "@synnaxlabs/client";
import {
  Align,
  Channel,
  Device,
  Form,
  Header,
  Input,
  List,
  Observe,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { deep, primitiveIsZero } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { nanoid } from "nanoid";
import { ReactElement, useCallback, useState } from "react";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { CSS } from "@/css";
import { enrich } from "@/hardware/ni/device/enrich/enrich";
import { Properties } from "@/hardware/ni/device/types";
import {
  ChannelListEmptyContent,
  ChannelListHeader,
  Controls,
  EnableDisableButton,
} from "@/hardware/ni/task/common";
import {
  Chan,
  DIChan,
  DIGITAL_READ_TYPE,
  DigitalRead,
  DigitalReadConfig,
  digitalReadConfigZ,
  DigitalReadPayload,
  DigitalReadStateDetails,
  DigitalReadType,
  ZERO_DI_CHAN,
  ZERO_DIGITAL_READ_PAYLOAD,
} from "@/hardware/ni/task/types";
import { wrapTaskLayout } from "@/hardware/task/TaskWrapper";
import { Layout } from "@/layout";

interface ConfigureDigitalReadArgs {
  create: boolean;
}

export const configureDigitalReadLayout = (
  create: boolean = false,
): Layout.State<ConfigureDigitalReadArgs> => ({
  name: "Configure NI Digital Read Task",
  type: DIGITAL_READ_TYPE,
  key: nanoid(),
  windowKey: DIGITAL_READ_TYPE,
  location: "mosaic",
  args: { create },
});

interface InternalProps {
  layoutKey: string;
  task?: DigitalRead;
  initialValues: DigitalReadPayload;
}

const Internal = ({
  task: pTask,
  initialValues,
  layoutKey,
}: InternalProps): ReactElement => {
  const client = Synnax.use();
  const methods = Form.use({
    values: initialValues,
    schema: z.object({
      name: z.string(),
      config: digitalReadConfigZ,
    }),
  });

  const [task, setTask] = useState(pTask);
  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);
  const [selectedChannelIndex, setSelectedChannelIndex] = useState<number | null>(null);

  const taskState = Observe.useState({
    key: [task?.key],
    open: async () => await task?.openStateObserver<DigitalReadStateDetails>(),
    initialValue: task?.state,
  });

  const dispatch = useDispatch();

  const configure = useMutation({
    mutationKey: [client?.key, "configure"],
    mutationFn: async () => {
      if (!(await methods.validateAsync()) || client == null) return;
      const rack = await client.hardware.racks.retrieve("sy_node_1_rack");
      const { name, config } = methods.value();

      const dev = await client.hardware.devices.retrieve<Properties>(config.device);
      dev.properties = enrich(dev.model, dev.properties);

      let modified = false;
      let shouldCreateIndex = primitiveIsZero(dev.properties.digitalInput.index);
      if (!shouldCreateIndex) {
        try {
          await client.channels.retrieve(dev.properties.digitalInput.index);
        } catch (e) {
          if (e instanceof QueryError) shouldCreateIndex = true;
          else throw e;
        }
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
      methods.set("config", config);

      const t = await rack.createTask<
        DigitalReadConfig,
        DigitalReadStateDetails,
        DigitalReadType
      >({
        key: task?.key,
        name,
        type: DIGITAL_READ_TYPE,
        config,
      });
      dispatch(Layout.setAltKey({ key: layoutKey, altKey: t.key }));
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

  return (
    <Align.Space className={CSS.B("task-configure")} direction="y" grow empty>
      <Align.Space>
        <Form.Form {...methods}>
          <Form.Field<string> path="name">
            {(p) => <Input.Text variant="natural" level="h1" {...p} />}
          </Form.Field>
          <Align.Space direction="x">
            <Form.Field<string> path="config.device" label="Device" grow>
              {(p) => (
                <Device.SelectSingle
                  allowNone={false}
                  grow
                  {...p}
                  searchOptions={{ makes: ["NI"] }}
                />
              )}
            </Form.Field>
            <Form.NumericField label="Sample Rate" path="config.sampleRate" />
            <Form.NumericField label="Stream Rate" path="config.streamRate" />
            <Form.SwitchField label="Data Saving" path="config.dataSaving" />
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
          state={taskState}
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
}

const ChannelForm = ({ selectedChannelIndex }: ChannelFormProps): ReactElement => {
  if (selectedChannelIndex == -1) return <></>;
  const prefix = `config.channels.${selectedChannelIndex}`;
  return (
    <Align.Space direction="y" className={CSS.B("channel-form-content")} empty>
      <Align.Space direction="x" grow>
        <Form.NumericField path={`${prefix}.port`} label="Port" grow />
        <Form.NumericField path={`${prefix}.line`} label="Line" grow />
      </Align.Space>
    </Align.Space>
  );
};

interface ChannelListProps {
  path: string;
  onSelect: (keys: string[], index: number) => void;
  selected: string[];
}

const ChannelList = ({ path, selected, onSelect }: ChannelListProps): ReactElement => {
  const { value, push } = Form.useFieldArray<DIChan>({ path });
  const handleAdd = (): void => {
    const availableLine = Math.max(0, ...value.map((v) => v.line)) + 1;
    push({
      ...deep.copy(ZERO_DI_CHAN),
      port: 0,
      line: availableLine,
      key: nanoid(),
    });
  };
  return (
    <Align.Space className={CSS.B("channels")} grow empty>
      <ChannelListHeader onAdd={handleAdd} />
      <List.List<string, Chan>
        data={value}
        emptyContent={<ChannelListEmptyContent onAdd={handleAdd} />}
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
            {(props) => <ChannelListItem {...props} path={path} />}
          </List.Core>
        </List.Selector>
      </List.List>
    </Align.Space>
  );
};

const ChannelListItem = ({
  path,
  ...props
}: List.ItemProps<string, Chan> & {
  path: string;
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
    }).status.variant === "success";

  const portValid =
    Form.useField<number>({
      path: `${path}.${props.index}.port`,
    }).status.variant === "success";
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
              if (channelName === "No Channel") return "var(--pluto-warning-z)";
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
      />
    </List.ItemFrame>
  );
};

export const ConfigureDigitalRead = wrapTaskLayout<DigitalRead, DigitalReadPayload>(
  Internal,
  ZERO_DIGITAL_READ_PAYLOAD,
);
