// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Status, Synnax } from "@synnaxlabs/pluto";

import { type Notifications } from "@/platform/notifications";
import { Version } from "@/platform/version";

const isMismatch = (status: Notifications.NotificationProps["status"]): boolean => {
  const details = Synnax.statusDetailsSchema.safeParse(status.details);
  return details.success && details.data.type === Synnax.SERVER_VERSION_MISMATCH;
};

const UPDATE_CORE_ACTIONS = [
  <Button.Button
    key="update"
    variant="outlined"
    size="small"
    href="https://docs.synnaxlabs.com/reference/core/quick-start"
    target="_blank"
  >
    Update Core
  </Button.Button>,
];

const UPDATE_CONSOLE_ACTIONS = [<Version.OpenUpdateDialogAction key="update" />];

const Notification: Notifications.Notification = ({ status, silence }) => {
  const details = Synnax.statusDetailsSchema.safeParse(status.details);
  const oldServer =
    details.success &&
    details.data.type === Synnax.SERVER_VERSION_MISMATCH &&
    details.data.oldServer;
  return (
    <Status.Notification
      status={status}
      silence={silence}
      actions={oldServer ? UPDATE_CORE_ACTIONS : UPDATE_CONSOLE_ACTIONS}
    />
  );
};
Notification.match = isMismatch;

export const NOTIFICATIONS: Notifications.Notification[] = [Notification];
