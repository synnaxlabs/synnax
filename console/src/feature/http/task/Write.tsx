// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/http/task/Form.css";

import { channel, DisconnectedError, http, type icd } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { useExclusiveCallback } from "@synnaxlabs/lyra/hooks";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { ICD as PICD, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { deep, id } from "@synnaxlabs/x";
import { useState } from "react";

import { Device } from "@/feature/http/device";
import { DEFAULT_WRITE_NAME } from "@/feature/http/task/create";
import { EndpointLabel } from "@/feature/http/task/EndpointLabel";
import {
  deployWriteConfigZ,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteEndpoint,
  type WriteSchemas,
} from "@/feature/http/task/types";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { Empty } from "@/platform/empty";
import { Form as PlatformForm } from "@/platform/form";
import { ICD } from "@/platform/icd";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<WriteSchemas>>();

const Properties = () => <Device.Select />;

interface MethodSelectProps {
  path: deep.Target<http.Method>;
  onChange: (method: http.Method) => void;
}

const MethodSelect = ({ path, onChange }: MethodSelectProps) => (
  <PForm.Field path={path} label="Method" onChange={onChange}>
    {renderMethodSelect}
  </PForm.Field>
);

const SelectMethod = (p: Select.ButtonsProps<http.Method>) => (
  <Select.Buttons<http.Method> {...p}>
    <Select.Item<http.Method> itemKey="POST">POST</Select.Item>
    <Select.Item<http.Method> itemKey="PUT">PUT</Select.Item>
    <Select.Item<http.Method> itemKey="PATCH">PATCH</Select.Item>
  </Select.Buttons>
);

const renderMethodSelect = Component.renderProp(SelectMethod);

const EMPTY_ENDPOINTS = <Empty.Action message="No endpoints" />;

/** The method and path of the request of an endpoint. */
type RequestLine = Pick<WriteEndpoint, "method" | "path">;

interface RequestSectionProps {
  path: deep.Path<WriteEndpoint>;
  /** Called with each change to the method or path, before the form applies it. */
  onChange: (change: Partial<RequestLine>) => void;
}

const RequestSection = ({ path, onChange }: RequestSectionProps) => (
  <PForm.Section title="Request">
    <MethodSelect path={path.method} onChange={(method) => onChange({ method })} />
    <PForm.TextField
      path={path.path}
      label="Path"
      padHelpText={false}
      inputProps={PATH_INPUT_PROPS}
      onChange={(p) => onChange({ path: p })}
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
);

const PATH_INPUT_PROPS = { placeholder: "/api/control" } as const;

/** An endpoint whose request body is a message of an ICD. */
interface ICDEndpoint extends WriteEndpoint {
  icd: icd.Key;
  message: icd.WriteMessage;
}

const sendsICD = (ep: WriteEndpoint): ep is ICDEndpoint =>
  ep.icd != null && ep.message != null;

const icdPath = (key: string) => root.config.endpoints[key] as deep.Path<ICDEndpoint>;

/** The name deploying gives the command channel of a field of an endpoint. */
const fieldCommandName = (
  device: string,
  endpoint: string,
  field: icd.JSONField,
): string =>
  `${channel.escapeInvalidName(device)}${channel.escapeInvalidName(endpoint)}_` +
  `${channel.escapeInvalidName(field.name)}_cmd`;

/** @returns the name that an endpoint gives the message it sends. */
const messageName = ({ method, path }: RequestLine): string =>
  `${method} ${path}`.trim();

/** @returns whether name is the message name of request, with or without a number. */
const namedBy = (name: string, request: RequestLine): boolean => {
  const base = messageName(request);
  if (name === base) return true;
  return name.startsWith(`${base} `) && /^\d+$/.test(name.slice(base.length + 1));
};

/** @returns the name of the ICD of a task named name. */
const icdName = (name: string): string =>
  name.trim() === "" ? DEFAULT_WRITE_NAME : name;

const EndpointItem = (props: Task.ChannelListItemProps) => {
  const path = root.config.endpoints[props.itemKey];
  const disabled = PForm.useFieldValue(path.disabled);
  return (
    <Select.Item
      {...props}
      y
      align="start"
      gap="small"
      className={CSS.cls(CSS.B("endpoint-item"), disabled && CSS.M("off"))}
    >
      <Flex.Box x align="center" justify="between" gap="small" full="x">
        <EndpointLabel path={path} />
        <Task.EnabledCheckbox path={path.disabled} />
      </Flex.Box>
    </Select.Item>
  );
};

const endpointItem = Component.renderProp(EndpointItem);

interface DetailsProps {
  epKey: string;
  deviceName: string;
}

/** The message of an endpoint that sends one, and the binding of each of its fields. */
const MessageDetails = ({ epKey, deviceName }: DetailsProps) => {
  const path = icdPath(epKey);
  const { set } = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const formats = ICD.useFormats();
  const [opened, setOpened] = useState<icd.FieldKey>();
  const ep = PForm.useFieldValue(path);
  const message = ICD.useMessage(ep.icd, ep.message.message);
  const handleFormatChange = (icdKey: icd.Key, picked: icd.Message) =>
    set(path, { ...ep, icd: icdKey, message: ICD.Write.create(picked) });
  // The Suit of the endpoint binds the new field.
  const handleAddField = () => {
    const fields = ICD.jsonFields(message);
    const field = ICD.createJSONField(fields);
    ICD.formatOf(formats, ep.icd).insert(
      ICD.fieldsPath(ep.message.message),
      field,
      fields.at(-1)?.key,
    );
    setOpened(field.key);
  };
  return (
    <>
      <PForm.Section
        title="Message"
        actions={
          <ICD.AddField
            onClick={handleAddField}
            disabled={message?.payload.format !== "json"}
          />
        }
      >
        <ICD.MessageFormatField
          icdKey={ep.icd}
          message={ep.message.message}
          onChange={handleFormatChange}
          preview={isPreview}
        />
      </PForm.Section>
      <ICD.MessageStatus
        message={message}
        missingDescription={ICD.Write.MISSING_DESCRIPTION}
      />
      <ICD.Write.Bindings
        path={path.message.bindings}
        icdKey={ep.icd}
        message={ep.message.message}
        channelName={(field) => fieldCommandName(deviceName, ep.path, field)}
        skipped={ep.disabled}
        opened={opened}
      />
    </>
  );
};

/** Picks the message of an endpoint that sends none. */
const MessagePick = ({ epKey }: { epKey: string }) => {
  const path = root.config.endpoints[epKey];
  const { set } = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const ep = PForm.useFieldValue(path);
  const handleFormatChange = (icdKey: icd.Key, picked: icd.Message) =>
    set(path, { ...ep, icd: icdKey, message: ICD.Write.create(picked) });
  return (
    <PForm.Section title="Message">
      <ICD.MessageFormatField onChange={handleFormatChange} preview={isPreview} />
    </PForm.Section>
  );
};

const EndpointDetails = ({ epKey, deviceName }: DetailsProps) => {
  const path = root.config.endpoints[epKey];
  const ctx = PForm.useContext();
  const formats = ICD.useFormats();
  const sends = sendsICD(PForm.useFieldValue(path));
  // A message that only this endpoint sends follows its request while it keeps the
  // name that the request gave it.
  const handleRequestChange = (change: Partial<RequestLine>) => {
    const ep = ctx.get(path).value;
    if (!sendsICD(ep)) return;
    const shared = ctx
      .get(root.config.endpoints)
      .value.some(
        (other) =>
          other.key !== ep.key &&
          other.icd === ep.icd &&
          other.message?.message === ep.message.message,
      );
    if (shared) return;
    const format = ICD.formatOf(formats, ep.icd);
    const { messages } = format.value();
    const message = messages.find(({ key }) => key === ep.message.message);
    if (message == null || !namedBy(message.name, ep)) return;
    format.set(
      PICD.formPath.messages[message.key].name,
      ICD.availableName(
        messageName({ ...ep, ...change }),
        messages.filter(({ key }) => key !== message.key),
      ),
    );
  };
  return (
    <Flex.Box y grow empty className={CSS.B("endpoint-details")}>
      <PForm.Sections className={CSS.B("endpoint-form")}>
        <RequestSection path={path} onChange={handleRequestChange} />
        {sends ? (
          <MessageDetails epKey={epKey} deviceName={deviceName} />
        ) : (
          <MessagePick epKey={epKey} />
        )}
      </PForm.Sections>
    </Flex.Box>
  );
};

const duplicateEndpoints = (
  endpoints: WriteEndpoint[],
  keys: string[],
): WriteEndpoint[] =>
  endpoints
    .filter(({ key }) => keys.includes(key))
    .map((ep) => ({
      ...ep,
      key: id.create(),
      message: ep.message == null ? undefined : ICD.Write.unbind(ep.message),
    }));

interface EndpointsProps {
  device: Device.Device;
}

const Endpoints = ({ device }: EndpointsProps) => {
  const client = PSynnax.use();
  const ctx = PForm.useContext();
  const handleError = Status.useErrorHandler();
  const isPreview = Task.useIsPreview();
  const formats = ICD.useFormats();
  const [selected, setSelected] = useState<string[]>([]);
  const endpoints = PForm.useFieldListUtils(root.config.endpoints);
  const sending = PForm.useFieldValue(root.config.endpoints).filter(sendsICD);

  /**
   * Adds an endpoint that sends a new message with no fields, in the ICD of the most
   * endpoints, or in a new ICD named after the task when no endpoint sends one.
   */
  const add = async () => {
    if (client == null) throw new DisconnectedError();
    const ep = http.writeEndpointZ.parse({});
    const {
      icd: key,
      messages: [message],
    } = await ICD.Write.addMessages({
      client,
      formats,
      targets: sending,
      name: icdName(ctx.get(root.name).value),
      messages: [{ name: messageName(ep), fields: [] }],
    });
    const added = { ...ep, icd: key, message: ICD.Write.create(message) };
    endpoints.push([added]);
    setSelected([added.key]);
  };
  // Adds run one at a time, so a second click cannot make a second ICD.
  const addExclusive = useExclusiveCallback(add);
  const handleAdd = () => handleError(addExclusive, "Failed to add endpoint");

  const current = selected.at(0);
  const currentPath = current == null ? null : root.config.endpoints[current];
  return (
    <Task.Views.Panes
      listTitle="Endpoints"
      list={
        <Task.Views.ChannelList<WriteEndpoint>
          path={root.config.endpoints}
          header={null}
          emptyContent={EMPTY_ENDPOINTS}
          footer={
            !isPreview && (
              <Button.CreateListItem size="small" onClick={handleAdd}>
                Add endpoint
              </Button.CreateListItem>
            )
          }
          createChannels={duplicateEndpoints}
          listItem={endpointItem}
          resolve={null}
          selected={selected}
          onSelect={setSelected}
          grow
        />
      }
      detailsPath={currentPath}
      title={currentPath != null && <EndpointLabel path={currentPath} />}
    >
      {current != null ? (
        <EndpointDetails key={current} epKey={current} deviceName={device.name} />
      ) : (
        <Flex.Box y grow align="center" justify="center">
          <Text.Text status="disabled">Select an endpoint to configure</Text.Text>
        </Flex.Box>
      )}
      {sending.map(({ key, icd: icdKey }) => (
        <ICD.Write.Suit key={key} icdKey={icdKey} path={icdPath(key).message} />
      ))}
    </Task.Views.Panes>
  );
};

const Content = ({ device }: PlatformDevice.TaskFormContentProps<Device.Device>) => {
  const icds = PForm.useFieldValue(root.config.endpoints)
    .filter(sendsICD)
    .map((ep) => ep.icd);
  return (
    <ICD.WithFormats icdKeys={icds}>
      <Endpoints device={device} />
    </ICD.WithFormats>
  );
};

/** The endpoints of the task, each with the message of an ICD that it sends. */
const Form = PlatformDevice.wrapTaskForm({
  use: Device.use,
  useConfigure: Device.useConnectModal,
  useNoneSelected: Device.useNoneSelected,
  Content,
});

/** Gives each channel binding of an enabled endpoint a command channel. */
const onConfigure: Task.OnConfigure<WriteSchemas["config"]> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: Device.SCHEMAS,
  });
  const before = deep.copy(dev.properties);
  try {
    for (const ep of config.endpoints.filter(sendsICD)) {
      if (ep.disabled) continue;
      await ICD.Write.configureChannels(client, dev.properties.write, ep, {
        key: (field) => `${ep.path}#${field.key}`,
        name: (field) => fieldCommandName(dev.name, ep.path, field),
      });
    }
  } finally {
    await PlatformDevice.setProperties(client, dev, before, Device.SCHEMAS);
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
