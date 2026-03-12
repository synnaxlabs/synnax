// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/http/task/Write.css";

import { channel } from "@synnaxlabs/client";
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
  onDuplicate: (keys: string[]) => void;
}

const EndpointContextMenu = ({
  keys,
  onDelete,
  onDuplicate,
}: EndpointContextMenuProps) => {
  const isSnapshot = Common.Task.useIsSnapshot();
  const canAct = keys.length > 0;
  const handleSelect: Record<string, () => void> = {
    duplicate: () => onDuplicate(keys),
    delete: () => onDelete(keys),
  };
  return (
    <PMenu.Menu onChange={handleSelect} level="small">
      {!isSnapshot && canAct && (
        <>
          <PMenu.Item itemKey="duplicate">
            <Icon.Copy />
            Duplicate
          </PMenu.Item>
          <PMenu.Item itemKey="delete">
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

// ─── Endpoint List Item ───

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

// ─── Selectors ───

const TIME_FORMAT_DATA: Select.StaticEntry<string>[] = [
  { key: "iso8601", name: "ISO 8601" },
  { key: "unix_sec", name: "Unix (s)" },
  { key: "unix_ms", name: "Unix (ms)" },
  { key: "unix_us", name: "Unix (μs)" },
  { key: "unix_ns", name: "Unix (ns)" },
];

type WriteHTTPMethod = "POST" | "PUT" | "PATCH";
const WRITE_METHOD_KEYS: WriteHTTPMethod[] = ["POST", "PUT", "PATCH"];

const JSON_TYPE_DATA: Select.StaticEntry<string>[] = [
  { key: "number", name: "Number" },
  { key: "string", name: "String" },
  { key: "boolean", name: "Boolean" },
];

const GENERATOR_DATA: Select.StaticEntry<string>[] = [
  { key: "uuid", name: "UUID" },
  { key: "timestamp", name: "Timestamp" },
];

// ─── Method Select ───

const WriteMethodSelect: FC<{ path: string }> = ({ path }) => (
  <PForm.Field<WriteHTTPMethod> path={path} label="Method">
    {({ value, onChange }) => (
      <Select.Buttons<WriteHTTPMethod>
        value={value}
        onChange={onChange}
        keys={WRITE_METHOD_KEYS}
      >
        <Select.Button<WriteHTTPMethod> itemKey="POST">POST</Select.Button>
        <Select.Button<WriteHTTPMethod> itemKey="PUT">PUT</Select.Button>
        <Select.Button<WriteHTTPMethod> itemKey="PATCH">PATCH</Select.Button>
      </Select.Buttons>
    )}
  </PForm.Field>
);

// ─── Channel Field Section ───

const ChannelFieldSection: FC<{ epPath: string }> = ({ epPath }) => {
  const channelPath = `${epPath}.channel`;
  const channelKey = PForm.useFieldValue<number>(`${channelPath}.channel`);

  return (
    <Flex.Box y gap="small" style={{ padding: "0 1rem" }}>
      <Header.Header>
        <Header.Title weight={500} color={10}>
          Channel
        </Header.Title>
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

// ─── Additional Fields (static + generated) ───

const FieldListItem = (props: List.ItemProps<string> & { epKey: string }) => {
  const { itemKey, epKey } = props;
  const path = `config.endpoints.${epKey}.fields.${itemKey}`;
  const fieldType = PForm.useFieldValue<string>(`${path}.type`);
  const generator = PForm.useFieldValue<GeneratorType | undefined>(
    `${path}.generator`,
    { optional: true },
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
        <PForm.Field<string>
          path={`${path}.jsonType`}
          showLabel={false}
          showHelpText={false}
          style={{ width: 100 }}
        >
          {({ value, onChange }) => (
            <Select.Static<string, Select.StaticEntry<string>>
              value={value}
              onChange={onChange}
              data={JSON_TYPE_DATA}
              resourceName="type"
            />
          )}
        </PForm.Field>
      )}
      {fieldType === "static" && (
        <PForm.TextField
          path={`${path}.value`}
          showLabel={false}
          showHelpText={false}
          inputProps={{ placeholder: "value" }}
          style={{ width: 120 }}
        />
      )}
      {fieldType === "generated" && (
        <PForm.Field<string>
          path={`${path}.generator`}
          showLabel={false}
          showHelpText={false}
          style={{ width: 120 }}
        >
          {({ value, onChange }) => (
            <Select.Static<string, Select.StaticEntry<string>>
              value={value ?? "uuid"}
              onChange={onChange}
              data={GENERATOR_DATA}
              resourceName="generator"
            />
          )}
        </PForm.Field>
      )}
      {fieldType === "generated" && generator === "timestamp" && (
        <PForm.Field<string>
          path={`${path}.timeFormat`}
          showLabel={false}
          showHelpText={false}
          style={{ width: 130 }}
        >
          {({ value, onChange }) => (
            <Select.Static<string, Select.StaticEntry<string>>
              value={value ?? "iso8601"}
              onChange={onChange}
              data={TIME_FORMAT_DATA}
              resourceName="format"
            />
          )}
        </PForm.Field>
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
      <PMenu.ContextMenu
        {...menuProps}
        menu={(p) => (
          <EndpointContextMenu
            keys={p.keys}
            onDelete={handleDelete}
            onDuplicate={() => {}}
          />
        )}
      >
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
            emptyContent={<EmptyAction message="No additional fields." action="" />}
          >
            {listItem}
          </List.Items>
        </Select.Frame>
      </PMenu.ContextMenu>
    </Flex.Box>
  );
};

// ─── Endpoint Details ───

const EndpointDetails: FC<{ epKey: string }> = ({ epKey }) => {
  const path = `config.endpoints.${epKey}`;
  return (
    <Flex.Box
      y
      grow
      empty
      style={{ overflowY: "auto" }}
      className={CSS.B("http-write-endpoint")}
    >
      <Flex.Box x align="end" gap="large" style={{ padding: "1rem" }}>
        <WriteMethodSelect path={`${path}.method`} />
        <PForm.TextField
          path={`${path}.path`}
          label="Path"
          grow
          inputProps={{ placeholder: "/api/control" }}
        />
      </Flex.Box>
      <Divider.Divider x padded />
      <Flex.Box y style={{ padding: "0 1rem" }}>
        <KeyValueEditor
          path={`${path}.headers`}
          label="Headers"
          keyPlaceholder="Header Name"
          valuePlaceholder="Header Value"
        />
      </Flex.Box>
      <Divider.Divider x padded />
      <ChannelFieldSection epPath={path} />
      <Divider.Divider x padded />
      <AdditionalFields key={epKey} epKey={epKey} />
    </Flex.Box>
  );
};

// ─── Properties Panel ───

const Properties = () => (
  <>
    <Device.Select />
    <Flex.Box x grow>
      <Common.Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

// ─── Main Form ───

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

  return (
    <Flex.Box x grow empty>
      <Flex.Box y style={{ width: 250, flexShrink: 0 }}>
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
        <PMenu.ContextMenu
          {...menuProps}
          menu={(p) => (
            <EndpointContextMenu
              keys={p.keys}
              onDelete={handleDeleteEndpoints}
              onDuplicate={handleDuplicateEndpoints}
            />
          )}
        >
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

// ─── Configure Handler ───

const getInitialValues: Common.Task.GetInitialValues<WriteSchemas> = ({
  deviceKey,
}) => ({
  ...ZERO_WRITE_PAYLOAD,
  config: {
    ...ZERO_WRITE_PAYLOAD.config,
    device: deviceKey ?? ZERO_WRITE_PAYLOAD.config.device,
  },
});

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

  // Ensure writeIndexes exists.
  if (dev.properties.writeIndexes == null) {
    dev.properties.writeIndexes = {};
    modified = true;
  }

  for (const ep of config.endpoints) {
    const ch = ep.channel;
    if (ch.channel !== 0) continue;

    const escapedPath = channel.escapeInvalidName(ep.path);
    const indexName = `${safeDevName}_${escapedPath}_cmd_time`;
    const cmdName = `${safeDevName}_${escapedPath}_cmd`;

    // Check if we already have a write index for this endpoint path.
    const existingIndex = dev.properties.writeIndexes[ep.path];
    let indexKey: number;
    if (primitive.isNonZero(existingIndex)) indexKey = existingIndex;
    else {
      const existing = await client.channels.retrieve({ names: [indexName] });
      if (existing.length > 0) indexKey = existing[0].key;
      else {
        const newIndexCh = await client.channels.create({
          name: indexName,
          dataType: "timestamp",
          isIndex: true,
        });
        indexKey = newIndexCh.key;
      }
      dev.properties.writeIndexes[ep.path] = indexKey;
      modified = true;
    }

    // Create or retrieve the command channel.
    const existingCmd = await client.channels.retrieve({ names: [cmdName] });
    if (existingCmd.length > 0) ch.channel = existingCmd[0].key;
    else {
      const newCh = await client.channels.create({
        name: cmdName,
        dataType: ch.dataType,
        index: indexKey,
      });
      ch.channel = newCh.key;
    }
  }

  if (modified) await client.devices.create(dev, Device.SCHEMAS);
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
