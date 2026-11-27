// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/kafka/task/Task.css";

import { channel, DisconnectedError, type icd, kafka } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { useExclusiveCallback } from "@synnaxlabs/lyra/hooks";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import type { Tree } from "@synnaxlabs/lyra/tree";
import { Synnax as PSynnax } from "@synnaxlabs/pluto";
import { deep, id, type json, unique } from "@synnaxlabs/x";
import { type Dispatch, type SetStateAction, useState } from "react";

import { Device } from "@/feature/kafka/device";
import { DEFAULT_WRITE_NAME } from "@/feature/kafka/task/create";
import { createDiscoverModal, type Format } from "@/feature/kafka/task/Discover";
import { RecordLabel, TopicLabel } from "@/feature/kafka/task/Labels";
import {
  deployWriteConfigZ,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteSchemas,
  type WriteTarget,
} from "@/feature/kafka/task/types";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { ICD } from "@/platform/icd";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<WriteSchemas>>();

const TOPIC_INPUT_PROPS = { placeholder: "synnax-samples" } as const;

const commandPrefix = (
  deviceName: string,
  { topic, recordKey }: Pick<WriteTarget, "topic" | "recordKey">,
): string => {
  const prefix = `${channel.escapeInvalidName(deviceName)}_${channel.escapeInvalidName(topic)}`;
  return recordKey === ""
    ? prefix
    : `${prefix}_${channel.escapeInvalidName(recordKey)}`;
};

/** The key of the command channel of a target's field in the device's write record. */
const writeKey = (
  { topic, recordKey }: Pick<WriteTarget, "topic" | "recordKey">,
  field: Pick<icd.JSONField, "key">,
): string => `${topic}#${recordKey}#${field.key}`;

/** Keys and names the command channels of a target, as deploying does. */
const namingOf = (deviceName: string, target: WriteTarget): ICD.Write.ChannelNaming => {
  const prefix = commandPrefix(deviceName, target);
  return {
    key: (field) => writeKey(target, field),
    name: (field) => ICD.Write.commandName(prefix, field),
  };
};

/**
 * Each channel binding of an enabled target gets a command channel, which the device
 * keeps for the next deploy.
 */
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
      await ICD.Write.configureChannels(
        client,
        dev.properties.write,
        target,
        namingOf(dev.name, target),
      );
    }
  } finally {
    await PlatformDevice.setProperties(client, dev, before, Device.SCHEMAS);
  }
  return [config, dev.rack];
};

const Properties = () => <Device.Select />;

/** Stands in for the record key of a target whose records have no key. */
const NO_KEY = "No key";

const RECORD_KEY_INPUT_PROPS = { placeholder: NO_KEY } as const;

/** A topic in the tree. It only groups the targets that produce to it. */
const TopicItem = ({ name, ...props }: ICD.Write.GroupItemProps) => (
  <Task.Views.TreeGroupItem {...props} label={<TopicLabel name={name} />} />
);

const TOPICS: ICD.Write.Groups<WriteTarget> = {
  of: ({ topic }) => topic,
  Item: TopicItem,
};

/** A target in the tree, named by its record key. */
const TargetItem = (props: Tree.ItemRenderProps<string>) => {
  const path = root.config.targets[props.itemKey];
  const disabled = PForm.useFieldValue(path.disabled, { optional: true });
  const recordKey = PForm.useFieldValue(path.recordKey, { optional: true });
  const icdKey = PForm.useFieldValue(path.icd, { optional: true });
  const message = PForm.useFieldValue(path.message.message, { optional: true });
  if (disabled == null || recordKey == null || icdKey == null || message == null)
    return null;
  return (
    <Task.Views.TreeGroupItem
      {...props}
      label={<RecordLabel recordKey={recordKey} emptyLabel={NO_KEY} />}
      off={disabled}
    >
      <ICD.MessageMark
        icdKey={icdKey}
        messages={[message]}
        missingDescription={ICD.Write.MISSING_DESCRIPTION}
      />
      <Task.EnabledCheckbox path={path.disabled} />
    </Task.Views.TreeGroupItem>
  );
};

interface TargetDetailsProps {
  path: deep.Path<WriteTarget>;
  onAddField: () => void;
}

/** Where a target produces, and the format of its message. */
const TargetDetails = ({ path, onAddField }: TargetDetailsProps) => {
  const { set } = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const target = PForm.useFieldValue(path);
  const message = ICD.useMessage(target.icd, target.message.message);
  const handleFormatChange = (icdKey: icd.Key, picked: icd.Message) =>
    set(path, { ...target, icd: icdKey, message: ICD.Write.create(picked) });
  const handleAddressChange = () => set(path.message, ICD.Write.unbind(target.message));
  return (
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      <PForm.Sections>
        <PForm.Section title="Produce">
          <PForm.TextField
            path={path.topic}
            label="Topic"
            padHelpText={false}
            inputProps={TOPIC_INPUT_PROPS}
            onChange={handleAddressChange}
          />
          <PForm.TextField
            path={path.recordKey}
            label="Record key"
            padHelpText={false}
            inputProps={RECORD_KEY_INPUT_PROPS}
            onChange={handleAddressChange}
          />
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
          <ICD.MessageWarning
            message={message}
            missingDescription={ICD.Write.MISSING_DESCRIPTION}
          />
        </PForm.Section>
      </PForm.Sections>
    </Flex.Box>
  );
};

/** @returns the name of the ICD of a task named name. */
const icdName = (name: string): string =>
  name.trim() === "" ? DEFAULT_WRITE_NAME : name;

const keyLabel = (recordKey: string): string => (recordKey === "" ? NO_KEY : recordKey);

/** @returns the key that sent every record of shape, or "" for none or several. */
const recordKeyOf = (shape: json.Shape | null): string =>
  shape?.sources.length === 1 ? shape.sources[0] : "";

const useAddTargetsModal = createDiscoverModal({
  title: "Add targets",
  hint: "Select the topics to write. Each one becomes a target.",
  noun: "target",
  verb: "Write",
  whole: true,
});

const duplicate = (target: WriteTarget): WriteTarget => ({
  ...target,
  key: id.create(),
  message: ICD.Write.unbind(target.message),
});

interface TargetsProps {
  device: Device.Device;
  selected: string[];
  onSelect: Dispatch<SetStateAction<string[]>>;
  expanded: string[];
  onExpand: Dispatch<SetStateAction<string[]>>;
}

/** The targets of the task as a tree of topics, beside the selected item's details. */
const Targets = ({ device, selected, onSelect, expanded, onExpand }: TargetsProps) => {
  const client = PSynnax.use();
  const ctx = PForm.useContext();
  const handleError = Status.useErrorHandler();
  const formats = ICD.useFormats();
  const targets = PForm.useFieldListUtils(root.config.targets);
  const all = PForm.useFieldValue(root.config.targets);
  const addTargets = useAddTargetsModal();
  const commandChannelsOf = ICD.Write.useCommandChannels();
  const specs = all
    .filter(({ disabled }) => !disabled)
    .flatMap((t) => commandChannelsOf(t, namingOf(device.name, t)));
  const plans = Task.useCommandChannelPlans(device.properties.write, specs);
  const messageOf = (target: WriteTarget): ICD.Write.TargetMessage => {
    const naming = namingOf(device.name, target);
    return {
      icd: target.icd,
      message: target.message.message,
      path: root.config.targets[target.key].message,
      naming,
    };
  };
  const { tree, handleSelect, contextMenu, renderItem, pane, currentGroup } =
    ICD.Write.useTree<WriteTarget>({
      path: root.config.targets,
      messageOf,
      plans,
      groups: TOPICS,
      TargetItem,
      targetPane: (target, onAddField) => ({
        path: root.config.targets[target.key],
        title: <RecordLabel recordKey={target.recordKey} emptyLabel={NO_KEY} />,
        details: (
          <TargetDetails
            key={target.key}
            path={root.config.targets[target.key]}
            onAddField={onAddField}
          />
        ),
      }),
      duplicate,
      selected,
      onSelect,
      expanded,
      onExpand,
    });

  const add = async () => {
    if (client == null) throw new DisconnectedError();
    const result = await addTargets({
      device,
      topic: currentGroup ?? "",
      topics: targets.value().map(({ topic }) => topic),
      label: keyLabel,
    });
    if (result == null || result.whole.length === 0) return;
    const { whole, format } = result;
    let sent: Format[];
    if (format != null) sent = whole.map(() => format);
    else {
      const { icd: icdKey, messages } = await ICD.Write.addMessages({
        client,
        formats,
        targets: targets.value(),
        name: icdName(ctx.get(root.name).value),
        messages: whole.map(({ topic, shape }) => ({
          name: topic,
          fields: ICD.createFields(shape?.fields ?? []),
        })),
      });
      sent = messages.map((message) => ({ icd: icdKey, message }));
    }
    const added = whole.map(({ topic, shape }, i) =>
      kafka.writeTargetZ.parse({
        topic,
        recordKey: recordKeyOf(shape),
        icd: sent[i].icd,
        message: ICD.Write.create(sent[i].message),
      }),
    );
    targets.push(added);
    onSelect([added[0].key]);
    onExpand((prev) =>
      unique.unique([...prev, ...ICD.Write.expandedOf(added, TOPICS)]),
    );
  };
  // Adds run one at a time, so a second click cannot make a second ICD.
  const addExclusive = useExclusiveCallback(add);
  const handleAdd = () => handleError(addExclusive, "Failed to add targets");

  return (
    <Task.Views.Panes
      listTitle="Topics"
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
      detailsPath={pane?.path ?? null}
      title={
        pane?.title ??
        (currentGroup == null ? null : <TopicLabel name={currentGroup} />)
      }
    >
      {pane?.details ?? (
        <Flex.Box y grow align="center" justify="center">
          <Text.Text status="disabled">
            {currentGroup == null
              ? "Add a target to send records to a topic"
              : "Select a record key of the topic"}
          </Text.Text>
        </Flex.Box>
      )}
      {all.map(({ key, icd: icdKey }) => (
        <ICD.Write.Suit
          key={key}
          icdKey={icdKey}
          path={root.config.targets[key].message}
        />
      ))}
    </Task.Views.Panes>
  );
};

const Content = ({ device }: PlatformDevice.TaskFormContentProps<Device.Device>) => {
  const targets = PForm.useFieldValue(root.config.targets);
  const [selected, setSelected] = useState<string[]>([]);
  const [expanded, setExpanded] = useState<string[]>(() =>
    ICD.Write.expandedOf(targets, TOPICS),
  );
  return (
    <ICD.WithFormats icdKeys={targets.map(({ icd }) => icd)}>
      <Targets
        device={device}
        selected={selected}
        onSelect={setSelected}
        expanded={expanded}
        onExpand={setExpanded}
      />
    </ICD.WithFormats>
  );
};

const Form = PlatformDevice.wrapTaskForm({
  use: Device.use,
  useConfigure: Device.useConnectModal,
  useNoneSelected: Device.useNoneSelected,
  Content,
});

export const Write = Task.wrapForm({
  Properties,
  Form,
  schemas: WRITE_SCHEMAS,
  deployConfigZ: deployWriteConfigZ,
  type: WRITE_TYPE,
  onConfigure,
});
