// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/mqtt/task/Form.css";

import {
  channel,
  DisconnectedError,
  type icd,
  mqtt,
  type Synnax,
} from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Synnax as PSynnax } from "@synnaxlabs/pluto";
import { deep, errors, id, primitive, unique } from "@synnaxlabs/x";
import { type FC, type ReactElement, useRef, useState } from "react";

import { Device } from "@/feature/mqtt/device";
import { DEFAULT_WRITE_NAME } from "@/feature/mqtt/task/create";
import type { Format, Kind } from "@/feature/mqtt/task/discover/Discover";
import { createDiscoverModal } from "@/feature/mqtt/task/discover/modal";
import type { TopicPick } from "@/feature/mqtt/task/discover/Topics";
import { namePrefix } from "@/feature/mqtt/task/entries";
import { EntryLabel } from "@/feature/mqtt/task/EntryLabel";
import type { Leaf } from "@/feature/mqtt/task/leaves";
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
  type HeldEntry,
  type PlainWriteTarget,
  type SparkplugTag,
  type SparkplugWriteTarget,
  WRITE_SCHEMAS,
  type WriteSchemas,
  type WriteTarget,
} from "@/feature/mqtt/task/types";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { Empty } from "@/platform/empty";
import { ICD } from "@/platform/icd";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<WriteSchemas>>();

const Properties = () => <Device.Select />;

/** The name deploying gives the command channel of a field of a plain target. */
const commandName = (prefix: string, field: icd.JSONField): string =>
  `${prefix}_${channel.escapeInvalidName(field.name)}_cmd`;

/** The name deploying gives a tag's command channel while it has none. */
const defaultCommandName = (devName: string, target: SparkplugWriteTarget): string =>
  `${sparkplugChannelName(devName, target)}_cmd`;

interface TagChannelNameProps {
  itemKey: string;
  target: SparkplugWriteTarget;
}

const TagChannelName = ({ itemKey, target }: TagChannelNameProps) => {
  const device = Device.useFromConfig();
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

const MISSING_DESCRIPTION =
  "The ICD no longer has this message. Pick another message format.";

const PlainTargetPane = ({
  path,
  deviceName,
}: PaneProps<PlainWriteTarget>): ReactElement => {
  const { set } = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const target = PForm.useFieldValue(path);
  const message = ICD.useMessage(target.icd, target.message.message);
  const prefix = namePrefix(deviceName, target.topic);
  const handleFormatChange = (icdKey: icd.Key, picked: icd.Message) =>
    set(path, { ...target, icd: icdKey, message: ICD.Write.create(picked) });
  return (
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
      <PForm.Section title="Message">
        <ICD.MessageFormatField
          icdKey={target.icd}
          message={target.message.message}
          onChange={handleFormatChange}
          preview={isPreview}
        />
      </PForm.Section>
      <ICD.MessageStatus message={message} missingDescription={MISSING_DESCRIPTION} />
      <ICD.Write.Bindings
        path={path.message.bindings}
        fields={ICD.jsonFields(message)}
        channelName={(field) => commandName(prefix, field)}
        skipped={target.disabled}
      />
    </PForm.Sections>
  );
};

const TOPIC_INPUT_PROPS = { placeholder: "plant/line1/valve/set" } as const;

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

const createSparkplugTarget = (tag?: SparkplugTag): SparkplugWriteTarget => {
  const target = mqtt.sparkplugWriteTargetZ.parse({ type: "sparkplug" });
  if (tag == null) return target;
  const { dataType, ...tagID } = tag;
  return { ...target, ...tagID, tagType: toSparkplugDataType(dataType) };
};

const createPlainTarget = (topic: string, { icd, message }: Format): PlainWriteTarget =>
  mqtt.plainWriteTargetZ.parse({
    type: "plain",
    topic,
    icd,
    message: ICD.Write.create(message),
  });

/** A topic to write, and the message format of its target. */
interface Draft {
  type: "plain";
  topic: string;
  /** The fields of the format that the task infers, when the dialog picked none. */
  fields: icd.JSONField[];
  /** The message that the target sends, when the dialog picked one. */
  format: Format | null;
}

/** What the discovery dialog adds: topics to write and Sparkplug B tags. */
type Addition = Draft | SparkplugWriteTarget;

const isDraft = (addition: Addition): addition is Draft => addition.type === "plain";

const createDraft = (topic: string, leaves: Leaf[], format: Format | null): Draft => ({
  type: "plain",
  topic,
  fields: format == null ? ICD.createFields(leaves) : [],
  format,
});

/** @returns the picked leaves of the shape that the most messages of a topic had. */
const formatLeaves = (pick: TopicPick): Leaf[] => {
  const most = ICD.Write.mostCommon(
    [...pick.values()],
    ({ shape }) => shape.count ?? 0,
  );
  if (most == null) return [];
  return most.shape.leaves.filter(({ pointer }) => most.pointers.has(pointer));
};

const DISCOVER_KIND: Kind<Addition> = {
  title: "Add targets",
  hint: "Select the topics to write. Each one becomes a target.",
  tagHint: "Select the tags to write. Each one becomes a target.",
  noun: "target",
  whole: true,
  formatPickable: true,
  create: ({ topics, tags, format }) => [
    ...[...topics].map(([topic, pick]) =>
      createDraft(topic, formatLeaves(pick), format),
    ),
    ...[...tags.values()].map((t) => createSparkplugTarget(t)),
  ],
  createTopic: (topic, { format }) => createDraft(topic, [], format),
  createTag: () => createSparkplugTarget(),
};

/** @returns the name of the ICD of a task named name. */
const icdName = (name: string): string =>
  name.trim() === "" ? DEFAULT_WRITE_NAME : name;

const heldEntry = (target: WriteTarget): HeldEntry =>
  target.type === "plain"
    ? { type: "plain", topic: target.topic, messages: [] }
    : target;

const useDiscoverModal = createDiscoverModal(DISCOVER_KIND);

const unbind = (binding: icd.WriteBinding): icd.WriteBinding =>
  binding.type === "channel" ? { ...binding, channel: 0 } : binding;

const duplicateTargets = (targets: WriteTarget[], keys: string[]): WriteTarget[] =>
  targets
    .filter(({ key }) => keys.includes(key))
    .map((target) =>
      target.type === "sparkplug"
        ? { ...target, key: id.create(), channel: 0, name: "" }
        : {
            ...target,
            key: id.create(),
            message: {
              ...target.message,
              bindings: target.message.bindings.map(unbind),
            },
          },
    );

interface TargetsProps {
  device: Device.Device;
  selected: string[];
  onSelect: (selected: string[]) => void;
}

const Targets = ({ device, selected, onSelect }: TargetsProps) => {
  const client = PSynnax.use();
  const ctx = PForm.useContext();
  const handleError = Status.useErrorHandler();
  const isPreview = Task.useIsPreview();
  const formats = ICD.useFormats();
  const targets = PForm.useFieldListUtils(root.config.targets);
  const plain = PForm.useFieldValue(root.config.targets).filter(
    (t) => t.type === "plain",
  );
  const discover = useDiscoverModal();

  // Adds run one at a time, so a second click cannot make a second ICD.
  const adding = useRef(false);
  const add = async () => {
    if (client == null) throw new DisconnectedError();
    const additions = await discover({
      device,
      entries: targets.value().map(heldEntry),
    });
    if (additions == null || additions.length === 0) return;
    const inferred = additions.filter(isDraft).filter(({ format }) => format == null);
    const made: Format[] = [];
    if (inferred.length > 0) {
      const { icd: key, messages } = await ICD.Write.addMessages({
        client,
        formats,
        targets: plain,
        name: icdName(ctx.get(root.name).value),
        messages: inferred.map(({ topic, fields }) => ({ name: topic, fields })),
      });
      made.push(...messages.map((message) => ({ icd: key, message })));
    }
    const added = additions.map((a) =>
      isDraft(a)
        ? createPlainTarget(a.topic, a.format ?? made[inferred.indexOf(a)])
        : a,
    );
    targets.push(added);
    onSelect([added[0].key]);
  };
  const handleAdd = () =>
    handleError(async () => {
      if (adding.current) return;
      adding.current = true;
      try {
        await add();
      } catch (e) {
        adding.current = false;
        throw errors.fromUnknown(e);
      }
      adding.current = false;
    }, "Failed to add targets");

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
              <Button.CreateListItem size="small" onClick={handleAdd}>
                Add targets
              </Button.CreateListItem>
            )
          }
          createChannels={duplicateTargets}
          listItem={targetItem}
          contextMenuItems={contextMenuItems}
          resolve={resolve}
          selected={selected}
          onSelect={onSelect}
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
      {plain.map(({ key, icd: icdKey }) => (
        <ICD.Write.Suit
          key={key}
          icdKey={icdKey}
          path={deep.variant(root.config.targets[key], "type", "plain").message}
        />
      ))}
    </Task.Views.Panes>
  );
};

const Content = ({ device }: PlatformDevice.TaskFormContentProps<Device.Device>) => {
  const [selected, setSelected] = useState<string[]>([]);
  const icds = unique.unique(
    PForm.useFieldValue(root.config.targets).flatMap((t) =>
      t.type === "plain" ? [t.icd] : [],
    ),
  );
  return icds.reduceRight(
    (child, key) => (
      <ICD.WithFormat key={key} icdKey={key}>
        {child}
      </ICD.WithFormat>
    ),
    <Targets device={device} selected={selected} onSelect={setSelected} />,
  );
};

const Form = PlatformDevice.wrapTaskForm({
  use: Device.use,
  useConfigure: Device.useConnectModal,
  useNoneSelected: Device.useNoneSelected,
  Content,
});

/**
 * Gives each channel binding of target a command channel that its field takes: the one
 * it has, the one that dev stores for its field, or a new one. A new channel takes the
 * data type of the binding, or the default of its field when the binding names none or
 * one that the field no longer takes. Updates the write properties of dev in place.
 */
const configureBindings = async (
  client: Synnax,
  dev: Device.Device,
  target: PlainWriteTarget,
): Promise<void> => {
  const bindings = target.message.bindings.filter(
    (b): b is icd.ChannelWriteBinding => b.type === "channel",
  );
  if (bindings.length === 0) return;
  const messages = await ICD.retrieveMessages(client, [target.icd]);
  const fields = ICD.jsonFields(
    messages.get(target.icd)?.find(({ key }) => key === target.message.message),
  );
  const prefix = namePrefix(dev.name, target.topic);
  for (const binding of bindings) {
    const field = fields.find(({ key }) => key === binding.field);
    // The Core rejects a binding to a field that the message lacks.
    if (field == null) continue;
    const { dataType = "" } = binding;
    const suited = dataType !== "" && ICD.encodes(field, dataType);
    binding.channel = await Task.configureCommandChannel(client, dev.properties.write, {
      propertiesKey: `${target.topic}#${binding.field}`,
      channel: binding.channel,
      name: commandName(prefix, field),
      dataType: ICD.bindingDataType(suited ? binding : {}, field),
      suits: (ch) => ICD.encodes(field, ch.dataType.toString()),
    });
  }
};

const onConfigure: Task.OnConfigure<WriteSchemas["config"]> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: Device.SCHEMAS,
  });
  const before = deep.copy(dev.properties);
  try {
    for (const target of config.targets) {
      if (target.disabled) continue;
      if (target.type === "plain") {
        await configureBindings(client, dev, target);
        continue;
      }
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
    await PlatformDevice.setProperties(client, dev, before, Device.SCHEMAS);
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
