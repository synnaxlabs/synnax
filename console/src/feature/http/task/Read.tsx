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
import { SCHEMAS } from "@/feature/http/device/types";
import { EndpointLabel } from "@/feature/http/task/EndpointLabel";
import {
  deployReadConfigZ,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadEndpoint,
  type ReadField,
  type ReadMethod,
  type ReadSchemas,
} from "@/feature/http/task/types";
import { CSS } from "@/platform/css";
import { Form as PlatformForm } from "@/platform/form";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const RATE_INPUT_PROPS = {
  endContent: "Hz",
  className: CSS.B("rate-input"),
} as const;

const NAN_FILL_INPUT_PROPS = {
  tooltip: "Write NaN for invalid floats instead of skipping the sample",
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
      <PForm.SwitchField
        label="NaN fill"
        path="config.nanFillEnabled"
        inputProps={NAN_FILL_INPUT_PROPS}
      />
    </Flex.Box>
  </>
);

const isTimingField = (f: ReadField): boolean => f.timeFormat != null;

/** The name configure gives a field's channel when the field carries none. */
const defaultChannelName = (devName: string, epPath: string, pointer: string): string =>
  channel.escapeInvalidName(devName) + channel.escapeInvalidName(epPath + pointer);

const useDefaultChannelName = (epKey: string, pointer: string): string => {
  const epPath = PForm.useFieldValue<string>(`config.endpoints.${epKey}.path`);
  const dev = useFromConfig();
  return dev == null ? "" : defaultChannelName(dev.name, epPath, pointer);
};

const POINTER_INPUT_PROPS = { placeholder: "/temperature" } as const;

const MethodSelect: FC<{ path: string; epPath: string }> = ({ path, epPath }) => {
  const { set } = PForm.useContext();
  const handleChange = (method: ReadMethod) => {
    set(path, method);
    // GET carries no request body. Clearing it here keeps a body typed under POST
    // from staying in the saved config, where it would set a content type on a
    // request that sends nothing.
    if (method !== "POST") set(`${epPath}.body`, "");
  };
  const renderMethod = (p: Select.ButtonsProps<ReadMethod>) => (
    <Select.Buttons<ReadMethod> {...p} onChange={handleChange}>
      <Select.Item<ReadMethod> itemKey="GET">GET</Select.Item>
      <Select.Item<ReadMethod> itemKey="POST">POST</Select.Item>
    </Select.Buttons>
  );
  return (
    <PForm.Field<ReadMethod> path={path} label="Method">
      {renderMethod}
    </PForm.Field>
  );
};

type TimingMode = "software" | "value";

const TimestampFields: FC<{ path: string }> = ({ path }) => {
  const fields = PForm.useFieldValue<ReadField[]>(`${path}.fields`);
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
      set(`${path}.fields`, [...fields, indexF]);
      set(`${path}.index`, indexF.key);
    } else if (mode === "software" && isValueTiming) {
      set(
        `${path}.fields`,
        fields.filter((f) => !isTimingField(f)),
      );
      set(`${path}.index`, "");
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
            path={`${path}.fields.${indexField.key}.pointer`}
            label="Pointer"
            padHelpText={false}
            inputProps={TIMESTAMP_POINTER_INPUT_PROPS}
          />
          <Task.Payload.TimeFormatField
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
            resourceName="parameter"
            label="Query parameters"
            keyField="parameter"
            keyPlaceholder="limit"
            valuePlaceholder="100"
          />
          <PlatformForm.KeyValueEditor
            path={`${path}.headers`}
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

const FieldPane: FC<{ epKey: string; fieldKey: string }> = ({ epKey, fieldKey }) => {
  const path = `config.endpoints.${epKey}.fields.${fieldKey}`;
  const { channel: fieldChannel, pointer } = PForm.useFieldValue<ReadField>(path);
  const defaultName = useDefaultChannelName(epKey, pointer);
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
          <Task.Payload.DataTypeField
            path={`${path}.dataType`}
            channel={fieldChannel}
          />
          <Task.ChannelNameField
            channel={fieldChannel}
            namePath={`${path}.name`}
            defaultName={defaultName}
          />
        </PForm.Section>
        <Task.Payload.EnumMappingSection path={`${path}.enumValues`} />
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

const endpointLabel = Component.renderProp(({ groupKey }: Task.Views.GroupProps) => (
  <EndpointLabel epKey={groupKey} />
));

const endpointDetails = Component.renderProp(({ groupKey }: Task.Views.GroupProps) => (
  <EndpointDetails epKey={groupKey} />
));

const fieldDetails = Component.renderProp(
  ({ groupKey, fieldKey }: Task.Views.FieldProps) => (
    <FieldPane epKey={groupKey} fieldKey={fieldKey} />
  ),
);

const Form: FC = () => {
  const endpoints = PForm.useFieldValue<ReadEndpoint[]>("config.endpoints");
  const read = useFromConfig()?.properties.read ?? null;
  return (
    <>
      <Task.Views.GroupTree<ReadEndpoint, ReadField>
        path="config.endpoints"
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
          path={`config.endpoints.${key}`}
          propertiesKeyField="path"
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
