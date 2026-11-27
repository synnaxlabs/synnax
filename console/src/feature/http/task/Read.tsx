// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/http/task/Form.css";

import { channel, DisconnectedError, http, icd } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { useExclusiveCallback } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import type { Tree } from "@synnaxlabs/lyra/tree";
import { ICD as PICD, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { deep, uuid } from "@synnaxlabs/x";
import {
  type Dispatch,
  type FC,
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  type RefObject,
  type SetStateAction,
  useRef,
  useState,
} from "react";

import { Device } from "@/feature/http/device";
import { DEFAULT_READ_NAME } from "@/feature/http/task/create";
import { EndpointItem, EndpointLabel } from "@/feature/http/task/EndpointLabel";
import {
  deployReadConfigZ,
  READ_SCHEMAS,
  type ReadEndpoint,
  type ReadSchemas,
} from "@/feature/http/task/types";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { Form as PlatformForm } from "@/platform/form";
import { ICD } from "@/platform/icd";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<ReadSchemas>>();

const RATE_INPUT_PROPS = {
  endContent: "Hz",
  className: CSS.B("rate-input"),
} as const;

const Properties = () => (
  <>
    <Device.Select />
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

/** @returns the prefix of the channel names that deploying creates for a path. */
const endpointPrefix = (deviceName: string, path: string): string =>
  channel.escapeInvalidName(deviceName) + channel.escapeInvalidName(path);

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

interface RequestSectionProps extends PropsWithChildren {
  path: deep.Path<ReadEndpoint>;
  actions?: ReactNode;
}

/** The request that polls an endpoint. Children follow the path and body. */
const RequestSection: FC<RequestSectionProps> = ({ path, actions, children }) => {
  const method = PForm.useFieldValue(path.method);
  return (
    <PForm.Section title="Request" actions={actions}>
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
      {children}
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
};

const PATH_INPUT_PROPS = { placeholder: "/api/data" } as const;

const REQUEST_BODY_INPUT_PROPS = {
  placeholder: '{"query": "latest"}',
  area: true,
  className: CSS.B("request-body"),
} as const;

/** An endpoint that reads the messages of an ICD. */
interface ICDEndpoint extends ReadEndpoint {
  icd: icd.Key;
}

const readsICD = (ep: ReadEndpoint): ep is ICDEndpoint => ep.icd != null;

/**
 * @returns the path of an endpoint that reads an ICD. Nothing checks it at run time.
 */
const icdPath = (key: string) => root.config.endpoints[key] as deep.Path<ICDEndpoint>;

/**
 * @returns where the device stores the channels of the messages of an endpoint. An
 * older Console stored them under the bare path.
 */
const storedSource = ({
  path,
  messages,
}: Pick<ICDEndpoint, "path" | "messages">): ICD.Stored.Source => ({
  key: path,
  legacy: path,
  messages,
});

/**
 * @returns stored with the channels that an older Console stored under the path of
 * endpoints split among the messages that read the path. An endpoint that reads no ICD
 * keeps them under the path.
 */
const splitStored = (
  stored: Record<string, Task.ReadChannelProps>,
  endpoints: ReadEndpoint[],
): Record<string, Task.ReadChannelProps> =>
  ICD.Stored.split(
    stored,
    endpoints.map((ep) =>
      readsICD(ep)
        ? { ...storedSource(ep), disabled: ep.disabled }
        : { key: ep.path, legacy: ep.path, messages: [] },
    ),
  );

/**
 * @returns whether ep stores and reuses the channels of message: no endpoint before it
 * in endpoints reads message from the path of ep.
 */
const ownsStored = (
  endpoints: ReadEndpoint[],
  ep: ICDEndpoint,
  message: icd.MessageKey,
): boolean =>
  endpoints.find(
    ({ path, messages }) =>
      path === ep.path && messages.some((read) => read.message === message),
  )?.key === ep.key;

/**
 * @returns stored without the channels of messages that other endpoints with the path
 * of ep read, which the device stores beside the channels of ep. It keeps a message
 * that ep reads only when ep owns its channels.
 */
const storedFor = (
  stored: Record<string, Task.ReadChannelProps>,
  endpoints: ReadEndpoint[],
  ep: ICDEndpoint,
): Record<string, Task.ReadChannelProps> => {
  const owned = new Set(
    ep.messages
      .filter(({ message }) => ownsStored(endpoints, ep, message))
      .map(({ message }) => ICD.Stored.key(ep.path, message)),
  );
  const others = new Set(
    endpoints
      .filter(({ key, path }) => key !== ep.key && path === ep.path)
      .flatMap(({ messages }) =>
        messages.map(({ message }) => ICD.Stored.key(ep.path, message)),
      )
      .filter((at) => !owned.has(at)),
  );
  return Object.fromEntries(Object.entries(stored).filter(([at]) => !others.has(at)));
};

/**
 * @returns the channels that endpoints bind. A disabled endpoint keeps its channels for
 * when it is enabled again.
 */
const writtenChannels = (endpoints: ReadEndpoint[]): Set<channel.Key> => {
  const written = new Set(
    endpoints.flatMap(({ messages }) =>
      messages.flatMap(({ bindings }) => bindings.map((b) => b.channel)),
    ),
  );
  written.delete(0);
  return written;
};

/**
 * @returns a name for a new message of an endpoint, after its method and path, that no
 * message in taken has. A taken name gets a number, such as `GET /status 2`.
 */
const messageName = (
  { method, path }: Pick<ReadEndpoint, "method" | "path">,
  taken: icd.Message[],
): string => {
  const names = new Set(taken.map(({ name }) => name));
  const base = `${method} ${path}`.trim();
  let name = base;
  for (let n = 2; names.has(name); n++) name = `${base} ${n}`;
  return name;
};

/** @returns the name of the ICD of a task named name. */
const icdName = (name: string): string =>
  name.trim() === "" ? DEFAULT_READ_NAME : name;

/** @returns the prefix of the channel names of the messages of an endpoint. */
const usePrefix = (deviceName: string, path: deep.Path<ICDEndpoint>): string =>
  endpointPrefix(deviceName, PForm.useFieldValue(path.path));

const MISSING_DESCRIPTION =
  "The ICD no longer has this message. Add a message to the endpoint, then remove this one.";

const MessageMark = ({ itemKey }: { itemKey: string }) => {
  const path = icdPath(itemKey);
  const icdKey = PForm.useFieldValue(path.icd, { optional: true });
  const messages = PForm.useFieldValue(path.messages, { optional: true });
  if (icdKey == null || messages == null) return null;
  return (
    <ICD.MessageMark
      icdKey={icdKey}
      messages={messages.map(({ message }) => message)}
      missingDescription={MISSING_DESCRIPTION}
    />
  );
};

const SourceItem = (props: Tree.ItemRenderProps<string>) => (
  <EndpointItem {...props} path={root.config.endpoints[props.itemKey]}>
    <MessageMark itemKey={props.itemKey} />
  </EndpointItem>
);

/** @returns the props that the panels of a message of an endpoint share. */
const useMessageProps = (
  device: Device.Device,
  path: deep.Path<ICDEndpoint>,
  read: number,
): Omit<ICD.MessageSectionsProps, "format"> => {
  const { index, bindings } = PForm.useFieldValue(path.messages[read]);
  const prefix = usePrefix(device.name, path);
  return {
    path,
    read,
    prefix,
    // A new index field keeps the channel of the old one.
    indexChannel: bindings.find(({ field }) => field === index)?.channel ?? 0,
    sourceLabel: "Poll time",
    missingDescription: MISSING_DESCRIPTION,
  };
};

interface MessagePanelProps {
  path: deep.Path<ICDEndpoint>;
  /** The index of the message in the endpoint's reads. */
  read: number;
  format: ICD.FormatForm;
  device: Device.Device;
}

const MessageSections = ({ device, path, read, format }: MessagePanelProps) => {
  const props = useMessageProps(device, path, read);
  return <ICD.MessageSections {...props} format={format} />;
};

interface MessageDetailsProps extends MessagePanelProps {
  onAddField: () => void;
  bindDisabled: boolean;
}

const MessageDetails = ({
  device,
  path,
  read,
  format,
  onAddField,
  bindDisabled,
}: MessageDetailsProps) => {
  const props = useMessageProps(device, path, read);
  return (
    <ICD.MessageDetails
      {...props}
      format={format}
      matchTitle="Responses"
      anyLabel="Any response"
      onAddField={onAddField}
      addFieldDisabled={bindDisabled}
    />
  );
};

interface FieldDetailsProps extends MessagePanelProps {
  field: ICD.TreeField;
}

const FieldDetails = ({ device, ...rest }: FieldDetailsProps) => {
  const prefix = usePrefix(device.name, rest.path);
  return <ICD.FieldDetails {...rest} prefix={prefix} />;
};

interface EndpointDetailsProps {
  /** The key of the endpoint. */
  endpoint: string;
  /** The open form of the ICD of the endpoint. Absent when the endpoint reads none. */
  format?: ICD.FormatForm;
  device: Device.Device;
  /** Adds a field to the message of an endpoint that reads one. */
  onAddField: () => void;
  /** Adds a message to the endpoint, for responses that its messages do not cover. */
  onAddMessage: () => void;
  /** Reads a message of an ICD instead of the one message that the endpoint reads. */
  onFormatChange: (icdKey: icd.Key, message: icd.Message) => void;
  bindDisabled: boolean;
}

/**
 * The request of an endpoint. An endpoint that reads one message also shows its
 * timestamp, which each message of an endpoint that reads several shows on its own.
 */
const EndpointDetails: FC<EndpointDetailsProps> = ({
  endpoint,
  format,
  device,
  onAddField,
  onAddMessage,
  onFormatChange,
  bindDisabled,
}) => {
  const isPreview = Task.useIsPreview();
  const path = root.config.endpoints[endpoint];
  const reads = PForm.useFieldValue(path.messages);
  const icdKey = PForm.useFieldValue(icdPath(endpoint).icd);
  const encoding = PForm.useFieldValue(
    PICD.formPath.messages[reads.length === 1 ? reads[0].message : ""].payload.format,
    { optional: true, ctx: format },
  );
  return (
    <Flex.Box y grow empty className={CSS.B("endpoint-details")}>
      <PForm.Sections className={CSS.B("endpoint-form")}>
        <RequestSection
          path={path}
          actions={
            !isPreview && (
              <>
                {encoding === "json" && (
                  <ICD.AddField onClick={onAddField} disabled={bindDisabled} />
                )}
                <Button.Button variant="text" size="small" onClick={onAddMessage}>
                  <Icon.Add />
                  Add message
                </Button.Button>
              </>
            )
          }
        >
          {reads.length === 1 && (
            <ICD.MessageFormatField
              icdKey={icdKey}
              message={reads[0].message}
              onChange={onFormatChange}
              disabled={bindDisabled}
            />
          )}
        </RequestSection>
        {format != null && reads.length === 1 && (
          <MessageSections
            path={icdPath(endpoint)}
            read={0}
            format={format}
            device={device}
          />
        )}
      </PForm.Sections>
    </Flex.Box>
  );
};

interface EndpointsProps {
  device: Device.Device;
  /** The key of each selected tree item. */
  selected: string[];
  onSelect: Dispatch<SetStateAction<string[]>>;
  expanded: string[];
  onExpand: Dispatch<SetStateAction<string[]>>;
  /** The key of each message that the task made. */
  made: RefObject<Set<string>>;
}

/** The endpoints of the task as a tree, beside the details of the selected item. */
const Endpoints: FC<EndpointsProps> = ({
  device,
  selected,
  onSelect,
  expanded,
  onExpand,
  made,
}) => {
  const client = PSynnax.use();
  const ctx = PForm.useContext();
  const handleError = Status.useErrorHandler();
  const formats = ICD.useFormats();
  const { push } = PForm.useFieldList(root.config.endpoints);
  const endpoints = PForm.useFieldValue(root.config.endpoints);
  const stored = device.properties.read;
  const { dataTypes, bindDisabled } = ICD.Stored.useDataTypes(stored);
  const {
    tree,
    handleSelect,
    contextMenu,
    renderItem,
    current,
    addField,
    addMessage,
    setMessage,
  } = ICD.Tree.use({
    sources: root.config.endpoints,
    sourceOf: (ep) => (readsICD(ep) ? ep : null),
    pathOf: icdPath,
    SourceItem,
    missingDescription: MISSING_DESCRIPTION,
    tareable: true,
    // Starting reuses the channel that the device stores for the field's pointer,
    // unless an endpoint reads it, another endpoint owns its message, or it no longer
    // exists or fits the field. An endpoint that reads one message also reuses the
    // channels of its earlier messages, so a new message format keeps its channels.
    bindDisabled,
    bindingOf: (key, read, field) => {
      const ep = ctx.get(icdPath(key)).value;
      const all = ctx.get(root.config.endpoints).value;
      const held = storedFor(splitStored(stored, all), all, ep);
      const channels =
        ep.messages.length === 1
          ? ICD.Stored.channels(held, storedSource(ep), ep.messages[0].message)
          : (ICD.Stored.of(held, storedSource(ep), read)?.channels ?? {});
      return ICD.Stored.bindingsOf(
        channels,
        [field],
        writtenChannels(all),
        dataTypes,
      )[0];
    },
    messageName: (key, held) =>
      messageName(ctx.get(root.config.endpoints[key]).value, held),
    made,
    selected,
    onSelect,
    expanded,
    onExpand,
  });

  /**
   * @returns ep reading a new message with no fields, in the ICD that the most
   * endpoints read, or in a new ICD named after the task when no endpoint reads one.
   */
  const withMessage = async (ep: ReadEndpoint): Promise<ICDEndpoint> => {
    if (client == null) throw new DisconnectedError();
    const home = ICD.mostRead(ctx.get(root.config.endpoints).value.filter(readsICD));
    const format =
      home == null ? null : { key: home, form: ICD.formatOf(formats, home) };
    const held = format?.form.value().messages ?? [];
    const message: icd.Message = {
      key: uuid.create(),
      name: messageName(ep, held),
      payload: { format: "json", fields: [] },
    };
    let icdKey: icd.Key;
    if (format == null) {
      const name = icdName(ctx.get(root.name).value);
      icdKey = (await client.icds.create({ name, messages: [message] })).key;
    } else {
      icdKey = format.key;
      format.form.insert(PICD.formPath.messages, message, held.at(-1)?.key);
    }
    made.current.add(message.key);
    return {
      ...ep,
      icd: icdKey,
      messages: [icd.readMessageZ.parse({ message: message.key })],
    };
  };

  // Adds run one at a time, so a second click cannot make a second ICD.
  const runExclusive = useExclusiveCallback(
    async (add: () => Promise<void>) => await add(),
  );
  const runAdd = (add: () => Promise<void>, message: string) =>
    handleError(async () => await runExclusive(add), message);

  const handleAdd = () =>
    runAdd(async () => {
      const ep = await withMessage(http.readEndpointZ.parse({}));
      push(ep);
      onSelect([ep.key]);
      tree.expand(ep.key);
    }, "Failed to add endpoint");

  const handleAddMessage = (key: string) => {
    const path = root.config.endpoints[key];
    const ep = ctx.get(path).value;
    if (readsICD(ep)) {
      addMessage(key);
      return;
    }
    runAdd(async () => ctx.set(path, await withMessage(ep)), "Failed to add message");
  };

  const ep =
    current == null ? undefined : endpoints.find(({ key }) => key === current.source);
  const format = ep != null && readsICD(ep) ? formats.get(ep.icd) : undefined;
  let details = null;
  let title = null;
  let detailsPath: deep.Target<unknown> | null = null;
  if (ep != null && format != null && current?.kind === "field") {
    const read = icdPath(ep.key).messages[current.read];
    const i = ep.messages[current.read].bindings.findIndex(
      ({ field }) => field === current.field.key,
    );
    detailsPath = i < 0 ? read : read.bindings[i];
    title = (
      <Text.Text level="p" weight={500} overflow="fade">
        {current.field.name}
      </Text.Text>
    );
    details = (
      <FieldDetails
        key={selected[0]}
        path={icdPath(ep.key)}
        read={current.read}
        field={current.field}
        format={format}
        device={device}
      />
    );
  } else if (ep != null && format != null && current?.kind === "message") {
    detailsPath = icdPath(ep.key).messages[current.read];
    title = <ICD.MessageLabel path={icdPath(ep.key)} read={current.read} />;
    details = (
      <MessageDetails
        key={selected[0]}
        path={icdPath(ep.key)}
        read={current.read}
        format={format}
        device={device}
        onAddField={() => addField(ep.key, current.read)}
        bindDisabled={bindDisabled}
      />
    );
  } else if (ep != null) {
    detailsPath = root.config.endpoints[ep.key];
    title = <EndpointLabel path={root.config.endpoints[ep.key]} />;
    details = (
      <EndpointDetails
        key={ep.key}
        endpoint={ep.key}
        format={format}
        device={device}
        onAddField={() => addField(ep.key, 0)}
        onAddMessage={() => handleAddMessage(ep.key)}
        onFormatChange={(icdKey, message) => setMessage(ep.key, icdKey, message)}
        bindDisabled={bindDisabled}
      />
    );
  }
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
      detailsPath={detailsPath}
      title={title ?? undefined}
    >
      {details ?? (
        <Flex.Box y grow align="center" justify="center">
          <Text.Text status="disabled">
            Select an endpoint or field to configure
          </Text.Text>
        </Flex.Box>
      )}
    </Task.Views.Panes>
  );
};

/** A message that an endpoint reads. */
type BoundMessage = ICD.BoundMessage<ICDEndpoint>;

/** @returns the key of the device properties that store the channels of a message. */
const storedKeyOf = ({ source, read }: BoundMessage): string =>
  ICD.Stored.key(source.path, source.messages[read].message);

/**
 * @returns the channels that stored, split by splitStored, holds for b, or undefined
 * when an earlier endpoint of endpoints owns them.
 */
const storedPropsOf = (
  stored: Record<string, Task.ReadChannelProps>,
  endpoints: ReadEndpoint[],
  b: BoundMessage,
): Task.ReadChannelProps | undefined =>
  ownsStored(endpoints, b.source, b.source.messages[b.read].message)
    ? ICD.Stored.of(stored, storedSource(b.source), b.read)
    : undefined;

/**
 * Plans the channels of each message of endpoints in bound, in deploy order with one
 * planner. A message plans with the channels that earlier ones keep, so it leaves a
 * stored channel that another binding writes.
 */
const planMessages = (
  planner: Task.Planner,
  dev: Device.Device,
  endpoints: ReadEndpoint[],
  bound: BoundMessage[],
): ICD.MessagePlan[] => {
  const written = writtenChannels(endpoints);
  const stored = splitStored(dev.properties.read, endpoints);
  const plans: ICD.MessagePlan[] = [];
  for (const b of bound) {
    const plan = ICD.planMessage(
      planner,
      b,
      endpointPrefix(dev.name, b.source.path),
      storedPropsOf(stored, endpoints, b),
      written,
      bound,
    );
    ICD.keptChannels(plan).forEach((ch) => written.add(ch));
    plans.push(plan);
  }
  return plans;
};

interface MessagePlansProps extends PropsWithChildren {
  device: Device.Device;
}

/**
 * Plans the channels of every message that the task reads, together and as deploying
 * plans them, for the tree and the panes.
 */
const MessagePlans = ({ device, children }: MessagePlansProps): ReactElement => {
  const endpoints = PForm.useFieldValue(root.config.endpoints);
  const messages = ICD.useMessages();
  const bound = endpoints
    .filter(readsICD)
    .filter(({ disabled }) => !disabled)
    .flatMap((source) =>
      source.messages.flatMap((read, r): BoundMessage[] => {
        const message = ICD.messageOf(messages, source, r);
        if (message == null) return [];
        const fields = ICD.readFields(ICD.jsonFields(message), read);
        return [{ source, read: r, message, fields }];
      }),
    );
  const stored = splitStored(device.properties.read, endpoints);
  const sources = bound.map((b) => ({
    key: b.source.key,
    position: b.read,
    read: b.source.messages[b.read],
    message: b.message,
    prefix: endpointPrefix(device.name, b.source.path),
    props: storedPropsOf(stored, endpoints, b),
  }));
  const plans = Task.useChannelPlans(sources, (planner) =>
    planMessages(planner, device, endpoints, bound),
  );
  const reads = bound.map(({ source, read }, i): ICD.PlannedRead => ({
    path: icdPath(source.key),
    read,
    plan: plans[i],
  }));
  return <ICD.Plans plans={reads}>{children}</ICD.Plans>;
};

const Content = ({ device }: { device: Device.Device }) => {
  const endpoints = PForm.useFieldValue(root.config.endpoints);
  const [selected, setSelected] = useState<string[]>([]);
  const [expanded, setExpanded] = useState<string[]>(() =>
    ICD.Tree.initialExpanded(endpoints),
  );
  const made = useRef(new Set<string>());
  const reading = endpoints.filter(readsICD);
  const body = (
    <MessagePlans device={device}>
      <Endpoints
        device={device}
        selected={selected}
        onSelect={setSelected}
        expanded={expanded}
        onExpand={setExpanded}
        made={made}
      />
      {reading.flatMap(({ key, messages }) =>
        messages.map(({ message }, read) => (
          <ICD.SuitRead key={`${key}/${message}`} path={icdPath(key)} read={read} />
        )),
      )}
    </MessagePlans>
  );
  return (
    <ICD.WithFormats icdKeys={reading.map((ep) => ep.icd)}>{body}</ICD.WithFormats>
  );
};

/** The endpoints of the task, each with the messages of its ICD that it reads. */
const Form = PlatformDevice.wrapTaskForm({
  use: Device.use,
  useConfigure: Device.useConnectModal,
  useNoneSelected: Device.useNoneSelected,
  Content,
});

/** Gives each field that an enabled endpoint reads a channel, and binds it. */
const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (client, config) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: Device.SCHEMAS,
  });
  const enabled = config.endpoints.filter(readsICD).filter(({ disabled }) => !disabled);
  const messages = await ICD.retrieveMessages(
    client,
    enabled.map((ep) => ep.icd),
  );
  // Every format is checked before any channel is created.
  const bound = enabled.flatMap((ep) =>
    ep.messages.map((_, read) =>
      ICD.boundMessage(messages, ep, read, `endpoint ${ep.method} ${ep.path}`),
    ),
  );
  // A deploy that fails partway keeps the channels it made for the next one.
  const before = deep.copy(dev.properties);
  try {
    // Each endpoint gets its share of the channels stored under its path, even when it
    // is disabled or a create below fails.
    dev.properties.read = splitStored(dev.properties.read, config.endpoints);
    const plans = await Task.planChannels(client, (planner) =>
      planMessages(planner, dev, config.endpoints, bound),
    );
    for (const [i, b] of bound.entries()) {
      const message = b.source.messages[b.read].message;
      const stored = ownsStored(config.endpoints, b.source, message)
        ? { read: dev.properties.read, key: storedKeyOf(b) }
        : undefined;
      await ICD.applyMessage(client, b, plans[i], stored);
    }
  } finally {
    await PlatformDevice.setProperties(client, dev, before, Device.SCHEMAS);
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
