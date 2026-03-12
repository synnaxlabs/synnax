// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/http/task/Write.css";

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
import { id, primitive } from "@synnaxlabs/x";
import { type FC, useCallback, useState } from "react";

import { EmptyAction, Menu } from "@/components";
import { KeyValueEditor } from "@/components/form/KeyValueEditor";
import { CSS } from "@/css";
import { Common } from "@/hardware/common";
import { Device } from "@/hardware/http/device";
import {
  type GeneratorType,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteEndpoint,
  type WriteField,
  type WriteHTTPMethod,
  type WritePayload,
  type WriteSchemas,
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

// ─── Endpoint Context Menu ───

interface EndpointContextMenuProps {
  keys: string[];
  onDelete: (keys: string[]) => void;
  onDuplicate?: (keys: string[]) => void;
}

const EndpointContextMenu = ({
  keys,
  onDuplicate,
  onDelete,
}: EndpointContextMenuProps) => {
  const isSnapshot = Common.Task.useIsSnapshot();
  const canAct = keys.length > 0;
  const canDuplicate = onDuplicate != null;
  return (
    <PMenu.Menu level="small">
      {!isSnapshot && canAct && (
        <>
          {canDuplicate && (
            <PMenu.Item itemKey="duplicate" onClick={() => onDuplicate?.(keys)}>
              <Icon.Copy />
              Duplicate
            </PMenu.Item>
          )}
          <PMenu.Item itemKey="delete" onClick={() => onDelete(keys)}>
            <Icon.Close />
            Delete
          </PMenu.Item>
          <PMenu.Divider />
        </>
      )}
      <Menu.ReloadConsoleItem />
    </PMenu.Menu>
  );
};

const EndpointListItem = (props: List.ItemProps<string>) => {
  const { itemKey } = props;
  const method = PForm.useFieldValue<string>(`config.endpoints.${itemKey}.method`);
  const epPath = PForm.useFieldValue<string>(`config.endpoints.${itemKey}.path`);
  return (
    <Select.ListItem {...props} justify="between" align="center" x>
      <Text.Text level="small" weight={500}>
        {method} {epPath || "(no path)"}
      </Text.Text>
    </Select.ListItem>
  );
};

const endpointListItem = Component.renderProp(EndpointListItem);

const WRITE_METHOD_KEYS: WriteHTTPMethod[] = ["POST", "PUT", "PATCH"];

const JSON_TYPE_DATA: Select.StaticEntry<string>[] = [
  { key: "number", name: "Number" },
  { key: "string", name: "String" },
  { key: "boolean", name: "Boolean" },
];

const GENERATOR_DATA: Select.StaticEntry<string>[] = [
  { key: "uuid", name: "UUID" },
  { key: "iso8601", name: "Timestamp (ISO 8601)" },
  { key: "unix_sec", name: "Timestamp (s)" },
  { key: "unix_ms", name: "Timestamp (ms)" },
  { key: "unix_us", name: "Timestamp (μs)" },
  { key: "unix_ns", name: "Timestamp (ns)" },
];

const WriteMethodSelect: FC<{ path: string }> = ({ path }) => (
  <PForm.Field<WriteHTTPMethod> path={path} label="Method">
    {renderWriteMethodSelect}
  </PForm.Field>
);

const renderWriteMethodSelect = Component.renderProp(
  (p: Omit<Select.ButtonsProps<WriteHTTPMethod>, "keys">) => (
    <Select.Buttons<WriteHTTPMethod> {...p} keys={WRITE_METHOD_KEYS}>
      <Select.Button<WriteHTTPMethod> itemKey="POST">POST</Select.Button>
      <Select.Button<WriteHTTPMethod> itemKey="PUT">PUT</Select.Button>
      <Select.Button<WriteHTTPMethod> itemKey="PATCH">PATCH</Select.Button>
    </Select.Buttons>
  ),
);

const ChannelFieldSection: FC<{ epPath: string; epKey: string }> = ({
  epPath,
  epKey,
}) => {
  const channelPath = `${epPath}.channel`;
  const channelKey = PForm.useFieldValue<number>(`${channelPath}.channel`);

  return (
    <Flex.Box y gap="small" style={{ padding: "0 1rem" }}>
      <Header.Header>
        <Header.Title weight={500} color={10}>
          Channel
        </Header.Title>
        <Flex.Box x align="center" grow justify="end">
          <Common.Task.ChannelName
            channel={channelKey}
            namePath={`${channelPath}.name`}
            id={Common.Task.getChannelNameID(epKey)}
          />
        </Flex.Box>
      </Header.Header>
      <Flex.Box x align="end" gap="large">
        <PForm.TextField
          path={`${channelPath}.pointer`}
          label="JSON Pointer"
          grow
          inputProps={{ placeholder: "/value" }}
        />
        <PForm.Field<string>
          path={`${channelPath}.jsonType`}
          label="JSON Type"
          style={{ width: 140 }}
        >
          {({ value, onChange }) => (
            <Select.Static<string, Select.StaticEntry<string>>
              value={value}
              onChange={onChange}
              data={JSON_TYPE_DATA}
              resourceName="JSON type"
            />
          )}
        </PForm.Field>
      </Flex.Box>
      <Flex.Box x align="end" gap="large">
        {channelKey === 0 && (
          <PForm.Field<string>
            path={`${channelPath}.dataType`}
            label="Data Type"
            showHelpText={false}
          >
            {({ value, onChange }) => (
              <Telem.SelectDataType
                value={value}
                onChange={onChange}
                hideVariableDensity
                location="bottom"
              />
            )}
          </PForm.Field>
        )}
      </Flex.Box>
    </Flex.Box>
  );
};

const generatorDisplayKey = (
  generator: string | null | undefined,
  timeFormat: string | null | undefined,
): string => {
  if (generator === "timestamp") return timeFormat ?? "iso8601";
  return "uuid";
};

const ZERO_JSON_VALUES: Record<string, string | number | boolean> = {
  string: "",
  number: 0,
  boolean: false,
};

const FieldListItem = (props: List.ItemProps<string> & { epKey: string }) => {
  const { itemKey, epKey } = props;
  const path = `config.endpoints.${epKey}.fields.${itemKey}`;
  const fieldType = PForm.useFieldValue<string>(`${path}.type`);
  const jsonType = PForm.useFieldValue<string | undefined>(`${path}.jsonType`, {
    optional: true,
  });
  const generator = PForm.useFieldValue<GeneratorType | undefined>(
    `${path}.generator`,
    { optional: true },
  );
  const timeFormat = PForm.useFieldValue<string | undefined>(`${path}.timeFormat`, {
    optional: true,
  });
  const { set } = PForm.useContext();

  const handleJSONTypeChange = useCallback(
    (value: string) => {
      set(`${path}.jsonType`, value);
      set(`${path}.value`, ZERO_JSON_VALUES[value]);
    },
    [set, path],
  );

  const handleGeneratorChange = useCallback(
    (key: string) => {
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
        inputProps={{ placeholder: "/field" }}
        grow
      />
      {fieldType === "static" && (
        <Select.Static<string, Select.StaticEntry<string>>
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
          inputProps={{ placeholder: "value" }}
          style={{ width: 120 }}
        />
      )}
      {fieldType === "static" && jsonType === "number" && (
        <PForm.NumericField
          path={`${path}.value`}
          showLabel={false}
          showHelpText={false}
          style={{ width: 120 }}
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
        <Select.Static<string, Select.StaticEntry<string>>
          value={generatorDisplayKey(generator, timeFormat)}
          onChange={handleGeneratorChange}
          data={GENERATOR_DATA}
          resourceName="generator"
        />
      )}
      <Text.Text level="small" color={7} style={{ whiteSpace: "nowrap" }}>
        {fieldType}
      </Text.Text>
    </Select.ListItem>
  );
};

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
      <EndpointContextMenu keys={p.keys} onDelete={handleDelete} />
    ),
    [handleDelete],
  );

  return (
    <Flex.Box y grow empty>
      <Header.Header>
        <Header.Title weight={500} color={10}>
          Additional Fields
        </Header.Title>
        {!isSnapshot && (
          <Header.Actions>
            <Button.Button
              onClick={handleAddStatic}
              variant="text"
              contrast={2}
              tooltip="Add Static Field"
              sharp
            >
              <Icon.Add />
            </Button.Button>
            <Button.Button
              onClick={handleAddGenerated}
              variant="text"
              contrast={2}
              tooltip="Add Generated Field"
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
            className={menuProps.className}
            onContextMenu={menuProps.open}
            emptyContent={emptyContent}
          >
            {listItem}
          </List.Items>
        </Select.Frame>
      </PMenu.ContextMenu>
    </Flex.Box>
  );
};

const emptyContent = <EmptyAction message="No additional fields." action="" />;

const EndpointDetails: FC<{ epKey: string }> = ({ epKey }) => {
  const path = `config.endpoints.${epKey}`;
  return (
    <Flex.Box y grow empty className={CSS.B("endpoint-details")}>
      <Flex.Box gap="small" empty className={CSS.B("endpoint-details-form")}>
        <Flex.Box x align="end" gap="large">
          <WriteMethodSelect path={`${path}.method`} />
          <PForm.TextField
            path={`${path}.path`}
            label="Path"
            grow
            inputProps={{ placeholder: "/api/control" }}
          />
        </Flex.Box>
        <KeyValueEditor
          path={`${path}.headers`}
          label="Headers"
          className={CSS.B("endpoint-details-headers")}
          keyPlaceholder="Header Name"
          valuePlaceholder="Header Value"
        />
      </Flex.Box>
      <Divider.Divider x padded />
      <ChannelFieldSection epPath={path} epKey={epKey} />
      <Divider.Divider x />
      <AdditionalFields key={epKey} epKey={epKey} />
    </Flex.Box>
  );
};

const Properties = () => (
  <>
    <Device.Select />
    <Flex.Box x grow>
      <Common.Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

const Form: FC<Common.Task.FormProps<WriteSchemas>> = () => {
  const [selectedEndpoints, setSelectedEndpoints] = useState<string[]>([]);
  const { data, push, remove } = PForm.useFieldList<string, WriteEndpoint>(
    "config.endpoints",
  );
  const ctx = PForm.useContext();
  const isSnapshot = Common.Task.useIsSnapshot();

  const handleAddEndpoint = useCallback(() => {
    const ep: WriteEndpoint = { ...ZERO_WRITE_ENDPOINT, key: id.create() };
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

  const menuProps = PMenu.useContextMenu();
  const menuRenderProp = useCallback(
    (p: PMenu.ContextMenuMenuProps) => (
      <EndpointContextMenu
        keys={p.keys}
        onDelete={handleDeleteEndpoints}
        onDuplicate={handleDuplicateEndpoints}
      />
    ),
    [handleDeleteEndpoints, handleDuplicateEndpoints],
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
                tooltip="Add Endpoint"
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
              {endpointListItem}
            </List.Items>
          </Select.Frame>
        </PMenu.ContextMenu>
      </Flex.Box>
      <Divider.Divider y />
      {selectedEndpoints.length > 0 ? (
        <EndpointDetails epKey={selectedEndpoints[0]} />
      ) : (
        <Flex.Box y grow align="center" justify="center">
          <Text.Text level="small" status="disabled">
            Select an endpoint to configure
          </Text.Text>
        </Flex.Box>
      )}
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

      const newIndexCh = await client.channels.create({
        name: `${safeDevName}${escapedPath}_cmd_time`,
        dataType: "timestamp",
        isIndex: true,
      });
      const newCmdCh = await client.channels.create({
        name: `${safeDevName}${escapedPath}_cmd`,
        dataType: ep.channel.dataType,
        index: newIndexCh.key,
      });
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
