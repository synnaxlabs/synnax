// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/mqtt/task/Form.css";

import { channel, DisconnectedError, mqtt } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Synnax } from "@synnaxlabs/pluto";
import { type DataType, deep, id, type payload, primitive } from "@synnaxlabs/x";
import { type FC, useState } from "react";

import { use, useFromConfig } from "@/feature/mqtt/device/queries";
import { Select as SelectDevice, useNoneSelected } from "@/feature/mqtt/device/Select";
import { type Device, MAKE, SCHEMAS } from "@/feature/mqtt/device/types";
import { useConnectModal } from "@/feature/mqtt/device/useConnectModal";
import { kindOf } from "@/feature/mqtt/task/createReadFields";
import { createDiscoverModal, type Kind } from "@/feature/mqtt/task/discover/Discover";
import { EntryLabel } from "@/feature/mqtt/task/EntryLabel";
import { QoSField } from "@/feature/mqtt/task/QoSField";
import {
  fromSparkplugDataType,
  sparkplugChannelName,
  sparkplugPropertiesKey,
  toSparkplugDataType,
} from "@/feature/mqtt/task/sparkplug";
import { SparkplugTagFields } from "@/feature/mqtt/task/SparkplugTagFields";
import { SparkplugTypeField } from "@/feature/mqtt/task/SparkplugTypeField";
import {
  type ChannelField,
  deployWriteConfigZ,
  type PlainWriteTarget,
  type ReadField,
  type SparkplugTag,
  type SparkplugWriteTarget,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteSchemas,
  type WriteTarget,
} from "@/feature/mqtt/task/types";
import { Button as PlatformButton } from "@/platform/button";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { Empty } from "@/platform/empty";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<WriteSchemas>>();

const Properties = () => <SelectDevice />;

/** The name deploying gives the command channel of a plain target. */
const commandChannelName = (devName: string, topic: string): string =>
  `${channel.escapeInvalidName(devName)}_${channel.escapeInvalidName(topic)}_cmd`;

/** The name deploying gives a target's command channel while it has none. */
const defaultCommandName = (devName: string, target: WriteTarget): string =>
  target.type === "plain"
    ? commandChannelName(devName, target.topic)
    : `${sparkplugChannelName(devName, target)}_cmd`;

/** A plain target holds its channel in a field; a Sparkplug B target is the field. */
const channelPaths = (path: deep.Path<WriteTarget>, type: WriteTarget["type"]) => {
  if (type === "plain") {
    const { channel } = deep.variant(path, "type", "plain");
    return { key: channel.channel, name: channel.name };
  }
  const target = deep.variant(path, "type", "sparkplug");
  return { key: target.channel, name: target.name };
};

const TargetItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = root.config.targets[itemKey];
  const target = PForm.useFieldValue(path);
  const device = useFromConfig();
  const paths = channelPaths(path, target.type);
  const channelKey = PForm.useFieldValue(paths.key);
  return (
    <Select.Item
      {...props}
      y
      align="start"
      gap="small"
      className={CSS.cls(target.disabled && CSS.M("off"))}
    >
      <Flex.Box x align="center" gap="small" full="x">
        <EntryLabel path={path} />
        <Task.EnabledCheckbox path={path.disabled} />
      </Flex.Box>
      <Task.ChannelName
        channel={channelKey}
        namePath={paths.name}
        defaultName={
          device == null ? undefined : defaultCommandName(device.name, target)
        }
        skipped={target.disabled}
        id={Task.getChannelNameID(itemKey)}
        level="small"
        weight={450}
        color={9}
        overflow="fade"
      />
    </Select.Item>
  );
};

const targetItem = Component.renderProp(TargetItem);

const EMPTY_TARGETS = <Empty.Action message="No targets" />;

interface PaneProps<T> {
  path: deep.Path<T>;
  deviceName: string;
}

const SparkplugTargetPane: FC<PaneProps<SparkplugWriteTarget>> = ({
  path,
  deviceName,
}) => {
  const target = PForm.useFieldValue(path);
  return (
    <PForm.Sections>
      <PForm.Section title="Tag">
        <SparkplugTagFields path={path} />
      </PForm.Section>
      <PForm.Section title="Channel">
        <Task.ChannelNameField
          channel={target.channel}
          namePath={path.name}
          defaultName={defaultCommandName(deviceName, target)}
          skipped={target.disabled}
        />
        <SparkplugTypeField
          path={path.tagType}
          label="Sparkplug B type"
          padHelpText={false}
        />
      </PForm.Section>
    </PForm.Sections>
  );
};

const PlainChannelNameField: FC<PaneProps<PlainWriteTarget>> = ({
  path,
  deviceName,
}) => {
  const channelKey = PForm.useFieldValue(path.channel.channel);
  const topic = PForm.useFieldValue(path.topic);
  const disabled = PForm.useFieldValue(path.disabled);
  return (
    <Task.ChannelNameField
      channel={channelKey}
      namePath={path.channel.name}
      defaultName={commandChannelName(deviceName, topic)}
      skipped={disabled}
    />
  );
};

/** Returns field written as kind, with the defaults of kind. */
const asKind = (field: ChannelField, kind: payload.Kind): ChannelField =>
  mqtt.channelFieldZ.parse({
    pointer: field.pointer,
    channel: field.channel,
    name: field.name,
    dataType: field.dataType,
    type: kind,
    ...(kind === "timestamp" ? { timeFormat: "iso8601" } : {}),
  });

/** A command channel can hold any data type a payload writes. */
const NO_HIDDEN_DATA_TYPES: DataType[] = [];

const ChannelSections: FC<PaneProps<PlainWriteTarget>> = ({ path, deviceName }) => {
  const field = PForm.useFieldValue(path.channel);
  const dataType = Task.Payload.useDataType(field.channel, field.dataType);
  return (
    <>
      <PForm.Section title="Channel">
        <PlainChannelNameField path={path} deviceName={deviceName} />
        <Task.Payload.DataTypeField
          path={deep.present(path.channel.dataType)}
          channel={field.channel}
          hideDataTypes={NO_HIDDEN_DATA_TYPES}
        />
        <PForm.TextField
          path={path.channel.pointer}
          label="Pointer"
          padHelpText={false}
          inputProps={POINTER_INPUT_PROPS}
        />
        <Task.Payload.KindFields
          path={path.channel}
          dataType={dataType}
          asKind={asKind}
        />
      </PForm.Section>
      <Task.Payload.EntriesSection path={path.channel} />
    </>
  );
};

const PlainTargetPane: FC<PaneProps<PlainWriteTarget>> = ({ path, deviceName }) => (
  <PForm.Sections>
    <PForm.Section title="Publish">
      <PForm.TextField
        path={path.topic}
        label="Topic"
        padHelpText={false}
        inputProps={TOPIC_INPUT_PROPS}
      />
      <QoSField path={path.qos} />
      <PForm.CheckboxField path={path.retained} label="Retain" padHelpText={false} />
    </PForm.Section>
    <ChannelSections path={path} deviceName={deviceName} />
    <Task.Payload.AdditionalFields key={String(path)} path={path.fields} />
  </PForm.Sections>
);

const TOPIC_INPUT_PROPS = { placeholder: "plant/line1/valve/set" } as const;

const POINTER_INPUT_PROPS = { placeholder: "Empty sends the bare value" } as const;

const TargetPane: FC<PaneProps<WriteTarget>> = ({ path, deviceName }) => {
  const type = PForm.useFieldValue(path.type);
  return (
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      {type === "plain" ? (
        <PlainTargetPane
          path={deep.variant(path, "type", "plain")}
          deviceName={deviceName}
        />
      ) : (
        <SparkplugTargetPane
          path={deep.variant(path, "type", "sparkplug")}
          deviceName={deviceName}
        />
      )}
    </Flex.Box>
  );
};

/** Creates a target that publishes to a topic, at the pointer of a browsed field. */
const createTarget = (topic: string, field?: ReadField): PlainWriteTarget => {
  const target = mqtt.plainWriteTargetZ.parse({ type: "plain", topic });
  const channel =
    field == null
      ? { type: "float", dataType: DEFAULT_DATA_TYPE }
      : { type: kindOf(field), pointer: field.pointer, dataType: field.dataType };
  return { ...target, channel: mqtt.channelFieldZ.parse(channel) };
};

/** The data type of the command channel of a target that names no sample. */
const DEFAULT_DATA_TYPE = "float64";

const createSparkplugTarget = (tag?: SparkplugTag): SparkplugWriteTarget => {
  const target = mqtt.sparkplugWriteTargetZ.parse({ type: "sparkplug" });
  if (tag == null) return target;
  const { dataType, ...tagID } = tag;
  return { ...target, ...tagID, tagType: toSparkplugDataType(dataType) };
};

const DISCOVER_KIND: Kind<WriteTarget> = {
  title: "Add targets",
  hint: "Select the values to write. Each one becomes a target.",
  noun: "target",
  single: true,
  create: ({ topics, tags }) => [
    ...[...topics].map(([topic, { leaves, pointers }]) =>
      createTarget(
        topic,
        leaves.find((l) => pointers.has(l.pointer)),
      ),
    ),
    ...[...tags.values()].map((t) => createSparkplugTarget(t)),
  ],
  createTopic: (topic) => createTarget(topic),
  createTag: () => createSparkplugTarget(),
};

const useDiscoverModal = createDiscoverModal(DISCOVER_KIND);

const duplicateTargets = (targets: WriteTarget[], keys: string[]): WriteTarget[] =>
  targets
    .filter(({ key }) => keys.includes(key))
    .map((target) =>
      target.type === "sparkplug"
        ? { ...target, key: id.create(), channel: 0, name: "" }
        : {
            ...target,
            key: id.create(),
            channel: { ...target.channel, channel: 0 },
            fields: target.fields.map((f) => ({ ...f, key: id.create() })),
          },
    );

const Content = ({ device }: PlatformDevice.TaskFormContentProps<Device>) => {
  const [selected, setSelected] = useState<string[]>([]);
  const isPreview = Task.useIsPreview();
  const targets = PForm.useFieldListUtils(root.config.targets);
  const discover = useDiscoverModal();
  const client = Synnax.use();
  const { get, set } = PForm.useContext();
  const handleError = Status.useErrorHandler();

  // The Core clears the data type of a target with a channel, so a copy, which gets a
  // new channel, reads the data type of the channel it copies. That channel exists: the
  // Core rejects a config that names a missing channel, and a used one is not deleted.
  // The read sets the data type only while the copy still holds the one it started
  // with, so it does not replace a type the user picked.
  const handleDuplicate = (all: WriteTarget[], keys: string[]): WriteTarget[] => {
    const copies = duplicateTargets(all, keys);
    const sources = all.filter(({ key }) => keys.includes(key));
    copies.forEach((copy, i) => {
      const source = sources[i];
      if (copy.type !== "plain" || source.type !== "plain") return;
      if (source.channel.channel === 0) return;
      const key = source.channel.channel;
      const initial = copy.channel.dataType;
      const path = deep.variant(root.config.targets[copy.key], "type", "plain");
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
        const ch = await client.channels.retrieve(key);
        const current = get(root.config.targets[copy.key], { optional: true })?.value;
        if (
          current?.type !== "plain" ||
          current.channel.channel !== 0 ||
          current.channel.dataType !== initial
        )
          return;
        set(path.channel.dataType, ch.dataType.toString());
      }, "Failed to copy the data type of the command channel");
    });
    return copies;
  };

  const handleAdd = async () => {
    const added = await discover({ device, entries: targets.value() });
    if (added == null) return;
    targets.push(added);
    setSelected([added[0].key]);
  };

  const resolve = (target: WriteTarget): Partial<WriteTarget> => {
    const { write } = device.properties;
    if (target.type === "sparkplug")
      return { channel: write[sparkplugPropertiesKey(target)] ?? 0 };
    return { channel: { ...target.channel, channel: write[target.topic] ?? 0 } };
  };

  const current = selected.at(0);
  const currentPath = current == null ? null : root.config.targets[current];

  return (
    <Task.Views.Panes
      listTitle="Targets"
      list={
        <Task.Views.ChannelList<WriteTarget>
          path={root.config.targets}
          header={null}
          emptyContent={EMPTY_TARGETS}
          footer={
            !isPreview && (
              <PlatformButton.CreateListItem
                size="small"
                onClick={() => void handleAdd()}
              >
                Add targets
              </PlatformButton.CreateListItem>
            )
          }
          createChannels={handleDuplicate}
          listItem={targetItem}
          contextMenuItems={Task.readChannelContextMenuItem}
          resolve={resolve}
          selected={selected}
          onSelect={setSelected}
          grow
        />
      }
      detailsPath={currentPath}
      title={currentPath != null && <EntryLabel path={currentPath} />}
    >
      {currentPath != null ? (
        <TargetPane path={currentPath} deviceName={device.name} />
      ) : (
        <Flex.Box y grow align="center" justify="center">
          <Text.Text status="disabled">Select a target to configure</Text.Text>
        </Flex.Box>
      )}
    </Task.Views.Panes>
  );
};

const Form = PlatformDevice.wrapTaskForm({
  use,
  useConfigure: useConnectModal,
  useNoneSelected,
  Content,
});

const getInitialValues: Task.GetInitialValues<WriteSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = WRITE_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: "MQTT write task", type: WRITE_TYPE, config: cfg };
};

const onConfigure: Task.OnConfigure<WriteSchemas["config"]> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({ key: config.device, schemas: SCHEMAS });
  const before = deep.copy(dev.properties);
  try {
    for (const target of config.targets) {
      if (target.disabled) continue;
      if (target.type === "plain") {
        const { channel: field, topic } = target;
        field.channel = await Task.configureCommandChannel(
          client,
          dev.properties.write,
          {
            propertiesKey: topic,
            channel: field.channel,
            name: primitive.isNonZero(field.name)
              ? field.name
              : defaultCommandName(dev.name, target),
            dataType: field.dataType,
          },
        );
      } else
        target.channel = await Task.configureCommandChannel(
          client,
          dev.properties.write,
          {
            propertiesKey: sparkplugPropertiesKey(target),
            channel: target.channel,
            name: primitive.isNonZero(target.name)
              ? target.name
              : defaultCommandName(dev.name, target),
            dataType: fromSparkplugDataType(target.tagType),
          },
        );
    }
  } finally {
    await PlatformDevice.setProperties(client, dev, before, SCHEMAS);
  }
  return [config, dev.rack];
};

export const Write = Task.wrapForm({
  Properties,
  Form,
  schemas: WRITE_SCHEMAS,
  deployConfigZ: deployWriteConfigZ,
  type: "mqtt_write",
  onConfigure,
});

export const useCreateWrite = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const WriteSelectable: Selector.Selectable = {
  type: WRITE_TYPE,
  title: "MQTT write task",
  label: "Write",
  icon: <Icon.Logo.MQTT />,
  useOnSelect: useCreateWrite,
};
