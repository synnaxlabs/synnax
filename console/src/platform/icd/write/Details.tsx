// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { channel, icd } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { ICD as PICD } from "@synnaxlabs/pluto";
import {
  bounds,
  DataType,
  deep,
  type payload,
  primitive,
  TimeSpan,
  TimeStamp,
  uuid,
} from "@synnaxlabs/x";
import type { FC, ReactElement, ReactNode } from "react";

import { CSS } from "@/platform/css";
import { defaultDataType, encodes, valueKind } from "@/platform/icd/dataType";
import {
  BinaryField,
  JSONField,
  renamedPointer,
  TextField,
} from "@/platform/icd/Fields";
import { formatOf, heldMessage, useFormats } from "@/platform/icd/Formats";
import { OpenInEditor } from "@/platform/icd/OpenInEditor";
import { formatDefault } from "@/platform/icd/Options";
import { rawBounds } from "@/platform/icd/range";
import { type Field, fieldsPath, jsonFields } from "@/platform/icd/read";
import { ValueField } from "@/platform/icd/ValueField";
import { Task } from "@/platform/task";

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
  [K in payload.Kind]: (field: Field) => Extract<icd.WriteValue, { type: K }>;
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

const hiddenDataTypes = (field: Field): DataType[] =>
  DataType.ALL.filter((dt) => !encodes(field, dt.toString()));

interface FieldSectionProps {
  payload: deep.Path<icd.Message["payload"]>;
  field: icd.FieldKey;
  commanded: boolean;
  autoFocus: boolean;
  onNameChange?: (name: string) => void;
  actions: ReactNode;
}

const FIELD_SECTIONS: Record<
  payload.Format,
  (props: FieldSectionProps) => ReactElement
> = {
  json: ({ payload, field, ...rest }) => (
    <JSONField path={deep.variant(payload, "format", "json").fields[field]} {...rest} />
  ),
  text: ({ payload, field, commanded, actions }) => (
    <TextField
      path={deep.variant(payload, "format", "text").fields[field]}
      commanded={commanded}
      actions={actions}
    />
  ),
  binary: ({ payload, field, commanded, actions }) => (
    <BinaryField
      path={deep.variant(payload, "format", "binary").fields[field]}
      commanded={commanded}
      actions={actions}
    />
  ),
};

const ONLY_CHANNEL_HELP = "A message needs at least one field on a channel.";

export interface FieldDetailsProps {
  /** The bindings of the message that the target sends. */
  path: deep.Path<icd.WriteBinding[]>;
  /** The ICD of the message, which an enclosing WithFormats opens. */
  icdKey: icd.Key;
  message: icd.MessageKey;
  field: icd.FieldKey;
  /** The name of the command channel that deploying creates for the field. */
  channelName: string;
  /** The stored channel that deploying keeps when the field has none, or 0. */
  stored?: channel.Key;
  /** Deploying creates no channel, because the target is disabled. */
  skipped?: boolean;
  /** The name of the field takes the focus on mount. */
  autoFocus?: boolean;
  /**
   * The task form created the field, so a rename moves its pointer while the pointer
   * is the pointer of its name.
   */
  created?: boolean;
}

/**
 * The format of a field, which each edit saves to the ICD at once, then what fills the
 * field: a command channel, a fixed value, or its default.
 */
export const FieldDetails: FC<FieldDetailsProps> = ({
  path,
  icdKey,
  message,
  field: key,
  channelName,
  stored = 0,
  skipped = false,
  autoFocus = false,
  created = false,
}) => {
  const { set } = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const formats = useFormats();
  const format = formatOf(formats, icdKey);
  const payloadPath = PICD.formPath.messages[message].payload;
  const field = PForm.useFieldValue(fieldsPath(message)[key], { ctx: format });
  const json = "pointer" in field ? field : null;
  const bindings = PForm.useFieldValue(path);
  const i = bindings.findIndex((b) => b.field === key);
  const binding = i >= 0 ? bindings[i] : undefined;
  const bound = binding != null;
  const shown =
    binding?.type === "channel" && primitive.isNonZero(binding.channel)
      ? binding.channel
      : stored;
  // The Core refuses a message that binds no channels.
  const only =
    binding?.type === "channel" &&
    bindings.filter(({ type }) => type === "channel").length === 1;
  const handleMode = (mode: Mode) => {
    const others = bindings.filter((b) => b.field !== key);
    if (mode === "default") {
      set(path, others);
      return;
    }
    const next: icd.WriteBinding =
      mode === "channel"
        ? { type: "channel", field: key, channel: 0 }
        : { type: "value", field: key, value: INITIAL_VALUES[valueKind(field)](field) };
    set(path, bound ? bindings.with(i, next) : [...bindings, next]);
  };
  const rename = (renamed: icd.JSONField, name: string) => {
    const pointer = renamedPointer(
      renamed,
      name,
      jsonFields(heldMessage(formats, icdKey, message)),
    );
    if (pointer != null)
      format.set(
        deep.variant(payloadPath, "format", "json").fields[key].pointer,
        pointer,
      );
  };
  const messageFormat = PForm.useFieldValue(payloadPath.format, { ctx: format });
  const modes = (
    <Select.Buttons<Mode>
      value={bound ? binding.type : "default"}
      onChange={handleMode}
      preview={isPreview}
      size="small"
    >
      <Select.Item<Mode> itemKey="channel">Channel</Select.Item>
      <Select.Item<Mode> itemKey="value" disabled={only}>
        Value
      </Select.Item>
      {field.default != null && (
        <Select.Item<Mode> itemKey="default" disabled={only}>
          Default
        </Select.Item>
      )}
    </Select.Buttons>
  );
  return (
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      <PForm.Sections>
        <PForm.Form<typeof PICD.formSchema> {...format}>
          {FIELD_SECTIONS[messageFormat]({
            payload: payloadPath,
            field: key,
            commanded: binding?.type === "channel",
            autoFocus,
            onNameChange:
              created && json != null ? (name) => rename(json, name) : undefined,
            actions: (
              <OpenInEditor icdKey={icdKey} message={message} field={field.key} />
            ),
          })}
        </PForm.Form>
        <PForm.Section title="Source" actions={modes} role="group" aria-label="Source">
          {only && (
            <Text.Text level="small" color={9}>
              {ONLY_CHANNEL_HELP}
            </Text.Text>
          )}
          {bound && binding.type === "channel" && (
            <>
              <Task.ChannelNameField
                channel={shown}
                defaultName={channelName}
                skipped={skipped}
              />
              <Task.Payload.DataTypeField
                path={deep.variant(path[i], "type", "channel").dataType}
                channel={shown}
                defaultDataType={defaultDataType(field)}
                hideDataTypes={hiddenDataTypes(field)}
              />
            </>
          )}
          {bound && binding.type === "value" && (
            <ValueField
              type={valueKind(field)}
              path={deep.variant(path[i], "type", "value").value.value}
            />
          )}
          {!bound && (
            <Text.Text color={9}>
              Sends its default,{" "}
              {"pointer" in field ? formatDefault(field) : String(field.default)}.
            </Text.Text>
          )}
        </PForm.Section>
      </PForm.Sections>
    </Flex.Box>
  );
};
