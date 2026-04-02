// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/http/task/Form.css";

import { channel, type Synnax as Client } from "@synnaxlabs/client";
import {
  Button,
  Component,
  Divider,
  Flex,
  Form as PForm,
  Header,
  Icon,
  List,
  Menu as PMenu,
  Select,
  Telem,
  Text,
} from "@synnaxlabs/pluto";
import { DataType, id, json, primitive } from "@synnaxlabs/x";
import { type FC, useCallback, useMemo, useState } from "react";

import { EmptyAction } from "@/components";
import { KeyValueEditor } from "@/components/form/KeyValueEditor";
import { CSS } from "@/css";
import { Common } from "@/hardware/common";
import { Device } from "@/hardware/http/device";
import { ContextMenu } from "@/hardware/http/task/ContextMenu";
import { EndpointListItem } from "@/hardware/http/task/EndpointListItem";
import {
  type GeneratorType,
  type TimeFormat,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteEndpoint,
  type WriteField,
  type WriteMethod,
  type WritePayload,
  type WriteSchemas,
  ZERO_CHANNEL_FIELD,
  ZERO_WRITE_ENDPOINT,
  ZERO_WRITE_PAYLOAD,
} from "@/hardware/http/task/types";
import { Selector } from "@/selector";

export const WRITE_LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  type: WRITE_TYPE,
  name: ZERO_WRITE_PAYLOAD.name,
  icon: "Logo.HTTP",
};

export const WriteSelectable = Selector.createSimpleItem({
  title: "HTTP Write Task",
  icon: <Icon.Logo.HTTP />,
  layout: WRITE_LAYOUT,
});

const Properties = () => (
  <>
    <Device.Select />
    <Flex.Box x grow>
      <Common.Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

const WRITE_METHOD_KEYS: WriteMethod[] = ["POST", "PUT", "PATCH"];

const JSON_TYPE_DATA: Select.StaticEntry<json.PrimitiveType>[] = [
  { key: "number", name: "Number" },
  { key: "string", name: "String" },
  { key: "boolean", name: "Boolean" },
];

const GENERATOR_DATA: Select.StaticEntry<GeneratorType | TimeFormat>[] = [
  { key: "uuid", name: "UUID" },
  { key: "iso8601", name: "Timestamp (ISO 8601)" },
  { key: "unix_sec", name: "Timestamp (s)" },
  { key: "unix_ms", name: "Timestamp (ms)" },
  { key: "unix_us", name: "Timestamp (μs)" },
  { key: "unix_ns", name: "Timestamp (ns)" },
];

const MethodSelect: FC<{ path: string }> = ({ path }) => (
  <PForm.Field<WriteMethod> path={path} label="Method">
    {renderMethodSelect}
  </PForm.Field>
);

const renderMethodSelect = Component.renderProp(
  (p: Omit<Select.ButtonsProps<WriteMethod>, "keys">) => (
    <Select.Buttons<WriteMethod> {...p} keys={WRITE_METHOD_KEYS}>
      <Select.Button<WriteMethod> itemKey="POST">POST</Select.Button>
      <Select.Button<WriteMethod> itemKey="PUT">PUT</Select.Button>
      <Select.Button<WriteMethod> itemKey="PATCH">PATCH</Select.Button>
    </Select.Buttons>
  ),
);

const getEndpointChannelNameID = (epKey: string) => `write-ep-ch-${epKey}`;

const WriteEndpointListItem = (props: List.ItemProps<string>) => {
  const { itemKey } = props;
  const channel = PForm.useFieldValue<number>(
    `config.endpoints.${itemKey}.channel.channel`,
  );
  const extraNode = useMemo(
    () => (
      <Common.Task.ChannelName
        channel={channel}
        namePath={`config.endpoints.${itemKey}.channel.name`}
        id={getEndpointChannelNameID(itemKey)}
        weight={600}
        color={10}
      />
    ),
    [channel, itemKey],
  );
  return <EndpointListItem {...props} extra={extraNode} y textProps={TEXT_PROPS} />;
};

const TEXT_PROPS = { weight: 450, color: 9 } as const;

const writeEndpointListItem = Component.renderProp(WriteEndpointListItem);

const EnumValuesEditor: FC<{ channelPath: string }> = ({ channelPath }) => (
  <KeyValueEditor
    path={`${channelPath}.enumValues`}
    label="Enum mappings"
    keyField="label"
    keyPlaceholder="String (e.g. ON)"
    valueType="number"
    valueFirst
  />
);

const ChannelFieldSection: FC<{ epPath: string }> = ({ epPath }) => {
  const channelPath = `${epPath}.channel`;
  const channelKey = PForm.useFieldValue<number>(`${channelPath}.channel`);
  const jsonType = PForm.useFieldValue<string>(`${channelPath}.jsonType`);

  return (
    <>
      <Header.Header>
        <Header.Title weight={500} color={9}>
          Channel
        </Header.Title>
      </Header.Header>
      <Flex.Box className={CSS.B("channel-field-section")}>
        <Flex.Box x align="end" gap="large">
          <PForm.TextField
            path={`${channelPath}.pointer`}
            label="JSON pointer"
            grow
            inputProps={JSON_POINTER_INPUT_PROPS}
          />
          <PForm.Field<string>
            path={`${channelPath}.jsonType`}
            label="JSON type"
            className={CSS.B("json-type-select")}
          >
            {renderSelectJSONType}
          </PForm.Field>
        </Flex.Box>
        {channelKey === 0 && (
          <PForm.Field<string>
            path={`${channelPath}.dataType`}
            label="Synnax data type"
            showHelpText={false}
            className={CSS.B("data-type-select")}
          >
            {renderSelectDataType}
          </PForm.Field>
        )}
        {jsonType === "string" && <EnumValuesEditor channelPath={channelPath} />}
      </Flex.Box>
    </>
  );
};

const renderSelectJSONType = Component.renderProp(
  (
    p: Omit<
      Select.StaticProps<string, Select.StaticEntry<json.PrimitiveType>>,
      "data" | "resourceName"
    >,
  ) => (
    <Select.Static<string, Select.StaticEntry<json.PrimitiveType>>
      {...p}
      data={JSON_TYPE_DATA}
      resourceName="JSON type"
    />
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

const FieldListItem = (props: List.ItemProps<string> & { epKey: string }) => {
  const { itemKey, epKey } = props;
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
    <Select.ListItem {...props} justify="between" align="center" x>
      <PForm.TextField
        path={`${path}.pointer`}
        showLabel={false}
        showHelpText={false}
        inputProps={STATIC_FILED_INPUT_PROPS}
        grow
      />
      {fieldType === "static" && (
        <Select.Static<json.PrimitiveType, Select.StaticEntry<json.PrimitiveType>>
          value={jsonType ?? "string"}
          onChange={handleJSONTypeChange}
          data={JSON_TYPE_DATA}
          resourceName="type"
        />
      )}
      {fieldType === "static" && jsonType === "string" && (
        <PForm.TextField
          path={`${path}.value`}
          showLabel={false}
          showHelpText={false}
          inputProps={STRING_INPUT_PROPS}
          className={CSS.B("static-field-value")}
        />
      )}
      {fieldType === "static" && jsonType === "number" && (
        <PForm.NumericField
          path={`${path}.value`}
          showLabel={false}
          showHelpText={false}
          className={CSS.B("static-field-value")}
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
        <Select.Static<
          GeneratorType | TimeFormat,
          Select.StaticEntry<GeneratorType | TimeFormat>
        >
          value={generatorDisplayKey(generator, timeFormat)}
          onChange={handleGeneratorChange}
          data={GENERATOR_DATA}
          resourceName="generator"
        />
      )}
      <Text.Text level="small" color={7}>
        {fieldType}
      </Text.Text>
    </Select.ListItem>
  );
};

const STATIC_FILED_INPUT_PROPS = { placeholder: "field" } as const;

const STRING_INPUT_PROPS = { placeholder: "value" } as const;

const AdditionalFields: FC<{ epKey: string }> = ({ epKey }) => {
  const path = `config.endpoints.${epKey}.fields`;
  const { data, push, remove } = PForm.useFieldList<string, WriteField>(path);
  const [selected, setSelected] = useState<string[]>([]);
  const isSnapshot = Common.Task.useIsSnapshot();

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

  const handleDelete = useCallback(
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

  const menuProps = PMenu.useContextMenu();
  const menuRenderProp = useCallback(
    (p: PMenu.ContextMenuMenuProps) => (
      <ContextMenu keys={p.keys} onDelete={handleDelete} />
    ),
    [handleDelete],
  );

  return (
    <Flex.Box y grow empty>
      <Header.Header>
        <Header.Title weight={500} color={9}>
          Additional fields
        </Header.Title>
        {!isSnapshot && (
          <Header.Actions>
            <Button.Button
              onClick={handleAddStatic}
              variant="text"
              contrast={2}
              tooltip="Add static field"
              sharp
            >
              <Icon.Add />
            </Button.Button>
            <Button.Button
              onClick={handleAddGenerated}
              variant="text"
              contrast={2}
              tooltip="Add generated field"
              sharp
            >
              <Icon.Time />
            </Button.Button>
          </Header.Actions>
        )}
      </Header.Header>
      <PMenu.ContextMenu {...menuProps} menu={menuRenderProp}>
        <Select.Frame<string, WriteField>
          multiple
          data={data}
          value={selected}
          onChange={setSelected}
          replaceOnSingle
          allowNone
        >
          <List.Items<string, WriteField>
            full="y"
            className={CSS(menuProps.className, CSS.B("field-list-items"))}
            onContextMenu={menuProps.open}
            emptyContent={EMPTY_CONTENT}
          >
            {listItem}
          </List.Items>
        </Select.Frame>
      </PMenu.ContextMenu>
    </Flex.Box>
  );
};

const EMPTY_CONTENT = <EmptyAction message="No additional fields." action="" />;

const EndpointDetails: FC<{ epKey: string }> = ({ epKey }) => {
  const path = `config.endpoints.${epKey}`;
  return (
    <Flex.Box y grow empty className={CSS.B("endpoint-details")}>
      <Flex.Box gap="small" empty className={CSS.B("endpoint-details-form")}>
        <Flex.Box x align="end" gap="large">
          <MethodSelect path={`${path}.method`} />
          <PForm.TextField
            path={`${path}.path`}
            label="Path"
            required
            grow
            inputProps={PATH_INPUT_PROPS}
          />
        </Flex.Box>
        <Divider.Divider x />
        <KeyValueEditor
          path={`${path}.headers`}
          label="Headers"
          keyField="name"
          className={CSS.B("headers-kv-editor")}
          keyPlaceholder="Name"
          valuePlaceholder="Value"
        />
        <Divider.Divider x />
        <KeyValueEditor
          path={`${path}.queryParams`}
          label="Query parameters"
          keyField="parameter"
          className={CSS.B("query-params-kv-editor")}
          keyPlaceholder="Parameter"
          valuePlaceholder="Value"
        />
      </Flex.Box>
      <Divider.Divider x />
      <ChannelFieldSection epPath={path} />
      <Divider.Divider x />
      <AdditionalFields key={epKey} epKey={epKey} />
    </Flex.Box>
  );
};

const PATH_INPUT_PROPS = { placeholder: "/api/control" };

const Form: FC<Common.Task.FormProps<WriteSchemas>> = () => {
  const [selectedEndpoints, setSelectedEndpoints] = useState<string[]>([]);
  const { data, push, remove } = PForm.useFieldList<string, WriteEndpoint>(
    "config.endpoints",
  );
  const ctx = PForm.useContext();
  const isSnapshot = Common.Task.useIsSnapshot();

  const handleAddEndpoint = useCallback(() => {
    const ep: WriteEndpoint = {
      ...ZERO_WRITE_ENDPOINT,
      key: id.create(),
      channel: { ...ZERO_CHANNEL_FIELD },
      fields: [],
    };
    push(ep);
    setSelectedEndpoints([ep.key]);
  }, [push]);

  const handleDeleteEndpoints = useCallback(
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

  const menuProps = PMenu.useContextMenu();
  const menuRenderProp = useCallback(
    (p: PMenu.ContextMenuMenuProps) => (
      <ContextMenu
        keys={p.keys}
        onDelete={handleDeleteEndpoints}
        onDuplicate={handleDuplicateEndpoints}
        onRename={handleRenameChannel}
      />
    ),
    [handleDeleteEndpoints, handleDuplicateEndpoints, handleRenameChannel],
  );

  return (
    <Flex.Box x grow empty>
      <Flex.Box y className={CSS.B("endpoint-list")} empty>
        <Header.Header>
          <Header.Title weight={500} color={10}>
            Endpoints
          </Header.Title>
          {!isSnapshot && (
            <Header.Actions>
              <Button.Button
                onClick={handleAddEndpoint}
                variant="text"
                contrast={2}
                tooltip="Add endpoint"
                sharp
              >
                <Icon.Add />
              </Button.Button>
            </Header.Actions>
          )}
        </Header.Header>
        <PMenu.ContextMenu {...menuProps} menu={menuRenderProp}>
          <Select.Frame<string, WriteEndpoint>
            multiple
            data={data}
            value={selectedEndpoints}
            onChange={setSelectedEndpoints}
            replaceOnSingle
            allowNone={false}
            autoSelectOnNone
          >
            <List.Items<string, WriteEndpoint>
              full="y"
              className={menuProps.className}
              onContextMenu={menuProps.open}
              emptyContent={
                <EmptyAction
                  message="No endpoints."
                  action="Add an endpoint"
                  onClick={isSnapshot ? undefined : handleAddEndpoint}
                />
              }
            >
              {writeEndpointListItem}
            </List.Items>
          </Select.Frame>
        </PMenu.ContextMenu>
      </Flex.Box>
      <Divider.Divider y />
      <Flex.Box y grow empty>
        <Common.Task.Layouts.DetailsHeader
          path={
            selectedEndpoints.length > 0
              ? `config.endpoints.${selectedEndpoints[0]}`
              : ""
          }
          disabled={selectedEndpoints.length === 0}
        />
        {selectedEndpoints.length > 0 ? (
          <EndpointDetails epKey={selectedEndpoints[0]} />
        ) : (
          <Flex.Box y grow align="center" justify="center">
            <Text.Text status="disabled">Select an endpoint to configure</Text.Text>
          </Flex.Box>
        )}
      </Flex.Box>
    </Flex.Box>
  );
};

const getInitialValues: Common.Task.GetInitialValues<WriteSchemas> = ({
  deviceKey,
  config,
}) => {
  if (config != null) {
    const pld: WritePayload = {
      ...ZERO_WRITE_PAYLOAD,
      config: WRITE_SCHEMAS.config.parse(config),
    };
    if (deviceKey != null) pld.config.device = deviceKey;
    return pld;
  }
  const pld: WritePayload = { ...ZERO_WRITE_PAYLOAD };
  if (deviceKey != null) pld.config = { ...pld.config, device: deviceKey };
  return pld;
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

const onConfigure: Common.Task.OnConfigure<WriteSchemas["config"]> = async (
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
    if (modified) await client.devices.create(dev, Device.SCHEMAS);
  }
  return [config, dev.rack];
};

export const Write = Common.Task.wrapForm({
  Properties,
  Form,
  schemas: WRITE_SCHEMAS,
  type: WRITE_TYPE,
  getInitialValues,
  onConfigure,
});
