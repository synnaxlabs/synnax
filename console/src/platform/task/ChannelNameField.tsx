// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Access, Channel } from "@synnaxlabs/pluto";
import { type deep, primitive } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode } from "react";

import {
  isMissing,
  Message,
  MISSING_DESCRIPTIONS,
  NEW_DESCRIPTION,
  STATE_MESSAGES,
  StateIcon,
} from "@/platform/task/ChannelState";
import { useIsPreview } from "@/platform/task/useIsPreview";

export interface ChannelNameFieldProps {
  /** The channel the name belongs to, or 0 before deploying creates it. */
  channel: channel.Key;
  /** Holds the name deploying the task gives the channel it creates. */
  namePath: deep.Target<string | undefined>;
  /** The name deploying falls back to; shown as the placeholder. */
  defaultName?: string;
  /** Deploying the task does not create the channel, because it is disabled. */
  skipped?: boolean;
}

/**
 * A labeled input that names a channel. Until deploying creates the channel, or while
 * the channel is missing, the name lives in the form at namePath. After, an edit
 * renames the channel itself.
 */
export const ChannelNameField = ({
  channel: key,
  defaultName = "",
  skipped = false,
  ...rest
}: ChannelNameFieldProps): ReactElement =>
  key === 0 ? (
    <Unbound {...rest} defaultName={defaultName} skipped={skipped} />
  ) : (
    <Bound {...rest} channel={key} defaultName={defaultName} skipped={skipped} />
  );

interface StoredProps extends Required<Omit<ChannelNameFieldProps, "channel">> {
  startContent?: ReactNode;
}

const Stored = ({ namePath, defaultName, startContent }: StoredProps) => (
  <Form.TextField
    path={namePath}
    label="Channel"
    padHelpText={false}
    inputProps={{
      placeholder: defaultName === "" ? "Channel name" : defaultName,
      startContent,
    }}
  />
);

const Unbound = (props: StoredProps) => {
  const name = Form.useFieldValue(props.namePath, { optional: true });
  const named = props.defaultName !== "" || primitive.isNonZero(name);
  if (props.skipped || !named) return <Stored {...props} />;
  return (
    <Stored
      {...props}
      startContent={
        <Message
          variant="info"
          message={STATE_MESSAGES.new}
          description={NEW_DESCRIPTION}
        >
          <StateIcon state="new" />
        </Message>
      }
    />
  );
};

interface BoundProps extends StoredProps {
  channel: channel.Key;
}

const Bound = ({ channel: key, ...rest }: BoundProps) => {
  const result = Channel.useResultName({ key });
  const { update } = Channel.useRename();
  const canRename = Access.useUpdateGranted(channel.TYPE_RESOURCE_ID);
  const isPreview = useIsPreview();
  if (isMissing(result))
    return (
      <Stored
        {...rest}
        startContent={
          <Message
            variant="warning"
            message={STATE_MESSAGES.missing}
            description={MISSING_DESCRIPTIONS[rest.skipped ? "skips" : "creates"]}
          >
            <StateIcon state="missing" />
          </Message>
        }
      />
    );
  const name = result.data ?? "";
  const handleChange = (next: string) => {
    if (next.length > 0 && next !== name) update({ key, name: next });
  };
  return (
    <Input.Item label="Channel" padHelpText={false}>
      <Input.Text
        value={name}
        onChange={handleChange}
        onlyChangeOnBlur
        disabled={isPreview || !canRename}
      />
    </Input.Item>
  );
};
