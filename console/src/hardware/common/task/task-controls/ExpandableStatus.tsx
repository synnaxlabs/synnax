// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Flex, Icon, Text, Tooltip } from "@synnaxlabs/pluto";
import { type status, TimeStamp } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useRef, useState } from "react";

import { CSS } from "@/css";

export interface ExpandableStatusProps extends Omit<Flex.BoxProps, "children"> {
  /** The status to display */
  status: status.Status;
  /** Whether the status is expanded */
  expanded: boolean;
  /** Callback when expand/collapse is toggled */
  onToggle: () => void;
  /** Whether the status is being hovered (for preview) */
  hovered?: boolean;
  /** Callback when hover state changes */
  onHoverChange?: (hovered: boolean) => void;
  /** Number of lines to show when collapsed. Defaults to 1. */
  collapsedLines?: number;
  /** Whether to show hover preview. Defaults to true. */
  hoverPreview?: boolean;
  /** Fallback message when status.message is empty */
  fallbackMessage?: string;
}

/** Formats a status into copyable text */
const formatStatusForCopy = (stat: status.Status): string => {
  const parts: string[] = [];
  if (stat.message) parts.push(stat.message);
  if (stat.description) parts.push(stat.description);
  return parts.join("\n\n");
};

/** Checks if text content is truncated */
const isTextTruncated = (element: HTMLElement | null): boolean => {
  if (!element) return false;
  return element.scrollHeight > element.clientHeight;
};

export const ExpandableStatus = ({
  status: stat,
  expanded,
  onToggle,
  hovered = false,
  onHoverChange,
  collapsedLines = 1,
  hoverPreview = true,
  fallbackMessage,
  className,
  ...props
}: ExpandableStatusProps): ReactElement => {
  const [copied, setCopied] = useState(false);
  const messageRef = useRef<HTMLDivElement>(null);
  const hoverTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleCopy = useCallback(async () => {
    const text = formatStatusForCopy(stat);
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }, [stat]);

  const handleClick = useCallback(() => {
    onToggle();
    onHoverChange?.(false);
  }, [onToggle, onHoverChange]);

  const handleMouseEnter = useCallback(() => {
    if (expanded || !hoverPreview || onHoverChange == null) return;
    hoverTimeoutRef.current = setTimeout(() => {
      const shouldPreview =
        isTextTruncated(messageRef.current) || Boolean(stat.description);
      if (shouldPreview) onHoverChange(true);
    }, 300);
  }, [expanded, hoverPreview, stat.description, onHoverChange]);

  const handleMouseLeave = useCallback(() => {
    if (hoverTimeoutRef.current) {
      clearTimeout(hoverTimeoutRef.current);
      hoverTimeoutRef.current = null;
    }
    onHoverChange?.(false);
  }, [onHoverChange]);

  const showExpanded = expanded || hovered;
  const hasDescription = Boolean(stat.description);
  const hasContent = Boolean(stat.message) || hasDescription;
  const icon = stat.variant === "loading" ? <Icon.Loading /> : <Icon.Circle />;
  const message = stat.message || fallbackMessage;

  return (
    <Flex.Box
      className={CSS(CSS.B("task-state"), className)}
      x={!showExpanded}
      y={showExpanded}
      onClick={handleClick}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      {...props}
    >
      {/* Collapsed view */}
      {!showExpanded && (
        <>
          <Flex.Box
            x
            align="center"
            grow
            gap="small"
            className={CSS.BE("task-state", "collapsed")}
          >
            <Text.Text level="p" status={stat.variant}>
              {icon}
            </Text.Text>
            <Text.Text
              ref={messageRef}
              level="p"
              status={stat.variant}
              lineClamp={collapsedLines}
              className={CSS.BE("task-state", "message")}
            >
              {message}
            </Text.Text>
          </Flex.Box>
          <Icon.Expand className={CSS.BE("task-state", "expand-indicator")} />
        </>
      )}

      {/* Expanded view */}
      {showExpanded && (
        <Flex.Box y className={CSS.BE("task-state", "expanded")} gap="small">
          <Flex.Box x justify="between" align="start" gap="small">
            <Flex.Box y gap="tiny" grow style={{ minWidth: 0 }}>
              <Flex.Box x align="center" gap="small">
                <Text.Text level="p" status={stat.variant}>
                  {icon}
                </Text.Text>
                <Text.Text level="p" status={stat.variant} weight={500}>
                  {message}
                </Text.Text>
              </Flex.Box>
              {hasDescription && (
                <Text.Text
                  level="small"
                  color={8}
                  className={CSS.BE("task-state", "description")}
                >
                  {stat.description}
                </Text.Text>
              )}
            </Flex.Box>
            <Flex.Box x align="center" gap="small">
              {hasContent && (
                <Tooltip.Dialog>
                  {copied ? "Copied!" : "Copy to clipboard"}
                  <Button.Button
                    variant="text"
                    size="small"
                    onClick={(e) => {
                      e.stopPropagation();
                      void handleCopy();
                    }}
                  >
                    {copied ? <Icon.Check /> : <Icon.Copy />}
                  </Button.Button>
                </Tooltip.Dialog>
              )}
              <Icon.Collapse
                className={CSS.BE("task-state", "expand-indicator")}
              />
            </Flex.Box>
          </Flex.Box>

          <Flex.Box x justify="between" align="center">
            <Text.Text level="small" color={6}>
              {new TimeStamp(stat.time).toString("time", "local")}
            </Text.Text>
          </Flex.Box>
        </Flex.Box>
      )}
    </Flex.Box>
  );
};
