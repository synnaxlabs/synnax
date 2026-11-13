// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/http/task/Form.css";

import { channel, DisconnectedError, http } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Synnax } from "@synnaxlabs/pluto";
import { type DataType, deep, id, type payload, primitive } from "@synnaxlabs/x";
import { type FC, useState } from "react";

import { useFromConfig } from "@/feature/http/device/queries";
import { Select as SelectDevice } from "@/feature/http/device/Select";
import { MAKE, SCHEMAS } from "@/feature/http/device/types";
import { EndpointLabel } from "@/feature/http/task/EndpointLabel";
import {
  type ChannelField,
  deployWriteConfigZ,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteEndpoint,
  writeEndpointZ,
  type WriteSchemas,
} from "@/feature/http/task/types";
import { CSS } from "@/platform/css";
import { Device } from "@/platform/device";
import { Empty } from "@/platform/empty";
import { Form as PlatformForm } from "@/platform/form";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<WriteSchemas>>();

const Properties = () => <SelectDevice />;

const MethodSelect: FC<{ path: deep.Target<http.Method> }> = ({ path }) => (
  <PForm.Field path={path} label="Method">
    {renderMethodSelect}
  </PForm.Field>
);

const renderMethodSelect = Component.renderProp(
  (p: Select.ButtonsProps<http.Method>) => (
    <Select.Buttons<http.Method> {...p}>
      <Select.Item<http.Method> itemKey="POST">POST</Select.Item>
      <Select.Item<http.Method> itemKey="PUT">PUT</Select.Item>
      <Select.Item<http.Method> itemKey="PATCH">PATCH</Select.Item>
    </Select.Buttons>
  ),
);

/** The name deploying gives an endpoint's command channel while it has none. */
const commandChannelName = (device: string, endpoint: string): string =>
  `${channel.escapeInvalidName(device)}${channel.escapeInvalidName(endpoint)}_cmd`;

const EndpointItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = root.config.endpoints[itemKey];
  const channelKey = PForm.useFieldValue(path.channel.channel);
  const endpointPath = PForm.useFieldValue(path.path);
  const disabled = PForm.useFieldValue(path.disabled);
  const dev = useFromConfig();
  return (
    <Select.Item
      {...props}
      y
      align="start"
      gap="small"
      className={CSS.B("endpoint-item")}
    >
      <Flex.Box x align="center" gap="small" full="x">
        <EndpointLabel path={path} />
      </Flex.Box>
      <Flex.Box x align="center" justify="between" gap="small" full="x">
        <Task.ChannelName
          channel={channelKey}
          namePath={path.channel.name}
          defaultName={
            dev == null ? undefined : commandChannelName(dev.name, endpointPath)
          }
          skipped={disabled}
          id={Task.getChannelNameID(itemKey)}
          level="small"
          weight={450}
          color={9}
          overflow="fade"
        />
        <Task.EnabledCheckbox path={path.disabled} />
      </Flex.Box>
    </Select.Item>
  );
};

const endpointItem = Component.renderProp(EndpointItem);

const EMPTY_ENDPOINTS = <Empty.Action message="No endpoints" />;

/** Returns field written as kind, with the defaults of kind. */
const asKind = (field: ChannelField, kind: payload.Kind): ChannelField =>
  http.channelFieldZ.parse({
    pointer: field.pointer,
    channel: field.channel,
    name: field.name,
    dataType: field.dataType,
    type: kind,
    ...(kind === "timestamp" ? { timeFormat: "iso8601" } : {}),
  });

/** A command channel can hold any data type a body writes. */
const NO_HIDDEN_DATA_TYPES: DataType[] = [];

const POINTER_INPUT_PROPS = { placeholder: "/value" } as const;

const ChannelSections: FC<{ path: deep.Path<WriteEndpoint> }> = ({ path }) => {
  const field = PForm.useFieldValue(path.channel);
  const dataType = Task.Payload.useDataType(field.channel, field.dataType);
  return (
    <>
      <PForm.Section title="Channel">
        <Task.Payload.DataTypeField
          path={deep.present(path.channel.dataType)}
          channel={field.channel}
          hideDataTypes={NO_HIDDEN_DATA_TYPES}
        />
        <PForm.TextField
          path={path.channel.pointer}
          label="Pointer"
          padHelpText={false}
          inputProps={POINTER_INPUT_PROPS}
        />
        <Task.Payload.KindFields
          path={path.channel}
          dataType={dataType}
          asKind={asKind}
        />
      </PForm.Section>
      <Task.Payload.EntriesSection path={path.channel} />
    </>
  );
};

const EndpointDetails: FC<{ epKey: string }> = ({ epKey }) => {
  const path = root.config.endpoints[epKey];
  return (
    <Flex.Box y grow empty className={CSS.B("endpoint-details")}>
      <PForm.Sections className={CSS.B("endpoint-form")}>
        <PForm.Section title="Request">
          <MethodSelect path={path.method} />
          <PForm.TextField
            path={path.path}
            label="Path"
            padHelpText={false}
            inputProps={PATH_INPUT_PROPS}
          />
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
        <ChannelSections path={path} />
        <Task.Payload.AdditionalFields key={epKey} path={path.fields} />
      </PForm.Sections>
    </Flex.Box>
  );
};

const PATH_INPUT_PROPS = { placeholder: "/api/control" } as const;

/** The data type of the command channel of a new endpoint. */
const DEFAULT_DATA_TYPE = "float64";

const createEndpoint = (): WriteEndpoint =>
  writeEndpointZ.parse({ channel: { type: "float", dataType: DEFAULT_DATA_TYPE } });

const duplicateEndpoints = (
  endpoints: WriteEndpoint[],
  keys: string[],
): WriteEndpoint[] =>
  endpoints
    .filter(({ key }) => keys.includes(key))
    .map((ep) => ({
      ...ep,
      key: id.create(),
      channel: { ...ep.channel, channel: 0 },
      fields: ep.fields.map((f) => ({ ...f, key: id.create() })),
    }));

const Form: FC = () => {
  const [selected, setSelected] = useState<string[]>([]);
  const dev = useFromConfig();
  const client = Synnax.use();
  const { get, set } = PForm.useContext();
  const handleError = Status.useErrorHandler();

  // The Core clears the data type of an endpoint with a channel, so a copy, which gets
  // a new channel, reads the data type of the channel it copies. The read sets the data
  // type only while the copy still holds the one it started with.
  const handleDuplicate = (all: WriteEndpoint[], keys: string[]): WriteEndpoint[] => {
    const copies = duplicateEndpoints(all, keys);
    const sources = all.filter(({ key }) => keys.includes(key));
    copies.forEach((copy, i) => {
      const key = sources[i].channel.channel;
      if (key === 0) return;
      const initial = copy.channel.dataType;
      const path = root.config.endpoints[copy.key];
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
        const ch = await client.channels.retrieve(key);
        const current = get(path, { optional: true })?.value;
        if (
          current == null ||
          current.channel.channel !== 0 ||
          current.channel.dataType !== initial
        )
          return;
        set(path.channel.dataType, ch.dataType.toString());
      }, "Failed to copy the data type of the command channel");
    });
    return copies;
  };
  const resolve = (ep: WriteEndpoint) =>
    dev == null
      ? null
      : { channel: { ...ep.channel, channel: dev.properties.write[ep.path] ?? 0 } };
  const current = selected.length > 0 ? selected[0] : null;
  return (
    <Task.Views.Panes
      listTitle="Endpoints"
      list={
        <Task.Views.ChannelList<WriteEndpoint>
          path={root.config.endpoints}
          header={null}
          emptyContent={EMPTY_ENDPOINTS}
          addLabel="Add endpoint"
          createChannel={createEndpoint}
          createChannels={handleDuplicate}
          listItem={endpointItem}
          contextMenuItems={Task.readChannelContextMenuItem}
          resolve={resolve}
          selected={selected}
          onSelect={setSelected}
          grow
        />
      }
      detailsPath={current != null ? root.config.endpoints[current] : null}
      title={current != null && <EndpointLabel path={root.config.endpoints[current]} />}
    >
      {current != null ? (
        <EndpointDetails epKey={current} />
      ) : (
        <Flex.Box y grow align="center" justify="center">
          <Text.Text status="disabled">Select an endpoint to configure</Text.Text>
        </Flex.Box>
      )}
    </Task.Views.Panes>
  );
};

const getInitialValues: Task.GetInitialValues<WriteSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = WRITE_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: "HTTP write task", type: WRITE_TYPE, config: cfg };
};

const onConfigure: Task.OnConfigure<WriteSchemas["config"]> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: SCHEMAS,
  });
  const before = deep.copy(dev.properties);
  try {
    for (const ep of config.endpoints) {
      if (ep.disabled) continue;
      ep.channel.channel = await Task.configureCommandChannel(
        client,
        dev.properties.write,
        {
          propertiesKey: ep.path,
          channel: ep.channel.channel,
          name: primitive.isNonZero(ep.channel.name)
            ? ep.channel.name
            : commandChannelName(dev.name, ep.path),
          dataType: ep.channel.dataType,
        },
      );
    }
  } finally {
    await Device.setProperties(client, dev, before, SCHEMAS);
  }
  return [config, dev.rack];
};

export const Write = Task.wrapForm({
  Properties,
  Form,
  schemas: WRITE_SCHEMAS,
  deployConfigZ: deployWriteConfigZ,
  type: WRITE_TYPE,
  onConfigure,
});

export const useCreateWrite = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const WriteSelectable: Selector.Selectable = {
  type: WRITE_TYPE,
  title: "HTTP write task",
  label: "Write",
  icon: <Icon.Logo.HTTP />,
  useOnSelect: useCreateWrite,
};
