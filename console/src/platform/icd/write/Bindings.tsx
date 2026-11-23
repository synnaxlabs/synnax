// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { bounds, DataType, deep, TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { defaultDataType, encodes } from "@/platform/icd/dataType";
import { formatDefault } from "@/platform/icd/Options";
import { rawBounds } from "@/platform/icd/range";
import { ValueField } from "@/platform/icd/ValueField";
import { Task } from "@/platform/task";

type Kind = icd.JSONField["type"];

/** What fills a field: a command channel, a fixed value, or the field's default. */
type Mode = icd.WriteBindingType | "default";

/** @returns the number in b nearest zero, or zero when b holds none. */
const nearZero = (b: bounds.Bounds): number =>
  b.lower > b.upper ? 0 : bounds.clamp(b, 0);

/** @returns the whole numbers of b, at or above floor. */
const whole = ({ lower, upper }: bounds.Bounds, floor = -Infinity): bounds.Bounds => ({
  lower: Math.max(Math.ceil(lower), floor),
  upper: Math.floor(upper),
});

/** The fixed value that a field starts with, one the Core accepts when one fits. */
const INITIAL_VALUES: {
  [K in Kind]: (field: icd.JSONField) => Extract<icd.WriteValue, { type: K }>;
} = {
  float: (f) => ({ type: "float", value: nearZero(rawBounds(f)) }),
  integer: (f) => ({ type: "integer", value: String(nearZero(whole(rawBounds(f)))) }),
  unsigned: (f) => ({
    type: "unsigned",
    value: String(nearZero(whole(rawBounds(f), 0))),
  }),
  string: () => ({ type: "string", value: "" }),
  boolean: () => ({ type: "boolean", value: false }),
  timestamp: () => ({
    type: "timestamp",
    value: TimeStamp.now().truncate(TimeSpan.SECOND),
  }),
  json: () => ({ type: "json", value: "null" }),
  uuid: () => ({ type: "uuid", value: uuid.create() }),
  bytes: () => ({ type: "bytes", value: "" }),
};

const hiddenDataTypes = (field: icd.JSONField): DataType[] =>
  DataType.ALL.filter((dt) => !encodes(field, dt.toString()));

const ONLY_CHANNEL_TOOLTIP = "A message needs at least one field on a channel.";

interface FieldBindingProps {
  path: deep.Path<icd.WriteBinding[]>;
  field: icd.JSONField;
  channelName: string;
  skipped: boolean;
}

const FieldBinding = ({
  path,
  field,
  channelName,
  skipped,
}: FieldBindingProps): ReactElement => {
  const { set } = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const bindings = PForm.useFieldValue(path);
  const i = bindings.findIndex((b) => b.field === field.key);
  const binding = i >= 0 ? bindings[i] : undefined;
  const bound = binding != null;
  // The Core refuses a message that binds no channels.
  const only =
    binding?.type === "channel" &&
    bindings.filter(({ type }) => type === "channel").length === 1;
  const handleMode = (mode: Mode) => {
    const others = bindings.filter((b) => b.field !== field.key);
    if (mode === "default") {
      set(path, others);
      return;
    }
    const next: icd.WriteBinding =
      mode === "channel"
        ? { type: "channel", field: field.key, channel: 0 }
        : { type: "value", field: field.key, value: INITIAL_VALUES[field.type](field) };
    set(path, bound ? bindings.with(i, next) : [...bindings, next]);
  };
  const modes = (
    <Select.Buttons<Mode>
      value={bound ? binding.type : "default"}
      onChange={handleMode}
      preview={isPreview}
      size="small"
    >
      <Select.Item<Mode> itemKey="channel">Channel</Select.Item>
      <Select.Item<Mode>
        itemKey="value"
        disabled={only}
        tooltip={only ? ONLY_CHANNEL_TOOLTIP : undefined}
      >
        Value
      </Select.Item>
      {field.default != null && (
        <Select.Item<Mode>
          itemKey="default"
          disabled={only}
          tooltip={only ? ONLY_CHANNEL_TOOLTIP : undefined}
        >
          Default
        </Select.Item>
      )}
    </Select.Buttons>
  );
  return (
    <PForm.Section
      title={field.name}
      actions={modes}
      role="group"
      aria-label={field.name}
    >
      {bound && binding.type === "channel" && (
        <>
          <Task.ChannelNameField
            channel={binding.channel}
            defaultName={channelName}
            skipped={skipped}
          />
          <Task.Payload.DataTypeField
            path={deep.variant(path[i], "type", "channel").dataType}
            channel={binding.channel}
            defaultDataType={defaultDataType(field)}
            hideDataTypes={hiddenDataTypes(field)}
          />
        </>
      )}
      {bound && binding.type === "value" && (
        <ValueField
          type={field.type}
          path={deep.variant(path[i], "type", "value").value.value}
        />
      )}
      {!bound && (
        <Text.Text color={9}>Sends its default, {formatDefault(field)}.</Text.Text>
      )}
    </PForm.Section>
  );
};

export interface BindingsProps {
  path: deep.Path<icd.WriteBinding[]>;
  /** The fields of the message. */
  fields: icd.JSONField[];
  /** @returns the name of the command channel that deploying creates for the field. */
  channelName: (field: icd.JSONField) => string;
  /** Deploying creates no channels, because the target is disabled. */
  skipped?: boolean;
}

/** Fills each field of a message from a channel, a fixed value, or its default. */
export const Bindings = ({
  path,
  fields,
  channelName,
  skipped = false,
}: BindingsProps): ReactElement => (
  <>
    {fields.map((field) => (
      <FieldBinding
        key={field.key}
        path={path}
        field={field}
        channelName={channelName(field)}
        skipped={skipped}
      />
    ))}
  </>
);
