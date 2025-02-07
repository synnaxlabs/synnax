// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/notifications/Notifications.css";

import { type Button, List, Status } from "@synnaxlabs/pluto";
import { TimeSpan } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { createPortal } from "react-dom";

import { CSS } from "@/css";

export interface Sugared extends Status.NotificationSpec {
  actions?: ReactElement | Button.ButtonProps[];
  content?: ReactElement;
}

export interface Adapter {
  (status: Status.NotificationSpec, silence: (key: string) => void): null | Sugared;
}

const DEFAULT_EXPIRATION = TimeSpan.seconds(7);

interface NotificationsProps {
  adapters?: Adapter[];
}

export const Notifications = ({ adapters }: NotificationsProps): ReactElement => {
  const { statuses, silence } = Status.useNotifications({
    expiration: DEFAULT_EXPIRATION,
  });
  const sugared = statuses.map((status) => {
    if (adapters == null || adapters.length === 0) return status;
    for (const adapter of adapters) {
      const result = adapter(status, silence);
      if (result != null) return result;
    }
    return status;
  });
  return createPortal(
    <List.List<string, Status.NotificationSpec | Sugared> data={sugared}>
      <List.Core<string, Sugared> className={CSS(CSS.B("notifications"))} size="medium">
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
    </List.List>,
    document.getElementById("root") as HTMLElement,
  );
};
