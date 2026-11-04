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
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { DataType, deep, id } from "@synnaxlabs/x";
import { type FC } from "react";

import { useFromConfig } from "@/feature/http/device/queries";
import { Select as SelectDevice } from "@/feature/http/device/Select";
import { MAKE, SCHEMAS } from "@/feature/http/device/types";
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
import { Form as PlatformForm } from "@/platform/form";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<ReadSchemas>>();

const RATE_INPUT_PROPS = {
  endContent: "Hz",
  className: CSS.B("rate-input"),
} as const;

const Properties = () => (
  <>
    <SelectDevice />
    <Flex.Box x grow>
      <PForm.NumericField
        path={root.config.rate}
        label="Rate"
        inputProps={RATE_INPUT_PROPS}
      />
      <Task.Fields.NaNFill />
    </Flex.Box>
  </>
);

const isTimingField = (f: ReadField): boolean => f.timeFormat != null;

/** The name configure gives a field's channel when the field carries none. */
const defaultChannelName = (devName: string, epPath: string, pointer: string): string =>
  channel.escapeInvalidName(devName) + channel.escapeInvalidName(epPath + pointer);

const useDefaultChannelName = (endpoint: deep.Path<ReadEndpoint>, pointer: string) => {
  const epPath = PForm.useFieldValue(endpoint.path);
  const dev = useFromConfig();
  return dev == null ? "" : defaultChannelName(dev.name, epPath, pointer);
};

const POINTER_INPUT_PROPS = { placeholder: "/temperature" } as const;

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
      const indexF: ReadField = {
        ...http.readFieldZ.parse({}),
        key: id.create(),
        timeFormat: "unix_sec",
      };
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
            path={path.fields[indexField.key].timeFormat}
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
  const { channel: fieldChannel, pointer } = PForm.useFieldValue(path);
  const defaultName = useDefaultChannelName(endpoint, pointer);
  return (
    <Flex.Box y grow empty className={CSS.B("endpoint-details")}>
      <PForm.Sections className={CSS.B("endpoint-form")}>
        <PForm.Section title="Field">
          <PForm.TextField
            path={path.pointer}
            label="Pointer"
            padHelpText={false}
            inputProps={POINTER_INPUT_PROPS}
          />
          <Task.Payload.DataTypeField path={path.dataType} channel={fieldChannel} />
          <Task.ChannelNameField
            channel={fieldChannel}
            namePath={path.name}
            defaultName={defaultName}
          />
        </PForm.Section>
        <Task.Payload.EnumMappingSection path={path.enumValues} />
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

const createField = (): Omit<ReadField, "key"> => http.readFieldZ.parse({});

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
        <Task.BindReadChannels
          key={key}
          path={root.config.endpoints[key]}
          propertiesKey={root.config.endpoints[key].path}
          read={read}
        />
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
  const saved = deep.copy(dev.properties);
  try {
    for (const ep of config.endpoints)
      await Task.configureReadChannels({
        client,
        read: dev.properties.read,
        propertiesKey: ep.path,
        fields: ep.fields,
        namePrefix: `${safeDevName}${channel.escapeInvalidName(ep.path)}`,
        indexKey: ep.index,
        indexed: (f) => !isTimingField(f) && !new DataType(f.dataType).isVariable,
      });
  } finally {
    if (!deep.equal(saved, dev.properties)) await client.devices.create(dev, SCHEMAS);
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

export const ReadSelectable = Selector.createSelectable({
  type: READ_TYPE,
  title: "HTTP read task",
  icon: <Icon.Logo.HTTP />,
  useOnSelect: useCreateRead,
});
