// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/mqtt/task/Form.css";

import { DisconnectedError, type icd, mqtt } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { useExclusiveCallback } from "@synnaxlabs/lyra/hooks";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import type { Tree } from "@synnaxlabs/lyra/tree";
import { Synnax as PSynnax } from "@synnaxlabs/pluto";
import { deep, id, primitive, unique } from "@synnaxlabs/x";
import {
  type Dispatch,
  type FC,
  type ReactElement,
  type SetStateAction,
  useState,
} from "react";

import { Device } from "@/feature/mqtt/device";
import { DEFAULT_WRITE_NAME } from "@/feature/mqtt/task/create";
import type { Format, Kind } from "@/feature/mqtt/task/discover/Discover";
import { createDiscoverModal } from "@/feature/mqtt/task/discover/modal";
import { formatOf, type TopicPick } from "@/feature/mqtt/task/discover/Topics";
import { namePrefix } from "@/feature/mqtt/task/entries";
import { EntryItem, EntryLabel } from "@/feature/mqtt/task/EntryLabel";
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
import { ICD } from "@/platform/icd";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<WriteSchemas>>();

const Properties = () => <Device.Select />;

/** The name deploying gives a tag's command channel while it has none. */
const defaultCommandName = (devName: string, target: SparkplugWriteTarget): string =>
  `${sparkplugChannelName(devName, target)}_cmd`;

/** Keys and names the command channels of a plain target, as deploying does. */
const plainNaming = (
  devName: string,
  { topic }: PlainWriteTarget,
): ICD.Write.ChannelNaming => {
  const prefix = namePrefix(devName, topic);
  return {
    key: (field) => `${topic}#${field.key}`,
    name: (field) => ICD.Write.commandName(prefix, field),
  };
};

/** What the command channel of a tag needs from deploying. */
const sparkplugCommand = (
  devName: string,
  target: SparkplugWriteTarget,
): Task.CommandChannelSpec => ({
  propertiesKey: sparkplugPropertiesKey(target),
  channel: target.channel,
  name: primitive.isNonZero(target.name)
    ? target.name
    : defaultCommandName(devName, target),
  dataType: fromSparkplugDataType(target.tagType),
});

const plainPath = (key: string) =>
  deep.variant(root.config.targets[key], "type", "plain");

const TargetItem = (props: Tree.ItemRenderProps<string>) => (
  <EntryItem {...props} path={root.config.targets[props.itemKey]} />
);

interface PaneProps<T> {
  path: deep.Path<T>;
  deviceName: string;
}

interface SparkplugTargetPaneProps extends PaneProps<SparkplugWriteTarget> {
  /** The name that deploying gives the command channel it creates, once planned. */
  plannedName?: string;
}

const SparkplugTargetPane: FC<SparkplugTargetPaneProps> = ({
  path,
  deviceName,
  plannedName,
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
          defaultName={plannedName ?? defaultCommandName(deviceName, target)}
          plannedName={plannedName}
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
  onAddField: () => void;
}

const PlainTargetPane = ({ path, onAddField }: PlainTargetPaneProps): ReactElement => {
  const { set } = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const target = PForm.useFieldValue(path);
  const message = ICD.useMessage(target.icd, target.message.message);
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
      <PForm.Section
        title="Message"
        actions={
          <ICD.AddField
            onClick={onAddField}
            disabled={message?.payload.format !== "json"}
          />
        }
      >
        <ICD.MessageFormatField
          icdKey={target.icd}
          message={target.message.message}
          onChange={handleFormatChange}
          preview={isPreview}
        />
      </PForm.Section>
      <ICD.MessageStatus
        message={message}
        missingDescription={ICD.Write.MISSING_DESCRIPTION}
      />
    </PForm.Sections>
  );
};

const TOPIC_INPUT_PROPS = { placeholder: "plant/line1/valve/set" } as const;

interface TargetPaneProps
  extends PaneProps<WriteTarget>, Pick<SparkplugTargetPaneProps, "plannedName"> {
  onAddField: () => void;
}

const TargetPane: FC<TargetPaneProps> = ({
  path,
  deviceName,
  plannedName,
  onAddField,
}) => {
  const type = PForm.useFieldValue(path.type);
  return (
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      {type === "plain" ? (
        <PlainTargetPane
          path={deep.variant(path, "type", "plain")}
          onAddField={onAddField}
        />
      ) : (
        <SparkplugTargetPane
          path={deep.variant(path, "type", "sparkplug")}
          deviceName={deviceName}
          plannedName={plannedName}
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

/** @returns the picked leaves of the shape that sets the format of a topic. */
const formatLeaves = (pick: TopicPick): Leaf[] => {
  const format = formatOf(pick);
  if (format == null) return [];
  return format.shape.leaves.filter(({ pointer }) => format.pointers.has(pointer));
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

const duplicate = (target: WriteTarget): WriteTarget =>
  target.type === "sparkplug"
    ? { ...target, key: id.create(), channel: 0, name: "" }
    : {
        ...target,
        key: id.create(),
        message: ICD.Write.unbind(target.message),
      };

const messageOf = (target: WriteTarget): ICD.Write.TargetMessage | null =>
  target.type === "plain"
    ? {
        icd: target.icd,
        message: target.message.message,
        bindings: plainPath(target.key).message.bindings,
      }
    : null;

interface TargetsProps {
  device: Device.Device;
  selected: string[];
  onSelect: Dispatch<SetStateAction<string[]>>;
  expanded: string[];
  onExpand: Dispatch<SetStateAction<string[]>>;
}

/** The targets of the task as a tree, beside the details of the selected item. */
const Targets = ({ device, selected, onSelect, expanded, onExpand }: TargetsProps) => {
  const client = PSynnax.use();
  const ctx = PForm.useContext();
  const handleError = Status.useErrorHandler();
  const formats = ICD.useFormats();
  const list = PForm.useFieldListUtils(root.config.targets);
  const targets = PForm.useFieldValue(root.config.targets);
  const plain = targets.filter((t) => t.type === "plain");
  const discover = useDiscoverModal();
  const { tree, handleSelect, contextMenu, renderItem, current, addField, added } =
    ICD.Write.useTree({
      path: root.config.targets,
      messageOf,
      TargetItem,
      duplicate,
      selected,
      onSelect,
      expanded,
      onExpand,
    });
  const commandChannelsOf = ICD.Write.useCommandChannels();
  const specs = targets
    .filter(({ disabled }) => !disabled)
    .flatMap((t) =>
      t.type === "plain"
        ? commandChannelsOf(t, plainNaming(device.name, t))
        : [sparkplugCommand(device.name, t)],
    );
  const names = Task.useCommandChannelNames(device.properties.write, specs);

  const add = async () => {
    if (client == null) throw new DisconnectedError();
    const additions = await discover({
      device,
      entries: list.value().map(heldEntry),
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
    const created = additions.map((a) =>
      isDraft(a)
        ? createPlainTarget(a.topic, a.format ?? made[inferred.indexOf(a)])
        : a,
    );
    list.push(created);
    onSelect([created[0].key]);
    onExpand((prev) => unique.unique([...prev, ...ICD.Write.expandedOf(created)]));
  };
  // Adds run one at a time, so a second click cannot make a second ICD.
  const addExclusive = useExclusiveCallback(add);
  const handleAdd = () => handleError(addExclusive, "Failed to add targets");

  const target =
    current == null ? undefined : targets.find(({ key }) => key === current.target);
  let details = null;
  let title = null;
  let detailsPath: deep.Target<unknown> | null = null;
  if (target?.type === "plain" && current?.kind === "field") {
    const path = plainPath(target.key).message;
    const naming = plainNaming(device.name, target);
    const i = target.message.bindings.findIndex(
      ({ field }) => field === current.field.key,
    );
    detailsPath = i < 0 ? path : path.bindings[i];
    title = (
      <Text.Text level="p" weight={500} overflow="fade">
        {current.field.name}
      </Text.Text>
    );
    details = (
      <ICD.Write.FieldDetails
        key={selected[0]}
        path={path.bindings}
        icdKey={target.icd}
        message={target.message.message}
        field={current.field.key}
        channelName={names.get(naming.key(current.field)) ?? naming.name(current.field)}
        skipped={target.disabled}
        autoFocus={current.field.key === added}
      />
    );
  } else if (target != null) {
    detailsPath = root.config.targets[target.key];
    title = <EntryLabel path={root.config.targets[target.key]} />;
    details = (
      <TargetPane
        key={target.key}
        path={root.config.targets[target.key]}
        deviceName={device.name}
        plannedName={
          target.type === "sparkplug"
            ? names.get(sparkplugPropertiesKey(target))
            : undefined
        }
        onAddField={() => addField(target.key)}
      />
    );
  }

  return (
    <Task.Views.Panes
      listTitle="Targets"
      list={
        <Task.Views.TreeList
          {...tree}
          onSelect={handleSelect}
          menu={contextMenu}
          emptyMessage="No targets"
          addLabel="Add targets"
          onAdd={handleAdd}
        >
          {renderItem}
        </Task.Views.TreeList>
      }
      detailsPath={detailsPath}
      title={title ?? undefined}
    >
      {details ?? (
        <Flex.Box y grow align="center" justify="center">
          <Text.Text status="disabled">
            Add targets to write to a topic or a tag
          </Text.Text>
        </Flex.Box>
      )}
      {plain.map(({ key, icd: icdKey }) => (
        <ICD.Write.Suit key={key} icdKey={icdKey} path={plainPath(key).message} />
      ))}
    </Task.Views.Panes>
  );
};

const Content = ({ device }: PlatformDevice.TaskFormContentProps<Device.Device>) => {
  const targets = PForm.useFieldValue(root.config.targets);
  const [selected, setSelected] = useState<string[]>([]);
  const [expanded, setExpanded] = useState<string[]>(() =>
    ICD.Write.expandedOf(targets),
  );
  const icds = targets.flatMap((t) => (t.type === "plain" ? [t.icd] : []));
  const resolve = (target: WriteTarget): Partial<WriteTarget> | null =>
    target.type === "sparkplug"
      ? { channel: device.properties.write[sparkplugPropertiesKey(target)] ?? 0 }
      : null;
  return (
    <>
      <ICD.WithFormats icdKeys={icds}>
        <Targets
          device={device}
          selected={selected}
          onSelect={setSelected}
          expanded={expanded}
          onExpand={setExpanded}
        />
      </ICD.WithFormats>
      <Task.BindChannels path={root.config.targets} resolve={resolve} />
    </>
  );
};

const Form = PlatformDevice.wrapTaskForm({
  use: Device.use,
  useConfigure: Device.useConnectModal,
  useNoneSelected: Device.useNoneSelected,
  Content,
});

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
        await ICD.Write.configureChannels(
          client,
          dev.properties.write,
          target,
          plainNaming(dev.name, target),
        );
        continue;
      }
      target.channel = await Task.configureCommandChannel(
        client,
        dev.properties.write,
        sparkplugCommand(dev.name, target),
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
