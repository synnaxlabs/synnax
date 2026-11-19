// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type icd } from "@synnaxlabs/client";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Channel, ICD as PICD } from "@synnaxlabs/pluto";
import { deep, type payload } from "@synnaxlabs/x";
import { type FC, type ReactElement, useEffect } from "react";

import { converts } from "@/platform/icd/dataType";
import { useFormats } from "@/platform/icd/Formats";
import {
  isReadable,
  jsonFields,
  type ReadBinding,
  type ReadSource,
  suit,
} from "@/platform/icd/read";
import { Task } from "@/platform/task";

export interface MessageLabelProps {
  path: deep.Path<ReadSource>;
  /** The index of the message in the source's reads. */
  read: number;
}

/** Names a message that a source reads by its name in the ICD. */
export const MessageLabel = ({ path, read }: MessageLabelProps): ReactElement => {
  const formats = useFormats();
  const icdKey = PForm.useFieldValue(path.icd);
  const message = PForm.useFieldValue(path.messages[read].message);
  const name = PForm.useFieldValue(PICD.formPath.messages[message].name, {
    optional: true,
    ctx: formats.get(icdKey),
  });
  return (
    <Text.Text level="small" weight={500} color={name == null ? 8 : 10} overflow="fade">
      <Icon.JSON />
      {name ?? "Message not found"}
    </Text.Text>
  );
};

const UNREADABLE_TOOLTIP =
  "A bytes field needs a JSON pointer. Give it one in the message format.";

export interface ReadCheckboxProps {
  bindings: deep.Target<ReadBinding[]>;
  field: icd.JSONField;
  /** Starts or stops reading the field. */
  onChange: (on: boolean) => void;
}

/** Reads the field into a channel, or stops reading it. */
export const ReadCheckbox: FC<ReadCheckboxProps> = ({ bindings, field, onChange }) => {
  const isPreview = Task.useIsPreview();
  const bound = PForm.useFieldValue(bindings).some(
    ({ field: key }) => key === field.key,
  );
  const readable = isReadable(field);
  let tooltip: string | undefined = bound ? "Disable" : "Enable";
  if (isPreview) tooltip = undefined;
  else if (!readable) tooltip = UNREADABLE_TOOLTIP;
  return (
    <Input.Checkbox
      value={bound}
      onChange={onChange}
      checkedIcon={<Status.Indicator variant="success" />}
      uncheckedIcon={<Status.Indicator variant="disabled" />}
      size="small"
      disabled={isPreview || !readable}
      aria-label={`Read ${field.name}`}
      tooltip={tooltip}
    />
  );
};

interface UnbindProps {
  path: deep.Target<channel.Key>;
  kind: payload.Kind;
  channel: channel.Key;
}

/**
 * Unbinds a channel that the field's kind can no longer convert to, so deploying
 * creates a new one.
 */
const Unbind = ({ path, kind, channel: key }: UnbindProps) => {
  const { set } = PForm.useContext();
  const dataType = Channel.useResultDataType({ key }).data?.toString();
  useEffect(() => {
    if (dataType != null && !converts(kind, dataType)) set(path, 0);
  }, [dataType, kind]);
  return null;
};

export interface SuitReadProps {
  path: deep.Path<ReadSource>;
  /** The index of the message in the source's reads. */
  read: number;
}

/**
 * Keeps the read of a message suited to its format, whose edits can leave a read that
 * the Core refuses to save. Unbinds each channel that its field can no longer convert
 * to.
 */
export const SuitRead: FC<SuitReadProps> = ({ path, read }) => {
  const { set } = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const formats = useFormats();
  const icdKey = PForm.useFieldValue(path.icd);
  const readPath = path.messages[read];
  const readMessage = PForm.useFieldValue(readPath);
  const message = PForm.useFieldValue(PICD.formPath.messages[readMessage.message], {
    optional: true,
    ctx: formats.get(icdKey),
  });
  const fields = jsonFields(message);
  useEffect(() => {
    if (isPreview || message?.payload.format !== "json") return;
    const suited = suit(readMessage, fields);
    if (!deep.equal(suited, readMessage)) set(readPath, suited);
  }, [isPreview, readMessage, message]);
  if (isPreview) return null;
  return readMessage.bindings.map((b, i) => {
    const field = fields.find(({ key }) => key === b.field);
    if (field == null || b.channel === 0) return null;
    return (
      <Unbind
        key={b.field}
        path={readPath.bindings[i].channel}
        kind={field.type}
        channel={b.channel}
      />
    );
  });
};
