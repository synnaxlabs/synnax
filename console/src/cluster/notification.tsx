// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/pluto";

import {
  NotificationAdapter,
  SugaredNotification,
} from "@/notifications/Notifications";
import { OpenUpdateDialogAction } from "@/version/Updater";

export const versionOutdatedAdapter: NotificationAdapter = (status) => {
  if (status.data == null) return null;
  if (status.data.type !== "serverVersionMismatch") return null;
  const oldSever = status.data.oldServer as boolean;
  const nextStatus: SugaredNotification = { ...status };
  if (oldSever)
    nextStatus.actions = [
      <Button.Link
        key="update"
        variant="outlined"
        size="small"
        href={"https://docs.synnaxlabs.com/reference/cluster/quick-start"}
        target="_blank"
      >
        Update Cluster
      </Button.Link>,
    ];
  else nextStatus.actions = [<OpenUpdateDialogAction key="update" />];
  return nextStatus;
};
