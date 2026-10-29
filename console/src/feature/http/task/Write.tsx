// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/http/task/Form.css";

import { channel } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { deep, id, primitive } from "@synnaxlabs/x";
import { type FC, useState } from "react";

import { useFromConfig } from "@/feature/http/device/queries";
import { Select as SelectDevice } from "@/feature/http/device/Select";
import { SCHEMAS } from "@/feature/http/device/types";
import { EndpointLabel } from "@/feature/http/task/EndpointLabel";
import {
  deployWriteConfigZ,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteEndpoint,
  writeEndpointZ,
  type WriteMethod,
  type WriteSchemas,
} from "@/feature/http/task/types";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { Form as PlatformForm } from "@/platform/form";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const Properties = () => (
  <>
    <SelectDevice />
    <Flex.Box x grow>
      <Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

const MethodSelect: FC<{ path: string }> = ({ path }) => (
  <PForm.Field<WriteMethod> path={path} label="Method">
    {renderMethodSelect}
  </PForm.Field>
);

const renderMethodSelect = Component.renderProp(
  (p: Select.ButtonsProps<WriteMethod>) => (
    <Select.Buttons<WriteMethod> {...p}>
      <Select.Item<WriteMethod> itemKey="POST">POST</Select.Item>
      <Select.Item<WriteMethod> itemKey="PUT">PUT</Select.Item>
      <Select.Item<WriteMethod> itemKey="PATCH">PATCH</Select.Item>
    </Select.Buttons>
  ),
);

const EndpointItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = `config.endpoints.${itemKey}`;
  const channel = PForm.useFieldValue<number>(`${path}.channel.channel`);
  return (
    <Select.Item
      {...props}
      y
      align="start"
      gap="small"
      className={CSS.B("endpoint-item")}
    >
      <Flex.Box x align="center" gap="small" full="x">
        <EndpointLabel epKey={itemKey} />
      </Flex.Box>
      <Flex.Box x align="center" justify="between" gap="small" full="x">
        <Task.ChannelName
          channel={channel}
          namePath={`${path}.channel.name`}
          id={Task.getChannelNameID(itemKey)}
          level="small"
          weight={450}
          color={9}
          overflow="fade"
        />
        <Task.EnabledCheckbox path={`${path}.disabled`} />
      </Flex.Box>
    </Select.Item>
  );
};

const endpointItem = Component.renderProp(EndpointItem);

const EMPTY_ENDPOINTS = <Empty.Action message="No endpoints" />;

const EndpointDetails: FC<{ epKey: string }> = ({ epKey }) => {
  const path = `config.endpoints.${epKey}`;
  return (
    <Flex.Box y grow empty className={CSS.B("endpoint-details")}>
      <PForm.Sections className={CSS.B("endpoint-form")}>
        <PForm.Section title="Request">
          <MethodSelect path={`${path}.method`} />
          <PForm.TextField
            path={`${path}.path`}
            label="Path"
            padHelpText={false}
            inputProps={PATH_INPUT_PROPS}
          />
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
        <Task.Payload.ChannelSections
          path={`${path}.channel`}
          pointerPlaceholder="/value"
        />
        <Task.Payload.AdditionalFields key={epKey} path={`${path}.fields`} />
      </PForm.Sections>
    </Flex.Box>
  );
};

const PATH_INPUT_PROPS = { placeholder: "/api/control" } as const;

const createEndpoint = (): WriteEndpoint => writeEndpointZ.parse({});

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
          path="config.endpoints"
          title={null}
          emptyContent={EMPTY_ENDPOINTS}
          addLabel="Add endpoint"
          createChannel={createEndpoint}
          createChannels={duplicateEndpoints}
          listItem={endpointItem}
          contextMenuItems={Task.readChannelContextMenuItem}
          resolve={resolve}
          selected={selected}
          onSelect={setSelected}
          grow
        />
      }
      detailsPath={current != null ? `config.endpoints.${current}` : null}
      title={current != null && <EndpointLabel epKey={current} />}
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
  const safeDevName = channel.escapeInvalidName(dev.name);
  const saved = deep.copy(dev.properties);
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
            : `${safeDevName}${channel.escapeInvalidName(ep.path)}_cmd`,
          dataType: ep.channel.dataType,
        },
      );
    }
  } finally {
    if (!deep.equal(saved, dev.properties)) await client.devices.create(dev, SCHEMAS);
  }
  return [config, dev.rack];
};

export const Write = Task.wrapForm({
  Properties,
  Form,
  schemas: WRITE_SCHEMAS,
  deployConfigZ: deployWriteConfigZ,
  type: WRITE_TYPE,
  getInitialValues,
  onConfigure,
});

export const useCreateWrite = Task.createUseCreate({
  getInitialValues,
});

export const WriteSelectable = Selector.createSelectable({
  type: WRITE_TYPE,
  title: "HTTP write task",
  icon: <Icon.Logo.HTTP />,
  useOnSelect: useCreateWrite,
});
