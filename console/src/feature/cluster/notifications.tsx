// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Status, Synnax } from "@synnaxlabs/pluto";

import { Version } from "@/platform/version";
import { type Notifications } from "@/platform/notifications";

const isMismatch = (status: Notifications.NotificationProps["status"]): boolean => {
  const details = Synnax.statusDetailsSchema.safeParse(status.details);
  return details.success && details.data.type === Synnax.SERVER_VERSION_MISMATCH;
};

export const Notification: Notifications.Notification = ({ status, silence }) => {
  const details = Synnax.statusDetailsSchema.safeParse(status.details);
  const oldServer =
    details.success &&
    details.data.type === Synnax.SERVER_VERSION_MISMATCH &&
    details.data.oldServer;
  return (
    <Status.Notification
      status={status}
      silence={silence}
      actions={
        oldServer
          ? [
              <Button.Button
                key="update"
                variant="outlined"
                size="small"
                href="https://docs.synnaxlabs.com/reference/core/quick-start"
                target="_blank"
              >
                Update Core
              </Button.Button>,
            ]
          : [<Version.OpenUpdateDialogAction key="update" />]
      }
    />
  );
};
Notification.match = isMismatch;

export const NOTIFICATIONS: Notifications.Notification[] = [Notification];
