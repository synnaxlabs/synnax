// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/http/task/Form.css";

import {
  channel,
  http,
  NotFoundError,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Access, Channel as PChannel, Telem } from "@synnaxlabs/pluto";
import { DataType, errors, id, primitive } from "@synnaxlabs/x";
import { type FC } from "react";

import { useFromConfig } from "@/feature/http/device/queries";
import { Select as SelectDevice } from "@/feature/http/device/Select";
import { SCHEMAS } from "@/feature/http/device/types";
import { EndpointLabel } from "@/feature/http/task/EndpointLabel";
import { TimeFormatField } from "@/feature/http/task/TimeFormatField";
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
      <PForm.CheckboxField
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

const HIDDEN_DATA_TYPES = [
  DataType.TIMESTAMP,
  DataType.UUID,
  DataType.JSON,
  DataType.BYTES,
];

const renderTelemSelectDataType = Component.renderProp(
  (p: Telem.SelectDataTypeProps) => (
    <Telem.SelectDataType {...p} hideDataTypes={HIDDEN_DATA_TYPES} location="bottom" />
  ),
);

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

/**
 * Binds an endpoint's fields to the channels the device stores for its path. Mounted by
 * the form for every endpoint, as the details pane renders only the selected one.
 */
const FieldBinder = ({ epKey }: { epKey: string }) => {
  const dev = useFromConfig();
  const epPath = PForm.useFieldValue<string>(`config.endpoints.${epKey}.path`);
  const indexKey = PForm.useFieldValue<string>(`config.endpoints.${epKey}.index`);
  const resolve = (field: ReadField) => {
    if (dev == null) return null;
    const props = dev.properties.read[epPath];
    if (props == null) return { channel: 0 };
    return {
      channel:
        field.key === indexKey ? props.index : (props.channels[field.pointer] ?? 0),
    };
  };
  return (
    <Task.BindChannels<ReadField>
      path={`config.endpoints.${epKey}.fields`}
      resolve={resolve}
    />
  );
};

interface ChannelNameFieldProps {
  path: string;
  channel: channel.Key;
  defaultName: string;
}

/**
 * Names a field's channel. Until configure creates the channel, the name lives on the
 * field; after, an edit renames the channel itself.
 */
const ChannelNameField = ({
  path,
  channel: key,
  defaultName,
}: ChannelNameFieldProps) =>
  key === 0 ? (
    <PForm.TextField
      path={`${path}.name`}
      label="Channel"
      padHelpText={false}
      inputProps={{ placeholder: defaultName === "" ? "Channel name" : defaultName }}
    />
  ) : (
    <ExistingChannelNameField channel={key} />
  );

const ExistingChannelNameField = ({ channel: key }: { channel: channel.Key }) => {
  const { data: name = "" } = PChannel.useResultName({ key });
  const { update } = PChannel.useRename();
  const canRename = Access.useUpdateGranted(channel.TYPE_ONTOLOGY_ID);
  const isPreview = Task.useIsPreview();
  const handleChange = (next: string) => {
    if (next.length > 0 && next !== name) update({ key, name: next });
  };
  return (
    <Input.Item label="Channel" padHelpText={false}>
      <Input.Text
        value={name}
        onChange={handleChange}
        onlyChangeOnBlur
        disabled={isPreview || !canRename}
      />
    </Input.Item>
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
          <TimeFormatField
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
  const bound = fieldChannel !== 0;
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
          <PForm.Field<string>
            path={`${path}.dataType`}
            label="Data type"
            padHelpText={false}
            helpText={bound ? "Set on the channel" : undefined}
          >
            {(p) => renderTelemSelectDataType({ ...p, disabled: bound })}
          </PForm.Field>
          <ChannelNameField
            path={path}
            channel={fieldChannel}
            defaultName={defaultName}
          />
        </PForm.Section>
        <PForm.Section title="Enum mapping">
          <PlatformForm.KeyValueEditor
            path={`${path}.enumValues`}
            resourceName="mapping"
            keyField="label"
            keyPlaceholder="String (e.g. ON)"
            valueType="number"
          />
        </PForm.Section>
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

const createEndpoint = (): Omit<ReadEndpoint, "key"> => http.readEndpointZ.parse({});

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
  return (
    <>
      <Task.Views.GroupTree<ReadEndpoint, ReadField>
        path="config.endpoints"
        listTitle="Endpoints"
        groupName="endpoint"
        placeholder="Select an endpoint or field to configure"
        createGroup={createEndpoint}
        createField={createField}
        groupLabel={endpointLabel}
        groupDetails={endpointDetails}
        fieldDetails={fieldDetails}
      />
      {endpoints.map(({ key }) => (
        <FieldBinder key={key} epKey={key} />
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

const retrieveChannel = async (
  client: Client,
  key: channel.Key,
): Promise<channel.Channel | null> => {
  try {
    return await client.channels.retrieve(key);
  } catch (e) {
    if (NotFoundError.matches(e)) return null;
    throw errors.fromUnknown(e);
  }
};

const channelExists = async (client: Client, key: channel.Key): Promise<boolean> =>
  (await retrieveChannel(client, key)) != null;

const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (client, config) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: SCHEMAS,
  });
  const safeDevName = channel.escapeInvalidName(dev.name);
  let modified = false;
  try {
    for (const ep of config.endpoints) {
      dev.properties.read[ep.path] ??= { index: 0, channels: {} };
      const epProps = dev.properties.read[ep.path];

      const needsIndex = ep.fields.some(
        (f) => !isTimingField(f) && !new DataType(f.dataType).isVariable,
      );

      if (needsIndex) {
        let shouldCreateIndex = !primitive.isNonZero(epProps.index);
        shouldCreateIndex ||= !(await channelExists(client, epProps.index));
        if (shouldCreateIndex) {
          // check if any existing data channels share an index we can reuse
          let recoveredIndex = 0;
          for (const storedKey of Object.values(epProps.channels)) {
            if (!primitive.isNonZero(storedKey)) continue;
            const ch = await retrieveChannel(client, storedKey);
            if (ch != null && primitive.isNonZero(ch.index)) {
              const indexCh = await retrieveChannel(client, ch.index);
              if (indexCh != null) {
                recoveredIndex = ch.index;
                break;
              }
            }
          }
          if (primitive.isNonZero(recoveredIndex)) {
            epProps.index = recoveredIndex;
            modified = true;
          } else {
            modified = true;
            const newIndexCh = await client.channels.create({
              name: `${safeDevName}${channel.escapeInvalidName(ep.path)}_time`,
              dataType: "timestamp",
              isIndex: true,
            });
            epProps.index = newIndexCh.key;
          }
        }
      }

      const potentialTimingKey = ep.index;
      for (const field of ep.fields) {
        if (field.key === potentialTimingKey && epProps.index !== 0) {
          field.channel = epProps.index;
          continue;
        }

        if (field.channel !== 0 && (await channelExists(client, field.channel)))
          continue;

        const storedKey = epProps.channels[field.pointer];
        if (
          primitive.isNonZero(storedKey) &&
          (await channelExists(client, storedKey))
        ) {
          field.channel = storedKey;
          continue;
        }

        const dt = new DataType(field.dataType);
        const chName = primitive.isNonZero(field.name)
          ? field.name
          : defaultChannelName(dev.name, ep.path, field.pointer);
        const newCh = await client.channels.create({
          name: chName,
          dataType: field.dataType,
          ...(dt.isVariable ? { virtual: true } : { index: epProps.index }),
        });
        modified = true;
        field.channel = newCh.key;
        epProps.channels[field.pointer] = newCh.key;
      }
    }
  } finally {
    if (modified) await client.devices.create(dev, SCHEMAS);
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
