// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Status } from "@synnaxlabs/lyra/status";

import { Notifications } from "@/platform/notifications";
import { STATUS_KEY_PREFIX } from "@/platform/version/check";
import { useInfoModal } from "@/platform/version/useInfoModal";

export const Notification: Notifications.Notification = ({ status, silence }) => (
  <Status.Notification
    status={status}
    silence={silence}
    actions={[<OpenUpdateDialogAction key="update" />]}
  />
);
Notification.match = Notifications.matchPrefix(STATUS_KEY_PREFIX);

const OpenUpdateDialogAction = () => {
  const openInfo = useInfoModal();
  return (
    <Button.Button variant="outlined" size="small" onClick={() => openInfo()}>
      Update
    </Button.Button>
  );
};
