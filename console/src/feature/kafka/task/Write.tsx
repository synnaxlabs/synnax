// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/kafka/task/Task.css";

import { DisconnectedError, kafka } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Channel as PChannel, Synnax } from "@synnaxlabs/pluto";
import { deep, id, type payload, primitive } from "@synnaxlabs/x";
import { type FC, useEffect, useRef } from "react";

import { Select as SelectDevice } from "@/feature/kafka/device/Select";
import { MAKE, SCHEMAS } from "@/feature/kafka/device/types";
import {
  deployWriteConfigZ,
  type RecordKey,
  type TimeFormat,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteChannel,
  type WriteSchemas,
} from "@/feature/kafka/task/types";
import { CSS } from "@/platform/css";
import { Form as PlatformForm } from "@/platform/form";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<WriteSchemas>>();

const SELECT_TRIGGER_PROPS = { className: CSS.BE("kafka-select", "trigger") };

const TEXT_FIELD_CLASS = CSS.BE("kafka-properties", "text");

const SELECT_DIALOG_PROPS = { className: CSS.BE("kafka-select", "dialog") };

const renderRecordKeySelect = Component.renderProp(
  (p: Omit<Select.SingleSimpleProps<RecordKey>, "children" | "resourceName">) => (
    <Select.Simple<RecordKey>
      {...p}
      resourceName="record key"
      triggerProps={SELECT_TRIGGER_PROPS}
      dialogProps={SELECT_DIALOG_PROPS}
    >
      <Select.Item itemKey="channel_name">Channel name</Select.Item>
      <Select.Item itemKey="none">None</Select.Item>
    </Select.Simple>
  ),
);

const renderTimeFormatSelect = Component.renderProp(
  (p: Omit<Select.SingleSimpleProps<TimeFormat>, "children" | "resourceName">) => (
    <Select.Simple<TimeFormat>
      {...p}
      resourceName="time format"
      triggerProps={SELECT_TRIGGER_PROPS}
      dialogProps={SELECT_DIALOG_PROPS}
    >
      <Select.Item itemKey="iso8601">ISO 8601</Select.Item>
      <Select.Item itemKey="unix_sec">Unix (s)</Select.Item>
      <Select.Item itemKey="unix_ms">Unix (ms)</Select.Item>
      <Select.Item itemKey="unix_us">Unix (µs)</Select.Item>
      <Select.Item itemKey="unix_ns">Unix (ns)</Select.Item>
    </Select.Simple>
  ),
);

const Properties = () => (
  <>
    <SelectDevice />
    <Flex.Box x grow wrap>
      <PForm.TextField
        className={TEXT_FIELD_CLASS}
        path={root.config.topic}
        label="Topic"
        inputProps={TOPIC_INPUT_PROPS}
        grow
      />
      <PForm.Field path={root.config.recordKey} label="Record key">
        {renderRecordKeySelect}
      </PForm.Field>
    </Flex.Box>
    <Flex.Box x grow wrap>
      <PForm.TextField
        className={TEXT_FIELD_CLASS}
        path={root.config.record.valuePointer}
        label="Value pointer"
        inputProps={VALUE_POINTER_INPUT_PROPS}
        grow
      />
      <PForm.TextField
        className={TEXT_FIELD_CLASS}
        path={root.config.record.channelPointer}
        label="Channel pointer"
        inputProps={CHANNEL_POINTER_INPUT_PROPS}
        grow
      />
      <PForm.TextField
        className={TEXT_FIELD_CLASS}
        path={root.config.record.timestampPointer}
        label="Timestamp pointer"
        inputProps={TIMESTAMP_POINTER_INPUT_PROPS}
        grow
      />
      <PForm.Field path={root.config.record.timeFormat} label="Time format">
        {renderTimeFormatSelect}
      </PForm.Field>
    </Flex.Box>
  </>
);

const TOPIC_INPUT_PROPS = { placeholder: "synnax-samples" } as const;

const VALUE_POINTER_INPUT_PROPS = { placeholder: "/value" } as const;

const CHANNEL_POINTER_INPUT_PROPS = { placeholder: "Omitted" } as const;

const TIMESTAMP_POINTER_INPUT_PROPS = { placeholder: "Omitted" } as const;

const ChannelListItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = root.config.channels[itemKey];
  const { channel: key, type, disabled } = PForm.useFieldValue(path);
  return (
    <Select.Item
      {...props}
      justify="between"
      align="center"
      x
      className={CSS.cls(CSS.B("kafka-item"), disabled && CSS.M("off"))}
    >
      <Flex.Box x align="center" gap="small" className={CSS.BE("kafka-item", "label")}>
        <Task.ChannelName
          channel={key}
          picked
          level="p"
          weight={500}
          color={10}
          className={CSS.BE("kafka-item", "name")}
        />
        <Text.Text level="small" color={9}>
          {Task.Payload.kindName(type)}
        </Text.Text>
      </Flex.Box>
      <Task.EnabledCheckbox path={path.disabled} />
    </Select.Item>
  );
};

const listItem = Component.renderProp(ChannelListItem);

type NumericKind = "float" | "integer" | "unsigned";

const isNumericKind = (kind: payload.Kind): kind is NumericKind =>
  kind === "float" || kind === "integer" || kind === "unsigned";

interface NumericEntry {
  label: string;
  value: number | string;
}

const numericEntries = (ch: WriteChannel): NumericEntry[] =>
  ch.type === "float" || ch.type === "integer" || ch.type === "unsigned"
    ? ch.enumValues
    : [];

const MAX_EXACT_FLOAT = 2n ** 53n;

const RANGES: Record<"integer" | "unsigned", [bigint, bigint]> = {
  integer: [-(2n ** 63n), 2n ** 63n - 1n],
  unsigned: [0n, 2n ** 64n - 1n],
};

const exactInteger = (value: number | string): bigint | null => {
  if (typeof value === "number") return Number.isInteger(value) ? BigInt(value) : null;
  return /^-?[0-9]+$/.test(value) ? BigInt(value) : null;
};

/** Returns value as an entry value of kind, or null when it does not stay exact. */
const convertValue = (
  value: number | string,
  kind: NumericKind,
): number | string | null => {
  if (kind === "float" && typeof value === "number") return value;
  const n = exactInteger(value);
  if (n == null) return null;
  if (kind === "float")
    return n >= -MAX_EXACT_FLOAT && n <= MAX_EXACT_FLOAT ? Number(n) : null;
  const [min, max] = RANGES[kind];
  return n >= min && n <= max ? n.toString() : null;
};

/**
 * Returns ch written as kind, with the defaults of kind. Between numeric kinds, each
 * entry that stays exact carries over.
 */
const asKind = (ch: WriteChannel, kind: payload.Kind): WriteChannel =>
  kafka.writeChannelZ.parse({
    key: ch.key,
    disabled: ch.disabled,
    channel: ch.channel,
    type: kind,
    ...(kind === "timestamp" ? { timeFormat: "iso8601" } : {}),
    ...(isNumericKind(kind)
      ? {
          enumValues: numericEntries(ch).flatMap(({ label, value }) => {
            const converted = convertValue(value, kind);
            return converted == null ? [] : [{ label, value: converted }];
          }),
        }
      : {}),
  });

interface KindSelectProps {
  path: deep.Target<WriteChannel>;
  kinds: payload.Kind[];
}

const KindSelect = ({ path, kinds }: KindSelectProps) => (
  <PForm.Field<WriteChannel> path={path} label="Output type" padHelpText={false}>
    {({ value, onChange }) => (
      <Select.Simple<payload.Kind>
        value={value.type}
        onChange={(kind: payload.Kind) => onChange(asKind(value, kind))}
        resourceName="kind"
      >
        {kinds.map((kind) => (
          <Select.Item key={kind} itemKey={kind}>
            {Task.Payload.kindName(kind)}
          </Select.Item>
        ))}
      </Select.Simple>
    )}
  </PForm.Field>
);

interface EntriesSectionProps {
  path: deep.Target<WriteChannel>;
  type: payload.Kind;
}

const EntriesSection = ({ path, type }: EntriesSectionProps) => {
  switch (type) {
    case "float":
      return (
        <Task.Payload.EnumMappingSection
          path={deep.variant(path, "type", "float").enumValues}
          valueFirst
        />
      );
    case "integer":
    case "unsigned":
    case "string":
      return (
        <PForm.Section title="Enum mapping">
          <PlatformForm.KeyValueEditor
            path={deep.variant(path, "type", type).enumValues}
            resourceName="mapping"
            keyField="label"
            keyPlaceholder="String (e.g. ON)"
            valuePlaceholder={type === "string" ? "Text" : "Integer"}
            valueFirst
          />
        </PForm.Section>
      );
    case "boolean":
      return (
        <PForm.Section title="Enum mapping">
          <PlatformForm.KeyValueEditor
            path={deep.variant(path, "type", "boolean").enumValues}
            resourceName="mapping"
            keyField="label"
            keyPlaceholder="String (e.g. ON)"
            valueType="boolean"
            valueFirst
          />
        </PForm.Section>
      );
    default:
      return null;
  }
};

/**
 * Selects the channel of the write channel at path. It sets the channel and an output
 * type that the channel allows in one change, so the Core never sees a pair it rejects.
 */
const ChannelField = ({ path }: Task.Views.DetailsProps<WriteChannel>) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const { get, set } = PForm.useContext();
  const latest = useRef(0);
  const handleChange = (key: number) => {
    const pick = ++latest.current;
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      const { dataType } = await client.channels.retrieve(key);
      // A slow lookup for an earlier pick must not replace a later one.
      if (pick !== latest.current) return;
      const kinds = Task.Payload.channelKinds(dataType);
      const ch = { ...get<WriteChannel>(path).value, channel: key };
      set(path, kinds.includes(ch.type) ? ch : asKind(ch, kinds[0]));
    }, "Failed to select the channel");
  };
  return (
    <PForm.Field path={path.channel} label="Channel" required>
      {({ value }) => <PChannel.SelectSingle value={value} onChange={handleChange} />}
    </PForm.Field>
  );
};

const ChannelDetails = ({ path }: Task.Views.DetailsProps<WriteChannel>) => {
  const ch = PForm.useFieldValue(path);
  const { set } = PForm.useContext();
  const query = primitive.isNonZero(ch.channel) ? { key: ch.channel } : null;
  const { data: dataType } = PChannel.useResultDataType(query);
  const kinds = dataType == null ? null : Task.Payload.channelKinds(dataType);
  const allowed = kinds == null || kinds.includes(ch.type);
  useEffect(() => {
    if (kinds != null && !allowed) set(path, asKind(ch, kinds[0]));
  }, [kinds?.[0], allowed]);
  return (
    <PForm.Sections className={CSS.B("kafka-details")}>
      <PForm.Section title="Channel">
        <ChannelField path={path} />
        {kinds != null && kinds.length > 1 && <KindSelect path={path} kinds={kinds} />}
        {ch.type === "timestamp" && (
          <Task.Payload.TimeFormatField
            path={deep.variant(path, "type", "timestamp").timeFormat}
            label="Time format"
          />
        )}
      </PForm.Section>
      <EntriesSection path={path} type={ch.type} />
    </PForm.Sections>
  );
};

const details = Component.renderProp(ChannelDetails);

const ChannelTitle = ({ path }: Task.Views.DetailsProps<WriteChannel>) => {
  const key = PForm.useFieldValue(path.channel);
  return <Task.ChannelName channel={key} picked level="p" weight={500} color={10} />;
};

const detailsTitle = Component.renderProp(ChannelTitle);

const createChannel = (channels: WriteChannel[], keyToCopy?: string): WriteChannel => {
  const source = channels.find((c) => c.key === keyToCopy);
  return {
    ...(source != null
      ? { ...source, channel: 0 }
      : kafka.writeChannelZ.parse({ type: "float" })),
    key: id.create(),
  };
};

/** The channels are chosen in the form, so the device binds none. */
const resolveNothing = () => null;

const Form: FC = () => (
  <Task.Views.ListAndDetails<WriteChannel>
    path={root.config.channels}
    listItem={listItem}
    details={details}
    detailsTitle={detailsTitle}
    createChannel={createChannel}
    resolve={resolveNothing}
  />
);

const getInitialValues: Task.GetInitialValues<WriteSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = WRITE_SCHEMAS.config.parse(
    config ?? {
      record: { channelPointer: "/channel", timestampPointer: "/timestamp" },
    },
  );
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: "Kafka write task", type: WRITE_TYPE, config: cfg };
};

/** A cleared pointer is omitted from the record instead of placed at "". */
const onConfigure: Task.OnConfigure<WriteSchemas["config"]> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: SCHEMAS,
  });
  if (config.record.channelPointer === "") delete config.record.channelPointer;
  if (config.record.timestampPointer === "") delete config.record.timestampPointer;
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
  title: "Kafka write task",
  label: "Write",
  icon: <Icon.Logo.Kafka />,
  useOnSelect: useCreateWrite,
};
