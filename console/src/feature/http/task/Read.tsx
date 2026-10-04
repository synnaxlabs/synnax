// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/http/task/Form.css";

import { channel, http } from "@synnaxlabs/client";
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
import { DataType, id, type record } from "@synnaxlabs/x";
import { type FC, type MouseEvent, useCallback, useMemo, useState } from "react";

import { useFromConfig } from "@/feature/http/device/queries";
import { Select as SelectDevice } from "@/feature/http/device/Select";
import * as Device from "@/feature/http/device/types";
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
const defaultChannelName = (devName: string, epPath: string, pointer: string): string =>
  channel.escapeInvalidName(devName) + channel.escapeInvalidName(epPath + pointer);

const useDefaultChannelName = (epKey: string, pointer: string): string => {
  const epPath = PForm.useFieldValue<string>(`config.endpoints.${epKey}.path`);
  const dev = useFromConfig();
  return dev == null ? "" : defaultChannelName(dev.name, epPath, pointer);
};

interface EndpointTreeItemProps extends Tree.ItemRenderProps<string> {
  onAddField: (epKey: string) => void;
}

const EndpointTreeItem = ({ onAddField, ...props }: EndpointTreeItemProps) => {
  const { itemKey } = props;
  const isPreview = Task.useIsPreview();
  const handleAdd = useCallback(
    (e: MouseEvent) => {
      e.stopPropagation();
      onAddField(itemKey);
    },
    [onAddField, itemKey],
  );
  return (
    <Tree.Item {...props} className={CSS.B("endpoint-item")}>
      <EndpointLabel epKey={itemKey} />
      {!isPreview && (
        <Button.Button
          onClick={handleAdd}
          variant="text"
          size="tiny"
          tooltip="Add field"
          aria-label="Add field"
          tooltipLocation="right"
          className={CSS.BE("endpoint-item", "add")}
        >
          <Icon.Add />
        </Button.Button>
      )}
    </Tree.Item>
  );
};

interface FieldLabelProps extends Pick<Text.TextProps, "level"> {
  pointer: string;
}

/** Names a field by its pointer; an empty pointer reads as a new field. */
const FieldLabel = ({ pointer, level }: FieldLabelProps) => (
  <Text.Text
    level={level}
    weight={500}
    color={pointer === "" ? 8 : 10}
    overflow="ellipsis"
    className={CSS.B("field-pointer")}
  >
    {pointer === "" ? "New field" : pointer}
  </Text.Text>
);

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
      <FieldLabel pointer={pointer} level="small" />
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
    (p: Select.ButtonsProps<ReadMethod>) => (
      <Select.Buttons<ReadMethod> {...p} onChange={handleChange}>
        <Select.Item<ReadMethod> itemKey="GET">GET</Select.Item>
        <Select.Item<ReadMethod> itemKey="POST">POST</Select.Item>
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

/**
 * Binds an endpoint's fields to the channels the device stores for its path. Mounted by
 * the form for every endpoint, as the details pane renders only the selected one.
 */
const FieldBinder = ({ epKey }: { epKey: string }) => {
  const dev = useFromConfig();
  const epPath = PForm.useFieldValue<string>(`config.endpoints.${epKey}.path`);
  const indexKey = PForm.useFieldValue<string>(`config.endpoints.${epKey}.index`);
  const resolve = useCallback(
    (field: ReadField) => {
      if (dev == null) return null;
      const props = dev.properties.read[epPath];
      if (props == null) return { channel: 0 };
      return {
        channel:
          field.key === indexKey ? props.index : (props.channels[field.pointer] ?? 0),
      };
    },
    [dev, epPath, indexKey],
  );
  return (
    <Task.BindChannels<ReadField>
      path={`config.endpoints.${epKey}.fields`}
      resolve={resolve}
    />
  );
};

type TimingMode = "software" | "value";

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
        >
          <Select.Item<TimingMode> itemKey="software">Poll time</Select.Item>
          <Select.Item<TimingMode> itemKey="value">Response value</Select.Item>
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

const FieldTitle: FC<{ epKey: string; fieldKey: string }> = ({ epKey, fieldKey }) => (
  <FieldLabel
    pointer={PForm.useFieldValue<string>(
      `config.endpoints.${epKey}.fields.${fieldKey}.pointer`,
    )}
    level="p"
  />
);

const FieldPane: FC<{ epKey: string; fieldKey: string }> = ({ epKey, fieldKey }) => {
  const path = `config.endpoints.${epKey}.fields.${fieldKey}`;
  const { channel: fieldChannel, pointer } = PForm.useFieldValue<ReadField>(path);
  const defaultName = useDefaultChannelName(epKey, pointer);
  const bound = fieldChannel !== 0;
  return (
    <Flex.Box y grow empty className={CSS.B("endpoint-details")}>
      <PForm.Sections className={CSS.B("endpoint-form")}>
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
            defaultName={defaultName}
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
  // A fold that hides the selected field moves the selection up to its endpoint.
  const handleExpand = useCallback(
    ({ action, clicked }: Tree.HandleExpandProps<string>) => {
      if (action !== "contract") return;
      const hidesSelection = selected.some((k) => {
        const entry = entries.get(k);
        return entry?.kind === "field" && entry.epKey === clicked;
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
      if (clicked != null && entries.get(clicked)?.kind === "endpoint") expand(clicked);
    },
    [entries, expand],
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
        const copiedKeys = new Map(ep.fields.map((f) => [f.key, id.create()]));
        const copy: ReadEndpoint = {
          ...ep,
          key: id.create(),
          index: copiedKeys.get(ep.index) ?? ep.index,
          fields: ep.fields.map((f) => ({
            ...f,
            ...Task.READ_CHANNEL_OVERRIDE,
            key: copiedKeys.get(f.key) ?? id.create(),
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
        <Task.Views.ContextMenu
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
      return <EndpointTreeItem key={key} {...p} onAddField={handleAddField} />;
    },
    [entries, handleAddField],
  );

  const current = selected.length > 0 ? entries.get(selected[0]) : undefined;

  return (
    <>
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
                Add endpoint
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
      {nodes.map(({ key }) => (
        <FieldBinder key={key} epKey={key} />
      ))}
    </>
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
      const changed = await Task.configureReadChannels({
        client,
        props: dev.properties.read[ep.path],
        fields: ep.fields,
        namePrefix: `${safeDevName}${channel.escapeInvalidName(ep.path)}`,
        indexKey: ep.index,
        indexed: (f) => !isTimingField(f) && !new DataType(f.dataType).isVariable,
      });
      modified ||= changed;
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
