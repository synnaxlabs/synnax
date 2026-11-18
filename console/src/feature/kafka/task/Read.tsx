// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/kafka/task/Task.css";

import {
  channel,
  DisconnectedError,
  type icd,
  kafka,
  type Synnax,
  ValidationError,
} from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree } from "@synnaxlabs/lyra/tree";
import { ICD as PICD, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { DataType, deep, errors, type record, unique, uuid } from "@synnaxlabs/x";
import {
  type Dispatch,
  type FC,
  type SetStateAction,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";

import { use } from "@/feature/kafka/device/queries";
import { Select as SelectDevice, useNoneSelected } from "@/feature/kafka/device/Select";
import { type Device, MAKE, SCHEMAS } from "@/feature/kafka/device/types";
import { useConnectModal } from "@/feature/kafka/device/useConnectModal";
import {
  deployReadConfigZ,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadRecord,
  type ReadSchemas,
  type StartOffset,
} from "@/feature/kafka/task/types";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { ICD } from "@/platform/icd";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<ReadSchemas>>();

const DEFAULT_NAME = "Kafka read task";

const ANY_KEY = "Any key";

const recordPath = (key: string) => root.config.records[key];

const SELECT_TRIGGER_PROPS = { className: CSS.BE("kafka-select", "trigger") };

const SELECT_DIALOG_PROPS = { className: CSS.BE("kafka-select", "dialog") };

const TEXT_FIELD_CLASS = CSS.BE("kafka-properties", "text");

const TOPIC_INPUT_PROPS = { placeholder: "sensor-readings" } as const;

const GROUP_INPUT_PROPS = { placeholder: "Derived from the task" } as const;

const RECORD_KEY_INPUT_PROPS = { placeholder: ANY_KEY } as const;

const SelectStartOffset = (
  p: Omit<Select.SingleSimpleProps<StartOffset>, "children" | "resourceName">,
) => (
  <Select.Simple<StartOffset>
    {...p}
    resourceName="start offset"
    triggerProps={SELECT_TRIGGER_PROPS}
    dialogProps={SELECT_DIALOG_PROPS}
  >
    <Select.Item itemKey="latest">Latest</Select.Item>
    <Select.Item itemKey="earliest">Earliest</Select.Item>
    <Select.Item itemKey="none">Committed only</Select.Item>
  </Select.Simple>
);

const renderStartOffsetSelect = Component.renderProp(SelectStartOffset);

const Properties = () => (
  <>
    <SelectDevice />
    <Flex.Box x grow wrap>
      <PForm.TextField
        className={TEXT_FIELD_CLASS}
        path={root.config.topic}
        label="Topic"
        inputProps={TOPIC_INPUT_PROPS}
        grow
      />
      <PForm.TextField
        className={TEXT_FIELD_CLASS}
        path={root.config.group}
        label="Consumer group"
        inputProps={GROUP_INPUT_PROPS}
        grow
      />
      <PForm.Field path={root.config.startOffset} label="Start from">
        {renderStartOffsetSelect}
      </PForm.Field>
    </Flex.Box>
  </>
);

/** @returns a message that the record reads, from the record's own ICD. */
const messageOf = (
  messages: ICD.Messages,
  rec: ReadRecord,
  read: number,
): icd.Message | undefined =>
  messages.get(rec.icd)?.find(({ key }) => key === rec.messages.at(read)?.message);

/** @returns the record key, or a stand-in for a record that reads any key. */
const recordLabel = ({ recordKey }: Pick<ReadRecord, "recordKey">): string =>
  recordKey === "" ? ANY_KEY : recordKey;

/**
 * The prefix of the names of the channels that deploying creates for a message. A
 * message after the first adds its name to the prefix of the record.
 */
const messagePrefix = (
  deviceName: string,
  topic: string,
  recordKey: string,
  read: number,
  name: string,
): string => {
  let prefix = `${channel.escapeInvalidName(deviceName)}_${channel.escapeInvalidName(topic)}`;
  if (recordKey !== "") prefix += `_${channel.escapeInvalidName(recordKey)}`;
  return read === 0 ? prefix : `${prefix}_${channel.escapeInvalidName(name)}`;
};

/** @returns base, numbered from 2 until no message of the ICD has the name. */
const uniqueName = (base: string, taken: icd.Message[]): string => {
  const names = new Set(taken.map(({ name }) => name));
  if (!names.has(base)) return base;
  let n = 2;
  while (names.has(`${base} ${n}`)) n++;
  return `${base} ${n}`;
};

const createMessage = (name: string, match?: icd.JSONMatch): icd.Message => ({
  key: uuid.create(),
  name,
  payload: { format: "json", fields: [], ...(match == null ? {} : { match }) },
});

const recordOf = (icdKey: icd.Key, message: icd.MessageKey): ReadRecord =>
  kafka.readRecordZ.parse({ icd: icdKey, messages: [{ message }] });

/** @returns the name of the ICD of a task named name. */
const icdName = (name: string): string => (name.trim() === "" ? DEFAULT_NAME : name);

/** @returns the prefix of the channel names of a message, current with its name. */
const usePrefix = (
  deviceName: string,
  path: deep.Path<ReadRecord>,
  read: number,
): string => {
  const formats = ICD.useFormats();
  const topic = PForm.useFieldValue(root.config.topic);
  const recordKey = PForm.useFieldValue(path.recordKey);
  const icdKey = PForm.useFieldValue(path.icd);
  const message = PForm.useFieldValue(path.messages[read].message);
  const name = PForm.useFieldValue(PICD.formPath.messages[message].name, {
    optional: true,
    ctx: formats.get(icdKey),
  });
  return messagePrefix(deviceName, topic, recordKey, read, name ?? "");
};

/** Starts or stops reading a field of a message. */
const setRead = (
  ctx: PForm.ContextValue,
  rec: string,
  read: number,
  field: icd.JSONField,
  on: boolean,
): void => {
  const path = recordPath(rec).messages[read].bindings;
  const bindings = ctx.get(path).value;
  const bound = bindings.some(({ field: key }) => key === field.key);
  if (bound === on || (on && !ICD.isReadable(field))) return;
  ctx.set(
    path,
    on
      ? [...bindings, kafka.readBindingZ.parse({ field: field.key })]
      : bindings.filter(({ field: k }) => k !== field.key),
  );
};

interface RecordNode {
  kind: "record";
  rec: string;
}

/** A message of a record that reads several, in the tree under its record. */
interface MessageNode {
  kind: "message";
  /** The key of the record. */
  rec: string;
  /** The index of the message in the record's reads. */
  read: number;
}

/** A field of a message, in the tree under its message, or under its record. */
interface FieldNode {
  kind: "field";
  /** The key of the record. */
  rec: string;
  /** The index of the message in the record's reads. */
  read: number;
  field: icd.JSONField;
}

type Node = RecordNode | MessageNode | FieldNode;

interface Index {
  nodes: Tree.Node[];
  byKey: Map<string, Node>;
}

const messageNodeKey = (rec: string, message: icd.MessageKey): string =>
  `${rec}/${message}`;

const fieldNodeKey = (rec: string, message: icd.MessageKey, field: string): string =>
  `${rec}/${message}/${field}`;

/** Each record key with the message and shown fields of each read. */
type Structure = [string, [icd.MessageKey, icd.JSONField[]][]][];

/**
 * The records of the task as tree nodes. A record holds the fields of its message, or a
 * node for each message when it reads several. The index field stays hidden, as the
 * timestamp section edits it.
 */
const useIndex = (formats: ICD.Formats): Index => {
  const { bind, get } = PForm.useContext();
  const subscribe = (onChange: () => void) => {
    const unbinds = [
      bind(onChange),
      ...[...formats.values()].map((f) => f.bind(onChange)),
    ];
    return () => unbinds.forEach((unbind) => unbind());
  };
  const getStructure = () =>
    JSON.stringify(
      get(root.config.records).value.map((r): Structure[number] => [
        r.key,
        r.messages.map(({ message, index }) => [
          message,
          ICD.fieldsOf(formats, r.icd, message).filter(({ key }) => key !== index),
        ]),
      ]),
    );
  const structure = useSyncExternalStore(subscribe, getStructure, getStructure);
  const byKey = new Map<string, Node>();
  const nodes = (JSON.parse(structure) as Structure).map(([rec, reads]): Tree.Node => {
    byKey.set(rec, { kind: "record", rec });
    const fieldNodes = (
      read: number,
      message: icd.MessageKey,
      fields: icd.JSONField[],
    ): Tree.Node[] =>
      fields.map((field) => {
        const key = fieldNodeKey(rec, message, field.key);
        byKey.set(key, { kind: "field", rec, read, field });
        return { key };
      });
    if (reads.length === 1) return { key: rec, children: fieldNodes(0, ...reads[0]) };
    const children = reads.map(([message, fields], read): Tree.Node => {
      const key = messageNodeKey(rec, message);
      byKey.set(key, { kind: "message", rec, read });
      return { key, children: fieldNodes(read, message, fields) };
    });
    return { key: rec, children };
  });
  return { nodes, byKey };
};

/** Names a record by its record key. */
const RecordLabel = ({ path }: { path: deep.Path<ReadRecord> }) => {
  const recordKey = PForm.useFieldValue(path.recordKey);
  return (
    <Flex.Box x align="center" className={CSS.B("kafka-record-label")}>
      <Task.Views.ItemLabel
        kind="JSON"
        kindColor="var(--pluto-secondary-z)"
        color={recordKey === "" ? 8 : 10}
        variant="code"
      >
        {recordLabel({ recordKey })}
      </Task.Views.ItemLabel>
    </Flex.Box>
  );
};

const RecordItem = (props: Tree.ItemRenderProps<string>) => {
  const path = recordPath(props.itemKey);
  const disabled = PForm.useFieldValue(path.disabled, { optional: true });
  if (disabled == null) return null;
  return (
    <Task.Views.TreeGroupItem
      {...props}
      label={<RecordLabel path={path} />}
      off={disabled}
    >
      <Task.EnabledCheckbox path={path.disabled} />
    </Task.Views.TreeGroupItem>
  );
};

interface MessageItemProps extends Tree.ItemRenderProps<string> {
  node: MessageNode;
}

const MessageItem = ({ node: { rec, read }, ...props }: MessageItemProps) => (
  <Task.Views.TreeGroupItem
    {...props}
    label={<ICD.MessageLabel path={recordPath(rec)} read={read} />}
  />
);

interface FieldItemProps extends Tree.ItemRenderProps<string> {
  node: FieldNode;
}

const FieldItem = ({ node: { rec, read, field }, ...props }: FieldItemProps) => {
  const ctx = PForm.useContext();
  const path = recordPath(rec).messages[read];
  const bindings = PForm.useFieldValue(path.bindings);
  const recordOff = PForm.useFieldValue(recordPath(rec).disabled);
  const i = bindings.findIndex(({ field: key }) => key === field.key);
  return (
    <Task.Views.TreeFieldItem
      {...props}
      label={field.name}
      emptyLabel="Unnamed field"
      channel={i < 0 ? null : path.bindings[i].channel}
      off={i < 0}
      skipped={i < 0 || recordOff}
    >
      <ICD.ReadCheckbox
        bindings={path.bindings}
        field={field}
        onChange={(on) => setRead(ctx, rec, read, field, on)}
      />
    </Task.Views.TreeFieldItem>
  );
};

/** @returns the props that the panels of a message of a record share. */
const useMessageProps = (
  device: Device,
  path: deep.Path<ReadRecord>,
  read: number,
): Omit<ICD.MessageSectionsProps, "format"> => {
  const { index, bindings } = PForm.useFieldValue(path.messages[read]);
  const prefix = usePrefix(device.name, path, read);
  return {
    path,
    read,
    prefix,
    // A new index field keeps the channel of the old one.
    indexChannel: bindings.find(({ field }) => field === index)?.channel ?? 0,
    sourceLabel: "Kafka timestamp",
    missingDescription:
      "The ICD no longer has this message. Add a message to the record, then remove this one.",
  };
};

interface MessagePanelProps {
  path: deep.Path<ReadRecord>;
  /** The index of the message in the record's reads. */
  read: number;
  format: ICD.FormatForm;
  device: Device;
}

const MessageSections = ({ device, path, read, format }: MessagePanelProps) => {
  const props = useMessageProps(device, path, read);
  return <ICD.MessageSections {...props} format={format} />;
};

interface MessageDetailsProps extends MessagePanelProps {
  onAddField: () => void;
}

const MessageDetails = ({
  device,
  path,
  read,
  format,
  onAddField,
}: MessageDetailsProps) => {
  const props = useMessageProps(device, path, read);
  return (
    <ICD.MessageDetails
      {...props}
      format={format}
      matchTitle="Values"
      onAddField={onAddField}
    />
  );
};

interface FieldDetailsProps extends MessagePanelProps {
  field: icd.JSONField;
}

const FieldDetails = ({ device, ...rest }: FieldDetailsProps) => {
  const prefix = usePrefix(device.name, rest.path, rest.read);
  return <ICD.FieldDetails {...rest} prefix={prefix} />;
};

interface RecordDetailsProps {
  path: deep.Path<ReadRecord>;
  format: ICD.FormatForm;
  device: Device;
  /** Adds a field to the message of a record that reads one. */
  onAddField: () => void;
  /** Adds a message to the record, for values that its messages do not cover. */
  onAddMessage: () => void;
}

/**
 * The record key of a record. A record that reads one message also shows its timestamp,
 * which each message of a record that reads several shows on its own.
 */
const RecordDetails: FC<RecordDetailsProps> = ({
  path,
  format,
  device,
  onAddField,
  onAddMessage,
}) => {
  const isPreview = Task.useIsPreview();
  const reads = PForm.useFieldValue(path.messages).length;
  return (
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      <PForm.Sections>
        <PForm.Section
          title="Record"
          actions={
            !isPreview && (
              <>
                {reads === 1 && <ICD.AddField onClick={onAddField} />}
                <Button.Button variant="text" size="small" onClick={onAddMessage}>
                  <Icon.Add />
                  Add message
                </Button.Button>
              </>
            )
          }
        >
          <PForm.TextField
            path={path.recordKey}
            label="Record key"
            padHelpText={false}
            inputProps={RECORD_KEY_INPUT_PROPS}
          />
        </PForm.Section>
        {reads === 1 && (
          <MessageSections path={path} read={0} format={format} device={device} />
        )}
      </PForm.Sections>
    </Flex.Box>
  );
};

/**
 * @returns the expanded state of a task's tree as it opens: every record, and every
 * message of a record that reads several.
 */
const initialExpanded = (records: ReadRecord[]): string[] =>
  records.flatMap((r) =>
    r.messages.length > 1
      ? [r.key, ...r.messages.map(({ message }) => messageNodeKey(r.key, message))]
      : [r.key],
  );

interface RecordsProps {
  device: Device;
  /** The key of each selected tree item. */
  selected: string[];
  onSelect: Dispatch<SetStateAction<string[]>>;
  expanded: string[];
  onExpand: Dispatch<SetStateAction<string[]>>;
}

/** The records of the task as a tree, beside the details of the selected item. */
const Records: FC<RecordsProps> = ({
  device,
  selected,
  onSelect,
  expanded,
  onExpand,
}) => {
  const client = PSynnax.use();
  const ctx = PForm.useContext();
  const handleError = Status.useErrorHandler();
  const formats = ICD.useFormats();
  const { push, remove } = PForm.useFieldList(root.config.records);
  const records = PForm.useFieldValue(root.config.records);
  const { nodes, byKey } = useIndex(formats);
  const tree = Tree.use({
    nodes,
    selected,
    onSelectedChange: onSelect,
    expanded,
    onExpandedChange: onExpand,
    toggleOn: "caret",
  });

  /**
   * Adds a record that reads any key, with a new message in the ICD of the first
   * record. A task with no record gets a new ICD.
   */
  const add = async () => {
    if (client == null) throw new DisconnectedError();
    const first = records.at(0);
    let rec: ReadRecord;
    if (first == null) {
      const message = createMessage(ANY_KEY);
      const name = icdName(ctx.get(root.name).value);
      const { key } = await client.icds.create({ name, messages: [message] });
      rec = recordOf(key, message.key);
    } else {
      const format = ICD.formatOf(formats, first.icd);
      const held = format.value().messages;
      const message = createMessage(uniqueName(ANY_KEY, held));
      format.insert(PICD.formPath.messages, message, held.at(-1)?.key);
      rec = recordOf(first.icd, message.key);
    }
    push(rec);
    onSelect([rec.key]);
    tree.expand(rec.key);
  };
  // Adds run one at a time, so a second click cannot make a second ICD.
  const adding = useRef(false);
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
    }, "Failed to add record");

  /**
   * Adds a message with no fields to the record and to its ICD. The match tests the
   * pointer of the record's other matches, so the messages tell values apart.
   */
  const handleAddMessage = (recKey: string) => {
    const path = recordPath(recKey);
    const rec = ctx.get(path).value;
    const form = ICD.formatOf(formats, rec.icd);
    const held = form.value().messages;
    const rule = rec.messages
      .map(({ message }) => held.find(({ key }) => key === message)?.payload)
      .find((p) => p?.format === "json" && p.match != null);
    const match = rule?.format === "json" ? rule.match : undefined;
    const message = createMessage(
      uniqueName(`${recordLabel(rec)} ${rec.messages.length + 1}`, held),
      ICD.createMatch(match?.type ?? "string", match?.pointer ?? ""),
    );
    form.insert(PICD.formPath.messages, message, held.at(-1)?.key);
    ctx.set(path.messages, [
      ...rec.messages,
      kafka.readMessageZ.parse({ message: message.key }),
    ]);
    const shown = [...rec.messages.map((m) => m.message), message.key];
    onExpand((prev) =>
      unique.unique([...prev, recKey, ...shown.map((m) => messageNodeKey(recKey, m))]),
    );
    onSelect([messageNodeKey(recKey, message.key)]);
  };

  /** Adds a field to a message of a record and to its ICD, reads it, and selects it. */
  const handleAddField = (recKey: string, read: number) => {
    const rec = ctx.get(recordPath(recKey)).value;
    const { message } = rec.messages[read];
    const fields = ICD.fieldsOf(formats, rec.icd, message);
    const field = ICD.createJSONField(fields.map(({ name }) => name));
    ICD.formatOf(formats, rec.icd).insert(
      ICD.fieldsPath(message),
      field,
      fields.at(-1)?.key,
    );
    setRead(ctx, recKey, read, field, true);
    const parent = rec.messages.length > 1 ? [messageNodeKey(recKey, message)] : [];
    onExpand((prev) => unique.unique([...prev, recKey, ...parent]));
    onSelect([fieldNodeKey(recKey, message, field.key)]);
  };

  /** Removes each chosen field from its message in the ICD. */
  const handleRemoveFields = (chosen: FieldNode[]) => {
    for (const { rec, read, field } of chosen) {
      const { icd: icdKey, messages } = ctx.get(recordPath(rec)).value;
      ICD.formatOf(formats, icdKey).remove(
        ICD.fieldsPath(messages[read].message)[field.key],
      );
    }
    onSelect([]);
  };

  const handleSelect: Tree.TreeProps<string, record.Keyed<string>>["onSelect"] = (
    keys,
    { clicked },
  ) => {
    onSelect(keys);
    const kind = clicked == null ? undefined : byKey.get(clicked)?.kind;
    if (clicked != null && kind !== "field") tree.expand(clicked);
  };

  /** Sets whether the task reads each chosen field, and enables each chosen record. */
  const setEnabled = (chosen: (RecordNode | FieldNode)[], enabled: boolean) =>
    ctx.batch(() => {
      for (const node of chosen)
        if (node.kind === "record") ctx.set(recordPath(node.rec).disabled, !enabled);
        else setRead(ctx, node.rec, node.read, node.field, enabled);
    });
  const isEnabled = (node: RecordNode | FieldNode): boolean => {
    if (node.kind === "record") return !ctx.get(recordPath(node.rec).disabled).value;
    return ctx
      .get(recordPath(node.rec).messages[node.read].bindings)
      .value.some(({ field }) => field === node.field.key);
  };
  /** Removes each chosen record, and each chosen message from its record. */
  const handleRemove = (chosen: (RecordNode | MessageNode)[]) => {
    const removed = new Set(
      chosen.flatMap((n) => (n.kind === "record" ? [n.rec] : [])),
    );
    const dropped = new Map<string, Set<number>>();
    for (const n of chosen)
      if (n.kind === "message" && !removed.has(n.rec))
        dropped.set(n.rec, (dropped.get(n.rec) ?? new Set()).add(n.read));
    ctx.batch(() => {
      dropped.forEach((reads, rec) => {
        const path = recordPath(rec).messages;
        ctx.set(
          path,
          ctx.get(path).value.filter((_, r) => !reads.has(r)),
        );
      });
      if (removed.size > 0) remove([...removed]);
    });
    onSelect([]);
  };
  /**
   * @returns whether the chosen nodes can go: records, and messages that leave each of
   * their records at least one.
   */
  const isRemovable = (chosen: Node[]): chosen is (RecordNode | MessageNode)[] => {
    if (chosen.some(({ kind }) => kind === "field")) return false;
    const counts = new Map<string, number>();
    for (const n of chosen)
      if (n.kind === "message") counts.set(n.rec, (counts.get(n.rec) ?? 0) + 1);
    return [...counts].every(
      ([rec, count]) =>
        chosen.some((n) => n.kind === "record" && n.rec === rec) ||
        count < ctx.get(recordPath(rec).messages).value.length,
    );
  };
  const contextMenu = ({ keys }: Menu.ContextMenuMenuProps) => {
    const chosen = keys.flatMap((k) => byKey.get(k) ?? []);
    const switchable = chosen.filter(
      (n): n is RecordNode | FieldNode => n.kind !== "message",
    );
    const [only] = chosen;
    const single = chosen.length === 1 && only.kind === "record" ? only.rec : undefined;
    let addField: (() => void) | undefined;
    if (chosen.length === 1 && only.kind === "message")
      addField = () => handleAddField(only.rec, only.read);
    else if (single != null && ctx.get(recordPath(single).messages).value.length === 1)
      addField = () => handleAddField(single, 0);
    const fields = chosen.filter((n): n is FieldNode => n.kind === "field");
    let onRemove: (() => void) | undefined;
    if (fields.length > 0 && fields.length === chosen.length)
      onRemove = () => handleRemoveFields(fields);
    else if (isRemovable(chosen)) onRemove = () => handleRemove(chosen);
    return (
      <Task.Views.ContextMenu
        keys={keys}
        onRemove={onRemove}
        onEnable={
          switchable.some((n) => !isEnabled(n))
            ? () => setEnabled(switchable, true)
            : undefined
        }
        onDisable={
          switchable.some(isEnabled) ? () => setEnabled(switchable, false) : undefined
        }
        onAddField={addField}
        extraMenuItems={
          single != null && (
            <Menu.Item itemKey="addMessage" onClick={() => handleAddMessage(single)}>
              <Icon.Add />
              Add message
            </Menu.Item>
          )
        }
      />
    );
  };

  const renderItem = ({ key, ...p }: Tree.ItemRenderProps<string>) => {
    const node = byKey.get(p.itemKey);
    if (node?.kind === "field") return <FieldItem key={key} {...p} node={node} />;
    if (node?.kind === "message") return <MessageItem key={key} {...p} node={node} />;
    return <RecordItem key={key} {...p} />;
  };

  const current = selected.length > 0 ? byKey.get(selected[0]) : undefined;
  const rec =
    current == null ? undefined : records.find(({ key }) => key === current.rec);
  const format = rec == null ? undefined : formats.get(rec.icd);
  let details = null;
  let title = null;
  let detailsPath: deep.Target<unknown> | null = null;
  if (rec != null && format != null && current?.kind === "field") {
    const read = recordPath(rec.key).messages[current.read];
    const i = rec.messages[current.read].bindings.findIndex(
      ({ field }) => field === current.field.key,
    );
    detailsPath = i < 0 ? read : read.bindings[i];
    title = (
      <Text.Text level="p" weight={500} overflow="fade">
        {current.field.name}
      </Text.Text>
    );
    details = (
      <FieldDetails
        key={selected[0]}
        path={recordPath(rec.key)}
        read={current.read}
        field={current.field}
        format={format}
        device={device}
      />
    );
  } else if (rec != null && format != null && current?.kind === "message") {
    detailsPath = recordPath(rec.key).messages[current.read];
    title = <ICD.MessageLabel path={recordPath(rec.key)} read={current.read} />;
    details = (
      <MessageDetails
        key={selected[0]}
        path={recordPath(rec.key)}
        read={current.read}
        format={format}
        device={device}
        onAddField={() => handleAddField(rec.key, current.read)}
      />
    );
  } else if (rec != null) {
    detailsPath = recordPath(rec.key);
    title = <RecordLabel path={recordPath(rec.key)} />;
    if (format != null)
      details = (
        <RecordDetails
          key={rec.key}
          path={recordPath(rec.key)}
          format={format}
          device={device}
          onAddField={() => handleAddField(rec.key, 0)}
          onAddMessage={() => handleAddMessage(rec.key)}
        />
      );
  }
  return (
    <Task.Views.Panes
      listTitle="Records"
      list={
        <Task.Views.TreeList
          {...tree}
          onSelect={handleSelect}
          menu={contextMenu}
          emptyMessage="No records"
          addLabel="Add record"
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
          <Text.Text status="disabled">Add a record to read its messages</Text.Text>
        </Flex.Box>
      )}
    </Task.Views.Panes>
  );
};

const Content = ({ device }: PlatformDevice.TaskFormContentProps<Device>) => {
  const records = PForm.useFieldValue(root.config.records);
  const [selected, setSelected] = useState<string[]>([]);
  const [expanded, setExpanded] = useState<string[]>(() => initialExpanded(records));
  const body = (
    <>
      <Records
        device={device}
        selected={selected}
        onSelect={setSelected}
        expanded={expanded}
        onExpand={setExpanded}
      />
      {records.flatMap(({ key, messages }) =>
        messages.map(({ message }, read) => (
          <ICD.SuitRead key={`${key}/${message}`} path={recordPath(key)} read={read} />
        )),
      )}
    </>
  );
  return unique.unique(records.map((r) => r.icd)).reduceRight(
    (child, key) => (
      <ICD.WithFormat key={key} icdKey={key}>
        {child}
      </ICD.WithFormat>
    ),
    body,
  );
};

const Form = PlatformDevice.wrapTaskForm({
  use,
  useConfigure: useConnectModal,
  useNoneSelected,
  Content,
});

const getInitialValues: Task.GetInitialValues<ReadSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = READ_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: DEFAULT_NAME, type: READ_TYPE, config: cfg };
};

/** A message that a record reads, with the fields that the task reads of it. */
interface BoundMessage {
  rec: ReadRecord;
  /** The index of the message in the record's reads. */
  read: number;
  message: icd.Message;
  fields: icd.JSONField[];
}

/**
 * @returns a message that the record reads, with the fields that the task reads of it.
 * @throws {ValidationError} if the task cannot read the message.
 */
const boundMessage = (
  messages: ICD.Messages,
  rec: ReadRecord,
  read: number,
): BoundMessage => {
  const message = messageOf(messages, rec, read);
  const name = `record ${recordLabel(rec)}`;
  const where =
    message != null && rec.messages.length > 1 ? `${message.name} of ${name}` : name;
  if (message?.payload.format !== "json")
    throw new ValidationError(`The message format of ${where} is missing or not JSON`);
  const readMessage = rec.messages[read];
  const fields = ICD.readFields(message.payload.fields, readMessage);
  const index = fields.find(({ key }) => key === readMessage.index);
  if (index != null && !ICD.isIndexable(index))
    throw new ValidationError(
      `In ${where}, the index field ${index.name} must be a timestamp field with no default`,
    );
  if (fields.length === 0)
    throw new ValidationError(`The task reads no fields of ${where}`);
  const mixed = ICD.variableAmongFixed(fields);
  if (mixed.length > 0)
    throw new ValidationError(`In ${where}, ${ICD.mixMessage(mixed)}`);
  return { rec, read, message, fields };
};

/**
 * Gives each field that a message reads a channel: the one it has, or a new one of the
 * binding's data type. A channel that no longer suits its field is replaced.
 */
const configureMessage = async (
  client: Synnax,
  prefix: string,
  { rec, read: r, fields }: BoundMessage,
): Promise<void> => {
  const read = rec.messages[r];
  const index = fields.find(({ key }) => key === read.index)?.key;
  const bindingOf = (f: icd.JSONField): kafka.ReadBinding =>
    read.bindings.find(({ field }) => field === f.key) ??
    kafka.readBindingZ.parse({ field: f.key });
  const held = fields.map((f) => bindingOf(f).channel).filter((k) => k !== 0);
  const live = new Map(
    (await client.channels.retrieve(held)).map((ch) => [ch.key, ch]),
  );
  const channels = fields.map((f) => {
    const binding = bindingOf(f);
    const isIndex = f.key === index;
    const suited = binding.dataType !== "" && ICD.converts(f.type, binding.dataType);
    const dataType = suited ? binding.dataType : "";
    const ch = live.get(binding.channel);
    // An edit of the field's kind or index role since the last deploy needs a new
    // channel.
    const stale =
      ch != null &&
      (!ICD.converts(f.type, ch.dataType.toString()) || ch.isIndex !== isIndex);
    return {
      key: f.key,
      pointer: f.pointer,
      name: `${prefix}_${channel.escapeInvalidName(f.name)}`,
      dataType: isIndex
        ? DataType.TIMESTAMP.toString()
        : ICD.bindingDataType({ dataType }, f.type),
      channel: stale ? 0 : binding.channel,
      binding: { ...binding, dataType },
    };
  });
  // A cluster stores no channels, so the index comes from the live channels or is new.
  await Task.configureReadChannels({
    client,
    read: {},
    propertiesKey: "",
    fields: channels,
    namePrefix: prefix,
    indexKey: index ?? "",
  });
  rec.messages[r] = {
    message: read.message,
    index,
    bindings: channels.map(({ channel, binding }) => ({ ...binding, channel })),
  };
};

/** Gives each field that an enabled record reads a channel, and binds it. */
const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (client, config) => {
  const dev = await client.devices.retrieve({ key: config.device, schemas: SCHEMAS });
  const enabled = config.records.filter(({ disabled }) => !disabled);
  const messages = await ICD.retrieveMessages(
    client,
    enabled.map((r) => r.icd),
  );
  // Every format is checked before any channel is created.
  const bound = enabled.flatMap((rec) =>
    rec.messages.map((_, read) => boundMessage(messages, rec, read)),
  );
  for (const b of bound) {
    const prefix = messagePrefix(
      dev.name,
      config.topic,
      b.rec.recordKey,
      b.read,
      b.message.name,
    );
    await configureMessage(client, prefix, b);
  }
  return [config, dev.rack];
};

export const Read = Task.wrapForm({
  Properties,
  Form,
  schemas: READ_SCHEMAS,
  deployConfigZ: deployReadConfigZ,
  type: READ_TYPE,
  onConfigure,
});

export const useCreateRead = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const ReadSelectable: Selector.Selectable = {
  type: READ_TYPE,
  title: DEFAULT_NAME,
  label: "Read",
  icon: <Icon.Logo.Kafka />,
  useOnSelect: useCreateRead,
};
