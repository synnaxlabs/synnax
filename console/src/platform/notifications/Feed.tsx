// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/notifications/Feed.css";

import { type status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { location, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { type FC, type ReactElement, useEffect, useState } from "react";
import { createPortal } from "react-dom";

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
  /** Marks a renderer that never displays, so the feed skips it when counting. */
  suppress?: boolean;
}

const Default: Notification = ({ status, silence }) => (
  <Status.Notification status={status} silence={silence} />
);
Default.match = () => true;

export const createSuppressed = (match: Matcher): Notification => {
  const Suppressed: Notification = () => null;
  Suppressed.match = match;
  Suppressed.suppress = true;
  return Suppressed;
};

export const matchVariants =
  (...variants: status.Variant[]): Matcher =>
  (stat: Status.NotificationSpec) =>
    variants.includes(stat.variant);

export const matchPrefix =
  (prefix: string): Matcher =>
  (stat: Status.NotificationSpec) =>
    stat.key.startsWith(prefix);

export const composeMatchers =
  (...matchers: Matcher[]): Matcher =>
  (stat: Status.NotificationSpec) =>
    matchers.every((m) => m(stat));

export const createSuppressRoutineForPrefix = (prefix: string): Notification =>
  createSuppressed(
    composeMatchers(
      matchPrefix(prefix),
      matchVariants("success", "loading", "disabled"),
    ),
  );

interface FeedProps {
  notifications: Notification[];
}

const VISIBLE_CAP = 4;

const resolve = (
  statuses: Status.NotificationSpec[],
  notifications: Notification[],
): { status: Status.NotificationSpec; Render: Notification }[] =>
  statuses
    .map((status) => ({
      status,
      Render: notifications.find((n) => n.match(status)) ?? Default,
    }))
    .filter(({ Render }) => Render.suppress !== true);

export const Feed = ({ notifications }: FeedProps): ReactElement => {
  const { statuses, silence, silenceAll } = Status.useNotifications();
  const [expanded, setExpanded] = useState(false);
  const rendered = resolve(statuses, notifications);
  const overflow = rendered.length - VISIBLE_CAP;
  const showAll = expanded && overflow > 0;
  useEffect(() => {
    if (overflow <= 0) setExpanded(false);
  }, [overflow]);
  const visible = showAll ? rendered : rendered.slice(0, VISIBLE_CAP);
  return createPortal(
    <Flex.Box y className={CSS.B("notifications")}>
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
    </Flex.Box>,
    document.getElementById("root") as HTMLElement,
  );
};

const INBOX_EXPIRATION = TimeSpan.days(1);

/**
 * A bell button that holds notifications in a dropdown instead of popping them up. A
 * dot on the bell marks notifications that arrived since the dropdown last opened.
 */
export const Inbox = ({ notifications }: FeedProps): ReactElement => {
  const { statuses, silence, silenceAll } = Status.useNotifications({
    expiration: INBOX_EXPIRATION,
  });
  const [visible, setVisible] = useState(false);
  const [lastSeen, setLastSeen] = useState(() => TimeStamp.now());
  const rendered = resolve(statuses, notifications).sort((a, b) =>
    Number(b.status.time.valueOf() - a.status.time.valueOf()),
  );
  const unread = rendered.filter(({ status }) => status.time.after(lastSeen));
  const unreadVariant = unread.some(({ status }) => status.variant === "error")
    ? "error"
    : unread.some(({ status }) => status.variant === "warning")
      ? "warning"
      : "info";
  useEffect(() => setLastSeen(TimeStamp.now()), [visible]);
  return (
    <Dialog.Frame
      visible={visible}
      onVisibleChange={setVisible}
      location={location.BOTTOM_RIGHT}
    >
      <Dialog.Trigger
        hideCaret
        size="medium"
        variant="text"
        tooltip="Notifications"
        tooltipLocation={location.BOTTOM_LEFT}
        className={CSS.cls(
          CSS.BE("inbox", "trigger"),
          unread.length > 0 && CSS.M("unread"),
          CSS.M(unreadVariant),
        )}
      >
        <Icon.Notification />
      </Dialog.Trigger>
      <Dialog.Dialog background={1} pack={false} className={CSS.B("inbox")}>
        <Flex.Box
          x
          justify="between"
          align="center"
          className={CSS.BE("inbox", "header")}
        >
          <Text.Text level="small" weight={500} color={10}>
            Notifications
          </Text.Text>
          {rendered.length > 0 && (
            <Button.Button size="small" variant="text" onClick={silenceAll}>
              Clear all
            </Button.Button>
          )}
        </Flex.Box>
        <Flex.Box y gap="small" className={CSS.BE("inbox", "list")}>
          {rendered.length === 0 ? (
            <Flex.Box
              align="center"
              justify="center"
              className={CSS.BE("inbox", "empty")}
            >
              <Text.Text level="small" color={9}>
                No notifications
              </Text.Text>
            </Flex.Box>
          ) : (
            rendered.map(({ status, Render }) => (
              <Render key={status.key} status={status} silence={silence} />
            ))
          )}
        </Flex.Box>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
