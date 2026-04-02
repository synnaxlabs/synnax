// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/http/task/Form.css";

import { channel, NotFoundError, type Synnax as Client } from "@synnaxlabs/client";
import {
  Button,
  Component,
  Divider,
  Flex,
  Form as PForm,
  Header,
  Icon,
  Input,
  List,
  Menu as PMenu,
  Select,
  Telem,
  Text,
} from "@synnaxlabs/pluto";
import { DataType, id, primitive } from "@synnaxlabs/x";
import { type FC, useCallback, useState } from "react";

import { EmptyAction } from "@/components";
import { KeyValueEditor } from "@/components/form/KeyValueEditor";
import { CSS } from "@/css";
import { Common } from "@/hardware/common";
import { ChannelList as BaseChannelList } from "@/hardware/common/task/ChannelList";
import { Device } from "@/hardware/http/device";
import { ContextMenu } from "@/hardware/http/task/ContextMenu";
import { EndpointListItem } from "@/hardware/http/task/EndpointListItem";
import {
  READ_SCHEMAS,
  READ_TYPE,
  type ReadEndpoint,
  type ReadField,
  type ReadMethod,
  type ReadPayload,
  type ReadSchemas,
  type TimeFormat,
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

const RATE_INPUT_PROPS = {
  endContent: "Hz",
  className: CSS.B("rate-input"),
} as const;

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
      <Common.Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

const ReadEndpointListItem = (props: List.ItemProps<string>) => {
  const { itemKey } = props;
  const fields = PForm.useFieldValue<ReadField[]>(`config.endpoints.${itemKey}.fields`);
  return (
    <EndpointListItem
      {...props}
      extra={
        <Text.Text level="small" color={7}>
          {fields.length}
        </Text.Text>
      }
    />
  );
};

const readEndpointListItem = Component.renderProp(ReadEndpointListItem);

const TIME_FORMAT_DATA: Select.StaticEntry<TimeFormat>[] = [
  { key: "iso8601", name: "ISO 8601" },
  { key: "unix_sec", name: "Unix (s)" },
  { key: "unix_ms", name: "Unix (ms)" },
  { key: "unix_us", name: "Unix (μs)" },
  { key: "unix_ns", name: "Unix (ns)" },
];

const isTimingField = (f: ReadField): boolean => f.timestampFormat != null;

interface FieldListItemProps extends Common.Task.ChannelListItemProps {
  epKey: string;
}

const FieldListItem = ({ epKey, ...props }: FieldListItemProps) => {
  const { itemKey } = props;
  const path = `config.endpoints.${epKey}.fields.${itemKey}`;
  const fieldChannel = PForm.useFieldValue<number>(`${path}.channel`);
  const enumValues = PForm.useFieldValue<Record<string, number>[]>(
    `${path}.enumValues`,
    { defaultValue: [] },
  );
  const enumCount = enumValues.length;
  const enumCountText =
    enumCount === 0 ? "" : `${enumCount} enum${enumCount === 1 ? "" : "s"}`;
  return (
    <Select.ListItem {...props} justify="between" align="center" x>
      <PForm.TextField
        path={`${path}.pointer`}
        showLabel={false}
        showHelpText={false}
        inputProps={POINTER_INPUT_PROPS}
        grow
      />
      {fieldChannel === 0 && (
        <PForm.Field<string>
          path={`${path}.dataType`}
          showLabel={false}
          showHelpText={false}
          hideIfNull
        >
          {renderTelemSelectDataType}
        </PForm.Field>
      )}
      {enumCountText !== "" && (
        <Text.Text level="small" color={7}>
          {enumCountText}
        </Text.Text>
      )}
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

const METHOD_KEYS: ReadMethod[] = ["GET", "POST"];

const MethodSelect: FC<{ path: string; epPath: string }> = ({ path, epPath }) => {
  const { set } = PForm.useContext();
  const handleChange = useCallback(
    (method: ReadMethod) => {
      set(path, method);
      if (method === "POST") set(`${epPath}.body`, "");
    },
    [set, path, epPath],
  );
  const renderMethod = useCallback(
    (p: Omit<Select.ButtonsProps<ReadMethod>, "keys">) => (
      <Select.Buttons<ReadMethod> {...p} onChange={handleChange} keys={METHOD_KEYS}>
        <Select.Button<ReadMethod> itemKey="GET">GET</Select.Button>
        <Select.Button<ReadMethod> itemKey="POST">POST</Select.Button>
      </Select.Buttons>
    ),
    [handleChange],
  );
  return (
    <PForm.Field<ReadMethod> path={path} label="Method">
      {renderMethod}
    </PForm.Field>
  );
};

interface FieldListProps {
  epKey: string;
}

const FieldList = ({ epKey }: FieldListProps) => {
  const path = `config.endpoints.${epKey}.fields`;
  const { data: allData, push, remove } = PForm.useFieldList<string, ReadField>(path);
  const [selected, setSelected] = useState<string[]>([]);
  const ctx = PForm.useContext();
  const isSnapshot = Common.Task.useIsSnapshot();

  const allFields = PForm.useFieldValue<ReadField[]>(path);
  const indexKeys = new Set(allFields.filter(isTimingField).map((f) => f.key));
  const data = allData.filter((key) => !indexKeys.has(key));

  const handleAdd = useCallback(() => {
    const fields = ctx.get<ReadField[]>(path).value;
    const nonIndex = fields.filter((f) => !isTimingField(f));
    const last = nonIndex[nonIndex.length - 1];
    const field: ReadField = {
      ...(last != null
        ? { ...last, ...Common.Task.READ_CHANNEL_OVERRIDE }
        : ZERO_READ_FIELD),
      key: id.create(),
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

  const selectedFieldKey = selected.length === 1 ? selected[0] : null;
  const selectedFieldPath =
    selectedFieldKey != null ? `${path}.${selectedFieldKey}.enumValues` : null;

  return (
    <>
      <BaseChannelList<ReadField>
        data={data}
        remove={remove}
        onDuplicate={handleDuplicate}
        onSelect={setSelected}
        selected={selected}
        path={path}
        style={FIELD_LIST_STYLE}
        header={
          <Header.Header>
            <Header.Title weight={500} color={9}>
              Fields
            </Header.Title>
            {!isSnapshot && (
              <Header.Actions empty align="end">
                <Button.Button
                  onClick={handleAdd}
                  variant="text"
                  contrast={2}
                  tooltip="Add field"
                  size="small"
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
      {selectedFieldPath != null && (
        <Flex.Box y empty className={CSS.B("enum-mapping")}>
          <Divider.Divider x padded />
          <KeyValueEditor
            path={selectedFieldPath}
            label="Enum mapping"
            keyField="label"
            keyPlaceholder="String (e.g. ON)"
            valueType="number"
          />
        </Flex.Box>
      )}
    </>
  );
};

const FIELD_LIST_STYLE = {
  paddingBottom: "1rem",
  maxWidth: "100%",
  overflow: "visible",
} as const;

type TimingMode = "software" | "value";
const TIMING_MODE_KEYS: TimingMode[] = ["software", "value"];

const TimingToggle: FC<{ path: string }> = ({ path }) => {
  const fields = PForm.useFieldValue<ReadField[]>(`${path}.fields`);
  const { set } = PForm.useContext();
  const indexField = fields.find(isTimingField);
  const isValueTiming = indexField != null;

  const handleChange = useCallback(
    (mode: TimingMode) => {
      if (mode === "value" && !isValueTiming) {
        const indexF: ReadField = {
          ...ZERO_READ_FIELD,
          key: id.create(),
          timestampFormat: "unix_sec",
        };
        set(`${path}.fields`, [...fields, indexF]);
        set(`${path}.index`, indexF.key);
      } else if (mode === "software" && isValueTiming) {
        set(
          `${path}.fields`,
          fields.filter((f) => !isTimingField(f)),
        );
        set(`${path}.index`, null);
      }
    },
    [fields, isValueTiming, path, set],
  );

  return (
    <Flex.Box x align="end" wrap>
      <Input.Item label="Timing mode" padHelpText>
        <Select.Buttons<TimingMode>
          value={isValueTiming ? "value" : "software"}
          onChange={handleChange}
          keys={TIMING_MODE_KEYS}
        >
          <Select.Button<TimingMode> itemKey="software">Software</Select.Button>
          <Select.Button<TimingMode> itemKey="value">Value</Select.Button>
        </Select.Buttons>
      </Input.Item>
      {isValueTiming && indexField != null && (
        <>
          <PForm.TextField
            path={`${path}.fields.${indexField.key}.pointer`}
            label="Timestamp pointer"
            inputProps={TIMESTAMP_POINTER_INPUT_PROPS}
            grow
          />
          <PForm.Field<TimeFormat>
            path={`${path}.fields.${indexField.key}.timestampFormat`}
            label="Format"
            className={CSS.B("timestamp-format")}
          >
            {renderSelectTimeFormat}
          </PForm.Field>
        </>
      )}
    </Flex.Box>
  );
};

const TIMESTAMP_POINTER_INPUT_PROPS = { placeholder: "/timestamp" } as const;

const renderSelectTimeFormat = Component.renderProp(
  (
    p: Omit<
      Select.StaticProps<TimeFormat, Select.StaticEntry<TimeFormat>>,
      "data" | "resourceName"
    >,
  ) => (
    <Select.Static<TimeFormat, Select.StaticEntry<TimeFormat>>
      {...p}
      data={TIME_FORMAT_DATA}
      resourceName="time format"
    />
  ),
);

const EndpointDetails: FC<{ epKey: string }> = ({ epKey }) => {
  const path = `config.endpoints.${epKey}`;
  const method = PForm.useFieldValue<string>(`${path}.method`);
  return (
    <Flex.Box y grow empty className={CSS.B("endpoint-details")}>
      <Flex.Box gap="small" empty className={CSS.B("endpoint-details-form")}>
        <Flex.Box x align="end" gap="large">
          <MethodSelect path={`${path}.method`} epPath={path} />
          <PForm.TextField
            path={`${path}.path`}
            label="Path"
            grow
            inputProps={PATH_INPUT_PROPS}
          />
        </Flex.Box>
        {method === "POST" && (
          <Flex.Box>
            <PForm.TextField
              path={`${path}.body`}
              label="Request body"
              grow
              inputProps={REQUEST_BODY_INPUT_PROPS}
            />
          </Flex.Box>
        )}
        <TimingToggle path={path} />
        <Divider.Divider x />
        <KeyValueEditor
          path={`${path}.headers`}
          label="Headers"
          keyField="name"
          keyPlaceholder="Name"
          valuePlaceholder="Value"
          className={CSS.B("headers-kv-editor")}
        />
        <Divider.Divider x />
        <KeyValueEditor
          path={`${path}.queryParams`}
          label="Query parameters"
          keyField="parameter"
          keyPlaceholder="Parameter"
          valuePlaceholder="Value"
          className={CSS.B("query-params-kv-editor")}
        />
      </Flex.Box>
      <Divider.Divider x />
      <FieldList key={epKey} epKey={epKey} />
    </Flex.Box>
  );
};

const PATH_INPUT_PROPS = { placeholder: "/api/data" } as const;

const REQUEST_BODY_INPUT_PROPS = { placeholder: '{"query": "latest"}' } as const;

const Form: FC<Common.Task.FormProps<ReadSchemas>> = () => {
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
  const menuRenderProp = useCallback(
    (p: PMenu.ContextMenuMenuProps) => (
      <ContextMenu
        keys={p.keys}
        onDelete={handleDeleteEndpoints}
        onDuplicate={handleDuplicateEndpoints}
      />
    ),
    [handleDeleteEndpoints, handleDuplicateEndpoints],
  );

  return (
    <Flex.Box x grow empty>
      <Flex.Box className={CSS.B("endpoint-list")} y empty>
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
              {readEndpointListItem}
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

const getInitialValues: Common.Task.GetInitialValues<ReadSchemas> = ({
  deviceKey,
  config,
}) => {
  if (config != null) {
    const pld: ReadPayload = {
      ...ZERO_READ_PAYLOAD,
      config: READ_SCHEMAS.config.parse(config),
    };
    if (deviceKey != null) pld.config.device = deviceKey;
    return pld;
  }
  const pld: ReadPayload = { ...ZERO_READ_PAYLOAD };
  if (deviceKey != null) pld.config = { ...pld.config, device: deviceKey };
  return pld;
};

const retrieveChannel = async (
  client: Client,
  key: channel.Key,
): Promise<channel.Channel | null> => {
  try {
    return await client.channels.retrieve(key);
  } catch (e) {
    if (NotFoundError.matches(e)) return null;
    throw e;
  }
};

const channelExists = async (client: Client, key: channel.Key): Promise<boolean> =>
  (await retrieveChannel(client, key)) != null;

const onConfigure: Common.Task.OnConfigure<ReadSchemas["config"]> = async (
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

        // create a new channel
        const dt = new DataType(field.dataType);
        const chName = primitive.isNonZero(field.name)
          ? field.name
          : `${safeDevName}${channel.escapeInvalidName(ep.path + field.pointer)}`;
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
    if (modified) await client.devices.create(dev, Device.SCHEMAS);
  }
  return [config, dev.rack];
};

export const Read = Common.Task.wrapForm({
  Properties,
  Form,
  schemas: READ_SCHEMAS,
  type: "http_read",
  getInitialValues,
  onConfigure,
});
