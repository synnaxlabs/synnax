// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/http/task/Form.css";

import { channel, DisconnectedError, type http, type icd } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { useExclusiveCallback } from "@synnaxlabs/lyra/hooks";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import type { Tree } from "@synnaxlabs/lyra/tree";
import { ICD as PICD, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { deep, id, primitive, unique } from "@synnaxlabs/x";
import {
  createContext,
  type Dispatch,
  type FC,
  type PropsWithChildren,
  type ReactElement,
  type SetStateAction,
  useContext,
  useState,
} from "react";

import { Device } from "@/feature/http/device";
import { DEFAULT_WRITE_NAME } from "@/feature/http/task/create";
import { EndpointItem, EndpointLabel } from "@/feature/http/task/EndpointLabel";
import {
  deployWriteConfigZ,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteEndpoint,
  writeEndpointZ,
  type WriteSchemas,
} from "@/feature/http/task/types";
import { FLAGS } from "@/flags";
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
  onChange?: (method: http.Method) => void;
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

/** The name deploying gives an endpoint's command channel while it has none. */
const commandChannelName = (device: string, endpoint: string): string =>
  `${channel.escapeInvalidName(device)}${channel.escapeInvalidName(endpoint)}_cmd`;

/** What the command channel of an endpoint needs from deploying. */
const endpointCommand = (
  device: string,
  ep: WriteEndpoint,
): Task.CommandChannelSpec => ({
  propertiesKey: ep.path,
  channel: ep.channel.channel,
  name: primitive.isNonZero(ep.channel.name)
    ? ep.channel.name
    : commandChannelName(device, ep.path),
  dataType: ep.channel.dataType,
});

/** The name that deploying gives each command channel it creates, by endpoint path. */
const NamesContext = createContext<ReadonlyMap<string, string>>(new Map());
NamesContext.displayName = "HTTP.NamesContext";

/** Plans the command channels of every endpoint together, as deploying does. */
const CommandNames = ({ children }: PropsWithChildren): ReactElement => {
  const endpoints = PForm.useFieldValue(root.config.endpoints);
  const dev = Device.useFromConfig();
  const specs =
    dev == null
      ? []
      : endpoints
          .filter(({ disabled }) => !disabled)
          .map((ep) => endpointCommand(dev.name, ep));
  const names = Task.useCommandChannelNames(dev?.properties.write, specs);
  return <NamesContext value={names}>{children}</NamesContext>;
};

const ChannelEndpointItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = root.config.endpoints[itemKey];
  const channelKey = PForm.useFieldValue(path.channel.channel);
  const endpointPath = PForm.useFieldValue(path.path);
  const disabled = PForm.useFieldValue(path.disabled);
  const dev = Device.useFromConfig();
  const planned = useContext(NamesContext).get(endpointPath);
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
            dev == null
              ? undefined
              : (planned ?? commandChannelName(dev.name, endpointPath))
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

const endpointItem = Component.renderProp(ChannelEndpointItem);

const EMPTY_ENDPOINTS = <Empty.Action message="No endpoints" />;

/** The method and path of the request of an endpoint. */
type RequestLine = Pick<WriteEndpoint, "method" | "path">;

interface RequestSectionProps {
  path: deep.Path<WriteEndpoint>;
  /** Called with each change to the method or path, before the form applies it. */
  onChange?: (change: Partial<RequestLine>) => void;
}

const RequestSection = ({ path, onChange }: RequestSectionProps) => (
  <PForm.Section title="Request">
    <MethodSelect path={path.method} onChange={(method) => onChange?.({ method })} />
    <PForm.TextField
      path={path.path}
      label="Path"
      padHelpText={false}
      inputProps={PATH_INPUT_PROPS}
      onChange={(p) => onChange?.({ path: p })}
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

const EndpointDetails: FC<{ epKey: string }> = ({ epKey }) => {
  const path = root.config.endpoints[epKey];
  return (
    <Flex.Box y grow empty className={CSS.B("endpoint-details")}>
      <PForm.Sections className={CSS.B("endpoint-form")}>
        <RequestSection path={path} />
        <Task.Payload.ChannelSections path={path.channel} pointerPlaceholder="/value" />
        <Task.Payload.AdditionalFields key={epKey} path={path.fields} />
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
  const dev = Device.useFromConfig();
  const resolve = (ep: WriteEndpoint) =>
    dev == null
      ? null
      : { channel: { ...ep.channel, channel: dev.properties.write[ep.path] ?? 0 } };
  const current = selected.length > 0 ? selected[0] : null;
  return (
    <Task.Views.Panes
      listTitle="Endpoints"
      list={
        <CommandNames>
          <Task.Views.ChannelList<WriteEndpoint>
            path={root.config.endpoints}
            header={null}
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
        </CommandNames>
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
    for (const ep of config.endpoints) {
      if (ep.disabled) continue;
      ep.channel.channel = await Task.configureCommandChannel(
        client,
        dev.properties.write,
        endpointCommand(dev.name, ep),
      );
    }
  } finally {
    await PlatformDevice.setProperties(client, dev, before, Device.SCHEMAS);
  }
  return [config, dev.rack];
};

/** An endpoint whose request body is a message of an ICD. */
interface ICDEndpoint extends WriteEndpoint {
  icd: icd.Key;
  message: icd.WriteMessage;
}

const sendsICD = (ep: WriteEndpoint): ep is ICDEndpoint =>
  ep.icd != null && ep.message != null;

const icdPath = (key: string) => root.config.endpoints[key] as deep.Path<ICDEndpoint>;

/** Keys and names the command channels of fields of an endpoint, as deploying does. */
const namingOf = (device: string, { path }: WriteEndpoint): ICD.Write.ChannelNaming => {
  const prefix = channel.escapeInvalidName(device) + channel.escapeInvalidName(path);
  return {
    key: (field) => `${path}#${field.key}`,
    name: (field) => ICD.Write.commandName(prefix, field),
  };
};

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

const TargetItem = (props: Tree.ItemRenderProps<string>) => (
  <EndpointItem {...props} path={root.config.endpoints[props.itemKey]} />
);

const duplicate = (ep: WriteEndpoint): WriteEndpoint => ({
  ...ep,
  key: id.create(),
  message: ep.message == null ? undefined : ICD.Write.unbind(ep.message),
});

interface ICDDetailsProps {
  epKey: string;
  onAddField: () => void;
}

/** The message that an endpoint sends. */
const MessageDetails = ({ epKey, onAddField }: ICDDetailsProps) => {
  const path = icdPath(epKey);
  const { set } = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const ep = PForm.useFieldValue(path);
  const message = ICD.useMessage(ep.icd, ep.message.message);
  const handleFormatChange = (icdKey: icd.Key, picked: icd.Message) =>
    set(path, { ...ep, icd: icdKey, message: ICD.Write.create(picked) });
  return (
    <>
      <PForm.Section
        title="Message"
        actions={
          <ICD.AddField
            onClick={onAddField}
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

const ICDEndpointDetails = ({ epKey, onAddField }: ICDDetailsProps) => {
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
          <MessageDetails epKey={epKey} onAddField={onAddField} />
        ) : (
          <MessagePick epKey={epKey} />
        )}
      </PForm.Sections>
    </Flex.Box>
  );
};

interface EndpointsProps {
  device: Device.Device;
  selected: string[];
  onSelect: Dispatch<SetStateAction<string[]>>;
  expanded: string[];
  onExpand: Dispatch<SetStateAction<string[]>>;
}

/** The endpoints of the task as a tree, beside the details of the selected item. */
const Endpoints = ({
  device,
  selected,
  onSelect,
  expanded,
  onExpand,
}: EndpointsProps) => {
  const client = PSynnax.use();
  const ctx = PForm.useContext();
  const handleError = Status.useErrorHandler();
  const formats = ICD.useFormats();
  const list = PForm.useFieldListUtils(root.config.endpoints);
  const endpoints = PForm.useFieldValue(root.config.endpoints);
  const sending = endpoints.filter(sendsICD);
  const commandChannelsOf = ICD.Write.useCommandChannels();
  const specs = sending
    .filter(({ disabled }) => !disabled)
    .flatMap((ep) => commandChannelsOf(ep, namingOf(device.name, ep)));
  const names = Task.useCommandChannelNames(device.properties.write, specs);
  const messageOf = (ep: WriteEndpoint): ICD.Write.TargetMessage | null => {
    if (!sendsICD(ep)) return null;
    const naming = namingOf(device.name, ep);
    return {
      icd: ep.icd,
      message: ep.message.message,
      path: icdPath(ep.key).message,
      channelName: (field) => names.get(naming.key(field)) ?? naming.name(field),
    };
  };
  const { tree, handleSelect, contextMenu, renderItem, pane } = ICD.Write.useTree({
    path: root.config.endpoints,
    messageOf,
    TargetItem,
    targetPane: (ep, onAddField) => ({
      path: root.config.endpoints[ep.key],
      title: <EndpointLabel path={root.config.endpoints[ep.key]} />,
      details: (
        <ICDEndpointDetails key={ep.key} epKey={ep.key} onAddField={onAddField} />
      ),
    }),
    duplicate,
    selected,
    onSelect,
    expanded,
    onExpand,
  });

  /**
   * Adds an endpoint that sends a new message with no fields, in the ICD of the most
   * endpoints, or in a new ICD named after the task when no endpoint sends one.
   */
  const add = async () => {
    if (client == null) throw new DisconnectedError();
    const ep = writeEndpointZ.parse({});
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
    list.push([added]);
    onSelect([added.key]);
    onExpand((prev) => unique.unique([...prev, ...ICD.Write.expandedOf([added])]));
  };
  // Adds run one at a time, so a second click cannot make a second ICD.
  const addExclusive = useExclusiveCallback(add);
  const handleAdd = () => handleError(addExclusive, "Failed to add endpoint");

  return (
    <Task.Views.Panes
      listTitle="Endpoints"
      list={
        <Task.Views.TreeList
          {...tree}
          onSelect={handleSelect}
          menu={contextMenu}
          emptyMessage="No endpoints"
          addLabel="Add endpoint"
          onAdd={handleAdd}
        >
          {renderItem}
        </Task.Views.TreeList>
      }
      detailsPath={pane?.path ?? null}
      title={pane?.title}
    >
      {pane?.details ?? (
        <Flex.Box y grow align="center" justify="center">
          <Text.Text status="disabled">Add an endpoint to send requests to</Text.Text>
        </Flex.Box>
      )}
      {sending.map(({ key, icd: icdKey }) => (
        <ICD.Write.Suit key={key} icdKey={icdKey} path={icdPath(key).message} />
      ))}
    </Task.Views.Panes>
  );
};

const ICDContent = ({ device }: PlatformDevice.TaskFormContentProps<Device.Device>) => {
  const endpoints = PForm.useFieldValue(root.config.endpoints);
  const [selected, setSelected] = useState<string[]>([]);
  const [expanded, setExpanded] = useState<string[]>(() =>
    ICD.Write.expandedOf(endpoints),
  );
  const icds = endpoints.filter(sendsICD).map((ep) => ep.icd);
  return (
    <ICD.WithFormats icdKeys={icds}>
      <Endpoints
        device={device}
        selected={selected}
        onSelect={setSelected}
        expanded={expanded}
        onExpand={setExpanded}
      />
    </ICD.WithFormats>
  );
};

/** The endpoints of the task, each with the message of an ICD that it sends. */
const ICDForm = PlatformDevice.wrapTaskForm({
  use: Device.use,
  useConfigure: Device.useConnectModal,
  useNoneSelected: Device.useNoneSelected,
  Content: ICDContent,
});

/** Gives each channel binding of an enabled endpoint a command channel. */
const configureICD: Task.OnConfigure<WriteSchemas["config"]> = async (
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
      await ICD.Write.configureChannels(
        client,
        dev.properties.write,
        ep,
        namingOf(dev.name, ep),
      );
    }
  } finally {
    await PlatformDevice.setProperties(client, dev, before, Device.SCHEMAS);
  }
  return [config, dev.rack];
};

export const Write = Task.wrapForm({
  Properties,
  Form: FLAGS.icds ? ICDForm : Form,
  schemas: WRITE_SCHEMAS,
  deployConfigZ: deployWriteConfigZ,
  type: WRITE_TYPE,
  onConfigure: FLAGS.icds ? configureICD : onConfigure,
});
