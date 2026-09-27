// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/mqtt/task/Form.css";

import { channel, mqtt, type Synnax as Client } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree } from "@synnaxlabs/lyra/tree";
import { Telem } from "@synnaxlabs/pluto";
import { DataType, id, primitive, type record } from "@synnaxlabs/x";
import { type FC, type MouseEvent, useCallback, useMemo, useState } from "react";

import { Browser } from "@/feature/mqtt/device/Browser";
import { use } from "@/feature/mqtt/device/queries";
import { Select as SelectDevice } from "@/feature/mqtt/device/Select";
import { type SparkplugHaulTag } from "@/feature/mqtt/device/SparkplugBrowser";
import { type Device, SCHEMAS } from "@/feature/mqtt/device/types";
import { useConnectModal } from "@/feature/mqtt/device/useConnectModal";
import { createReadFields } from "@/feature/mqtt/task/createReadFields";
import { EntryLabel } from "@/feature/mqtt/task/EntryLabel";
import { AddEntryButtons, useEntryDrop } from "@/feature/mqtt/task/EntryList";
import { QoSField } from "@/feature/mqtt/task/QoSField";
import {
  fromSparkplugDataType,
  sparkplugChannelName,
  sparkplugPropertiesKey,
  toSparkplugDataType,
} from "@/feature/mqtt/task/sparkplug";
import { SparkplugTagFields } from "@/feature/mqtt/task/SparkplugTagFields";
import { TimeFormatField } from "@/feature/mqtt/task/TimeFormatField";
import {
  type BrowsedTopic,
  deployReadConfigZ,
  HIDDEN_DATA_TYPES,
  type PlainReadEntry,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadEntry,
  type ReadField,
  type ReadSchemas,
  type SparkplugReadEntry,
} from "@/feature/mqtt/task/types";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { Empty } from "@/platform/empty";
import { Form as PlatformForm } from "@/platform/form";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const Properties = () => (
  <>
    <SelectDevice />
    <Flex.Box x grow>
      <Task.Fields.DataSaving />
      <Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

const ENTRIES_PATH = "config.entries";

type TreeEntry =
  | { kind: "entry"; entryKey: string; type: ReadEntry["type"] }
  | { kind: "field"; entryKey: string; fieldKey: string };

interface TreeIndex {
  nodes: Tree.Node[];
  entries: Map<string, TreeEntry>;
}

/**
 * Entries as tree nodes. A plain entry holds its fields beneath it, less the timestamp
 * field; a Sparkplug B entry is a leaf.
 */
const useTreeIndex = (): TreeIndex => {
  // The form writes child paths into the same array, so the array's identity never
  // changes; the field state's does on every write beneath it.
  const state = PForm.useFieldState<ReadEntry[]>(ENTRIES_PATH);
  return useMemo(() => {
    const entries = new Map<string, TreeEntry>();
    const nodes = state.value.map((entry): Tree.Node => {
      entries.set(entry.key, { kind: "entry", entryKey: entry.key, type: entry.type });
      if (entry.type === "sparkplug") return { key: entry.key };
      const children = entry.fields
        .filter((f) => f.key !== entry.index)
        .map((f) => {
          entries.set(f.key, { kind: "field", entryKey: entry.key, fieldKey: f.key });
          return { key: f.key };
        });
      return { key: entry.key, children };
    });
    return { nodes, entries };
  }, [state]);
};

const entryPath = (entry: TreeEntry): string =>
  entry.kind === "entry"
    ? `${ENTRIES_PATH}.${entry.entryKey}`
    : `${ENTRIES_PATH}.${entry.entryKey}.fields.${entry.fieldKey}`;

/** The prefix configure gives the channels of a topic's fields. */
const namePrefix = (devName: string, topic: string): string =>
  `${channel.escapeInvalidName(devName)}_${channel.escapeInvalidName(topic)}`;

interface EntryTreeItemProps extends Tree.ItemRenderProps<string> {
  onAddField: (entryKey: string) => void;
}

const EntryTreeItem = ({ onAddField, ...props }: EntryTreeItemProps) => {
  const { itemKey } = props;
  const path = `${ENTRIES_PATH}.${itemKey}`;
  const type = PForm.useFieldValue<ReadEntry["type"]>(`${path}.type`);
  const disabled = PForm.useFieldValue<boolean>(`${path}.disabled`);
  const isPreview = Task.useIsPreview();
  const handleAdd = useCallback(
    (e: MouseEvent) => {
      e.stopPropagation();
      onAddField(itemKey);
    },
    [onAddField, itemKey],
  );
  return (
    <Tree.Item
      {...props}
      className={CSS.cls(
        CSS.B("entry-item"),
        type === "sparkplug" && CSS.M("leaf"),
        disabled && CSS.M("off"),
      )}
    >
      <EntryLabel path={path} />
      {type === "plain" && !isPreview && (
        <Button.Button
          onClick={handleAdd}
          variant="text"
          size="tiny"
          tooltip="Add field"
          aria-label="Add field"
          tooltipLocation="right"
          className={CSS.BE("entry-item", "add")}
        >
          <Icon.Add />
        </Button.Button>
      )}
      <Task.EnabledCheckbox path={`${path}.disabled`} />
    </Tree.Item>
  );
};

interface FieldLabelProps extends Pick<Text.TextProps, "level"> {
  pointer: string;
}

/** Names a field by its pointer. An empty pointer takes the whole payload. */
const FieldLabel = ({ pointer, level }: FieldLabelProps) => (
  <Text.Text
    level={level}
    weight={500}
    color={pointer === "" ? 8 : 10}
    overflow="ellipsis"
    className={CSS.B("field-pointer")}
  >
    {pointer === "" ? "Whole payload" : pointer}
  </Text.Text>
);

interface FieldTreeItemProps extends Tree.ItemRenderProps<string> {
  entryKey: string;
}

const FieldTreeItem = ({ entryKey, ...props }: FieldTreeItemProps) => {
  const { itemKey } = props;
  const path = `${ENTRIES_PATH}.${entryKey}.fields.${itemKey}`;
  const { disabled, pointer } = PForm.useFieldValue<ReadField>(path);
  return (
    <Tree.Item
      {...props}
      className={CSS.cls(CSS.B("field-item"), disabled && CSS.M("off"))}
    >
      <FieldLabel pointer={pointer} level="small" />
      <Task.EnabledCheckbox path={`${path}.disabled`} />
    </Tree.Item>
  );
};

const POINTER_INPUT_PROPS = { placeholder: "/temperature" } as const;

const renderTelemSelectDataType = Component.renderProp(
  (p: Telem.SelectDataTypeProps) => (
    <Telem.SelectDataType {...p} hideDataTypes={HIDDEN_DATA_TYPES} location="bottom" />
  ),
);

interface BinderProps {
  device: Device;
  entryKey: string;
}

/**
 * Binds a plain entry's fields to the channels the device stores for its topic.
 * Mounted for every entry, as the details pane renders only the selected one.
 */
const FieldBinder = ({ device, entryKey }: BinderProps) => {
  const path = `${ENTRIES_PATH}.${entryKey}`;
  const topic = PForm.useFieldValue<string>(`${path}.topic`);
  const indexKey = PForm.useFieldValue<string>(`${path}.index`);
  const resolve = useCallback(
    (field: ReadField) => {
      const props = device.properties.read[topic];
      if (props == null) return { channel: 0 };
      return {
        channel:
          field.key === indexKey ? props.index : (props.channels[field.pointer] ?? 0),
      };
    },
    [device, topic, indexKey],
  );
  return <Task.BindChannels<ReadField> path={`${path}.fields`} resolve={resolve} />;
};

type TimingMode = "arrival" | "payload";

const TimestampFields: FC<{ path: string }> = ({ path }) => {
  const index = PForm.useFieldValue<string>(`${path}.index`);
  const { get, set } = PForm.useContext();
  const isPayloadTiming = index !== "";

  const handleChange = useCallback(
    (mode: TimingMode) => {
      const { fields, index } = get<PlainReadEntry>(path).value;
      if (mode === "payload" && index === "") {
        const indexField: ReadField = {
          ...mqtt.readFieldZ.parse({}),
          dataType: DataType.TIMESTAMP.toString(),
          timeFormat: "unix_sec",
        };
        set(`${path}.fields`, [...fields, indexField]);
        set(`${path}.index`, indexField.key);
      } else if (mode === "arrival" && index !== "") {
        set(
          `${path}.fields`,
          fields.filter((f) => f.key !== index),
        );
        set(`${path}.index`, "");
      }
    },
    [path, get, set],
  );

  return (
    <>
      <Input.Item label="Source" padHelpText={false}>
        <Select.Buttons<TimingMode>
          value={isPayloadTiming ? "payload" : "arrival"}
          onChange={handleChange}
        >
          <Select.Item<TimingMode> itemKey="arrival">Arrival time</Select.Item>
          <Select.Item<TimingMode> itemKey="payload">Payload value</Select.Item>
        </Select.Buttons>
      </Input.Item>
      {isPayloadTiming && (
        <>
          <PForm.TextField
            path={`${path}.fields.${index}.pointer`}
            label="Pointer"
            padHelpText={false}
            inputProps={TIMESTAMP_POINTER_INPUT_PROPS}
          />
          <TimeFormatField path={`${path}.fields.${index}.timeFormat`} label="Format" />
        </>
      )}
    </>
  );
};

const TIMESTAMP_POINTER_INPUT_PROPS = { placeholder: "/timestamp" } as const;

const TOPIC_INPUT_PROPS = { placeholder: "plant/line1/temperature" } as const;

const PlainEntryPane: FC<{ path: string }> = ({ path }) => (
  <PForm.Sections>
    <PForm.Section title="Subscription">
      <PForm.TextField
        path={`${path}.topic`}
        label="Topic"
        padHelpText={false}
        inputProps={TOPIC_INPUT_PROPS}
      />
      <QoSField path={`${path}.qos`} />
      <PForm.SwitchField
        path={`${path}.retainedIgnored`}
        label="Ignore retained"
        padHelpText={false}
      />
    </PForm.Section>
    <PForm.Section title="Timestamp">
      <TimestampFields path={path} />
    </PForm.Section>
  </PForm.Sections>
);

const SPARKPLUG_HIDDEN_DATA_TYPES = [DataType.UUID, DataType.JSON, DataType.BYTES];

const renderSparkplugSelectDataType = Component.renderProp(
  (p: Telem.SelectDataTypeProps) => (
    <Telem.SelectDataType
      {...p}
      hideDataTypes={SPARKPLUG_HIDDEN_DATA_TYPES}
      location="bottom"
    />
  ),
);

interface PaneProps {
  path: string;
  deviceName: string;
}

const SparkplugEntryPane: FC<PaneProps> = ({ path, deviceName }) => {
  const entry = PForm.useFieldValue<SparkplugReadEntry>(path);
  const bound = entry.channel !== 0;
  return (
    <PForm.Sections>
      <PForm.Section title="Tag">
        <SparkplugTagFields path={path} />
      </PForm.Section>
      <PForm.Section title="Channel">
        <Task.ChannelNameField
          channel={entry.channel}
          namePath={`${path}.name`}
          defaultName={sparkplugChannelName(deviceName, entry)}
        />
        <PForm.Field<string>
          path={`${path}.dataType`}
          label="Data type"
          padHelpText={false}
          helpText={bound ? "Set on the channel" : undefined}
        >
          {(p) => renderSparkplugSelectDataType({ ...p, disabled: bound })}
        </PForm.Field>
      </PForm.Section>
    </PForm.Sections>
  );
};

const EntryPane: FC<PaneProps> = ({ path, deviceName }) => {
  const type = PForm.useFieldValue<ReadEntry["type"]>(`${path}.type`);
  return (
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      {type === "plain" ? (
        <PlainEntryPane path={path} />
      ) : (
        <SparkplugEntryPane path={path} deviceName={deviceName} />
      )}
    </Flex.Box>
  );
};

interface FieldPaneProps extends Pick<PaneProps, "deviceName"> {
  entryKey: string;
  fieldKey: string;
}

const FieldTitle: FC<Omit<FieldPaneProps, "deviceName">> = ({ entryKey, fieldKey }) => (
  <FieldLabel
    pointer={PForm.useFieldValue<string>(
      `${ENTRIES_PATH}.${entryKey}.fields.${fieldKey}.pointer`,
    )}
    level="p"
  />
);

const FieldPane: FC<FieldPaneProps> = ({ entryKey, fieldKey, deviceName }) => {
  const path = `${ENTRIES_PATH}.${entryKey}.fields.${fieldKey}`;
  const { channel: fieldChannel, pointer } = PForm.useFieldValue<ReadField>(path);
  const topic = PForm.useFieldValue<string>(`${ENTRIES_PATH}.${entryKey}.topic`);
  const bound = fieldChannel !== 0;
  return (
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      <PForm.Sections>
        <PForm.Section title="Field">
          <PForm.TextField
            path={`${path}.pointer`}
            label="Pointer"
            padHelpText={false}
            inputProps={POINTER_INPUT_PROPS}
          />
          <PForm.Field<string>
            path={`${path}.dataType`}
            label="Data type"
            padHelpText={false}
            helpText={bound ? "Set on the channel" : undefined}
          >
            {(p) => renderTelemSelectDataType({ ...p, disabled: bound })}
          </PForm.Field>
          <Task.ChannelNameField
            channel={fieldChannel}
            namePath={`${path}.name`}
            defaultName={
              namePrefix(deviceName, topic) + channel.escapeInvalidName(pointer)
            }
          />
        </PForm.Section>
        <PForm.Section title="Enum mapping">
          <PlatformForm.KeyValueEditor
            path={`${path}.enumValues`}
            keyField="label"
            keyPlaceholder="String (e.g. ON)"
            valueType="number"
          />
        </PForm.Section>
      </PForm.Sections>
    </Flex.Box>
  );
};

const createEntry = (topic?: BrowsedTopic): PlainReadEntry => ({
  ...mqtt.plainReadEntryZ.parse({ type: "plain" }),
  ...(topic != null && {
    topic: topic.topic,
    fields: createReadFields(topic.payload),
  }),
});

const createSparkplugEntry = (tag?: SparkplugHaulTag): SparkplugReadEntry => {
  const entry = mqtt.sparkplugReadEntryZ.parse({ type: "sparkplug" });
  if (tag == null) return entry;
  const { dataType, ...tagID } = tag;
  return {
    ...entry,
    ...tagID,
    dataType: fromSparkplugDataType(toSparkplugDataType(dataType)),
  };
};

const duplicateEntry = (entry: ReadEntry): ReadEntry => {
  if (entry.type === "sparkplug")
    return { ...entry, ...Task.READ_CHANNEL_OVERRIDE, key: id.create(), index: 0 };
  const fields = entry.fields.map((f) => ({
    ...f,
    ...Task.READ_CHANNEL_OVERRIDE,
    key: id.create(),
  }));
  const indexPosition = entry.fields.findIndex((f) => f.key === entry.index);
  return {
    ...entry,
    key: id.create(),
    fields,
    index: fields[indexPosition]?.key ?? "",
  };
};

const newField = (last?: ReadField): ReadField => ({
  ...(last != null
    ? { ...last, ...Task.READ_CHANNEL_OVERRIDE }
    : mqtt.readFieldZ.parse({})),
  key: id.create(),
});

const TREE_ITEM_HEIGHT = 36;

const EMPTY_CONTENT = <Empty.Action message="No entries" />;

const Content = ({ device }: PlatformDevice.TaskFormContentProps<Device>) => {
  const [selected, setSelected] = useState<string[]>([]);
  const { nodes, entries } = useTreeIndex();
  const ctx = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const [initialExpanded] = useState(() => nodes.map(({ key }) => key));
  // A fold that hides the selected field moves the selection up to its entry.
  const handleExpand = useCallback(
    ({ action, clicked }: Tree.HandleExpandProps<string>) => {
      if (action !== "contract") return;
      const hidesSelection = selected.some((k) => {
        const entry = entries.get(k);
        return entry?.kind === "field" && entry.entryKey === clicked;
      });
      if (hidesSelection) setSelected([clicked]);
    },
    [selected, entries],
  );
  const treeProps = Tree.use({
    nodes,
    selected,
    onSelectedChange: setSelected,
    initialExpanded,
    onExpand: handleExpand,
    toggleOn: "caret",
  });
  const { expand, shape } = treeProps;

  const handleSelect = useCallback<
    Tree.TreeProps<string, record.Keyed<string>>["onSelect"]
  >(
    (keys, { clicked }) => {
      setSelected(keys);
      if (clicked != null && entries.get(clicked)?.kind === "entry") expand(clicked);
    },
    [entries, expand],
  );

  const handleAddEntries = useCallback(
    (added: ReadEntry[]) => {
      const current = ctx.get<ReadEntry[]>(ENTRIES_PATH).value;
      ctx.set(ENTRIES_PATH, [...current, ...added]);
      setSelected([added[0].key]);
      for (const { key, type } of added) if (type === "plain") expand(key);
    },
    [ctx, expand],
  );
  const handleAddEntry = useCallback(
    () => handleAddEntries([createEntry()]),
    [handleAddEntries],
  );
  const handleAddSparkplugEntry = useCallback(
    () => handleAddEntries([createSparkplugEntry()]),
    [handleAddEntries],
  );
  const dropProps = useEntryDrop<ReadEntry>({
    path: ENTRIES_PATH,
    create: createEntry,
    createSparkplug: createSparkplugEntry,
    onAdd: handleAddEntries,
  });

  const handleAddField = useCallback(
    (entryKey: string) => {
      const path = `${ENTRIES_PATH}.${entryKey}`;
      const { fields, index } = ctx.get<PlainReadEntry>(path).value;
      const field = newField(fields.findLast((f) => f.key !== index));
      ctx.set(`${path}.fields`, [...fields, field]);
      setSelected([field.key]);
      expand(entryKey);
    },
    [ctx, expand],
  );

  const handleRemove = useCallback(
    (keys: string[]) => {
      const removed = new Set(keys);
      const current = ctx.get<ReadEntry[]>(ENTRIES_PATH).value;
      ctx.set(
        ENTRIES_PATH,
        current
          .filter((entry) => !removed.has(entry.key))
          .map((entry) =>
            entry.type === "plain"
              ? { ...entry, fields: entry.fields.filter((f) => !removed.has(f.key)) }
              : entry,
          ),
      );
      // Selection moves to the nearest survivor in the visible list, below first.
      const visible = shape.keys;
      const first = visible.findIndex((k) => removed.has(k));
      const after = visible.slice(first).find((k) => !removed.has(k));
      const before = visible
        .slice(0, Math.max(first, 0))
        .reverse()
        .find((k) => !removed.has(k));
      const next = after ?? before;
      setSelected(next == null ? [] : [next]);
    },
    [ctx, shape],
  );

  const handleDuplicate = useCallback(
    (keys: string[]) => {
      const chosen = new Set(keys);
      const current = ctx.get<ReadEntry[]>(ENTRIES_PATH).value;
      const next: ReadEntry[] = [];
      let first: string | null = null;
      for (const entry of current) {
        if (entry.type === "plain") {
          const fields: ReadField[] = [];
          for (const f of entry.fields) {
            fields.push(f);
            if (!chosen.has(f.key)) continue;
            const copy = newField(f);
            first ??= copy.key;
            fields.push(copy);
          }
          next.push({ ...entry, fields });
        } else next.push(entry);
        if (!chosen.has(entry.key)) continue;
        const copy = duplicateEntry(entry);
        first ??= copy.key;
        next.push(copy);
        if (copy.type === "plain") expand(copy.key);
      }
      ctx.set(ENTRIES_PATH, next);
      if (first != null) setSelected([first]);
    },
    [ctx, expand],
  );

  const handleSetEnabled = useCallback(
    (keys: string[], enabled: boolean) => {
      for (const key of keys) {
        const entry = entries.get(key);
        if (entry != null) ctx.set(`${entryPath(entry)}.disabled`, !enabled);
      }
    },
    [ctx, entries],
  );

  const menuProps = Menu.useContextMenu();
  const menuRenderProp = useCallback(
    ({ keys }: Menu.ContextMenuMenuProps) => {
      const entry = keys.length === 1 ? entries.get(keys[0]) : undefined;
      const disabled = keys
        .map((key) => entries.get(key))
        .filter((e) => e != null)
        .map((e) => ctx.get<boolean>(`${entryPath(e)}.disabled`).value);
      return (
        <Task.Views.ContextMenu
          keys={keys}
          onRemove={handleRemove}
          onDuplicate={handleDuplicate}
          onAddField={
            entry?.kind === "entry" && entry.type === "plain"
              ? () => handleAddField(entry.entryKey)
              : undefined
          }
          onEnable={
            disabled.includes(true) ? () => handleSetEnabled(keys, true) : undefined
          }
          onDisable={
            disabled.includes(false) ? () => handleSetEnabled(keys, false) : undefined
          }
        />
      );
    },
    [ctx, entries, handleRemove, handleDuplicate, handleAddField, handleSetEnabled],
  );

  const renderItem = useCallback(
    ({ key, ...p }: Tree.ItemRenderProps<string>) => {
      const entry = entries.get(p.itemKey);
      if (entry?.kind === "field")
        return <FieldTreeItem key={key} {...p} entryKey={entry.entryKey} />;
      return <EntryTreeItem key={key} {...p} onAddField={handleAddField} />;
    },
    [entries, handleAddField],
  );

  const resolve = useCallback(
    (entry: ReadEntry) => {
      if (entry.type !== "sparkplug") return {};
      const props = device.properties.read[sparkplugPropertiesKey(entry)];
      return { channel: props?.channels[""] ?? 0, index: props?.index ?? 0 };
    },
    [device],
  );

  const current = selected.length > 0 ? entries.get(selected[0]) : undefined;

  return (
    <Flex.Box x grow empty>
      {!isPreview && <Browser device={device} />}
      <Task.Views.Panes
        listTitle="Entries"
        list={
          <>
            <Menu.ContextMenu {...menuProps} menu={menuRenderProp}>
              <Tree.Tree<string, record.Keyed<string>>
                {...treeProps}
                {...dropProps}
                onSelect={handleSelect}
                itemHeight={TREE_ITEM_HEIGHT}
                className={menuProps.className}
                onContextMenu={menuProps.open}
                emptyContent={EMPTY_CONTENT}
                allowNone={false}
                autoSelectOnNone
              >
                {renderItem}
              </Tree.Tree>
            </Menu.ContextMenu>
            <AddEntryButtons
              onAdd={handleAddEntry}
              onAddSparkplug={handleAddSparkplugEntry}
            />
          </>
        }
        detailsPath={current != null ? entryPath(current) : null}
        title={
          current?.kind === "entry" ? (
            <EntryLabel path={entryPath(current)} />
          ) : current?.kind === "field" ? (
            <FieldTitle entryKey={current.entryKey} fieldKey={current.fieldKey} />
          ) : undefined
        }
      >
        {current == null ? (
          <Flex.Box y grow align="center" justify="center">
            <Text.Text status="disabled">
              Select an entry or field to configure
            </Text.Text>
          </Flex.Box>
        ) : current.kind === "entry" ? (
          <EntryPane path={entryPath(current)} deviceName={device.name} />
        ) : (
          <FieldPane
            entryKey={current.entryKey}
            fieldKey={current.fieldKey}
            deviceName={device.name}
          />
        )}
      </Task.Views.Panes>
      <Task.BindChannels<ReadEntry> path={ENTRIES_PATH} resolve={resolve} />
      {nodes.map(
        ({ key, children }) =>
          children != null && <FieldBinder key={key} device={device} entryKey={key} />,
      )}
    </Flex.Box>
  );
};

const Form = PlatformDevice.wrapTaskForm({
  use,
  useConfigure: useConnectModal,
  Content,
});

const getInitialValues: Task.GetInitialValues<ReadSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = READ_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: "MQTT read task", type: READ_TYPE, config: cfg };
};

/** @returns true when it stored new channels in the properties of the device. */
const configureSparkplugEntry = async (
  client: Client,
  dev: Device,
  entry: SparkplugReadEntry,
): Promise<boolean> => {
  const propertiesKey = sparkplugPropertiesKey(entry);
  const storedKey = dev.properties.read[propertiesKey]?.channels[""];
  for (const key of [entry.channel, storedKey]) {
    const ch = await Task.retrieveChannel(client, key);
    if (ch == null) continue;
    entry.channel = ch.key;
    entry.index = ch.index;
    return false;
  }
  const name = primitive.isNonZero(entry.name)
    ? entry.name
    : sparkplugChannelName(dev.name, entry);
  const ch = await Task.createChannel(client, name, entry.dataType);
  entry.channel = ch.key;
  entry.index = ch.index;
  dev.properties.read[propertiesKey] = { index: ch.index, channels: { "": ch.key } };
  return true;
};

const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (client, config) => {
  const dev = await client.devices.retrieve({ key: config.device, schemas: SCHEMAS });
  let modified = false;
  try {
    for (const entry of config.entries) {
      if (entry.disabled) continue;
      if (entry.type === "sparkplug") {
        if (await configureSparkplugEntry(client, dev, entry)) modified = true;
        continue;
      }
      dev.properties.read[entry.topic] ??= { index: 0, channels: {} };
      const changed = await Task.configureReadChannels({
        client,
        props: dev.properties.read[entry.topic],
        fields: entry.fields,
        namePrefix: namePrefix(dev.name, entry.topic),
        indexKey: entry.index,
      });
      modified ||= changed;
    }
  } finally {
    if (modified) await client.devices.create(dev, SCHEMAS);
  }
  return [config, dev.rack];
};

export const Read = Task.wrapForm({
  Properties,
  Form,
  schemas: READ_SCHEMAS,
  deployConfigZ: deployReadConfigZ,
  type: READ_TYPE,
  getInitialValues,
  onConfigure,
});

export const useCreateRead = Task.createUseCreate({
  getInitialValues,
});

export const ReadSelectable = Selector.createSelectable({
  type: READ_TYPE,
  title: "MQTT read task",
  icon: <Icon.Logo.MQTT />,
  useOnSelect: useCreateRead,
});
