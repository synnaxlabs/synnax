// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/notifications/Feed.css";

import { Flex, Status } from "@synnaxlabs/pluto";
import { type FC, type ReactElement } from "react";
import { createPortal } from "react-dom";

import { CSS } from "@/platform/css";

export interface NotificationProps {
  status: Status.NotificationSpec & { details?: unknown };
  silence: (key: string) => void;
}

export interface Notification extends FC<NotificationProps> {
  match: (status: Status.NotificationSpec & { details?: unknown }) => boolean;
}

const Default: Notification = ({ status, silence }) => (
  <Status.Notification status={status} silence={silence} />
);
Default.match = () => true;

export const createSuppressed = (match: Notification["match"]): Notification => {
  const Suppressed: Notification = () => null;
  Suppressed.match = match;
  return Suppressed;
};

interface FeedProps {
  notifications: Notification[];
}

export const Feed = ({ notifications }: FeedProps): ReactElement => {
  const { statuses, silence } = Status.useNotifications();
  return createPortal(
    <Flex.Box y className={CSS.B("notifications")}>
      {statuses.map((status) => {
        const Match = notifications.find((n) => n.match(status)) ?? Default;
        return <Match key={status.key} status={status} silence={silence} />;
      })}
    </Flex.Box>,
    document.getElementById("root") as HTMLElement,
  );
};
