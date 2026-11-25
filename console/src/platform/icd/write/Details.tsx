// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import type { ICD as PICD } from "@synnaxlabs/pluto";
import { bounds, DataType, deep, TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import type { FC } from "react";

import { CSS } from "@/platform/css";
import { defaultDataType, encodes } from "@/platform/icd/dataType";
import { JSONField } from "@/platform/icd/Fields";
import { formatOf, useFormats } from "@/platform/icd/Formats";
import { OpenInEditor } from "@/platform/icd/OpenInEditor";
import { formatDefault } from "@/platform/icd/Options";
import { rawBounds } from "@/platform/icd/range";
import { fieldsPath } from "@/platform/icd/read";
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

export interface FieldDetailsProps {
  /** The bindings of the message that the target sends. */
  path: deep.Path<icd.WriteBinding[]>;
  /** The ICD of the message, which an enclosing WithFormats opens. */
  icdKey: icd.Key;
  message: icd.MessageKey;
  field: icd.FieldKey;
  /** The name of the command channel that deploying creates for the field. */
  channelName: string;
  /** Deploying creates no channel, because the target is disabled. */
  skipped?: boolean;
  /** The name of the field takes the focus on mount. */
  autoFocus?: boolean;
  /** A rename of the field moves its pointer to the pointer of the new name. */
  pointerLinked?: boolean;
  /** Called when the user edits the pointer of the field. */
  onPointerChange?: () => void;
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
  skipped = false,
  autoFocus = false,
  pointerLinked = false,
  onPointerChange,
}) => {
  const { set } = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const format = formatOf(useFormats(), icdKey);
  const fieldPath = fieldsPath(message)[key];
  const field = PForm.useFieldValue(fieldPath, { ctx: format });
  const bindings = PForm.useFieldValue(path);
  const i = bindings.findIndex((b) => b.field === key);
  const binding = i >= 0 ? bindings[i] : undefined;
  const bound = binding != null;
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
        : { type: "value", field: key, value: INITIAL_VALUES[field.type](field) };
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
    <Flex.Box y grow empty className={CSS.BE("details", "form")}>
      <PForm.Sections>
        <PForm.Form<typeof PICD.formSchema> {...format}>
          <JSONField
            path={fieldPath}
            autoFocus={autoFocus}
            pointerLinked={pointerLinked}
            onPointerChange={onPointerChange}
            actions={<OpenInEditor icdKey={icdKey} message={message} />}
          />
        </PForm.Form>
        <PForm.Section title="Source" actions={modes} role="group" aria-label="Source">
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
      </PForm.Sections>
    </Flex.Box>
  );
};
