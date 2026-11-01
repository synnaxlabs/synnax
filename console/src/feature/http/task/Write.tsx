// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/http/task/Form.css";

import { channel, type http, type Synnax as Client } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Channel as PChannel, Telem } from "@synnaxlabs/pluto";
import { DataType, deep, id, json, primitive } from "@synnaxlabs/x";
import { type FC, useState } from "react";

import { useFromConfig } from "@/feature/http/device/queries";
import { Select as SelectDevice } from "@/feature/http/device/Select";
import { MAKE, SCHEMAS } from "@/feature/http/device/types";
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
  type WriteSchemas,
} from "@/feature/http/task/types";
import { Button as PlatformButton } from "@/platform/button";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { Form as PlatformForm } from "@/platform/form";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<WriteSchemas>>();

const Properties = () => <SelectDevice />;

const JSON_TYPE_ITEMS = (
  <>
    <Select.Item itemKey="number">Number</Select.Item>
    <Select.Item itemKey="string">String</Select.Item>
    <Select.Item itemKey="boolean">Boolean</Select.Item>
  </>
);

const MethodSelect: FC<{ path: deep.Target<http.Method> }> = ({ path }) => (
  <PForm.Field path={path} label="Method">
    {renderMethodSelect}
  </PForm.Field>
);

const renderMethodSelect = Component.renderProp(
  (p: Select.ButtonsProps<http.Method>) => (
    <Select.Buttons<http.Method> {...p}>
      <Select.Item<http.Method> itemKey="POST">POST</Select.Item>
      <Select.Item<http.Method> itemKey="PUT">PUT</Select.Item>
      <Select.Item<http.Method> itemKey="PATCH">PATCH</Select.Item>
    </Select.Buttons>
  ),
);

const getEndpointChannelNameID = (epKey: string) => `write-ep-ch-${epKey}`;

const EndpointItem = (props: List.ItemProps<string>) => {
  const { itemKey } = props;
  const channel = PForm.useFieldValue(root.config.endpoints[itemKey].channel.channel);
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
        namePath={root.config.endpoints[itemKey].channel.name}
        id={getEndpointChannelNameID(itemKey)}
        level="small"
        weight={450}
        color={9}
        overflow="fade"
      />
    </Select.Item>
  );
};

const endpointItem = Component.renderProp(EndpointItem);

const EMPTY_ENDPOINTS = <Empty.Action message="No endpoints" />;

const EnumValuesEditor: FC<{ path: deep.Path<WriteEndpoint["channel"]> }> = ({
  path,
}) => (
  <PForm.Section title="Enum mapping">
    <PlatformForm.KeyValueEditor
      path={path.enumValues}
      resourceName="mapping"
      keyField="label"
      keyPlaceholder="String (e.g. ON)"
      valueType="number"
      valueFirst
    />
  </PForm.Section>
);

const ChannelFieldSection: FC<{ path: deep.Path<WriteEndpoint["channel"]> }> = ({
  path: channelPath,
}) => {
  const channelKey = PForm.useFieldValue(channelPath.channel);
  const jsonType = PForm.useFieldValue(channelPath.jsonType);
  const channelQuery = primitive.isNonZero(channelKey) ? { key: channelKey } : null;
  const { data: dataType } = PChannel.useResultDataType(channelQuery);

  return (
    <>
      <PForm.Section title="Channel">
        <PForm.TextField
          path={channelPath.pointer}
          label="Pointer"
          padHelpText={false}
          inputProps={JSON_POINTER_INPUT_PROPS}
        />
        <PForm.Field path={channelPath.jsonType} label="JSON type" padHelpText={false}>
          {renderSelectJSONType}
        </PForm.Field>
        {channelKey === 0 && (
          <PForm.Field
            path={channelPath.dataType}
            label="Data type"
            padHelpText={false}
          >
            {renderSelectDataType}
          </PForm.Field>
        )}
        {dataType != null && DataType.TIMESTAMP.equals(dataType) && (
          <TimeFormatField path={channelPath.timeFormat} label="Time format" />
        )}
      </PForm.Section>
      {jsonType === "string" && <EnumValuesEditor path={channelPath} />}
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

type GeneratorKey = "uuid" | TimeFormat;

const generatorDisplayKey = (
  generator: GeneratorType | null | undefined,
  timeFormat: TimeFormat | null | undefined,
): GeneratorKey => {
  if (generator === "timestamp") return timeFormat ?? "iso8601";
  return "uuid";
};

const FieldListItem = ({
  epKey,
  ...props
}: List.ItemProps<string> & { epKey: string }) => {
  const { itemKey } = props;
  const field = root.config.endpoints[epKey].fields[itemKey];
  const staticField = deep.variant(field, "type", "static");
  const generatedField = deep.variant(field, "type", "generated");
  const fieldType = PForm.useFieldValue(field.type);
  const jsonType = PForm.useFieldValue(staticField.jsonType, { optional: true });
  const generator = PForm.useFieldValue(generatedField.generator, { optional: true });
  const timeFormat = PForm.useFieldValue(generatedField.timeFormat, {
    optional: true,
  });
  const { set } = PForm.useContext();

  const handleJSONTypeChange = (value: http.JSONType) => {
    set(staticField.jsonType, value);
    set(staticField.value, json.ZERO_PRIMITIVES[value]);
  };

  const handleGeneratorChange = (key: GeneratorKey) => {
    if (key === "uuid") {
      set(generatedField.generator, "uuid");
      set(generatedField.timeFormat, undefined);
    } else {
      set(generatedField.generator, "timestamp");
      set(generatedField.timeFormat, key);
    }
  };

  return (
    <Select.Item {...props} justify="between" align="center" x>
      <PForm.TextField
        path={field.pointer}
        showLabel={false}
        showHelpText={false}
        inputProps={POINTER_INPUT_PROPS}
      />
      {fieldType === "static" && (
        <Select.Simple<http.JSONType>
          value={jsonType ?? "string"}
          onChange={handleJSONTypeChange}
          resourceName="type"
        >
          {JSON_TYPE_ITEMS}
        </Select.Simple>
      )}
      {fieldType === "static" && jsonType === "string" && (
        <PForm.TextField
          path={staticField.value as deep.Target<string>}
          showLabel={false}
          showHelpText={false}
          inputProps={STRING_INPUT_PROPS}
        />
      )}
      {fieldType === "static" && jsonType === "number" && (
        <PForm.NumericField
          path={staticField.value as deep.Target<number>}
          showLabel={false}
          showHelpText={false}
          inputProps={NUMBER_INPUT_PROPS}
        />
      )}
      {fieldType === "static" && jsonType === "boolean" && (
        <PForm.Field
          path={staticField.value as deep.Target<boolean>}
          showLabel={false}
          showHelpText={false}
        >
          {(p) => <Input.Checkbox {...p} />}
        </PForm.Field>
      )}
      {fieldType === "generated" && (
        <Select.Simple<GeneratorKey>
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
  const { data, push, remove } = PForm.useFieldList(
    root.config.endpoints[epKey].fields,
  );
  const [selected, setSelected] = useState<string[]>([]);
  const isPreview = Task.useIsPreview();

  const handleAddStatic = () => {
    const field: WriteField = {
      key: id.create(),
      pointer: "",
      jsonType: "string",
      type: "static",
      value: "",
    };
    push(field);
    setSelected([field.key]);
  };

  const handleAddGenerated = () => {
    const field: WriteField = {
      key: id.create(),
      pointer: "",
      type: "generated",
      generator: "uuid",
    };
    push(field);
    setSelected([field.key]);
  };

  const handleRemove = (keys: string[]) => {
    remove(keys);
    setSelected([]);
  };

  const listItem = ({ key, ...p }: List.ItemProps<string>) => (
    <FieldListItem key={key} {...p} epKey={epKey} />
  );

  const menuProps = Menu.useContextMenu();
  const menuRenderProp = (p: Menu.ContextMenuMenuProps) => (
    <Task.Views.ContextMenu keys={p.keys} onRemove={handleRemove} />
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
  const endpoint = root.config.endpoints[epKey];
  return (
    <Flex.Box y grow empty className={CSS.B("endpoint-details")}>
      <PForm.Sections className={CSS.B("endpoint-form")}>
        <PForm.Section title="Request">
          <MethodSelect path={endpoint.method} />
          <PForm.TextField
            path={endpoint.path}
            label="Path"
            padHelpText={false}
            inputProps={PATH_INPUT_PROPS}
          />
          <PlatformForm.KeyValueEditor
            path={endpoint.queryParams}
            resourceName="parameter"
            label="Query parameters"
            keyField="parameter"
            keyPlaceholder="limit"
            valuePlaceholder="100"
          />
          <PlatformForm.KeyValueEditor
            path={endpoint.headers}
            resourceName="header"
            label="Headers"
            keyField="name"
            keyPlaceholder="Content-Type"
            valuePlaceholder="application/json"
          />
        </PForm.Section>
        <ChannelFieldSection path={endpoint.channel} />
        <AdditionalFields key={epKey} epKey={epKey} />
      </PForm.Sections>
    </Flex.Box>
  );
};

const PATH_INPUT_PROPS = { placeholder: "/api/control" } as const;

const Form: FC = () => {
  const [selectedEndpoints, setSelectedEndpoints] = useState<string[]>([]);
  const { data, push, remove } = PForm.useFieldList(root.config.endpoints);
  const dev = useFromConfig();
  const resolve = (ep: WriteEndpoint) =>
    dev == null
      ? null
      : { channel: { ...ep.channel, channel: dev.properties.write[ep.path] ?? 0 } };
  const ctx = PForm.useContext();
  const isPreview = Task.useIsPreview();

  const handleAddEndpoint = () => {
    const ep: WriteEndpoint = { ...writeEndpointZ.parse({}), key: id.create() };
    push(ep);
    setSelectedEndpoints([ep.key]);
  };

  const handleRemoveEndpoints = (keys: string[]) => {
    remove(keys);
    setSelectedEndpoints([]);
  };

  const handleDuplicateEndpoints = (keys: string[]) => {
    const allEndpoints = ctx.get(root.config.endpoints).value;
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
  };

  const handleRenameChannel = (key: string) => Text.edit(getEndpointChannelNameID(key));

  const menuProps = Menu.useContextMenu();
  const menuRenderProp = (p: Menu.ContextMenuMenuProps) => (
    <Task.Views.ContextMenu
      keys={p.keys}
      onRemove={handleRemoveEndpoints}
      onDuplicate={handleDuplicateEndpoints}
      onRename={handleRenameChannel}
    />
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
        detailsPath={selected != null ? root.config.endpoints[selected] : null}
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
      <Task.BindChannels<WriteEndpoint>
        path={root.config.endpoints}
        resolve={resolve}
      />
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

const retrieveChannel = async (
  client: Client,
  key: number,
): Promise<channel.Channel | null> => {
  try {
    return await client.channels.retrieve(key);
  } catch {
    return null;
  }
};

const channelExists = async (client: Client, key: number): Promise<boolean> =>
  (await retrieveChannel(client, key)) != null;

const onConfigure: Task.OnConfigure<WriteSchemas["config"]> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: SCHEMAS,
  });
  const safeDevName = channel.escapeInvalidName(dev.name);
  let modified = false;
  try {
    for (const ep of config.endpoints) {
      if (
        ep.channel.channel !== 0 &&
        (await channelExists(client, ep.channel.channel))
      ) {
        if (dev.properties.write[ep.path] === ep.channel.channel) continue;
        dev.properties.write[ep.path] = ep.channel.channel;
        modified = true;
        continue;
      }

      const escapedPath = channel.escapeInvalidName(ep.path);

      // Ensure the index channel exists for this endpoint.
      const storedCmdChannel = dev.properties.write[ep.path];
      if (
        primitive.isNonZero(storedCmdChannel) &&
        (await channelExists(client, storedCmdChannel))
      ) {
        ep.channel.channel = storedCmdChannel;
        continue;
      }

      // no channel in either device or config, create a new one
      const dt = new DataType(ep.channel.dataType);
      const cmdName = primitive.isNonZero(ep.channel.name)
        ? ep.channel.name
        : `${safeDevName}${escapedPath}_cmd`;
      let newCmdCh: channel.Channel;
      if (dt.isVariable)
        newCmdCh = await client.channels.create({
          name: cmdName,
          dataType: ep.channel.dataType,
          virtual: true,
        });
      else {
        const newIndexCh = await client.channels.create({
          name: `${cmdName}_time`,
          dataType: "timestamp",
          isIndex: true,
        });
        newCmdCh = await client.channels.create({
          name: cmdName,
          dataType: ep.channel.dataType,
          index: newIndexCh.key,
        });
      }
      ep.channel.channel = newCmdCh.key;
      dev.properties.write[ep.path] = newCmdCh.key;
      modified = true;
    }
  } finally {
    if (modified) await client.devices.create(dev, SCHEMAS);
  }
  return [config, dev.rack];
};

export const Write = Task.wrapForm({
  Properties,
  Form,
  schemas: WRITE_SCHEMAS,
  deployConfigZ: deployWriteConfigZ,
  type: WRITE_TYPE,
  onConfigure,
});

export const useCreateWrite = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const WriteSelectable = Selector.createSelectable({
  type: WRITE_TYPE,
  title: "HTTP write task",
  icon: <Icon.Logo.HTTP />,
  useOnSelect: useCreateWrite,
});
