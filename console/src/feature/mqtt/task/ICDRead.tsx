// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/mqtt/task/Form.css";

import { channel, icd, mqtt, type Synnax } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { ICD as PICD, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { deep, type payload, uuid } from "@synnaxlabs/x";
import { type FC } from "react";

import { ICD } from "@/feature/icd";
import { use } from "@/feature/mqtt/device/queries";
import { Select as SelectDevice, useNoneSelected } from "@/feature/mqtt/device/Select";
import { type Device, MAKE, SCHEMAS } from "@/feature/mqtt/device/types";
import { useConnectModal } from "@/feature/mqtt/device/useConnectModal";
import { namePrefix } from "@/feature/mqtt/task/entries";
import { QoSField } from "@/feature/mqtt/task/QoSField";
import {
  deployICDReadConfigZ,
  ICD_READ_SCHEMAS,
  ICD_READ_TYPE,
  type ICDReadEntry,
  type ICDReadSchemas,
} from "@/feature/mqtt/task/types";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { Panel } from "@/platform/panel";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<ICDReadSchemas>>();

/** The channel data type each field kind reads into. */
const DATA_TYPES: Record<payload.Kind, string> = {
  float: "float64",
  integer: "int64",
  unsigned: "uint64",
  boolean: "uint8",
  string: "string",
  timestamp: "timestamp",
  json: "json",
  uuid: "uuid",
  bytes: "bytes",
};

const TOPIC_INPUT_PROPS = { placeholder: "plant/line1/telemetry" } as const;

const Properties = () => <SelectDevice />;

interface MessageFormatProps {
  icdKey: icd.Key;
  messageKey: icd.MessageKey;
}

/** Edits the fields of one ICD message. Each edit reaches the ICD at once. */
const MessageFormat: FC<MessageFormatProps> = ({ icdKey, messageKey }) => {
  const { form } = PICD.useForm({ query: { key: icdKey } });
  const payloadPath = PICD.formPath.messages[messageKey].payload;
  const format = PForm.useFieldValue(payloadPath.format, {
    optional: true,
    ctx: form,
  });
  if (format == null) return null;
  if (format !== "json")
    return (
      <Text.Text color={9}>
        The message format is not JSON. Change it in the ICD editor.
      </Text.Text>
    );
  return (
    <PForm.Form<typeof PICD.formSchema> {...form}>
      <ICD.Fields
        path={deep.variant(payloadPath, "format", "json").fields}
        encoding="json"
      />
    </PForm.Form>
  );
};

interface EntryProps {
  entryKey: string;
  onRemove: () => void;
}

const Entry: FC<EntryProps> = ({ entryKey, onRemove }) => {
  const path = root.config.entries[entryKey];
  const { topic, icd: icdKey, messages } = PForm.useFieldValue(path);
  const openTab = Panel.useOpenTab();
  const messageKey = messages.at(0)?.message;
  return (
    <PForm.Section title={topic === "" ? "New topic" : topic}>
      <PForm.TextField
        path={path.topic}
        label="Topic"
        padHelpText={false}
        inputProps={TOPIC_INPUT_PROPS}
      />
      <QoSField path={path.qos} />
      <PForm.CheckboxField
        path={path.retainedIgnored}
        label="Ignore retained"
        padHelpText={false}
      />
      {messageKey != null && <MessageFormat icdKey={icdKey} messageKey={messageKey} />}
      <Flex.Box x gap="small">
        <Button.Button
          variant="text"
          onClick={() =>
            openTab({ variant: "resource", resource: icd.resourceID(icdKey) })
          }
        >
          <Icon.Library />
          Open ICD
        </Button.Button>
        <Button.Button variant="text" onClick={onRemove}>
          <Icon.Delete />
          Remove topic
        </Button.Button>
      </Flex.Box>
    </PForm.Section>
  );
};

/**
 * Adds a JSON message to the ICD of the task. Creates that ICD, named after the task,
 * when the task has none.
 * @returns a topic entry that reads the new message.
 */
const createEntry = async (
  client: Synnax,
  taskName: string,
  entries: ICDReadEntry[],
): Promise<ICDReadEntry> => {
  const message: icd.Message = {
    key: uuid.create(),
    name: `Topic ${uuid.create().slice(0, 8)}`,
    payload: { format: "json", fields: [] },
  };
  let icdKey = entries.at(0)?.icd;
  if (icdKey == null)
    ({ key: icdKey } = await client.icds.create({
      name: taskName,
      messages: [message],
    }));
  else {
    await client.icds.retrieve(icdKey);
    await client.icds.dispatch(
      icdKey,
      icd.insert({ path: ["messages"], value: message }),
    );
  }
  return mqtt.icdReadEntryZ.parse({
    icd: icdKey,
    messages: [{ message: message.key }],
  });
};

const Content = (_: PlatformDevice.TaskFormContentProps<Device>) => {
  const client = PSynnax.use();
  const { push, remove } = PForm.useFieldListUtils(root.config.entries);
  const { get } = PForm.useContext();
  const entries = PForm.useFieldValue(root.config.entries);
  const handleAdd = async () => {
    if (client == null) return;
    push(await createEntry(client, get(root.name).value, entries));
  };
  return (
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      <PForm.Sections>
        {entries.map(({ key }) => (
          <Entry key={key} entryKey={key} onRemove={() => remove(key)} />
        ))}
      </PForm.Sections>
      <Button.Button variant="text" onClick={() => void handleAdd()}>
        <Icon.Add />
        Add topic
      </Button.Button>
    </Flex.Box>
  );
};

const Form = PlatformDevice.wrapTaskForm({
  use,
  useConfigure: useConnectModal,
  useNoneSelected,
  Content,
});

const getInitialValues: Task.GetInitialValues<ICDReadSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = ICD_READ_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: "MQTT ICD read task", type: ICD_READ_TYPE, config: cfg };
};

/** Gives every field of each topic's message a channel, and binds it. */
const onConfigure: Task.OnConfigure<ICDReadSchemas["config"]> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({ key: config.device, schemas: SCHEMAS });
  const before = deep.copy(dev.properties);
  try {
    for (const entry of config.entries) {
      if (entry.disabled) continue;
      const [read] = entry.messages;
      const ic = await client.icds.retrieve(entry.icd);
      const message = ic.messages.find(({ key }) => key === read.message);
      if (message?.payload.format !== "json")
        throw new Error(`The message format of topic ${entry.topic} is not JSON`);
      const { fields } = message.payload;
      if (fields.length === 0)
        throw new Error(`The message format of topic ${entry.topic} has no fields`);
      const prefix = namePrefix(dev.name, entry.topic);
      const channels = fields.map((f) => ({
        key: f.key,
        pointer: f.pointer,
        name: `${prefix}_${channel.escapeInvalidName(f.name)}`,
        dataType: DATA_TYPES[f.type],
        channel: read.bindings.find(({ field }) => field === f.key)?.channel ?? 0,
      }));
      const index = fields.find(({ type }) => type === "timestamp")?.key;
      await Task.configureReadChannels({
        client,
        read: dev.properties.read,
        propertiesKey: entry.topic,
        fields: channels,
        namePrefix: prefix,
        indexKey: index ?? "",
      });
      entry.messages = [
        {
          message: read.message,
          index,
          bindings: channels.map(({ key, channel }) => ({ field: key, channel })),
        },
      ];
    }
  } finally {
    await PlatformDevice.setProperties(client, dev, before, SCHEMAS);
  }
  return [config, dev.rack];
};

export const ICDRead = Task.wrapForm({
  Properties,
  Form,
  schemas: ICD_READ_SCHEMAS,
  deployConfigZ: deployICDReadConfigZ,
  type: ICD_READ_TYPE,
  onConfigure,
});

export const useCreateICDRead = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const ICDReadSelectable: Selector.Selectable = {
  type: ICD_READ_TYPE,
  title: "MQTT ICD read task",
  label: "Read with message format",
  icon: <Icon.Logo.MQTT />,
  useOnSelect: useCreateICDRead,
};
