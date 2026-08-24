// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/http/task/Form.css";

import {
  channel,
  http,
  NotFoundError,
  type Synnax as Client,
} from "@synnaxlabs/client";
import {
  Button,
  Component,
  CSS as PCSS,
  Device as PDevice,
  Flex,
  Form as PForm,
  Icon,
  Input,
  Menu,
  Select,
  Telem,
  Text,
  Tree,
} from "@synnaxlabs/pluto";
import { DataType, errors, id, primitive, type record } from "@synnaxlabs/x";
import { type FC, type MouseEvent, useCallback, useMemo, useState } from "react";

import { Select as SelectDevice } from "@/feature/http/device/Select";
import * as Device from "@/feature/http/device/types";
import { ContextMenu } from "@/feature/http/task/ContextMenu";
import { EndpointLabel } from "@/feature/http/task/EndpointLabel";
import { TimeFormatField } from "@/feature/http/task/TimeFormatField";
import {
  deployReadConfigZ,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadEndpoint,
  type ReadField,
  type ReadMethod,
  type ReadSchemas,
} from "@/feature/http/task/types";
import { Button as PlatformButton } from "@/platform/button";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { Form as PlatformForm } from "@/platform/form";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const RATE_INPUT_PROPS = {
  endContent: "Hz",
  className: CSS.B("rate-input"),
} as const;

const Properties = () => (
  <>
    <SelectDevice />
    <Flex.Box x grow>
      <PForm.NumericField
        path="config.rate"
        label="Rate"
        inputProps={RATE_INPUT_PROPS}
      />
      <Task.Fields.DataSaving />
      <Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

const isTimingField = (f: ReadField): boolean => f.timeFormat != null;

type TreeEntry =
  | { kind: "endpoint"; epKey: string }
  | { kind: "field"; epKey: string; fieldKey: string };

interface TreeIndex {
  nodes: Tree.Node[];
  entries: Map<string, TreeEntry>;
}

/** Endpoints as tree nodes with their fields beneath; the timing field stays hidden. */
const useTreeIndex = (): TreeIndex => {
  // The form writes child paths into the same array, so the array's identity never
  // changes; the field state's does on every write beneath it.
  const state = PForm.useFieldState<ReadEndpoint[]>("config.endpoints");
  return useMemo(() => {
    const entries = new Map<string, TreeEntry>();
    const nodes = state.value.map((ep) => {
      entries.set(ep.key, { kind: "endpoint", epKey: ep.key });
      const children = ep.fields
        .filter((f) => !isTimingField(f))
        .map((f) => {
          entries.set(f.key, { kind: "field", epKey: ep.key, fieldKey: f.key });
          return { key: f.key };
        });
      return { key: ep.key, children };
    });
    return { nodes, entries };
  }, [state]);
};

const entryPath = (entry: TreeEntry): string =>
  entry.kind === "endpoint"
    ? `config.endpoints.${entry.epKey}`
    : `config.endpoints.${entry.epKey}.fields.${entry.fieldKey}`;

/** The name configure gives a field's channel when the field carries none. */
const useDefaultChannelName = (epKey: string, pointer: string): string => {
  const deviceKey = PForm.useFieldValue<string>("config.device");
  const epPath = PForm.useFieldValue<string>(`config.endpoints.${epKey}.path`);
  const dev = PDevice.useResult({ key: deviceKey }).data;
  if (dev == null) return "";
  return (
    channel.escapeInvalidName(dev.name) + channel.escapeInvalidName(epPath + pointer)
  );
};

const isCaretTarget = (target: EventTarget | null): boolean =>
  target instanceof Element &&
  target.closest(`.${PCSS.BE("tree", "expansion-indicator")}`) != null;

interface EndpointTreeItemProps extends Tree.ItemRenderProps<string> {
  onAddField: (epKey: string) => void;
  onToggle: (epKey: string) => void;
}

const EndpointTreeItem = ({
  onAddField,
  onToggle,
  ...props
}: EndpointTreeItemProps) => {
  const { itemKey } = props;
  const isPreview = Task.useIsPreview();
  const handleAdd = useCallback(
    (e: MouseEvent) => {
      e.stopPropagation();
      onAddField(itemKey);
    },
    [onAddField, itemKey],
  );
  // The caret folds the fields; the rest of the row opens the endpoint to edit it.
  const handleClickCapture = useCallback(
    (e: MouseEvent) => {
      if (!isCaretTarget(e.target)) return;
      e.stopPropagation();
      onToggle(itemKey);
    },
    [onToggle, itemKey],
  );
  return (
    <Tree.Item
      {...props}
      onClickCapture={handleClickCapture}
      className={CSS.B("endpoint-item")}
    >
      <EndpointLabel epKey={itemKey} />
      {!isPreview && (
        <Button.Button
          onClick={handleAdd}
          variant="text"
          size="tiny"
          tooltip="Add field"
          tooltipLocation="right"
          className={CSS.BE("endpoint-item", "add")}
        >
          <Icon.Add />
        </Button.Button>
      )}
    </Tree.Item>
  );
};

interface FieldTreeItemProps extends Tree.ItemRenderProps<string> {
  epKey: string;
}

const FieldTreeItem = ({ epKey, ...props }: FieldTreeItemProps) => {
  const { itemKey } = props;
  const path = `config.endpoints.${epKey}.fields.${itemKey}`;
  const { disabled, pointer } = PForm.useFieldValue<ReadField>(path);
  return (
    <Tree.Item
      {...props}
      className={CSS.cls(CSS.B("field-item"), disabled && CSS.M("off"))}
    >
      <Text.Text
        level="small"
        weight={500}
        color={pointer === "" ? 8 : 10}
        overflow="ellipsis"
        className={CSS.B("field-pointer")}
      >
        {pointer === "" ? "New field" : pointer}
      </Text.Text>
      <Task.EnabledCheckbox path={`${path}.disabled`} />
    </Tree.Item>
  );
};

const POINTER_INPUT_PROPS = { placeholder: "/temperature" } as const;

const HIDDEN_DATA_TYPES = [
  DataType.TIMESTAMP,
  DataType.UUID,
  DataType.JSON,
  DataType.BYTES,
];

const renderTelemSelectDataType = Component.renderProp(
  (p: Telem.SelectDataTypeProps) => (
    <Telem.SelectDataType {...p} hideDataTypes={HIDDEN_DATA_TYPES} location="bottom" />
  ),
);

const METHOD_KEYS: ReadMethod[] = ["GET", "POST"];

const MethodSelect: FC<{ path: string; epPath: string }> = ({ path, epPath }) => {
  const { set } = PForm.useContext();
  const handleChange = useCallback(
    (method: ReadMethod) => {
      set(path, method);
      // GET carries no request body. Clearing it here keeps a body typed under POST
      // from staying in the saved config, where it would set a content type on a
      // request that sends nothing.
      if (method !== "POST") set(`${epPath}.body`, "");
    },
    [set, path, epPath],
  );
  const renderMethod = useCallback(
    (p: Omit<Select.ButtonsProps<ReadMethod>, "keys">) => (
      <Select.Buttons<ReadMethod> {...p} onChange={handleChange} keys={METHOD_KEYS}>
        <Select.Button<ReadMethod> itemKey="GET">GET</Select.Button>
        <Select.Button<ReadMethod> itemKey="POST">POST</Select.Button>
      </Select.Buttons>
    ),
    [handleChange],
  );
  return (
    <PForm.Field<ReadMethod> path={path} label="Method">
      {renderMethod}
    </PForm.Field>
  );
};

const FieldDetails: FC<{ epKey: string; fieldKey: string }> = ({ epKey, fieldKey }) => {
  const path = `config.endpoints.${epKey}.fields.${fieldKey}`;
  const { channel: fieldChannel, pointer } = PForm.useFieldValue<ReadField>(path);
  const defaultName = useDefaultChannelName(epKey, pointer);
  const bound = fieldChannel !== 0;
  return (
    <>
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
        <PForm.TextField
          path={`${path}.name`}
          label="Channel"
          padHelpText={false}
          inputProps={{
            placeholder: defaultName === "" ? "Channel name" : defaultName,
          }}
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
    </>
  );
};

type TimingMode = "software" | "value";
const TIMING_MODE_KEYS: TimingMode[] = ["software", "value"];

const TimestampFields: FC<{ path: string }> = ({ path }) => {
  const fields = PForm.useFieldValue<ReadField[]>(`${path}.fields`);
  const { set } = PForm.useContext();
  const indexField = fields.find(isTimingField);
  const isValueTiming = indexField != null;

  const handleChange = useCallback(
    (mode: TimingMode) => {
      if (mode === "value" && !isValueTiming) {
        const indexF: ReadField = {
          ...http.readFieldZ.parse({}),
          key: id.create(),
          timeFormat: "unix_sec",
        };
        set(`${path}.fields`, [...fields, indexF]);
        set(`${path}.index`, indexF.key);
      } else if (mode === "software" && isValueTiming) {
        set(
          `${path}.fields`,
          fields.filter((f) => !isTimingField(f)),
        );
        set(`${path}.index`, "");
      }
    },
    [fields, isValueTiming, path, set],
  );

  return (
    <>
      <Input.Item label="Source" padHelpText={false}>
        <Select.Buttons<TimingMode>
          value={isValueTiming ? "value" : "software"}
          onChange={handleChange}
          keys={TIMING_MODE_KEYS}
        >
          <Select.Button<TimingMode> itemKey="software">Poll time</Select.Button>
          <Select.Button<TimingMode> itemKey="value">Response value</Select.Button>
        </Select.Buttons>
      </Input.Item>
      {isValueTiming && indexField != null && (
        <>
          <PForm.TextField
            path={`${path}.fields.${indexField.key}.pointer`}
            label="Pointer"
            padHelpText={false}
            inputProps={TIMESTAMP_POINTER_INPUT_PROPS}
          />
          <TimeFormatField
            path={`${path}.fields.${indexField.key}.timeFormat`}
            label="Format"
          />
        </>
      )}
    </>
  );
};

const TIMESTAMP_POINTER_INPUT_PROPS = { placeholder: "/timestamp" } as const;

const EndpointDetails: FC<{ epKey: string }> = ({ epKey }) => {
  const path = `config.endpoints.${epKey}`;
  const method = PForm.useFieldValue<string>(`${path}.method`);
  return (
    <Flex.Box y grow empty className={CSS.B("endpoint-details")}>
      <PForm.Sections className={CSS.B("endpoint-form")}>
        <PForm.Section title="Request">
          <MethodSelect path={`${path}.method`} epPath={path} />
          <PForm.TextField
            path={`${path}.path`}
            label="Path"
            padHelpText={false}
            inputProps={PATH_INPUT_PROPS}
          />
          {method === "POST" && (
            <PForm.TextField
              path={`${path}.body`}
              label="Body"
              padHelpText={false}
              inputProps={REQUEST_BODY_INPUT_PROPS}
            />
          )}
          <PlatformForm.KeyValueEditor
            path={`${path}.queryParams`}
            label="Query parameters"
            keyField="parameter"
            keyPlaceholder="limit"
            valuePlaceholder="100"
          />
          <PlatformForm.KeyValueEditor
            path={`${path}.headers`}
            label="Headers"
            keyField="name"
            keyPlaceholder="Content-Type"
            valuePlaceholder="application/json"
          />
        </PForm.Section>
        <PForm.Section title="Timestamp">
          <TimestampFields path={path} />
        </PForm.Section>
      </PForm.Sections>
    </Flex.Box>
  );
};

const FieldTitle: FC<{ epKey: string; fieldKey: string }> = ({ epKey, fieldKey }) => {
  const pointer = PForm.useFieldValue<string>(
    `config.endpoints.${epKey}.fields.${fieldKey}.pointer`,
  );
  return (
    <Text.Text
      level="p"
      weight={500}
      color={pointer === "" ? 8 : 10}
      overflow="ellipsis"
      className={CSS.B("field-pointer")}
    >
      {pointer === "" ? "New field" : pointer}
    </Text.Text>
  );
};

const FieldPane: FC<{ epKey: string; fieldKey: string }> = ({ epKey, fieldKey }) => (
  <Flex.Box y grow empty className={CSS.B("endpoint-details")}>
    <PForm.Sections className={CSS.B("endpoint-form")}>
      <FieldDetails epKey={epKey} fieldKey={fieldKey} />
    </PForm.Sections>
  </Flex.Box>
);

const PATH_INPUT_PROPS = { placeholder: "/api/data" } as const;

const REQUEST_BODY_INPUT_PROPS = {
  placeholder: '{"query": "latest"}',
  area: true,
  className: CSS.B("request-body"),
} as const;

const newField = (last?: ReadField): ReadField => ({
  ...(last != null
    ? { ...last, ...Task.READ_CHANNEL_OVERRIDE }
    : http.readFieldZ.parse({})),
  key: id.create(),
});

const TREE_ITEM_HEIGHT = 36;

const EMPTY_CONTENT = <Empty.Action message="No endpoints" />;

const Form: FC = () => {
  const [selected, setSelected] = useState<string[]>([]);
  const { nodes, entries } = useTreeIndex();
  const ctx = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const [initialExpanded] = useState(() => nodes.map(({ key }) => key));
  const treeProps = Tree.use({
    nodes,
    selected,
    onSelectedChange: setSelected,
    initialExpanded,
  });
  const { expand, contract, expanded, shape } = treeProps;

  const handleSelect = useCallback<
    Tree.TreeProps<string, record.Keyed<string>>["onSelect"]
  >(
    (keys, { clicked }) => {
      setSelected(keys);
      if (clicked != null && entries.get(clicked)?.kind === "endpoint") expand(clicked);
    },
    [entries, expand],
  );

  const handleToggle = useCallback(
    (epKey: string) => {
      if (expanded.includes(epKey)) contract(epKey);
      else expand(epKey);
    },
    [expanded, expand, contract],
  );

  const handleAddEndpoint = useCallback(() => {
    const ep: ReadEndpoint = { ...http.readEndpointZ.parse({}), key: id.create() };
    const endpoints = ctx.get<ReadEndpoint[]>("config.endpoints").value;
    ctx.set("config.endpoints", [...endpoints, ep]);
    setSelected([ep.key]);
    expand(ep.key);
  }, [ctx, expand]);

  const handleAddField = useCallback(
    (epKey: string) => {
      const path = `config.endpoints.${epKey}.fields`;
      const fields = ctx.get<ReadField[]>(path).value;
      const shown = fields.filter((f) => !isTimingField(f));
      const field = newField(shown[shown.length - 1]);
      ctx.set(path, [...fields, field]);
      setSelected([field.key]);
      expand(epKey);
    },
    [ctx, expand],
  );

  const handleRemove = useCallback(
    (keys: string[]) => {
      const removed = new Set(keys);
      const endpoints = ctx.get<ReadEndpoint[]>("config.endpoints").value;
      ctx.set(
        "config.endpoints",
        endpoints
          .filter((ep) => !removed.has(ep.key))
          .map((ep) => ({
            ...ep,
            fields: ep.fields.filter((f) => !removed.has(f.key)),
          })),
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
      const endpoints = ctx.get<ReadEndpoint[]>("config.endpoints").value;
      const next: ReadEndpoint[] = [];
      let first: string | null = null;
      for (const ep of endpoints) {
        const fields: ReadField[] = [];
        for (const f of ep.fields) {
          fields.push(f);
          if (!chosen.has(f.key)) continue;
          const copy = newField(f);
          first ??= copy.key;
          fields.push(copy);
        }
        next.push({ ...ep, fields });
        if (!chosen.has(ep.key)) continue;
        const copy: ReadEndpoint = {
          ...ep,
          key: id.create(),
          fields: ep.fields.map((f) => ({
            ...f,
            ...Task.READ_CHANNEL_OVERRIDE,
            key: id.create(),
          })),
        };
        first ??= copy.key;
        next.push(copy);
        expand(copy.key);
      }
      ctx.set("config.endpoints", next);
      if (first != null) setSelected([first]);
    },
    [ctx, expand],
  );

  // Only fields carry a disabled flag; an endpoint is dropped, never switched off.
  const handleSetEnabled = useCallback(
    (keys: string[], enabled: boolean) => {
      for (const key of keys) {
        const entry = entries.get(key);
        if (entry?.kind === "field") ctx.set(`${entryPath(entry)}.disabled`, !enabled);
      }
    },
    [ctx, entries],
  );

  const menuProps = Menu.useContextMenu();
  const menuRenderProp = useCallback(
    ({ keys }: Menu.ContextMenuMenuProps) => {
      const entry = keys.length === 1 ? entries.get(keys[0]) : undefined;
      const fields = keys
        .map((key) => entries.get(key))
        .filter((e) => e?.kind === "field");
      const disabledOf = (e: TreeEntry) =>
        ctx.get<boolean>(`${entryPath(e)}.disabled`).value;
      return (
        <ContextMenu
          keys={keys}
          onRemove={handleRemove}
          onDuplicate={handleDuplicate}
          onAddField={
            entry?.kind === "endpoint" ? () => handleAddField(entry.epKey) : undefined
          }
          onEnable={
            fields.some(disabledOf) ? () => handleSetEnabled(keys, true) : undefined
          }
          onDisable={
            fields.some((e) => !disabledOf(e))
              ? () => handleSetEnabled(keys, false)
              : undefined
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
        return <FieldTreeItem key={key} {...p} epKey={entry.epKey} />;
      return (
        <EndpointTreeItem
          key={key}
          {...p}
          onAddField={handleAddField}
          onToggle={handleToggle}
        />
      );
    },
    [entries, handleAddField, handleToggle],
  );

  const current = selected.length > 0 ? entries.get(selected[0]) : undefined;

  return (
    <Task.Views.Panes
      listTitle="Endpoints"
      list={
        <>
          <Menu.ContextMenu {...menuProps} menu={menuRenderProp}>
            <Tree.Tree<string, record.Keyed<string>>
              {...treeProps}
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
          {!isPreview && (
            <PlatformButton.CreateListItem size="small" onClick={handleAddEndpoint}>
              New endpoint
            </PlatformButton.CreateListItem>
          )}
        </>
      }
      detailsPath={current != null ? entryPath(current) : null}
      title={
        current?.kind === "endpoint" ? (
          <EndpointLabel epKey={current.epKey} />
        ) : current?.kind === "field" ? (
          <FieldTitle epKey={current.epKey} fieldKey={current.fieldKey} />
        ) : undefined
      }
    >
      {current == null ? (
        <Flex.Box y grow align="center" justify="center">
          <Text.Text status="disabled">
            Select an endpoint or field to configure
          </Text.Text>
        </Flex.Box>
      ) : current.kind === "endpoint" ? (
        <EndpointDetails epKey={current.epKey} />
      ) : (
        <FieldPane epKey={current.epKey} fieldKey={current.fieldKey} />
      )}
    </Task.Views.Panes>
  );
};

const getInitialValues: Task.GetInitialValues<ReadSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = READ_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: "HTTP read task", type: READ_TYPE, config: cfg };
};

const retrieveChannel = async (
  client: Client,
  key: channel.Key,
): Promise<channel.Channel | null> => {
  try {
    return await client.channels.retrieve(key);
  } catch (e) {
    if (NotFoundError.matches(e)) return null;
    throw errors.fromUnknown(e);
  }
};

const channelExists = async (client: Client, key: channel.Key): Promise<boolean> =>
  (await retrieveChannel(client, key)) != null;

const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (client, config) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: Device.SCHEMAS,
  });
  const safeDevName = channel.escapeInvalidName(dev.name);
  let modified = false;
  try {
    for (const ep of config.endpoints) {
      dev.properties.read[ep.path] ??= { index: 0, channels: {} };
      const epProps = dev.properties.read[ep.path];

      const needsIndex = ep.fields.some(
        (f) => !isTimingField(f) && !new DataType(f.dataType).isVariable,
      );

      if (needsIndex) {
        let shouldCreateIndex = !primitive.isNonZero(epProps.index);
        shouldCreateIndex ||= !(await channelExists(client, epProps.index));
        if (shouldCreateIndex) {
          // check if any existing data channels share an index we can reuse
          let recoveredIndex = 0;
          for (const storedKey of Object.values(epProps.channels)) {
            if (!primitive.isNonZero(storedKey)) continue;
            const ch = await retrieveChannel(client, storedKey);
            if (ch != null && primitive.isNonZero(ch.index)) {
              const indexCh = await retrieveChannel(client, ch.index);
              if (indexCh != null) {
                recoveredIndex = ch.index;
                break;
              }
            }
          }
          if (primitive.isNonZero(recoveredIndex)) {
            epProps.index = recoveredIndex;
            modified = true;
          } else {
            modified = true;
            const newIndexCh = await client.channels.create({
              name: `${safeDevName}${channel.escapeInvalidName(ep.path)}_time`,
              dataType: "timestamp",
              isIndex: true,
            });
            epProps.index = newIndexCh.key;
          }
        }
      }

      const potentialTimingKey = ep.index;
      for (const field of ep.fields) {
        if (field.key === potentialTimingKey && epProps.index !== 0) {
          field.channel = epProps.index;
          continue;
        }

        if (field.channel !== 0 && (await channelExists(client, field.channel)))
          continue;

        const storedKey = epProps.channels[field.pointer];
        if (
          primitive.isNonZero(storedKey) &&
          (await channelExists(client, storedKey))
        ) {
          field.channel = storedKey;
          continue;
        }

        const dt = new DataType(field.dataType);
        const chName = primitive.isNonZero(field.name)
          ? field.name
          : `${safeDevName}${channel.escapeInvalidName(ep.path + field.pointer)}`;
        const newCh = await client.channels.create({
          name: chName,
          dataType: field.dataType,
          ...(dt.isVariable ? { virtual: true } : { index: epProps.index }),
        });
        modified = true;
        field.channel = newCh.key;
        epProps.channels[field.pointer] = newCh.key;
      }
    }
  } finally {
    if (modified) await client.devices.create(dev, Device.SCHEMAS);
  }
  return [config, dev.rack];
};

export const Read = Task.wrapForm({
  Properties,
  Form,
  schemas: READ_SCHEMAS,
  deployConfigZ: deployReadConfigZ,
  type: "http_read",
  getInitialValues,
  onConfigure,
});

export const useCreateRead = Task.createUseCreate({
  getInitialValues,
});

export const ReadSelectable = Selector.createSelectable({
  type: READ_TYPE,
  title: "HTTP read task",
  icon: <Icon.Logo.HTTP />,
  useOnSelect: useCreateRead,
});
