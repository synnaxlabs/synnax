// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/notifications/Notifications.css";

import { type Button, Flex, Status } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";
import { createPortal } from "react-dom";

import { CSS } from "@/css";

export interface Sugared extends Status.NotificationSpec {
  actions?: ReactElement | Button.ButtonProps[];
  content?: ReactElement;
}

export interface Adapter<Details = never> {
  (
    status: Status.NotificationSpec<Details>,
    silence: (key: string) => void,
  ): null | Sugared;
}

interface NotificationsProps {
  adapters: Adapter[];
}

// Note: Hack to hide repeated device and rack success notifications.
const hideRackAndDeviceSuccesses = (status: Status.NotificationSpec) =>
  (status.variant !== "success" && status.variant !== "loading") ||
  (!status.key.startsWith("rack") &&
    !status.key.startsWith("device") &&
    !status.key.startsWith("task"));

export const Notifications = ({ adapters }: NotificationsProps): ReactElement => {
  const { statuses, silence } = Status.useNotifications();
  const sugared = statuses
    .map((status) => {
      for (const adapter of adapters) {
        const result = adapter(status, silence);
        if (result != null) return result;
      }
      return status;
    })
    .filter(hideRackAndDeviceSuccesses) as Sugared[];
  return createPortal(
    <Flex.Box y className={CSS.B("notifications")}>
      {sugared.map((status) => (
        <Status.Notification
          key={status.key}
          status={status}
          silence={silence}
          actions={status.actions}
        >
          {status.content}
        </Status.Notification>
      ))}
    </Flex.Box>,
    document.getElementById("root") as HTMLElement,
  );
};
