// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, NotFoundError } from "@synnaxlabs/client";
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
  Text,
} from "@synnaxlabs/pluto";
import { id, primitive } from "@synnaxlabs/x";
import { type FC, useCallback, useState } from "react";

import { EmptyAction, Menu } from "@/components";
import { Common } from "@/hardware/common";
import { ChannelList as BaseChannelList } from "@/hardware/common/task/ChannelList";
import { Device } from "@/hardware/http/device";
import {
  READ_SCHEMAS,
  READ_TYPE,
  type readConfigZ,
  type ReadEndpoint,
  type ReadField,
  type readStatusDataZ,
  type readTypeZ,
  ZERO_READ_ENDPOINT,
  ZERO_READ_FIELD,
  ZERO_READ_PAYLOAD,
} from "@/hardware/http/task/types";
import { Selector } from "@/selector";

export const READ_LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  type: READ_TYPE,
  name: ZERO_READ_PAYLOAD.name,
  icon: "Logo.HTTP",
};

export const ReadSelectable = Selector.createSimpleItem({
  title: "HTTP Read Task",
  icon: <Icon.Logo.HTTP />,
  layout: READ_LAYOUT,
});

const RATE_INPUT_PROPS = { endContent: "Hz", style: { maxWidth: "20rem" } } as const;

const Properties = () => (
  <>
    <Device.Select />
    <Flex.Box x grow>
      <PForm.NumericField
        path="config.rate"
        label="Rate"
        inputProps={RATE_INPUT_PROPS}
      />
      <Common.Task.Fields.DataSaving />
      <PForm.SwitchField path="config.strict" label="Strict" />
      <Common.Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

// ─── Endpoint Context Menu ───

const EndpointContextMenu: FC<{
  keys: string[];
  onDelete: (keys: string[]) => void;
  onDuplicate: (keys: string[]) => void;
}> = ({ keys, onDelete, onDuplicate }) => {
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
  const fields = PForm.useFieldValue<ReadField[]>(`config.endpoints.${itemKey}.fields`);
  return (
    <Select.ListItem {...props} justify="between" align="center" x>
      <Text.Text level="small" weight={500}>
        {method} {epPath || "(no path)"}
      </Text.Text>
      <Text.Text level="small" color={7}>
        {fields?.length ?? 0}
      </Text.Text>
    </Select.ListItem>
  );
};

const endpointListItem = Component.renderProp(EndpointListItem);

// ─── Time Format Data ───

const TIME_FORMAT_DATA: Select.StaticEntry<string>[] = [
  { key: "iso8601", name: "ISO 8601" },
  { key: "unix_sec", name: "Unix (s)" },
  { key: "unix_ms", name: "Unix (ms)" },
  { key: "unix_us", name: "Unix (μs)" },
  { key: "unix_ns", name: "Unix (ns)" },
];

// ─── Field List Item ───

interface FieldListItemProps extends Common.Task.ChannelListItemProps {
  epKey: string;
}

const FieldListItem = ({ epKey, ...props }: FieldListItemProps) => {
  const { itemKey } = props;
  const path = `config.endpoints.${epKey}.fields.${itemKey}`;
  const fieldChannel = PForm.useFieldValue<number>(`${path}.channel`);
  return (
    <Select.ListItem {...props} justify="between" align="center" x>
      <PForm.TextField
        path={`${path}.pointer`}
        showLabel={false}
        showHelpText={false}
        inputProps={{ placeholder: "/temperature" }}
        grow
      />
      <Flex.Box x align="center" grow justify="end">
        <Common.Task.ChannelName
          channel={fieldChannel}
          namePath={`${path}.name`}
          id={Common.Task.getChannelNameID(itemKey)}
        />
        <Common.Task.EnableDisableButton path={`${path}.enabled`} />
      </Flex.Box>
    </Select.ListItem>
  );
};

// ─── Method Toggle ───
const MethodToggle: FC<{ path: string }> = ({ path }) => {
  const { get, set } = PForm.useContext();
  const method = get<string>(path).value;
  return (
    <Flex.Box x align="center" gap="small">
      <Text.Text level="small" weight={500} style={{ marginRight: "0.25rem" }}>
        Method
      </Text.Text>
      <Button.Toggle
        value={method === "GET"}
        onChange={() => set(path, "GET")}
        size="small"
        checkedVariant="filled"
        uncheckedVariant="text"
      >
        GET
      </Button.Toggle>
      <Button.Toggle
        value={method === "POST"}
        onChange={() => set(path, "POST")}
        size="small"
        checkedVariant="filled"
        uncheckedVariant="text"
      >
        POST
      </Button.Toggle>
    </Flex.Box>
  );
};

// ─── Field List ───

const FieldList: FC<{ epKey: string }> = ({ epKey }) => {
  const path = `config.endpoints.${epKey}.fields`;
  const { data: allData, push, remove } = PForm.useFieldList<string, ReadField>(path);
  const [selected, setSelected] = useState<string[]>([]);
  const ctx = PForm.useContext();
  const isSnapshot = Common.Task.useIsSnapshot();

  const allFields = PForm.useFieldValue<ReadField[]>(path);
  const indexKeys = new Set(
    allFields.filter((f) => f.isIndex).map((f) => f.key),
  );
  const data = allData.filter((key) => !indexKeys.has(key));

  const handleAdd = useCallback(() => {
    const fields = ctx.get<ReadField[]>(path).value;
    const nonIndex = fields.filter((f) => !f.isIndex);
    const last = nonIndex[nonIndex.length - 1];
    const field: ReadField = {
      ...(last != null
        ? { ...last, ...Common.Task.READ_CHANNEL_OVERRIDE }
        : ZERO_READ_FIELD),
      key: id.create(),
      isIndex: false,
    };
    push(field);
    setSelected([field.key]);
  }, [push, ctx, path]);

  const handleDuplicate = useCallback(
    (channels: ReadField[], keys: string[]) => {
      const duplicated = channels
        .filter(({ key }) => keys.includes(key))
        .map((ch) => ({
          ...ch,
          ...Common.Task.READ_CHANNEL_OVERRIDE,
          key: id.create(),
          isIndex: false,
        }));
      push(duplicated);
    },
    [push],
  );

  const listItem = useCallback(
    ({ key, ...p }: Common.Task.ChannelListItemProps) => (
      <FieldListItem {...p} key={key} epKey={epKey} />
    ),
    [epKey],
  );

  return (
    <BaseChannelList<ReadField>
      data={data}
      remove={remove}
      onDuplicate={handleDuplicate}
      onSelect={setSelected}
      selected={selected}
      path={path}
      header={
        <Header.Header>
          <Header.Title weight={500} color={10}>
            Fields
          </Header.Title>
          {!isSnapshot && (
            <Header.Actions>
              <Button.Button
                onClick={handleAdd}
                variant="text"
                contrast={2}
                tooltip="Add Field"
                sharp
              >
                <Icon.Add />
              </Button.Button>
            </Header.Actions>
          )}
        </Header.Header>
      }
      emptyContent={
        <EmptyAction
          message="No fields."
          action="Add a field"
          onClick={isSnapshot ? undefined : handleAdd}
        />
      }
      listItem={listItem}
      contextMenuItems={Common.Task.readChannelContextMenuItem}
    />
  );
};

// ─── Endpoint Details ───

const TimingToggle: FC<{ path: string }> = ({ path }) => {
  const fields = PForm.useFieldValue<ReadField[]>(`${path}.fields`);
  const { set } = PForm.useContext();
  const indexField = fields.find((f) => f.isIndex);
  const isValueTiming = indexField != null;

  const handleToggle = useCallback(
    (value: boolean) => {
      if (value && !isValueTiming) {
        const indexF: ReadField = {
          ...ZERO_READ_FIELD,
          key: id.create(),
          isIndex: true,
          timestampFormat: "unix_sec",
        };
        set(`${path}.fields`, [...fields, indexF]);
      } else if (!value && isValueTiming) 
        set(
          `${path}.fields`,
          fields.filter((f) => !f.isIndex),
        );
      
    },
    [fields, isValueTiming, path, set],
  );

  return (
    <Flex.Box y gap="small" style={{ padding: "0.5rem 1rem" }}>
      <Flex.Box x align="center" gap="small">
        <Text.Text level="small" weight={500} style={{ marginRight: "0.25rem" }}>
          Timing
        </Text.Text>
        <Button.Toggle
          value={!isValueTiming}
          onChange={() => handleToggle(false)}
          size="small"
          checkedVariant="filled"
          uncheckedVariant="text"
        >
          Software
        </Button.Toggle>
        <Button.Toggle
          value={isValueTiming}
          onChange={() => handleToggle(true)}
          size="small"
          checkedVariant="filled"
          uncheckedVariant="text"
        >
          Value
        </Button.Toggle>
      </Flex.Box>
      {isValueTiming && indexField != null && (
        <Flex.Box x align="center" gap="large">
          <PForm.TextField
            path={`${path}.fields.${indexField.key}.pointer`}
            label="Timestamp Pointer"
            inputProps={{ placeholder: "/timestamp" }}
            grow
          />
          <PForm.Field<string>
            path={`${path}.fields.${indexField.key}.timestampFormat`}
            label="Format"
            style={{ width: 160 }}
          >
            {({ value, onChange }) => (
              <Select.Static<string, Select.StaticEntry<string>>
                value={value ?? "unix_sec"}
                onChange={onChange}
                data={TIME_FORMAT_DATA}
                resourceName="time format"
              />
            )}
          </PForm.Field>
        </Flex.Box>
      )}
    </Flex.Box>
  );
};

const EndpointDetails: FC<{ epKey: string }> = ({ epKey }) => {
  const path = `config.endpoints.${epKey}`;
  const method = PForm.useFieldValue<string>(`${path}.method`);
  return (
    <Flex.Box y grow empty>
      <Flex.Box x align="center" gap="large" style={{ padding: "1rem" }}>
        <MethodToggle path={`${path}.method`} />
        <PForm.TextField
          path={`${path}.path`}
          label="Path"
          grow
          inputProps={{ placeholder: "/api/data" }}
        />
      </Flex.Box>
      {method === "POST" && (
        <Flex.Box style={{ padding: "0 1rem" }}>
          <PForm.TextField
            path={`${path}.body`}
            label="Request Body"
            grow
            inputProps={{ placeholder: '{"query": "latest"}' }}
          />
        </Flex.Box>
      )}
      <TimingToggle path={path} />
      <Divider.Divider x />
      <FieldList key={epKey} epKey={epKey} />
    </Flex.Box>
  );
};

// ─── Main Form ───

const Form: FC<
  Common.Task.FormProps<typeof readTypeZ, typeof readConfigZ, typeof readStatusDataZ>
> = () => {
  const [selectedEndpoints, setSelectedEndpoints] = useState<string[]>([]);
  const { data, push, remove } = PForm.useFieldList<string, ReadEndpoint>(
    "config.endpoints",
  );
  const ctx = PForm.useContext();
  const isSnapshot = Common.Task.useIsSnapshot();

  const handleAddEndpoint = useCallback(() => {
    const ep: ReadEndpoint = { ...ZERO_READ_ENDPOINT, key: id.create() };
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
      const allEndpoints = ctx.get<ReadEndpoint[]>("config.endpoints").value;
      const duplicated = allEndpoints
        .filter(({ key }) => keys.includes(key))
        .map((ep) => ({
          ...ep,
          key: id.create(),
          fields: ep.fields.map((f) => ({
            ...f,
            key: id.create(),
            channel: 0,
            name: "",
          })),
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
          <Select.Frame<string, ReadEndpoint>
            multiple
            data={data}
            value={selectedEndpoints}
            onChange={setSelectedEndpoints}
            replaceOnSingle
            allowNone={false}
            autoSelectOnNone
          >
            <List.Items<string, ReadEndpoint>
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

const getInitialValues: Common.Task.GetInitialValues<
  typeof readTypeZ,
  typeof readConfigZ,
  typeof readStatusDataZ
> = ({ deviceKey }) => ({
  ...ZERO_READ_PAYLOAD,
  config: {
    ...ZERO_READ_PAYLOAD.config,
    device: deviceKey ?? ZERO_READ_PAYLOAD.config.device,
  },
});

const onConfigure: Common.Task.OnConfigure<typeof readConfigZ> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: Device.SCHEMAS,
  });

  const safeName = channel.escapeInvalidName(dev.name);
  let modified = false;

  const props = dev.properties as Record<string, unknown>;
  const readProps = (props.read ?? {}) as Record<string, unknown>;
  const channelMap = (readProps.channels ?? {}) as Record<string, number>;
  const endpointIndices = (readProps.endpointIndices ?? {}) as Record<string, number>;

  try {
    for (const ep of config.endpoints) {
      // Resolve or create the index channel for this endpoint.
      // If a field is marked isIndex, it IS the index channel. Otherwise, create
      // a software-timed index.
      const indexField = ep.fields.find((f) => f.enabled && f.isIndex);
      let indexKey: number | undefined;

      if (indexField != null) {
        // The index field itself is a timestamp channel — create/retrieve it.
        const mapKey = `${ep.path}:${indexField.pointer}`;
        const existingKey = channelMap[mapKey];
        if (existingKey != null)
          try {
            await client.channels.retrieve(existingKey.toString());
            indexField.channel = existingKey;
            indexKey = existingKey;
          } catch (e) {
            if (!NotFoundError.matches(e)) throw e;
          }

        if (indexKey == null) {
          modified = true;
          const name = primitive.isNonZero(indexField.name)
            ? indexField.name
            : `${safeName}_${indexField.pointer.replace(/\//g, "_").replace(/^_/, "")}_time`;
          const idx = await client.channels.create({
            name,
            dataType: "timestamp",
            isIndex: true,
          });
          indexKey = idx.key;
          channelMap[mapKey] = idx.key;
          indexField.channel = idx.key;
        }
      } else {
        // No explicit index field — use a software-timed index per endpoint.
        indexKey = endpointIndices[ep.key] as number | undefined;
        if (indexKey != null)
          try {
            await client.channels.retrieve(indexKey.toString());
          } catch (e) {
            if (NotFoundError.matches(e)) indexKey = undefined;
            else throw e;
          }

        if (indexKey == null) {
          modified = true;
          const pathSlug = ep.path.replace(/\//g, "_").replace(/^_/, "");
          const idx = await client.channels.create({
            name: `${safeName}_${pathSlug}_time`,
            dataType: "timestamp",
            isIndex: true,
          });
          indexKey = idx.key;
        }
        endpointIndices[ep.key] = indexKey;
      }

      // Create/retrieve data channels for non-index fields.
      for (const field of ep.fields) {
        if (!field.enabled || field.isIndex) continue;
        const mapKey = `${ep.path}:${field.pointer}`;

        const existingKey = channelMap[mapKey];
        if (existingKey != null)
          try {
            await client.channels.retrieve(existingKey.toString());
            field.channel = existingKey;
            continue;
          } catch (e) {
            if (!NotFoundError.matches(e)) throw e;
          }

        modified = true;
        const ch = await client.channels.create({
          name: primitive.isNonZero(field.name)
            ? field.name
            : `${safeName}_${field.pointer.replace(/\//g, "_").replace(/^_/, "")}`,
          dataType: "float64",
          index: indexKey,
        });
        channelMap[mapKey] = ch.key;
        field.channel = ch.key;
      }
    }

    readProps.channels = channelMap;
    readProps.endpointIndices = endpointIndices;
    props.read = readProps;
  } finally {
    if (modified) await client.devices.create(dev, Device.SCHEMAS);
  }

  return [config, dev.rack];
};

export const Read = Common.Task.wrapForm({
  Properties,
  Form,
  schemas: READ_SCHEMAS,
  type: READ_TYPE,
  getInitialValues,
  onConfigure,
});
