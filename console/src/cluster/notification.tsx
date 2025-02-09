// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Synnax } from "@synnaxlabs/pluto";

import { type Notifications } from "@/notifications";
import { Version } from "@/version";

export const versionOutdatedAdapter: Notifications.Adapter = (status) => {
  if (status.data == null) return null;
  if (status.data.type !== Synnax.SERVER_VERSION_MISMATCH) return null;
  const oldServer = status.data.oldServer as boolean;
  const nextStatus: Notifications.Sugared = { ...status };
  if (oldServer)
    nextStatus.actions = [
      <Button.Link
        key="update"
        variant="outlined"
        size="small"
        href="https://docs.synnaxlabs.com/reference/cluster/quick-start"
        target="_blank"
      >
        Update Cluster
      </Button.Link>,
    ];
  else nextStatus.actions = [<Version.OpenUpdateDialogAction key="update" />];
  return nextStatus;
};
