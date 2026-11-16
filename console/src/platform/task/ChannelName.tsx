// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/task/ChannelName.css";

import { type channel, channel as clientChannel } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Channel } from "@synnaxlabs/pluto";
import { type deep, type optional, primitive } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import {
  type Anchor,
  type Deploy,
  isMissing,
  Message,
  MISSING_DESCRIPTIONS,
  NEW_DESCRIPTION,
  SKIPPED_DESCRIPTION,
  STATE_MESSAGES,
  StateIcon,
} from "@/platform/task/ChannelState";
import { useIsPreview } from "@/platform/task/useIsPreview";

export interface ChannelNameProps extends optional.Optional<
  Omit<Text.MaybeEditableProps, "value" | "onChange">,
  "level"
> {
  channel: channel.Key;
  /** The name deploying the task gives the channel; shown while the form has none. */
  defaultName?: string;
  /**
   * Ends the name in the form when shown, as deploying the task ends it. A rename
   * stores the name without it.
   */
  suffix?: string;
  /**
   * Holds the name deploying the task gives the channel, shown until the channel
   * resolves. Required while the channel is 0, unless the channel is picked.
   */
  namePath?: deep.Target<string | undefined>;
  /** Leads the name. A new or missing channel shows its state icon in its place. */
  icon?: ReactElement;
  /** Deploying the task does not create the channel, because it is disabled. */
  skipped?: boolean;
  /**
   * The form picks an existing channel, so deploying the task never creates one, and
   * the name is read only.
   */
  picked?: boolean;
}

const NO_CHANNEL = "No channel";

/** @returns the name, ended in the suffix unless it already ends in it. */
export const withSuffix = (name: string, suffix: string): string =>
  name.endsWith(suffix) ? name : `${name}${suffix}`;

interface NameProps extends Omit<
  ChannelNameProps,
  "channel" | "defaultName" | "suffix" | "namePath" | "icon" | "skipped" | "picked"
> {
  name: string;
  /** Takes a rename. Without it, the name is read only. */
  onRename?: (name: string) => void;
}

const Name = ({ name, onRename, className, ...rest }: NameProps) => {
  const isPreview = useIsPreview();
  return (
    <Text.MaybeEditable
      className={CSS.cls(className, CSS.BE("task", "channel-name"))}
      level="small"
      value={name}
      onChange={onRename}
      disabled={isPreview}
      allowDoubleClick={false}
      overflow="fade"
      {...rest}
    />
  );
};

interface StoredProps extends Omit<NameProps, "name" | "onRename"> {
  namePath: deep.Target<string | undefined>;
  defaultName?: string;
  suffix?: string;
}

/** The name the form carries for a channel whose record isn't in hand. */
const Stored = ({
  namePath,
  defaultName = NO_CHANNEL,
  suffix = "",
  ...rest
}: StoredProps) => {
  // Use optional values here because for some reason, the namePath is not always
  // populated.
  const field = Form.useField(namePath, { optional: true });
  const name = primitive.isNonZero(field?.value)
    ? withSuffix(field.value, suffix)
    : defaultName;
  const handleRename =
    field == null
      ? undefined
      : (renamed: string) =>
          field.onChange(
            suffix !== "" && renamed.endsWith(suffix) && renamed !== suffix
              ? renamed.slice(0, -suffix.length)
              : renamed,
          );
  return <Name {...rest} name={name} onRename={handleRename} />;
};

interface UnresolvedProps extends Omit<StoredProps, "namePath" | "defaultName"> {
  namePath?: deep.Target<string | undefined>;
  defaultName: string;
}

const Unresolved = ({ namePath, defaultName, suffix, ...rest }: UnresolvedProps) =>
  namePath == null ? (
    <Name {...rest} name={defaultName} />
  ) : (
    <Stored {...rest} namePath={namePath} defaultName={defaultName} suffix={suffix} />
  );

// A plain function, not a component, so a wrapping tooltip's handlers reach the DOM.
const lead = (icon: ReactElement | undefined, name: Anchor): Anchor =>
  icon == null ? (
    name
  ) : (
    <Flex.Box x align="center" gap="small" className={CSS.BE("task", "channel-lead")}>
      {icon}
      {name}
    </Flex.Box>
  );

interface ResolvedProps extends Omit<ChannelNameProps, "skipped" | "picked"> {
  deploy: Deploy;
}

const Resolved = ({
  channel,
  namePath,
  defaultName = NO_CHANNEL,
  suffix,
  icon,
  deploy,
  ...rest
}: ResolvedProps) => {
  const canRename = Access.useUpdateGranted(clientChannel.TYPE_RESOURCE_ID);
  const { update } = Channel.useRename();
  const result = Channel.useResultName({ key: channel });
  if (result.variant === "success") {
    const handleRename =
      deploy === "never" || !canRename
        ? undefined
        : (name: string) => update({ key: channel, name });
    return lead(icon, <Name {...rest} name={result.data} onRename={handleRename} />);
  }
  const unresolved = { ...rest, namePath, defaultName, suffix };
  if (isMissing(result))
    return (
      <Message
        variant="warning"
        message={STATE_MESSAGES.missing}
        description={MISSING_DESCRIPTIONS[deploy]}
      >
        {lead(
          <StateIcon state="missing" />,
          <Unresolved
            {...unresolved}
            defaultName={deploy === "never" ? STATE_MESSAGES.missing : defaultName}
            color={8}
          />,
        )}
      </Message>
    );
  if (result.variant !== "error")
    return lead(icon, <Unresolved {...unresolved} status="loading" />);
  return (
    <Message
      variant="error"
      message="Failed to retrieve channel"
      description={result.status.details.error.message}
    >
      {lead(icon, <Unresolved {...unresolved} status="error" />)}
    </Message>
  );
};

interface UnboundProps extends StoredProps {
  icon?: ReactElement;
  skipped: boolean;
}

const Unbound = ({ namePath, defaultName, icon, skipped, ...rest }: UnboundProps) => {
  const stored = Form.useFieldValue(namePath, { optional: true });
  if (defaultName == null && !primitive.isNonZero(stored))
    return (
      <Message
        variant="info"
        message={NO_CHANNEL}
        description={
          skipped ? SKIPPED_DESCRIPTION : "Deploying the task creates and names it."
        }
      >
        {lead(icon, <Stored {...rest} namePath={namePath} color={8} />)}
      </Message>
    );
  const name = <Stored {...rest} namePath={namePath} defaultName={defaultName} />;
  if (skipped)
    return (
      <Message
        variant="info"
        message="Skipped channel"
        description={SKIPPED_DESCRIPTION}
      >
        {lead(icon, name)}
      </Message>
    );
  return (
    <Message variant="info" message={STATE_MESSAGES.new} description={NEW_DESCRIPTION}>
      {lead(<StateIcon state="new" />, name)}
    </Message>
  );
};

const deployOf = (skipped: boolean, picked: boolean): Deploy => {
  if (picked) return "never";
  return skipped ? "skips" : "creates";
};

export const ChannelName = ({
  channel,
  namePath,
  defaultName,
  suffix,
  icon,
  skipped = false,
  picked = false,
  ...rest
}: ChannelNameProps): ReactElement => {
  const named = { ...rest, namePath, defaultName, suffix, icon };
  if (channel !== 0)
    return <Resolved {...named} channel={channel} deploy={deployOf(skipped, picked)} />;
  if (picked) return lead(icon, <Name {...rest} name={NO_CHANNEL} color={8} />);
  if (namePath == null) throw new Error("ChannelName needs a namePath for channel 0");
  return <Unbound {...named} namePath={namePath} skipped={skipped} />;
};
