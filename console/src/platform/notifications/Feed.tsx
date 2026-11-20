// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/notifications/Feed.css";

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { usePopover } from "@synnaxlabs/lyra/hooks";
import { Portal } from "@synnaxlabs/lyra/portal";
import { Status } from "@synnaxlabs/lyra/status";
import { getRootElement } from "@synnaxlabs/lyra/util";
import { type FC, type ReactElement, useEffect, useRef, useState } from "react";

import { CSS } from "@/platform/css";

export interface NotificationProps {
  status: Status.NotificationSpec & { details?: unknown };
  silence: (key: string) => void;
}

export type Matcher = (
  status: Status.NotificationSpec & { details?: unknown },
) => boolean;

export interface Notification extends FC<NotificationProps> {
  match: Matcher;
}

const Default: Notification = ({ status, silence }) => (
  <Status.Notification status={status} silence={silence} />
);
Default.match = () => true;

export const matchPrefix =
  (prefix: string): Matcher =>
  (stat: Status.NotificationSpec) =>
    stat.key.startsWith(prefix);

interface FeedProps {
  notifications: Notification[];
}

const VISIBLE_CAP = 4;

export const Feed = ({ notifications }: FeedProps): ReactElement => {
  const { statuses, silence, silenceAll } = Status.useNotifications();
  const [expanded, setExpanded] = useState(false);
  const rendered = statuses.map((status) => ({
    status,
    Render: notifications.find((n) => n.match(status)) ?? Default,
  }));
  const overflow = rendered.length - VISIBLE_CAP;
  const showAll = expanded && overflow > 0;
  useEffect(() => {
    if (overflow <= 0) setExpanded(false);
  }, [overflow]);
  const visible = showAll ? rendered : rendered.slice(0, VISIBLE_CAP);
  // Portal.Above moves the feed when a modal opens or closes, which hides it.
  const [moves, setMoves] = useState(0);
  useEffect(() => {
    const handleMove = (): void => setMoves((m) => m + 1);
    document.addEventListener(Portal.MODAL_CHANGE_EVENT, handleMove);
    return () => document.removeEventListener(Portal.MODAL_CHANGE_EVENT, handleMove);
  }, []);
  // A new notification shows over every overlay open when it arrives.
  const ref = useRef<HTMLDivElement>(null);
  const shown = statuses.map(({ key, count }) => `${key}:${count}`).join();
  usePopover(ref, true, `${moves}|${shown}`);
  return (
    <Portal.Above element={getRootElement()}>
      <Flex.Box y className={CSS.B("notifications")} ref={ref} popover="manual">
        <Flex.Box y className={CSS.BE("notifications", "stack")}>
          {visible.map(({ status, Render }) => (
            <Render key={status.key} status={status} silence={silence} />
          ))}
        </Flex.Box>
        {overflow > 0 && (
          <Flex.Box
            x
            justify="end"
            gap="small"
            className={CSS.BE("notifications", "controls")}
          >
            <Button.Button
              size="small"
              variant="outlined"
              onClick={() => setExpanded(!showAll)}
            >
              {showAll ? "Show less" : `+${overflow} more`}
            </Button.Button>
            <Button.Button size="small" variant="outlined" onClick={silenceAll}>
              Clear all
            </Button.Button>
          </Flex.Box>
        )}
      </Flex.Box>
    </Portal.Above>
  );
};
