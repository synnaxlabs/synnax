// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/task/ChannelState.css";

import { NotFoundError, type status } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Flux } from "@synnaxlabs/pluto";
import { location, type state } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

/** A task channel that does not exist yet, or no longer does. */
type State = "new" | "missing";

/** What deploying the task does with a channel that does not exist. */
export type Deploy = "creates" | "skips" | "never";

const ICONS: Record<State, Icon.FC> = { new: Icon.New, missing: Icon.None };

export const STATE_MESSAGES: Record<State, string> = {
  new: "New channel",
  missing: "Channel not found",
};

export const NEW_DESCRIPTION = "Deploying the task creates it.";

export const SKIPPED_DESCRIPTION =
  "Deploying the task does not create a disabled channel.";

const DELETED = "It may have been deleted.";

export const MISSING_DESCRIPTIONS: Record<Deploy, string> = {
  creates: `${DELETED} Deploying the task creates a new channel with this name.`,
  skips: `${DELETED} ${SKIPPED_DESCRIPTION}`,
  never: `${DELETED} Select another channel.`,
};

interface StateIconProps extends Icon.IconProps {
  state: State;
}

export const StateIcon = ({
  state,
  className,
  ...rest
}: StateIconProps): ReactElement => {
  const C = ICONS[state];
  return (
    <C
      {...rest}
      role="img"
      aria-label={STATE_MESSAGES[state]}
      className={CSS.cls(
        className,
        CSS.BE("task", "channel-icon"),
        CSS.BEM("task", "channel-icon", state),
      )}
    />
  );
};

/** @returns whether a channel query failed because the channel is gone. */
export const isMissing = <Data extends state.State>(
  result: Flux.Result<Data>,
): boolean => {
  if (result.variant !== "error") return false;
  const { error } = result.status.details;
  return Flux.DeletedError.matches(error) || NotFoundError.matches(error.cause);
};

export type Anchor = Tooltip.DialogProps["children"][1];

interface MessageProps {
  variant: status.Variant;
  message: string;
  description: string;
  children: Anchor;
}

const LOCATION = {
  targetCorner: location.BOTTOM_LEFT,
  dialogCorner: location.TOP_LEFT,
};

/** A tooltip below its anchor, at the anchor's left edge. */
export const Message = ({
  variant,
  message,
  description,
  children,
}: MessageProps): ReactElement => (
  <Tooltip.Dialog location={LOCATION}>
    <Flex.Box y gap="small">
      <Text.Text status={variant} level="p" color={10} weight={500}>
        {message}
      </Text.Text>
      <Text.Text level="small" color={9} weight={450}>
        {description}
      </Text.Text>
    </Flex.Box>
    {children}
  </Tooltip.Dialog>
);
