// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/mqtt/task/Form.css";

import { mqtt } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { deep, id, primitive } from "@synnaxlabs/x";
import { type FC, useState } from "react";

import { use, useFromConfig } from "@/feature/mqtt/device/queries";
import { Select as SelectDevice, useNoneSelected } from "@/feature/mqtt/device/Select";
import { type Device, MAKE, SCHEMAS } from "@/feature/mqtt/device/types";
import { useConnectModal } from "@/feature/mqtt/device/useConnectModal";
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

/** The name deploying gives a tag's command channel while it has none. */
const defaultCommandName = (devName: string, target: SparkplugWriteTarget): string =>
  `${sparkplugChannelName(devName, target)}_cmd`;

interface TagChannelNameProps {
  itemKey: string;
  target: SparkplugWriteTarget;
}

const TagChannelName = ({ itemKey, target }: TagChannelNameProps) => {
  const device = useFromConfig();
  return (
    <Task.ChannelName
      channel={target.channel}
      namePath={deep.variant(root.config.targets[itemKey], "type", "sparkplug").name}
      defaultName={device == null ? undefined : defaultCommandName(device.name, target)}
      skipped={target.disabled}
      id={Task.getChannelNameID(itemKey)}
      level="small"
      weight={450}
      color={9}
      overflow="fade"
    />
  );
};

const TargetItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = root.config.targets[itemKey];
  const target = PForm.useFieldValue(path);
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
      {target.type === "sparkplug" && (
        <TagChannelName itemKey={itemKey} target={target} />
      )}
    </Select.Item>
  );
};

const targetItem = Component.renderProp(TargetItem);

// A plain target has no one command channel to rename.
const contextMenuItems = (props: Task.ContextMenuItemProps<WriteTarget>) => {
  const target = props.channels.find(({ key }) => key === props.keys[0]);
  if (target?.type !== "sparkplug") return null;
  return Task.readChannelContextMenuItem(props);
};

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

interface PlainTargetPaneProps {
  path: deep.Path<PlainWriteTarget>;
}

const PlainTargetPane = ({ path }: PlainTargetPaneProps) => (
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
  </PForm.Sections>
);

const TOPIC_INPUT_PROPS = { placeholder: "plant/line1/valve/set" } as const;

const TargetPane: FC<PaneProps<WriteTarget>> = ({ path, deviceName }) => {
  const type = PForm.useFieldValue(path.type);
  return (
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      {type === "plain" ? (
        <PlainTargetPane path={deep.variant(path, "type", "plain")} />
      ) : (
        <SparkplugTargetPane
          path={deep.variant(path, "type", "sparkplug")}
          deviceName={deviceName}
        />
      )}
    </Flex.Box>
  );
};

const createSparkplugTarget = (tag?: SparkplugTag): SparkplugWriteTarget => {
  const target = mqtt.sparkplugWriteTargetZ.parse({ type: "sparkplug" });
  if (tag == null) return target;
  const { dataType, ...tagID } = tag;
  return { ...target, ...tagID, tagType: toSparkplugDataType(dataType) };
};

const DISCOVER_KIND: Kind<SparkplugWriteTarget> = {
  title: "Add targets",
  hint: "Select the tags to write. Each one becomes a target.",
  noun: "target",
  create: ({ tags }) => [...tags.values()].map((t) => createSparkplugTarget(t)),
  createTag: () => createSparkplugTarget(),
};

const useDiscoverModal = createDiscoverModal(DISCOVER_KIND);

const duplicateTargets = (targets: WriteTarget[], keys: string[]): WriteTarget[] =>
  targets
    .filter(({ key }) => keys.includes(key))
    .map((target) =>
      target.type === "sparkplug"
        ? { ...target, key: id.create(), channel: 0, name: "" }
        : { ...target, key: id.create() },
    );

const Content = ({ device }: PlatformDevice.TaskFormContentProps<Device>) => {
  const [selected, setSelected] = useState<string[]>([]);
  const isPreview = Task.useIsPreview();
  const targets = PForm.useFieldListUtils(root.config.targets);
  const discover = useDiscoverModal();

  const handleAdd = async () => {
    const tags = targets.value().filter((t) => t.type === "sparkplug");
    const added = await discover({ device, entries: tags });
    if (added == null) return;
    targets.push(added);
    setSelected([added[0].key]);
  };

  const resolve = (target: WriteTarget): Partial<WriteTarget> | null =>
    target.type === "sparkplug"
      ? { channel: device.properties.write[sparkplugPropertiesKey(target)] ?? 0 }
      : null;

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
          createChannels={duplicateTargets}
          listItem={targetItem}
          contextMenuItems={contextMenuItems}
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
        <TargetPane
          key={String(currentPath)}
          path={currentPath}
          deviceName={device.name}
        />
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
      if (target.disabled || target.type === "plain") continue;
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
