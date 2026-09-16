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
import {
  Access,
  Channel,
  Errors,
  Flex,
  Flux,
  Form,
  Text,
  Tooltip,
  useSyncedRef,
} from "@synnaxlabs/pluto";
import { location, type optional, primitive } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { CSS } from "@/platform/css";
import { useIsPreview } from "@/platform/task/Form";
import { Session } from "@/session";

interface BoundProps extends optional.Optional<
  Omit<Text.MaybeEditableProps, "value">,
  "level"
> {
  channel: channel.Key;
  defaultName?: string;
  namePath: string;
}

export interface ChannelNameProps<D> extends BoundProps {
  /** The device whose saved map binds the row, or undefined while it loads. */
  device: D | undefined;
  /** Returns the channel the device map binds the row to, or 0 when it has none. */
  resolve: (device: D) => channel.Key;
}

interface NameProps extends Omit<BoundProps, "defaultName"> {
  name: string;
}

const Name = ({ channel, namePath, name, className, ...rest }: NameProps) => {
  // Use optional values here because for some reason, the namePath is not always
  // populated. Eventually we should strongly type the Form so we don't need to worry
  // about this.
  const onChange = Form.useField<string>(namePath, { optional: true })?.onChange;
  const canRename = Access.useUpdateGranted(clientChannel.TYPE_ONTOLOGY_ID);
  const isPreview = useIsPreview();
  const { update } = Channel.useRename();
  const handleRename = useCallback(
    (name: string) => {
      if (channel === 0) return onChange?.(name);
      update({ key: channel, name });
    },
    [channel, update, onChange],
  );
  return (
    <Text.MaybeEditable
      className={CSS.cls(className, CSS.BE("task", "channel-name"))}
      level="small"
      value={name}
      onChange={handleRename}
      disabled={isPreview || (channel !== 0 && !canRename)}
      allowDoubleClick={false}
      overflow="ellipsis"
      {...rest}
    />
  );
};

/** The name the form carries for a channel whose record isn't in hand. */
const Unresolved = ({
  defaultName = "No channel",
  namePath,
  ...rest
}: BoundProps) => {
  const formName = Form.useFieldValue<string>(namePath, { optional: true });
  const name = primitive.isNonZero(formName) ? formName : defaultName;
  return <Name namePath={namePath} name={name} {...rest} />;
};

const Resolved = ({ channel, defaultName, namePath, ...rest }: BoundProps) => {
  const range = Session.Range.useSelectSelectedKey();
  const query = { key: channel, rangeKey: range ?? undefined };
  Channel.useEnsure(query);
  const name = Channel.useAlias(query);
  return <Name channel={channel} namePath={namePath} name={name} {...rest} />;
};

interface MessageProps {
  variant: status.Variant;
  message: string;
  description?: string;
  children: Tooltip.DialogProps["children"][1];
}

const Message = ({ variant, message, description, children }: MessageProps) => (
  <Tooltip.Dialog location={location.CENTER_RIGHT}>
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

/**
 * Shows a row's channel name. The device's saved map decides which channel is shown
 * once the device resolves, since configure binds from it. The row's key stands in.
 */
export const ChannelName = <D,>({
  device,
  resolve,
  channel,
  ...rest
}: ChannelNameProps<D>): ReactElement => {
  const props: BoundProps = {
    ...rest,
    channel: device == null ? channel : resolve(device),
  };
  // Through a ref so the fallback keeps its identity: a new component every render
  // would remount the name field and drop an edit in progress.
  const propsRef = useSyncedRef(props);
  const renderFallback = useCallback(
    ({ error }: Errors.FallbackProps) => (
      <Message variant="error" {...describe(error)}>
        <Unresolved {...propsRef.current} status="error" />
      </Message>
    ),
    [propsRef],
  );
  if (props.channel === 0)
    return (
      <Message variant="warning" message="No channel selected">
        <Unresolved {...props} status="warning" />
      </Message>
    );
  return (
    <Errors.SuspenseBoundary
      loading={<Unresolved {...props} status="loading" />}
      FallbackComponent={renderFallback}
    >
      <Resolved {...props} />
    </Errors.SuspenseBoundary>
  );
};
