// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/http/task/Form.css";

import { channel, http, query } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Channel, Telem } from "@synnaxlabs/pluto";
import { DataType, deep, unique } from "@synnaxlabs/x";
import { type FC, Fragment, useEffect, useState } from "react";

import { useFromConfig } from "@/feature/http/device/queries";
import { Select as SelectDevice } from "@/feature/http/device/Select";
import { MAKE, SCHEMAS } from "@/feature/http/device/types";
import {
  dataTypeOf,
  type FillField,
  formatFill,
  hasFill,
  matchKind,
  parseFill,
  retype,
} from "@/feature/http/task/adapter";
import { EndpointLabel } from "@/feature/http/task/EndpointLabel";
import {
  deployReadConfigZ,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadEndpoint,
  type ReadField,
  type ReadSchemas,
} from "@/feature/http/task/types";
import { CSS } from "@/platform/css";
import { Device } from "@/platform/device";
import { Form as PlatformForm } from "@/platform/form";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<ReadSchemas>>();

const RATE_INPUT_PROPS = {
  endContent: "Hz",
  className: CSS.B("rate-input"),
} as const;

const Properties = () => (
  <>
    <SelectDevice />
    <Flex.Box x grow wrap>
      <PForm.NumericField
        path={root.config.rate}
        label="Rate"
        inputProps={RATE_INPUT_PROPS}
      />
      <Task.Fields.NaNFill />
    </Flex.Box>
  </>
);

const isTimingField = (f: ReadField): boolean => f.type === "timestamp";

/** The name configure gives a field's channel when the field carries none. */
const defaultChannelName = (devName: string, epPath: string, pointer: string): string =>
  channel.escapeInvalidName(devName) + channel.escapeInvalidName(epPath + pointer);

const useDefaultChannelName = (endpoint: deep.Path<ReadEndpoint>, pointer: string) => {
  const epPath = PForm.useFieldValue(endpoint.path);
  const dev = useFromConfig();
  return dev == null ? "" : defaultChannelName(dev.name, epPath, pointer);
};

const POINTER_INPUT_PROPS = { placeholder: "/temperature" } as const;

const OPTIONAL_INPUT_PROPS = {
  tooltip:
    "Write the fill value with no warning when the value is missing. A float channel with no fill value writes NaN.",
} as const;

interface FillFieldsProps {
  path: deep.Path<ReadField>;
  field: FillField;
}

// Checked with no fill on a kind other than float is required until a fill commits.
const FillFields: FC<FillFieldsProps> = ({ path, field }) => {
  const { set } = PForm.useContext();
  const [checked, setChecked] = useState(field.fill != null);
  const [invalid, setInvalid] = useState<string | null>(null);
  const optional = checked || field.fill != null;
  const text = invalid ?? formatFill(field);
  const handleOptionalChange = (next: boolean) => {
    setChecked(next);
    setInvalid(null);
    if (!next) set(path, { ...field, fill: undefined });
    else if (field.type === "float") set(path, { ...field, fill: NaN });
  };
  const handleFillChange = (next: string) => {
    // A blur reports unedited text too, which must not retype the stored value.
    if (next === text) return;
    const parsed = parseFill(field, next);
    setInvalid(parsed == null ? next : null);
    if (parsed != null) set(path, parsed);
  };
  return (
    <Flex.Box x grow wrap>
      <Input.Item label="Optional" padHelpText>
        <Input.Checkbox
          value={optional}
          onChange={handleOptionalChange}
          {...OPTIONAL_INPUT_PROPS}
        />
      </Input.Item>
      {optional && (
        <Input.Item
          label="Fill value"
          padHelpText={false}
          status={invalid == null ? undefined : "error"}
          helpText={invalid == null ? undefined : `Not a valid ${field.type} value`}
        >
          <Input.Text
            value={text}
            onChange={handleFillChange}
            onlyChangeOnBlur
            placeholder={field.type === "float" ? "NaN" : "Required"}
            className={CSS.B("fill-value")}
          />
        </Input.Item>
      )}
    </Flex.Box>
  );
};

const HIDDEN_DATA_TYPES = [
  DataType.TIMESTAMP,
  DataType.UUID,
  DataType.JSON,
  DataType.BYTES,
];

const NO_HIDDEN_DATA_TYPES: DataType[] = [];

const NO_CHANGE = () => {};

// The kind follows the data type, so the select cannot bind a path of the union.
const DataTypeSelect: FC<{ path: deep.Path<ReadField> }> = ({ path }) => {
  const field = PForm.useFieldValue(path);
  const { set } = PForm.useContext();
  const { data: bound } = Channel.useResultDataType(
    field.channel === 0 ? null : { key: field.channel },
  );
  if (bound != null)
    return (
      <Input.Item label="Data type" padHelpText={false} helpText="Set on the channel">
        <Telem.SelectDataType
          value={bound.toString()}
          onChange={NO_CHANGE}
          hideDataTypes={NO_HIDDEN_DATA_TYPES}
          disabled
          location="bottom"
        />
      </Input.Item>
    );
  return (
    <Input.Item label="Data type" padHelpText={false}>
      <Telem.SelectDataType
        value={dataTypeOf(field).toString()}
        onChange={(next: string) => set(path, retype(field, new DataType(next)))}
        hideDataTypes={HIDDEN_DATA_TYPES}
        location="bottom"
      />
    </Input.Item>
  );
};

/**
 * Gives each field bound to a channel the kind of the channel's data type, which the
 * Core requires. Renders nothing.
 */
const BindKinds: FC<{ path: deep.Path<ReadField[]> }> = ({ path }) => {
  const fields = PForm.useFieldValue(path);
  const { set } = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const keys = unique.unique(fields.map((f) => f.channel).filter((k) => k !== 0));
  const { data: channels } = Channel.useResultMultiple(
    isPreview || keys.length === 0 ? null : query.keysRequest(keys),
  );
  useEffect(() => {
    if (channels == null) return;
    const dataTypes = new Map(channels.map((ch) => [ch.key, ch.dataType]));
    const next = fields.map((f) => matchKind(f, dataTypes.get(f.channel)));
    if (next.some((f, i) => f !== fields[i])) set(path, next);
  }, [channels, fields, path, set]);
  return null;
};

const ENUM_LABEL_PLACEHOLDER = "String (e.g. ON)";

const createBooleanRow = (rows: Input.TableCell[][]): Input.TableCell[] => [
  Task.Payload.createLabel(rows.map(([label]) => String(label))),
  0,
];

// The table holds numbers, so a boolean value shows as 0 or 1.
const BooleanEnumSection: FC<{
  path: deep.Path<Extract<ReadField, { type: "boolean" }>>;
}> = ({ path }) => {
  const entries = PForm.useFieldValue(path.enumValues);
  const { set } = PForm.useContext();
  const preview = Task.useIsPreview() ? true : undefined;
  const handleChange = (rows: Input.TableCell[][]) =>
    set(
      path.enumValues,
      rows.map(([label, value]) => ({ label: String(label), value: value !== 0 })),
    );
  return (
    <PForm.Section title="Enum mapping">
      <Input.Item padHelpText={false}>
        <Input.Table
          value={entries.map(({ label, value }) => [label, value ? 1 : 0])}
          onChange={handleChange}
          preview={preview}
          resourceName="mapping"
          createRow={createBooleanRow}
        >
          <Input.TableColumn key="label" name="Label" type="string">
            {(p) => <Input.Text {...p} placeholder={ENUM_LABEL_PLACEHOLDER} />}
          </Input.TableColumn>
          <Input.TableColumn key="value" name="Value" />
        </Input.Table>
      </Input.Item>
    </PForm.Section>
  );
};

interface EnumSectionProps {
  path: deep.Path<ReadField>;
  field: ReadField;
}

const EnumSection: FC<EnumSectionProps> = ({ path, field }) => {
  switch (field.type) {
    case "float":
      return (
        <Task.Payload.EnumMappingSection
          path={deep.variant(path, "type", "float").enumValues}
        />
      );
    case "integer":
    case "unsigned":
    case "string":
      return (
        <Task.Payload.EnumMappingSection
          path={deep.variant(path, "type", field.type).enumValues}
          valueType="string"
          defaultValue={field.type === "string" ? "" : "0"}
        />
      );
    case "boolean":
      return <BooleanEnumSection path={deep.variant(path, "type", "boolean")} />;
    default:
      return null;
  }
};

const MethodSelect: FC<{ path: deep.Path<ReadEndpoint> }> = ({ path }) => {
  const { set } = PForm.useContext();
  const handleChange = (method: http.Method) => {
    set(path.method, method);
    // GET carries no request body. Clearing it here keeps a body typed under POST
    // from staying in the saved config, where it would set a content type on a
    // request that sends nothing.
    if (method !== "POST") set(path.body, "");
  };
  const renderMethod = (p: Select.ButtonsProps<http.Method>) => (
    <Select.Buttons<http.Method> {...p} onChange={handleChange}>
      <Select.Item<http.Method> itemKey="GET">GET</Select.Item>
      <Select.Item<http.Method> itemKey="POST">POST</Select.Item>
    </Select.Buttons>
  );
  return (
    <PForm.Field path={path.method} label="Method">
      {renderMethod}
    </PForm.Field>
  );
};

type TimingMode = "software" | "value";

const TimestampFields: FC<{ path: deep.Path<ReadEndpoint> }> = ({ path }) => {
  const fields = PForm.useFieldValue(path.fields);
  const { set } = PForm.useContext();
  const indexField = fields.find(isTimingField);
  const isValueTiming = indexField != null;

  const handleChange = (mode: TimingMode) => {
    if (mode === "value" && !isValueTiming) {
      const indexF = http.timestampReadFieldZ.parse({
        type: "timestamp",
        timeFormat: "unix_sec",
      });
      set(path.fields, [...fields, indexF]);
      set(path.index, indexF.key);
    } else if (mode === "software" && isValueTiming) {
      set(
        path.fields,
        fields.filter((f) => !isTimingField(f)),
      );
      set(path.index, "");
    }
  };

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
            path={path.fields[indexField.key].pointer}
            label="Pointer"
            padHelpText={false}
            inputProps={TIMESTAMP_POINTER_INPUT_PROPS}
          />
          <Task.Payload.TimeFormatField
            path={
              deep.variant(path.fields[indexField.key], "type", "timestamp").timeFormat
            }
            label="Format"
          />
        </>
      )}
    </>
  );
};

const TIMESTAMP_POINTER_INPUT_PROPS = { placeholder: "/timestamp" } as const;

const EndpointDetails: FC<{ path: deep.Path<ReadEndpoint> }> = ({ path }) => {
  const method = PForm.useFieldValue(path.method);
  return (
    <Flex.Box y grow empty className={CSS.B("endpoint-details")}>
      <PForm.Sections className={CSS.B("endpoint-form")}>
        <PForm.Section title="Request">
          <MethodSelect path={path} />
          <PForm.TextField
            path={path.path}
            label="Path"
            padHelpText={false}
            inputProps={PATH_INPUT_PROPS}
          />
          {method === "POST" && (
            <PForm.TextField
              path={path.body}
              label="Body"
              padHelpText={false}
              inputProps={REQUEST_BODY_INPUT_PROPS}
            />
          )}
          <PlatformForm.KeyValueEditor
            path={path.queryParams}
            resourceName="parameter"
            label="Query parameters"
            keyField="parameter"
            keyPlaceholder="limit"
            valuePlaceholder="100"
          />
          <PlatformForm.KeyValueEditor
            path={path.headers}
            resourceName="header"
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

interface FieldPaneProps {
  endpoint: deep.Path<ReadEndpoint>;
  path: deep.Path<ReadField>;
}

const FieldPane: FC<FieldPaneProps> = ({ endpoint, path }) => {
  const field = PForm.useFieldValue(path);
  const defaultName = useDefaultChannelName(endpoint, field.pointer);
  return (
    <Flex.Box y grow empty className={CSS.B("endpoint-details")}>
      <PForm.Sections className={CSS.cls(CSS.B("endpoint-form"), CSS.B("field-form"))}>
        <PForm.Section title="Field">
          <PForm.TextField
            path={path.pointer}
            label="Pointer"
            padHelpText={false}
            inputProps={POINTER_INPUT_PROPS}
          />
          <DataTypeSelect path={path} />
          <Task.ChannelNameField
            channel={field.channel}
            namePath={path.name}
            defaultName={defaultName}
            skipped={field.disabled}
          />
          {hasFill(field) && (
            <FillFields
              key={`${field.key}-${dataTypeOf(field).toString()}`}
              path={path}
              field={field}
            />
          )}
        </PForm.Section>
        <EnumSection path={path} field={field} />
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

const createEndpoints = async (): Promise<ReadEndpoint[]> => [
  http.readEndpointZ.parse({}),
];

const createField = (): Omit<ReadField, "key"> =>
  http.floatReadFieldZ.parse({ type: "float", dataType: "float64" });

const endpointLabel = Component.renderProp(
  ({ path }: Task.Views.GroupProps<ReadEndpoint>) => <EndpointLabel path={path} />,
);

const endpointDetails = Component.renderProp(
  ({ path }: Task.Views.GroupProps<ReadEndpoint>) => <EndpointDetails path={path} />,
);

const fieldDetails = Component.renderProp(
  ({ groupKey, path }: Task.Views.FieldProps<ReadField>) => (
    <FieldPane endpoint={root.config.endpoints[groupKey]} path={path} />
  ),
);

// BindReadChannels reads only the base fields, so this path types each field as a
// ReadChannelField.
const sources = deep.path<{
  config: { endpoints: { index: string; fields: Task.ReadChannelField[] }[] };
}>().config.endpoints;

const Form: FC = () => {
  const endpoints = PForm.useFieldValue(root.config.endpoints);
  const read = useFromConfig()?.properties.read ?? null;
  return (
    <>
      <Task.Views.GroupTree<ReadEndpoint, ReadField>
        path={root.config.endpoints}
        listTitle="Endpoints"
        emptyMessage="No endpoints"
        placeholder="Select an endpoint or field to configure"
        addLabel="Add endpoint"
        createGroups={createEndpoints}
        createField={createField}
        groupLabel={endpointLabel}
        groupDetails={endpointDetails}
        fieldDetails={fieldDetails}
      />
      {endpoints.map(({ key }) => (
        <Fragment key={key}>
          <Task.BindReadChannels
            path={sources[key]}
            propertiesKey={root.config.endpoints[key].path}
            read={read}
          />
          <BindKinds path={root.config.endpoints[key].fields} />
        </Fragment>
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
    schemas: SCHEMAS,
  });
  const safeDevName = channel.escapeInvalidName(dev.name);
  const before = deep.copy(dev.properties);
  try {
    for (const ep of config.endpoints) {
      const specs = ep.fields.map((field) => ({
        ...field,
        dataType: dataTypeOf(field).toString(),
        timing: isTimingField(field),
      }));
      await Task.configureReadChannels({
        client,
        read: dev.properties.read,
        propertiesKey: ep.path,
        fields: specs,
        namePrefix: `${safeDevName}${channel.escapeInvalidName(ep.path)}`,
        indexKey: ep.index,
        indexed: (f) => !f.timing && !new DataType(f.dataType).isVariable,
      });
      const keys = specs.map((s) => s.channel).filter((k) => k !== 0);
      const channels = keys.length === 0 ? [] : await client.channels.retrieve(keys);
      const dataTypes = new Map(channels.map((ch) => [ch.key, ch.dataType]));
      ep.fields = ep.fields.map((field, i) =>
        matchKind(
          { ...field, channel: specs[i].channel },
          dataTypes.get(specs[i].channel),
        ),
      );
    }
  } finally {
    await Device.setProperties(client, dev, before, SCHEMAS);
  }
  return [config, dev.rack];
};

export const Read = Task.wrapForm({
  Properties,
  Form,
  schemas: READ_SCHEMAS,
  deployConfigZ: deployReadConfigZ,
  type: "http_read",
  onConfigure,
});

export const useCreateRead = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const ReadSelectable: Selector.Selectable = {
  type: READ_TYPE,
  title: "HTTP read task",
  label: "Read",
  icon: <Icon.Logo.HTTP />,
  useOnSelect: useCreateRead,
};
