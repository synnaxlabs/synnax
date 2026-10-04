// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/http/task/Form.css";

import { channel } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Channel as PChannel, Telem } from "@synnaxlabs/pluto";
import { DataType, id, json, primitive } from "@synnaxlabs/x";
import { type FC, useCallback, useMemo, useState } from "react";

import { useFromConfig } from "@/feature/http/device/queries";
import { Select as SelectDevice } from "@/feature/http/device/Select";
import * as Device from "@/feature/http/device/types";
import { EndpointLabel } from "@/feature/http/task/EndpointLabel";
import { TimeFormatField } from "@/feature/http/task/TimeFormatField";
import {
  deployWriteConfigZ,
  type GeneratorType,
  type TimeFormat,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteEndpoint,
  writeEndpointZ,
  type WriteField,
  type WriteMethod,
  type WriteSchemas,
} from "@/feature/http/task/types";
import { Button as PlatformButton } from "@/platform/button";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { Form as PlatformForm } from "@/platform/form";
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

const JSON_TYPE_ITEMS = (
  <>
    <Select.Item itemKey="number">Number</Select.Item>
    <Select.Item itemKey="string">String</Select.Item>
    <Select.Item itemKey="boolean">Boolean</Select.Item>
  </>
);

const MethodSelect: FC<{ path: string }> = ({ path }) => (
  <PForm.Field<WriteMethod> path={path} label="Method">
    {renderMethodSelect}
  </PForm.Field>
);

const renderMethodSelect = Component.renderProp(
  (p: Select.ButtonsProps<WriteMethod>) => (
    <Select.Buttons<WriteMethod> {...p}>
      <Select.Item<WriteMethod> itemKey="POST">POST</Select.Item>
      <Select.Item<WriteMethod> itemKey="PUT">PUT</Select.Item>
      <Select.Item<WriteMethod> itemKey="PATCH">PATCH</Select.Item>
    </Select.Buttons>
  ),
);

const getEndpointChannelNameID = (epKey: string) => `write-ep-ch-${epKey}`;

const EndpointItem = (props: List.ItemProps<string>) => {
  const { itemKey } = props;
  const channel = PForm.useFieldValue<number>(
    `config.endpoints.${itemKey}.channel.channel`,
  );
  return (
    <Select.Item
      {...props}
      y
      align="start"
      gap="small"
      className={CSS.B("endpoint-item")}
    >
      <Flex.Box x align="center" gap="small" full="x">
        <EndpointLabel epKey={itemKey} />
      </Flex.Box>
      <Task.ChannelName
        channel={channel}
        namePath={`config.endpoints.${itemKey}.channel.name`}
        id={getEndpointChannelNameID(itemKey)}
        level="small"
        weight={450}
        color={9}
        overflow="ellipsis"
      />
    </Select.Item>
  );
};

const endpointItem = Component.renderProp(EndpointItem);

const EMPTY_ENDPOINTS = <Empty.Action message="No endpoints" />;

const EnumValuesEditor: FC<{ channelPath: string }> = ({ channelPath }) => (
  <PForm.Section title="Enum mapping">
    <PlatformForm.KeyValueEditor
      path={`${channelPath}.enumValues`}
      keyField="label"
      keyPlaceholder="String (e.g. ON)"
      valueType="number"
      valueFirst
    />
  </PForm.Section>
);

const ChannelFieldSection: FC<{ epPath: string }> = ({ epPath }) => {
  const channelPath = `${epPath}.channel`;
  const channelKey = PForm.useFieldValue<number>(`${channelPath}.channel`);
  const jsonType = PForm.useFieldValue<string>(`${channelPath}.jsonType`);
  const channelQuery = useMemo(
    () => (primitive.isNonZero(channelKey) ? { key: channelKey } : null),
    [channelKey],
  );
  const { data: dataType } = PChannel.useResultDataType(channelQuery);

  return (
    <>
      <PForm.Section title="Channel">
        <PForm.TextField
          path={`${channelPath}.pointer`}
          label="Pointer"
          padHelpText={false}
          inputProps={JSON_POINTER_INPUT_PROPS}
        />
        <PForm.Field<string>
          path={`${channelPath}.jsonType`}
          label="JSON type"
          padHelpText={false}
        >
          {renderSelectJSONType}
        </PForm.Field>
        {channelKey === 0 && (
          <PForm.Field<string>
            path={`${channelPath}.dataType`}
            label="Data type"
            padHelpText={false}
          >
            {renderSelectDataType}
          </PForm.Field>
        )}
        {dataType != null && DataType.TIMESTAMP.equals(dataType) && (
          <TimeFormatField path={`${channelPath}.timeFormat`} label="Time format" />
        )}
      </PForm.Section>
      {jsonType === "string" && <EnumValuesEditor channelPath={channelPath} />}
    </>
  );
};

const renderSelectJSONType = Component.renderProp(
  (p: Omit<Select.SingleSimpleProps<string>, "children" | "resourceName">) => (
    <Select.Simple<string> {...p} resourceName="JSON type">
      {JSON_TYPE_ITEMS}
    </Select.Simple>
  ),
);

const JSON_POINTER_INPUT_PROPS = { placeholder: "/value" } as const;

const renderSelectDataType = Component.renderProp((p: Telem.SelectDataTypeProps) => (
  <Telem.SelectDataType
    {...p}
    hideDataTypes={HIDDEN_DATA_TYPES}
    hideVariableDensity
    location="bottom"
  />
));

const HIDDEN_DATA_TYPES = [
  DataType.TIMESTAMP,
  DataType.JSON,
  DataType.BYTES,
  DataType.UUID,
];

const generatorDisplayKey = (
  generator: GeneratorType | null | undefined,
  timeFormat: TimeFormat | null | undefined,
): GeneratorType | TimeFormat => {
  if (generator === "timestamp") return timeFormat ?? "iso8601";
  return "uuid";
};

const FieldListItem = ({
  epKey,
  ...props
}: List.ItemProps<string> & { epKey: string }) => {
  const { itemKey } = props;
  const path = `config.endpoints.${epKey}.fields.${itemKey}`;
  const fieldType = PForm.useFieldValue<string>(`${path}.type`);
  const jsonType = PForm.useFieldValue<json.PrimitiveType | undefined>(
    `${path}.jsonType`,
    { optional: true },
  );
  const generator = PForm.useFieldValue<GeneratorType | undefined>(
    `${path}.generator`,
    { optional: true },
  );
  const timeFormat = PForm.useFieldValue<TimeFormat | undefined>(`${path}.timeFormat`, {
    optional: true,
  });
  const { set } = PForm.useContext();

  const handleJSONTypeChange = useCallback(
    (value: json.PrimitiveType) => {
      set(`${path}.jsonType`, value);
      set(`${path}.value`, json.ZERO_PRIMITIVES[value]);
    },
    [set, path],
  );

  const handleGeneratorChange = useCallback(
    (key: GeneratorType | TimeFormat) => {
      if (key === "uuid") {
        set(`${path}.generator`, "uuid");
        set(`${path}.timeFormat`, undefined);
      } else {
        set(`${path}.generator`, "timestamp");
        set(`${path}.timeFormat`, key);
      }
    },
    [set, path],
  );

  return (
    <Select.Item {...props} justify="between" align="center" x>
      <PForm.TextField
        path={`${path}.pointer`}
        showLabel={false}
        showHelpText={false}
        inputProps={POINTER_INPUT_PROPS}
      />
      {fieldType === "static" && (
        <Select.Simple<json.PrimitiveType>
          value={jsonType ?? "string"}
          onChange={handleJSONTypeChange}
          resourceName="type"
        >
          {JSON_TYPE_ITEMS}
        </Select.Simple>
      )}
      {fieldType === "static" && jsonType === "string" && (
        <PForm.TextField
          path={`${path}.value`}
          showLabel={false}
          showHelpText={false}
          inputProps={STRING_INPUT_PROPS}
        />
      )}
      {fieldType === "static" && jsonType === "number" && (
        <PForm.NumericField
          path={`${path}.value`}
          showLabel={false}
          showHelpText={false}
          inputProps={NUMBER_INPUT_PROPS}
        />
      )}
      {fieldType === "static" && jsonType === "boolean" && (
        <PForm.SwitchField
          path={`${path}.value`}
          showLabel={false}
          showHelpText={false}
        />
      )}
      {fieldType === "generated" && (
        <Select.Simple<GeneratorType | TimeFormat>
          value={generatorDisplayKey(generator, timeFormat)}
          onChange={handleGeneratorChange}
          resourceName="generator"
          variant="floating"
        >
          <Select.Item itemKey="uuid">UUID</Select.Item>
          <Select.Item itemKey="iso8601">Timestamp (ISO 8601)</Select.Item>
          <Select.Item itemKey="unix_sec">Timestamp (s)</Select.Item>
          <Select.Item itemKey="unix_ms">Timestamp (ms)</Select.Item>
          <Select.Item itemKey="unix_us">Timestamp (µs)</Select.Item>
          <Select.Item itemKey="unix_ns">Timestamp (ns)</Select.Item>
        </Select.Simple>
      )}
      <Text.Text level="small" color={9}>
        {fieldType}
      </Text.Text>
    </Select.Item>
  );
};

// The section layout dissolves each field's own box, so sizing lives on the inputs.
const POINTER_INPUT_PROPS = { placeholder: "field", grow: true } as const;

const STRING_INPUT_PROPS = {
  placeholder: "value",
  className: CSS.B("static-field-value"),
} as const;

const NUMBER_INPUT_PROPS = { className: CSS.B("static-field-value") } as const;

const AdditionalFields: FC<{ epKey: string }> = ({ epKey }) => {
  const path = `config.endpoints.${epKey}.fields`;
  const { data, push, remove } = PForm.useFieldList<string, WriteField>(path);
  const [selected, setSelected] = useState<string[]>([]);
  const isPreview = Task.useIsPreview();

  const handleAddStatic = useCallback(() => {
    const field: WriteField = {
      key: id.create(),
      pointer: "",
      jsonType: "string",
      type: "static",
      value: "",
    };
    push(field);
    setSelected([field.key]);
  }, [push]);

  const handleAddGenerated = useCallback(() => {
    const field: WriteField = {
      key: id.create(),
      pointer: "",
      type: "generated",
      generator: "uuid",
    };
    push(field);
    setSelected([field.key]);
  }, [push]);

  const handleRemove = useCallback(
    (keys: string[]) => {
      remove(keys);
      setSelected([]);
    },
    [remove],
  );

  const listItem = useCallback(
    ({ key, ...p }: List.ItemProps<string>) => (
      <FieldListItem {...p} key={key} epKey={epKey} />
    ),
    [epKey],
  );

  const menuProps = Menu.useContextMenu();
  const menuRenderProp = useCallback(
    (p: Menu.ContextMenuMenuProps) => (
      <Task.Views.ContextMenu keys={p.keys} onRemove={handleRemove} />
    ),
    [handleRemove],
  );

  const actions = !isPreview && (
    <>
      <Button.Button
        onClick={handleAddStatic}
        variant="text"
        size="small"
        tooltip="Add static field"
      >
        <Icon.Add />
        Static
      </Button.Button>
      <Button.Button
        onClick={handleAddGenerated}
        variant="text"
        size="small"
        tooltip="Add generated field"
      >
        <Icon.Time />
        Generated
      </Button.Button>
    </>
  );

  return (
    <PForm.Section title="Additional fields" actions={actions}>
      <Menu.ContextMenu {...menuProps} menu={menuRenderProp}>
        <Select.Frame<string, WriteField>
          multiple
          data={data}
          value={selected}
          onChange={setSelected}
          replaceOnSingle
          allowNone={false}
          autoSelectOnNone
        >
          <List.Scroll className={menuProps.className} onContextMenu={menuProps.open}>
            <List.Items<string, WriteField> emptyContent={EMPTY_CONTENT}>
              {listItem}
            </List.Items>
          </List.Scroll>
        </Select.Frame>
      </Menu.ContextMenu>
    </PForm.Section>
  );
};

const EMPTY_CONTENT = <Empty.Action message="No additional fields" />;

const EndpointDetails: FC<{ epKey: string }> = ({ epKey }) => {
  const path = `config.endpoints.${epKey}`;
  return (
    <Flex.Box y grow empty className={CSS.B("endpoint-details")}>
      <PForm.Sections className={CSS.B("endpoint-form")}>
        <PForm.Section title="Request">
          <MethodSelect path={`${path}.method`} />
          <PForm.TextField
            path={`${path}.path`}
            label="Path"
            padHelpText={false}
            inputProps={PATH_INPUT_PROPS}
          />
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
        <ChannelFieldSection epPath={path} />
        <AdditionalFields key={epKey} epKey={epKey} />
      </PForm.Sections>
    </Flex.Box>
  );
};

const PATH_INPUT_PROPS = { placeholder: "/api/control" } as const;

const Form: FC = () => {
  const [selectedEndpoints, setSelectedEndpoints] = useState<string[]>([]);
  const { data, push, remove } = PForm.useFieldList<string, WriteEndpoint>(
    "config.endpoints",
  );
  const dev = useFromConfig();
  const resolve = useCallback(
    (ep: WriteEndpoint) =>
      dev == null
        ? null
        : { channel: { ...ep.channel, channel: dev.properties.write[ep.path] ?? 0 } },
    [dev],
  );
  const ctx = PForm.useContext();
  const isPreview = Task.useIsPreview();

  const handleAddEndpoint = useCallback(() => {
    const ep: WriteEndpoint = { ...writeEndpointZ.parse({}), key: id.create() };
    push(ep);
    setSelectedEndpoints([ep.key]);
  }, [push]);

  const handleRemoveEndpoints = useCallback(
    (keys: string[]) => {
      remove(keys);
      setSelectedEndpoints([]);
    },
    [remove],
  );

  const handleDuplicateEndpoints = useCallback(
    (keys: string[]) => {
      const allEndpoints = ctx.get<WriteEndpoint[]>("config.endpoints").value;
      const duplicated = allEndpoints
        .filter(({ key }) => keys.includes(key))
        .map((ep) => ({
          ...ep,
          key: id.create(),
          channel: { ...ep.channel, channel: 0 },
          fields: ep.fields.map((f) => ({ ...f, key: id.create() })),
        }));
      push(duplicated);
      if (duplicated.length > 0) setSelectedEndpoints([duplicated[0].key]);
    },
    [ctx, push],
  );

  const handleRenameChannel = useCallback(
    (key: string) => Text.edit(getEndpointChannelNameID(key)),
    [],
  );

  const menuProps = Menu.useContextMenu();
  const menuRenderProp = useCallback(
    (p: Menu.ContextMenuMenuProps) => (
      <Task.Views.ContextMenu
        keys={p.keys}
        onRemove={handleRemoveEndpoints}
        onDuplicate={handleDuplicateEndpoints}
        onRename={handleRenameChannel}
      />
    ),
    [handleRemoveEndpoints, handleDuplicateEndpoints, handleRenameChannel],
  );

  const selected = selectedEndpoints.length > 0 ? selectedEndpoints[0] : null;
  return (
    <>
      <Task.Views.Panes
        listTitle="Endpoints"
        list={
          <>
            <Menu.ContextMenu {...menuProps} menu={menuRenderProp}>
              <Select.Frame<string, WriteEndpoint>
                multiple
                data={data}
                value={selectedEndpoints}
                onChange={setSelectedEndpoints}
                replaceOnSingle
                allowNone={false}
                autoSelectOnNone
              >
                <List.Scroll
                  full="y"
                  className={menuProps.className}
                  onContextMenu={menuProps.open}
                >
                  <List.Items<string, WriteEndpoint> emptyContent={EMPTY_ENDPOINTS}>
                    {endpointItem}
                  </List.Items>
                </List.Scroll>
              </Select.Frame>
            </Menu.ContextMenu>
            {!isPreview && (
              <PlatformButton.CreateListItem size="small" onClick={handleAddEndpoint}>
                Add endpoint
              </PlatformButton.CreateListItem>
            )}
          </>
        }
        detailsPath={selected != null ? `config.endpoints.${selected}` : null}
        title={selected != null && <EndpointLabel epKey={selected} />}
      >
        {selected != null ? (
          <EndpointDetails epKey={selected} />
        ) : (
          <Flex.Box y grow align="center" justify="center">
            <Text.Text status="disabled">Select an endpoint to configure</Text.Text>
          </Flex.Box>
        )}
      </Task.Views.Panes>
      <Task.BindChannels<WriteEndpoint> path="config.endpoints" resolve={resolve} />
    </>
  );
};

const getInitialValues: Task.GetInitialValues<WriteSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = WRITE_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: "HTTP write task", type: WRITE_TYPE, config: cfg };
};

const onConfigure: Task.OnConfigure<WriteSchemas["config"]> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: Device.SCHEMAS,
  });
  const safeDevName = channel.escapeInvalidName(dev.name);
  let modified = false;
  try {
    for (const ep of config.endpoints) {
      let changed: boolean;
      [ep.channel.channel, changed] = await Task.configureCommandChannel(
        client,
        dev.properties.write,
        {
          propertiesKey: ep.path,
          channel: ep.channel.channel,
          name: primitive.isNonZero(ep.channel.name)
            ? ep.channel.name
            : `${safeDevName}${channel.escapeInvalidName(ep.path)}_cmd`,
          dataType: ep.channel.dataType,
        },
      );
      modified ||= changed;
    }
  } finally {
    if (modified) await client.devices.create(dev, Device.SCHEMAS);
  }
  return [config, dev.rack];
};

export const Write = Task.wrapForm({
  Properties,
  Form,
  schemas: WRITE_SCHEMAS,
  deployConfigZ: deployWriteConfigZ,
  type: WRITE_TYPE,
  getInitialValues,
  onConfigure,
});

export const useCreateWrite = Task.createUseCreate({
  getInitialValues,
});

export const WriteSelectable = Selector.createSelectable({
  type: WRITE_TYPE,
  title: "HTTP write task",
  icon: <Icon.Logo.HTTP />,
  useOnSelect: useCreateWrite,
});
