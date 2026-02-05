// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Flex, Icon, Telem, Text } from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { CSS } from "@/css";

export interface ExpandableStatusProps extends Omit<Flex.BoxProps, "children"> {
  /** The status to display */
  status: status.Status;
  /** Whether the status is expanded */
  expanded: boolean;
  /** Callback when expand/collapse is toggled */
  onToggle: () => void;
  /** Number of lines to show when collapsed. Defaults to 1. */
  collapsedLines?: number;
  /** Fallback message when status.message is empty */
  fallbackMessage?: string;
}

export const Status = ({
  status: stat,
  expanded,
  onToggle,
  fallbackMessage,
  className,
  ...props
}: ExpandableStatusProps): ReactElement => {
  const hasDescription = Boolean(stat.description);
  const statusIcon = stat.variant === "loading" ? <Icon.Loading /> : <Icon.Circle />;
  const message = stat.message || fallbackMessage;
  const ExpansionIcon = expanded ? Icon.Collapse : Icon.Expand;

  const getCopyText = useCallback(() => status.toString(stat), [stat]);

  return (
    <Button.Button
      className={CSS(CSS.B("task-status"), className)}
      el="div"
      variant="text"
      direction={expanded ? "y" : "x"}
      onClick={onToggle}
      {...props}
    >
      <Flex.Box
        x
        align="center"
        gap="small"
        justify="between"
        grow
        className={CSS.BE("task-status", "message")}
      >
        <Flex.Box x align="center" gap="small">
          <Text.Text level="p" status={stat.variant}>
            {statusIcon}
          </Text.Text>
          <Text.Text level="p" status={stat.variant} weight={500}>
            {message}
          </Text.Text>
        </Flex.Box>
        {expanded && (
          <Flex.Box x>
            <Telem.Text.TimeStamp
              level="small"
              color={8}
              format="time"
              displayTZ="local"
            >
              {stat.time}
            </Telem.Text.TimeStamp>
            <Button.Copy
              className={CSS.BE("task-status", "copy-button")}
              text={getCopyText}
              variant="outlined"
              size="small"
              textColor={10}
            >
              Copy diagnostics
            </Button.Copy>
          </Flex.Box>
        )}
      </Flex.Box>
      <ExpansionIcon className={CSS.BE("task-status", "expand-indicator")} />
      {expanded && hasDescription && (
        <Text.Text
          level="p"
          color={9}
          variant="code"
          className={CSS.BE("task-status", "description")}
        >
          {stat.description}
        </Text.Text>
      )}
    </Button.Button>
  );
};
