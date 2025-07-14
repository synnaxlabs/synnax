// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/notifications/Notifications.css";

import { Align, type Button, Status } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";
import { createPortal } from "react-dom";

import { CSS } from "@/css";

export interface Sugared extends Status.NotificationSpec {
  actions?: ReactElement | Button.ButtonProps[];
  content?: ReactElement;
}

export interface Adapter<D = undefined> {
  (status: Status.NotificationSpec<D>, silence: (key: string) => void): null | Sugared;
}

interface NotificationsProps {
  adapters: Adapter[];
}

export const Notifications = ({ adapters }: NotificationsProps): ReactElement => {
  const { statuses, silence } = Status.useNotifications();
  const sugared = statuses.map((status) => {
    for (const adapter of adapters) {
      const result = adapter(status, silence);
      if (result != null) return result;
    }
    return status;
  }) as Sugared[];
  return createPortal(
    <Align.Space y className={CSS.B("notifications")}>
      {sugared.map((status) => (
        <Status.Notification key={status.key} status={status} silence={silence}>
          {status.content}
        </Status.Notification>
      ))}
    </Align.Space>,
    document.getElementById("root") as HTMLElement,
  );
};
