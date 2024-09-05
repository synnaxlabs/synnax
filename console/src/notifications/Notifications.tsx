// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/notifications/Notifications.css";

import { Status } from "@synnaxlabs/pluto";
import { Button } from "@synnaxlabs/pluto/button";
import { List } from "@synnaxlabs/pluto/list";
import { TimeSpan } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";

interface NotificationsProps {
  adapters?: NotificationAdapter[];
}

export interface SugaredNotification extends Status.NotificationSpec {
  actions?: ReactElement | Button.ButtonProps[];
  content?: ReactElement;
}

export type NotificationAdapter = (
  status: Status.NotificationSpec,
) => null | SugaredNotification;

const DEFAULT_EXPIRATION = TimeSpan.seconds(5000);

export const Notifications = ({ adapters }: NotificationsProps): ReactElement => {
  const { statuses, silence } = Status.useNotifications({
    expiration: DEFAULT_EXPIRATION,
  });
  const sugared = statuses.map((status) => {
    if (adapters == null || adapters.length === 0) return status;
    for (const adapter of adapters) {
      const result = adapter(status);
      if (result != null) return result;
    }
    return status;
  });
  return (
    <List.List<string, Status.NotificationSpec | SugaredNotification> data={sugared}>
      <List.Core<string, SugaredNotification>
        className={CSS(CSS.B("notifications"))}
        size="medium"
      >
        {({ entry }) => (
          <Status.Notification
            key={entry.key}
            status={entry}
            silence={silence}
            actions={entry.actions}
          >
            {entry.content}
          </Status.Notification>
        )}
      </List.Core>
    </List.List>
  );
};
