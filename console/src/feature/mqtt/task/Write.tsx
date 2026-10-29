// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/mqtt/task/Form.css";

import { channel, mqtt } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { deep, id, primitive } from "@synnaxlabs/x";
import { type FC, useState } from "react";

import { use } from "@/feature/mqtt/device/queries";
import { Select as SelectDevice } from "@/feature/mqtt/device/Select";
import { type Device, SCHEMAS } from "@/feature/mqtt/device/types";
import { useConnectModal } from "@/feature/mqtt/device/useConnectModal";
import { jsonTypeOf } from "@/feature/mqtt/task/createReadFields";
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
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const Properties = () => (
  <>
    <SelectDevice />
    <Flex.Box x grow>
      <Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

const TARGETS_PATH = "config.targets";

/** The name configure gives the command channel of a plain target. */
const commandChannelName = (devName: string, topic: string): string =>
  `${channel.escapeInvalidName(devName)}_${channel.escapeInvalidName(topic)}_cmd`;

/** A plain target holds its channel in a field; a Sparkplug B target is the field. */
const channelPaths = (path: string, type: WriteTarget["type"]) =>
  type === "plain"
    ? { key: `${path}.channel.channel`, name: `${path}.channel.name` }
    : { key: `${path}.channel`, name: `${path}.name` };

const TargetItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = `${TARGETS_PATH}.${itemKey}`;
  const type = PForm.useFieldValue<WriteTarget["type"]>(`${path}.type`);
  const disabled = PForm.useFieldValue<boolean>(`${path}.disabled`);
  const paths = channelPaths(path, type);
  const channelKey = PForm.useFieldValue<number>(paths.key);
  return (
    <Select.Item
      {...props}
      y
      align="start"
      gap="small"
      className={CSS.cls(disabled && CSS.M("off"))}
    >
      <Flex.Box x align="center" gap="small" full="x">
        <EntryLabel path={path} />
        <Task.EnabledCheckbox path={`${path}.disabled`} />
      </Flex.Box>
      <Task.ChannelName
        channel={channelKey}
        namePath={paths.name}
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

interface PaneProps {
  path: string;
  deviceName: string;
}

const SparkplugTargetPane: FC<PaneProps> = ({ path, deviceName }) => {
  const target = PForm.useFieldValue<SparkplugWriteTarget>(path);
  return (
    <PForm.Sections>
      <PForm.Section title="Tag">
        <SparkplugTagFields path={path} />
      </PForm.Section>
      <PForm.Section title="Channel">
        <Task.ChannelNameField
          channel={target.channel}
          namePath={`${path}.name`}
          defaultName={`${sparkplugChannelName(deviceName, target)}_cmd`}
        />
        <SparkplugTypeField
          path={`${path}.tagType`}
          label="Sparkplug B type"
          padHelpText={false}
        />
      </PForm.Section>
    </PForm.Sections>
  );
};

const PlainChannelNameField: FC<PaneProps> = ({ path, deviceName }) => {
  const channelKey = PForm.useFieldValue<number>(`${path}.channel.channel`);
  const topic = PForm.useFieldValue<string>(`${path}.topic`);
  return (
    <Task.ChannelNameField
      channel={channelKey}
      namePath={`${path}.channel.name`}
      defaultName={commandChannelName(deviceName, topic)}
    />
  );
};

const PlainTargetPane: FC<PaneProps> = ({ path, deviceName }) => (
  <PForm.Sections>
    <PForm.Section title="Publish">
      <PForm.TextField
        path={`${path}.topic`}
        label="Topic"
        padHelpText={false}
        inputProps={TOPIC_INPUT_PROPS}
      />
      <QoSField path={`${path}.qos`} />
      <PForm.SwitchField path={`${path}.retained`} label="Retain" padHelpText={false} />
    </PForm.Section>
    <Task.Payload.ChannelSections
      path={`${path}.channel`}
      pointerPlaceholder="Empty sends the bare value"
    >
      <PlainChannelNameField path={path} deviceName={deviceName} />
    </Task.Payload.ChannelSections>
    <Task.Payload.AdditionalFields key={path} path={`${path}.fields`} />
  </PForm.Sections>
);

const TOPIC_INPUT_PROPS = { placeholder: "plant/line1/valve/set" } as const;

const TargetPane: FC<PaneProps> = ({ path, deviceName }) => {
  const type = PForm.useFieldValue<WriteTarget["type"]>(`${path}.type`);
  return (
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      {type === "plain" ? (
        <PlainTargetPane path={path} deviceName={deviceName} />
      ) : (
        <SparkplugTargetPane path={path} deviceName={deviceName} />
      )}
    </Flex.Box>
  );
};

/** Creates a target that publishes to a topic, at the pointer of a browsed field. */
const createTarget = (topic: string, field?: ReadField): PlainWriteTarget => {
  const target = mqtt.plainWriteTargetZ.parse({ type: "plain", topic });
  if (field == null) return target;
  const { pointer, dataType } = field;
  return {
    ...target,
    channel: { ...target.channel, pointer, dataType, jsonType: jsonTypeOf(field) },
  };
};

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
  const targets = PForm.useFieldListUtils<string, WriteTarget>(TARGETS_PATH);
  const discover = useDiscoverModal();

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
  const currentPath = current == null ? null : `${TARGETS_PATH}.${current}`;

  return (
    <Task.Views.Panes
      listTitle="Targets"
      list={
        <Task.Views.ChannelList<WriteTarget>
          path={TARGETS_PATH}
          title={null}
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
          createChannels={duplicateTargets}
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
  const saved = deep.copy(dev.properties);
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
              : commandChannelName(dev.name, topic),
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
              : `${sparkplugChannelName(dev.name, target)}_cmd`,
            dataType: fromSparkplugDataType(target.tagType),
          },
        );
    }
  } finally {
    if (!deep.equal(saved, dev.properties)) await client.devices.create(dev, SCHEMAS);
  }
  return [config, dev.rack];
};

export const Write = Task.wrapForm({
  Properties,
  Form,
  schemas: WRITE_SCHEMAS,
  deployConfigZ: deployWriteConfigZ,
  type: "mqtt_write",
  getInitialValues,
  onConfigure,
});

export const useCreateWrite = Task.createUseCreate({
  getInitialValues,
});

export const WriteSelectable = Selector.createSelectable({
  type: WRITE_TYPE,
  title: "MQTT write task",
  icon: <Icon.Logo.MQTT />,
  useOnSelect: useCreateWrite,
});
