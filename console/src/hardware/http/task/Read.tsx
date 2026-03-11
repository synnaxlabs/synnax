// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/http/task/Read.css";

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
import { DataType, id, primitive } from "@synnaxlabs/x";
import { type FC, useCallback, useState } from "react";

import { EmptyAction, Menu } from "@/components";
import { KeyValueEditor } from "@/components/form/KeyValueEditor";
import { CSS } from "@/css";
import { Common } from "@/hardware/common";
import { ChannelList as BaseChannelList } from "@/hardware/common/task/ChannelList";
import { Device } from "@/hardware/http/device";
import {
  READ_SCHEMAS,
  READ_TYPE,
  type ReadEndpoint,
  type ReadField,
  type ReadMethod,
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
      <Common.Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

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
  return (
    <PMenu.Menu level="small">
      {!isSnapshot && canAct && (
        <>
          <PMenu.Item itemKey="duplicate" onClick={() => onDuplicate(keys)}>
            <Icon.Copy />
            Duplicate
          </PMenu.Item>
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
  const fields = PForm.useFieldValue<ReadField[]>(`config.endpoints.${itemKey}.fields`);
  const shownPath = epPath === "" ? "(no path)" : epPath;
  return (
    <Select.ListItem {...props} justify="between" align="center" x>
      <Text.Text level="small" weight={500}>
        {method} {shownPath}
      </Text.Text>
      <Text.Text level="small" color={7}>
        {fields.length}
      </Text.Text>
    </Select.ListItem>
  );
};

const endpointListItem = Component.renderProp(EndpointListItem);

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
  const enumValues = PForm.useFieldValue<Record<string, number>>(`${path}.enumValues`, {
    defaultValue: {},
  });
  const enumCount = Object.keys(enumValues).length;
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

const HIDDEN_DATA_TYPES = [DataType.UUID, DataType.JSON, DataType.BYTES];

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
        style={{ paddingBottom: "1rem", maxWidth: "100%" }}
        grow
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
      {selectedFieldPath != null && (
        <Flex.Box y empty style={{ flexShrink: 0, minHeight: 200, overflowY: "auto" }}>
          <Divider.Divider x padded />
          <KeyValueEditor
            path={selectedFieldPath}
            label="Enum Mapping"
            keyPlaceholder="String (e.g. ON)"
            valueType="number"
          />
        </Flex.Box>
      )}
    </>
  );
};

// ─── Endpoint Details ───

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
    <Flex.Box y gap={3} style={{ padding: "2rem", flexShrink: 0 }}>
      <Flex.Box x align="center" gap="small">
        <Text.Text level="small" weight={500} style={{ marginRight: "0.25rem" }}>
          Timing
        </Text.Text>
        <Select.Buttons<TimingMode>
          value={isValueTiming ? "value" : "software"}
          onChange={handleChange}
          keys={TIMING_MODE_KEYS}
        >
          <Select.Button<TimingMode> itemKey="software">Software</Select.Button>
          <Select.Button<TimingMode> itemKey="value">Value</Select.Button>
        </Select.Buttons>
      </Flex.Box>
      {isValueTiming && indexField != null && (
        <Flex.Box x align="center" gap="large">
          <PForm.TextField
            path={`${path}.fields.${indexField.key}.pointer`}
            label="Timestamp pointer"
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
    <Flex.Box y grow empty className={CSS.B("endpoint-details")}>
      <Flex.Box gap="small" empty className={CSS.B("endpoint-details-form")}>
        <Flex.Box x align="end" gap="large">
          <MethodSelect path={`${path}.method`} epPath={path} />
          <PForm.TextField
            path={`${path}.path`}
            label="Path"
            grow
            inputProps={{ placeholder: "/api/data" }}
          />
        </Flex.Box>
        {method === "POST" && (
          <Flex.Box>
            <PForm.TextField
              path={`${path}.body`}
              label="Request body"
              grow
              inputProps={{ placeholder: '{"query": "latest"}' }}
            />
          </Flex.Box>
        )}
        <KeyValueEditor
          path={`${path}.headers`}
          label="Headers"
          keyPlaceholder="Header Name"
          valuePlaceholder="Header Value"
          className={CSS.B("endpoint-details-headers")}
        />
        <KeyValueEditor
          path={`${path}.queryParams`}
          label="Query parameters"
          keyPlaceholder="Parameter"
          valuePlaceholder="Value"
        />
      </Flex.Box>
      <Divider.Divider x padded />
      <TimingToggle path={path} />
      <Divider.Divider x />
      <FieldList key={epKey} epKey={epKey} />
    </Flex.Box>
  );
};

// ─── Main Form ───

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
          <Text.Text status="disabled">Select an endpoint to configure</Text.Text>
        </Flex.Box>
      )}
    </Flex.Box>
  );
};

const getInitialValues: Common.Task.GetInitialValues<ReadSchemas> = ({
  deviceKey,
}) => ({
  ...ZERO_READ_PAYLOAD,
  config: {
    ...ZERO_READ_PAYLOAD.config,
    device: deviceKey ?? ZERO_READ_PAYLOAD.config.device,
  },
});

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
  for (const ep of config.endpoints) {
    // first, see if the user specified an index channel for this endpoint
    const devIndexKey = dev.properties.readIndexes[ep.path];
    if (primitive.isNonZero(devIndexKey)) continue;

    // we need to create an index channel for this endpoint.
    const newIndexCh = await client.channels.create({
      name: `${safeDevName}${channel.escapeInvalidName(ep.path)}_time`,
      dataType: "timestamp",
      isIndex: true,
    });
    modified = true;
    dev.properties.readIndexes[ep.path] = newIndexCh.key;
  }
  // now, we need to update any data channels as need be
  for (const ep of config.endpoints) {
    const index = dev.properties.readIndexes[ep.path];
    const potentialTimingKey = ep.index;
    for (const field of ep.fields) {
      if (field.channel !== 0) continue;
      if (field.key === potentialTimingKey) {
        field.channel = index;
        continue;
      }
      const newCh = await client.channels.create({
        name: `${safeDevName}_${channel.escapeInvalidName(ep.path + field.pointer)}`,
        dataType: field.dataType,
        index,
      });
      field.channel = newCh.key;
    }
  }
  if (modified) await client.devices.create(dev, Device.SCHEMAS);
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
