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
import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { ICD as PICD, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import type { FC, ReactElement } from "react";

import { converts, unbind } from "@/platform/icd/dataType";
import { useFormats } from "@/platform/icd/Formats";
import {
  isReadable,
  jsonFields,
  type ReadSource,
  suit,
  type TreeField,
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
  bindings: deep.Target<icd.ReadBinding[]>;
  field: TreeField;
  /** Starts or stops reading the field. */
  onChange: (on: boolean) => void;
  disabled?: boolean;
}

/** Reads the field into a channel, or stops reading it. */
export const ReadCheckbox: FC<ReadCheckboxProps> = ({
  bindings,
  field,
  onChange,
  disabled = false,
}) => {
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
      disabled={isPreview || !readable || disabled}
      aria-label={`Read ${field.name}`}
      tooltip={tooltip}
    />
  );
};

export interface SuitReadProps {
  path: deep.Path<ReadSource>;
  /** The index of the message in the source's reads. */
  read: number;
}

/**
 * Keeps the read of a message suited to its format, whose edits can leave a read that
 * the Core refuses to save. Unbinds each channel that its field can no longer convert
 * to. It corrects only on a format change, so a refused correction waits for the next
 * one.
 */
export const SuitRead: FC<SuitReadProps> = ({ path, read }) => {
  const client = PSynnax.use();
  const { get, set } = PForm.useContext();
  const isPreview = Task.useIsPreview();
  const handleError = Status.useAsyncErrorHandler();
  const formats = useFormats();
  const icdKey = PForm.useFieldValue(path.icd);
  const readPath = path.messages[read];
  const key = PForm.useFieldValue(readPath.message);
  const message = PForm.useFieldValue(PICD.formPath.messages[key], {
    optional: true,
    ctx: formats.get(icdKey),
  });
  useAsyncEffect(
    async (signal) => {
      if (isPreview || client == null || message?.payload.format !== "json") return;
      const fields = jsonFields(message);
      await handleError(async () => {
        const bound = get(readPath).value.bindings.map((b) => b.channel);
        const channels = await client.channels.retrieve(bound);
        if (signal.aborted) return;
        const current = get(readPath).value;
        const suited = suit(current, fields);
        const bindings = unbind(suited.bindings, fields, channels, converts);
        const next = { ...suited, bindings };
        if (!deep.equal(next, current)) set(readPath, next);
      }, "Failed to retrieve the channels of a message");
    },
    [isPreview, client, read, message],
  );
  return null;
};
