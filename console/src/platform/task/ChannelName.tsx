// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type channel,
  channel as clientChannel,
  NotFoundError,
  type status,
} from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Text } from "@synnaxlabs/lyra/text";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Access, Channel, Flux } from "@synnaxlabs/pluto";
import { type optional, primitive } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { useIsPreview } from "@/platform/task/Form";
import { Session } from "@/session";

export interface ChannelNameProps extends optional.Optional<
  Omit<Text.MaybeEditableProps, "value">,
  "level"
> {
  channel: channel.Key;
  /** The name configure will give the channel; shown as a preview while it is 0. */
  defaultName?: string;
  /**
   * Holds the name configure gives the channel, shown until the channel resolves.
   * Required while the channel is 0.
   */
  namePath?: string;
}

const NO_CHANNEL = "No channel";

interface NameProps extends Omit<ChannelNameProps, "defaultName" | "namePath"> {
  name: string;
  /** Takes a rename while the channel is 0. */
  onDraftRename?: (name: string) => void;
}

const Name = ({ channel, name, onDraftRename, className, ...rest }: NameProps) => {
  const canRename = Access.useUpdateGranted(clientChannel.TYPE_ONTOLOGY_ID);
  const isPreview = useIsPreview();
  const { update } = Channel.useRename();
  const handleRename = (name: string) => {
    if (channel === 0) return onDraftRename?.(name);
    update({ key: channel, name });
  };
  return (
    <Text.MaybeEditable
      className={CSS.cls(className, CSS.BE("task", "channel-name"))}
      level="small"
      value={name}
      onChange={handleRename}
      disabled={isPreview || (channel !== 0 && !canRename)}
      allowDoubleClick={false}
      overflow="fade"
      {...rest}
    />
  );
};

interface StoredProps extends Omit<ChannelNameProps, "namePath"> {
  namePath: string;
}

/** The name the form carries for a channel whose record isn't in hand. */
const Stored = ({ namePath, defaultName = NO_CHANNEL, ...rest }: StoredProps) => {
  // Use optional values here because for some reason, the namePath is not always
  // populated. Eventually we should strongly type the Form so we don't need to worry
  // about this.
  const field = Form.useField<string>(namePath, { optional: true });
  const name = primitive.isNonZero(field?.value) ? field.value : defaultName;
  return <Name {...rest} name={name} onDraftRename={field?.onChange} />;
};

const Resolved = ({
  namePath,
  defaultName = NO_CHANNEL,
  ...rest
}: ChannelNameProps) => {
  const range = Session.Range.useSelectSelectedKey();
  const result = Channel.useResultAlias({
    key: rest.channel,
    rangeKey: range ?? undefined,
  });
  if (result.variant === "success") return <Name {...rest} name={result.data} />;
  let status: Text.MaybeEditableProps["status"];
  if (result.variant === "error") status = "error";
  else if (result.variant === "loading") status = "loading";
  const unresolved =
    namePath == null ? (
      <Name {...rest} name={defaultName} status={status} />
    ) : (
      <Stored {...rest} namePath={namePath} defaultName={defaultName} status={status} />
    );
  if (result.variant !== "error") return unresolved;
  return (
    <Message variant="error" {...describe(result.status.details.error)}>
      {unresolved}
    </Message>
  );
};

interface MessageProps {
  variant: status.Variant;
  message: string;
  description?: string;
  children: Tooltip.DialogProps["children"][1];
}

const TOOLTIP_LOCATION = { y: "center" } as const;

const Message = ({ variant, message, description, children }: MessageProps) => (
  <Tooltip.Dialog location={TOOLTIP_LOCATION}>
    <Flex.Box y gap="small" className={CSS.B("task-channel-name")}>
      <Text.Text status={variant} level="p" color={10} weight={500}>
        {message}
      </Text.Text>
      {primitive.isNonZero(description) && (
        <Text.Text level="small" color={9} weight={450}>
          {description}
        </Text.Text>
      )}
    </Flex.Box>
    {children}
  </Tooltip.Dialog>
);

const MISSING_DESCRIPTION =
  "It may have been deleted. Configuring the task creates a new channel.";

const describe = (error: Error): Pick<MessageProps, "message" | "description"> => {
  if (Flux.DeletedError.matches(error) || NotFoundError.matches(error.cause))
    return {
      message: "Channel not found",
      description: MISSING_DESCRIPTION,
    };
  return { message: "Failed to retrieve channel", description: error.message };
};

export const ChannelName = ({ namePath, ...rest }: ChannelNameProps): ReactElement => {
  if (rest.channel !== 0) return <Resolved {...rest} namePath={namePath} />;
  if (namePath == null) throw new Error("ChannelName needs a namePath for channel 0");
  if (rest.defaultName != null)
    return (
      <Message variant="info" message="Created when the task is configured">
        <Stored {...rest} namePath={namePath} color={8} />
      </Message>
    );
  return (
    <Message variant="warning" message="No channel selected">
      <Stored {...rest} namePath={namePath} status="warning" />
    </Message>
  );
};
