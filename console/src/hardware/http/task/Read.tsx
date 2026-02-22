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
  Divider,
  Flex,
  Form as PForm,
  Header,
  Icon,
  Select,
  Text,
} from "@synnaxlabs/pluto";
import { id, primitive } from "@synnaxlabs/x";
import { type FC, useCallback, useState } from "react";

import { Common } from "@/hardware/common";
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

const EndpointListEntry: FC<{
  epKey: string;
  selected: boolean;
  onSelect: (key: string) => void;
}> = ({ epKey, selected, onSelect }) => {
  const method = PForm.useFieldValue<string>(`config.endpoints.${epKey}.method`);
  const epPath = PForm.useFieldValue<string>(`config.endpoints.${epKey}.path`);
  const fields = PForm.useFieldValue<ReadField[]>(`config.endpoints.${epKey}.fields`);
  return (
    <Flex.Box
      x
      align="center"
      style={{
        padding: "0.5rem 1rem",
        cursor: "pointer",
        background: selected ? "var(--pluto-primary-z-20)" : undefined,
      }}
      onClick={() => onSelect(epKey)}
    >
      <Flex.Box grow>
        <Text.Text level="small">
          {method} {epPath || "(no path)"}
        </Text.Text>
      </Flex.Box>
      <Text.Text level="small" style={{ opacity: 0.5 }}>
        {fields?.length ?? 0}
      </Text.Text>
    </Flex.Box>
  );
};

const TIME_FORMAT_DATA: Select.StaticEntry<string>[] = [
  { key: "iso8601", name: "ISO 8601" },
  { key: "unix_sec", name: "Unix (s)" },
  { key: "unix_ms", name: "Unix (ms)" },
  { key: "unix_us", name: "Unix (μs)" },
  { key: "unix_ns", name: "Unix (ns)" },
];

const FieldItem: FC<{ path: string }> = ({ path }) => {
  const isIndex = PForm.useFieldValue<boolean>(`${path}.isIndex`);

  return (
    <Flex.Box y style={{ padding: "0.5rem 1rem", borderBottom: "var(--pluto-border)" }}>
      <Flex.Box x align="start" gap="large" wrap>
        <PForm.TextField
          path={`${path}.pointer`}
          label="JSON Pointer"
          inputProps={{ placeholder: "/temperature" }}
          style={{ minWidth: 150 }}
          grow
        />
        <Common.Task.ChannelName
          channel={PForm.useFieldValue<number>(`${path}.channel`)}
          namePath={`${path}.name`}
          id={Common.Task.getChannelNameID(path)}
        />
        <Common.Task.EnableDisableButton path={`${path}.enabled`} />
      </Flex.Box>
      <Flex.Box x align="center" gap="large">
        <PForm.SwitchField path={`${path}.isIndex`} label="Index" size="small" />
        {isIndex && (
          <PForm.Field<string>
            path={`${path}.timestampFormat`}
            label="Format"
            style={{ width: 140 }}
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
        )}
      </Flex.Box>
    </Flex.Box>
  );
};

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

const EndpointDetails: FC<{ path: string }> = ({ path }) => {
  const { push, data: fieldKeys } = PForm.useFieldList<string, ReadField>(
    `${path}.fields`,
  );

  const handleAddField = useCallback(() => {
    push({
      ...ZERO_READ_FIELD,
      key: id.create(),
    });
  }, [push]);

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
      <Divider.Divider x />
      <Header.Header>
        <Header.Title weight={500} wrap={false} color={10}>
          Fields
        </Header.Title>
        <Header.Actions>
          <Button.Button variant="text" size="small" onClick={handleAddField}>
            <Icon.Add />
          </Button.Button>
        </Header.Actions>
      </Header.Header>
      <Flex.Box y grow style={{ overflowY: "auto" }}>
        {fieldKeys.map((fk) => (
          <FieldItem key={fk} path={`${path}.fields.${fk}`} />
        ))}
        {fieldKeys.length === 0 && (
          <Flex.Box align="center" justify="center" grow>
            <Text.Text level="small" style={{ opacity: 0.5 }}>
              No fields. Click + to add one.
            </Text.Text>
          </Flex.Box>
        )}
      </Flex.Box>
    </Flex.Box>
  );
};

const Form: FC<
  Common.Task.FormProps<typeof readTypeZ, typeof readConfigZ, typeof readStatusDataZ>
> = () => {
  const [selected, setSelected] = useState<string[]>([]);
  const { push, data: endpointKeys } = PForm.useFieldList<string, ReadEndpoint>(
    "config.endpoints",
  );

  const handleAddEndpoint = useCallback(() => {
    push({
      ...ZERO_READ_ENDPOINT,
      key: id.create(),
    });
  }, [push]);

  const handleSelect = useCallback((key: string) => {
    setSelected([key]);
  }, []);

  return (
    <Flex.Box x grow empty>
      <Flex.Box y style={{ width: 250, flexShrink: 0 }}>
        <Header.Header>
          <Header.Title weight={500} wrap={false} color={10}>
            Endpoints
          </Header.Title>
          <Header.Actions>
            <Button.Button variant="text" size="small" onClick={handleAddEndpoint}>
              <Icon.Add />
            </Button.Button>
          </Header.Actions>
        </Header.Header>
        <Flex.Box y grow style={{ overflowY: "auto" }}>
          {endpointKeys.map((ek) => (
            <EndpointListEntry
              key={ek}
              epKey={ek}
              selected={selected.includes(ek)}
              onSelect={handleSelect}
            />
          ))}
          {endpointKeys.length === 0 && (
            <Flex.Box align="center" justify="center" grow>
              <Text.Text level="small" style={{ opacity: 0.5 }}>
                No endpoints. Click + to add one.
              </Text.Text>
            </Flex.Box>
          )}
        </Flex.Box>
      </Flex.Box>
      <Divider.Divider y />
      {selected.length > 0 ? (
        <EndpointDetails path={`config.endpoints.${selected[0]}`} />
      ) : (
        <Flex.Box y grow align="center" justify="center">
          <Text.Text level="small" style={{ opacity: 0.5 }}>
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
